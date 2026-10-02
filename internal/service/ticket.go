package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const (
	defaultTicketLimit = 50
	maxTicketLimit     = 200
)

// TicketView is a ticket with values derived at read time.
type TicketView struct {
	domain.Ticket
	Overdue            bool                  `json:"overdue"`
	AllowedTransitions []domain.TicketStatus `json:"allowed_transitions"`
}

type TicketPage struct {
	Items  []TicketView `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type TicketService struct {
	tickets    TicketRepository
	categories CategoryRepository
	equipment  EquipmentRepository
	now        func() time.Time
}

func NewTicketService(tickets TicketRepository, categories CategoryRepository, equipment EquipmentRepository, now func() time.Time) *TicketService {
	return &TicketService{tickets: tickets, categories: categories, equipment: equipment, now: now}
}

func (s *TicketService) view(t domain.Ticket) TicketView {
	return TicketView{
		Ticket:             t,
		Overdue:            t.IsOverdue(s.now()),
		AllowedTransitions: t.Status.AllowedTransitions(),
	}
}

func (s *TicketService) List(ctx context.Context, f domain.TicketFilter) (TicketPage, error) {
	details := map[string]string{}
	if f.Status != "" && !f.Status.Valid() {
		details["status"] = "must be one of NEW, IN_PROGRESS, RESOLVED, CLOSED"
	}
	if f.Priority != "" && !f.Priority.Valid() {
		details["priority"] = "must be one of LOW, MEDIUM, HIGH, CRITICAL"
	}
	if f.CategoryID != "" && !domain.IsValidID(f.CategoryID) {
		details["category_id"] = "must be a valid UUID"
	}
	if f.EquipmentID != "" && !domain.IsValidID(f.EquipmentID) {
		details["equipment_id"] = "must be a valid UUID"
	}
	if f.Limit < 0 || f.Limit > maxTicketLimit {
		details["limit"] = fmt.Sprintf("must be between 1 and %d", maxTicketLimit)
	}
	if f.Offset < 0 {
		details["offset"] = "must not be negative"
	}
	if len(details) > 0 {
		return TicketPage{}, domain.NewValidationError(details)
	}
	if f.Limit == 0 {
		f.Limit = defaultTicketLimit
	}
	f.Now = s.now()

	tickets, total, err := s.tickets.List(ctx, f)
	if err != nil {
		return TicketPage{}, err
	}
	items := make([]TicketView, 0, len(tickets))
	for _, t := range tickets {
		items = append(items, s.view(t))
	}
	return TicketPage{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

func (s *TicketService) Get(ctx context.Context, id string) (TicketView, error) {
	t, err := s.tickets.Get(ctx, id)
	if err != nil {
		return TicketView{}, err
	}
	return s.view(t), nil
}

// Create implements rules 1 and 3: status NEW, due_at = now + category SLA,
// retired equipment is rejected.
func (s *TicketService) Create(ctx context.Context, in domain.CreateTicketInput) (TicketView, error) {
	if err := in.Validate(); err != nil {
		return TicketView{}, err
	}

	category, err := s.categories.Get(ctx, in.CategoryID)
	if err != nil {
		return TicketView{}, referenceError(err, "category_id", "category")
	}
	if in.EquipmentID != nil {
		if err := s.checkEquipmentAssignable(ctx, *in.EquipmentID); err != nil {
			return TicketView{}, err
		}
	}

	now := s.now().UTC()
	created, err := s.tickets.Create(ctx, domain.Ticket{
		Title:       in.Title,
		Description: in.Description,
		Status:      domain.StatusNew,
		Priority:    in.Priority,
		CategoryID:  category.ID,
		EquipmentID: in.EquipmentID,
		DueAt:       now.Add(time.Duration(category.SLAHours) * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return TicketView{}, err
	}
	return s.view(created), nil
}

// Update edits a ticket that is not CLOSED. Category and status cannot be changed here.
func (s *TicketService) Update(ctx context.Context, id string, in domain.UpdateTicketInput) (TicketView, error) {
	if err := in.Validate(); err != nil {
		return TicketView{}, err
	}

	current, err := s.tickets.Get(ctx, id)
	if err != nil {
		return TicketView{}, err
	}
	if current.Status == domain.StatusClosed {
		return TicketView{}, domain.ErrTicketClosed
	}
	// Re-check only a newly assigned equipment: a ticket already linked to
	// equipment that was retired later must stay editable.
	if in.EquipmentID != nil && !sameID(current.EquipmentID, in.EquipmentID) {
		if err := s.checkEquipmentAssignable(ctx, *in.EquipmentID); err != nil {
			return TicketView{}, err
		}
	}

	updated, err := s.tickets.Update(ctx, id, current.Status, in, s.now().UTC())
	if err != nil {
		return TicketView{}, err
	}
	return s.view(updated), nil
}

// ChangeStatus implements rule 2: only transitions from the state machine are allowed.
func (s *TicketService) ChangeStatus(ctx context.Context, id string, to domain.TicketStatus) (TicketView, error) {
	if !to.Valid() {
		return TicketView{}, domain.NewValidationError(map[string]string{
			"status": "must be one of NEW, IN_PROGRESS, RESOLVED, CLOSED",
		})
	}

	current, err := s.tickets.Get(ctx, id)
	if err != nil {
		return TicketView{}, err
	}
	if current.Status == domain.StatusClosed {
		return TicketView{}, domain.ErrTicketClosed
	}
	if !current.Status.CanTransitionTo(to) {
		return TicketView{}, domain.NewConflictError(
			"INVALID_STATUS_TRANSITION",
			fmt.Sprintf("cannot change status from %s to %s", current.Status, to),
		)
	}

	updated, err := s.tickets.UpdateStatus(ctx, id, current.Status, to, s.now().UTC())
	if err != nil {
		return TicketView{}, err
	}
	return s.view(updated), nil
}

// Delete implements rule 5: only NEW tickets can be deleted.
func (s *TicketService) Delete(ctx context.Context, id string) error {
	current, err := s.tickets.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.Status != domain.StatusNew {
		return domain.ErrTicketNotDeletable
	}
	return s.tickets.Delete(ctx, id, domain.StatusNew)
}

func (s *TicketService) checkEquipmentAssignable(ctx context.Context, id string) error {
	eq, err := s.equipment.Get(ctx, id)
	if err != nil {
		return referenceError(err, "equipment_id", "equipment")
	}
	if eq.Status == domain.EquipmentRetired {
		return domain.ErrEquipmentRetired
	}
	return nil
}

// referenceError turns "not found" of a referenced entity into a validation error:
// the request body is wrong, not the URL.
func referenceError(err error, field, entity string) error {
	if de, ok := domain.AsError(err); ok && de.Kind == domain.KindNotFound {
		return domain.NewValidationError(map[string]string{field: entity + " does not exist"})
	}
	return err
}

func sameID(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
