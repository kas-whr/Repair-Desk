package domain

import (
	"strings"
	"time"
)

type TicketStatus string

const (
	StatusNew        TicketStatus = "NEW"
	StatusInProgress TicketStatus = "IN_PROGRESS"
	StatusResolved   TicketStatus = "RESOLVED"
	StatusClosed     TicketStatus = "CLOSED"
)

func (s TicketStatus) Valid() bool {
	_, ok := transitions[s]
	return ok
}

// transitions is the status state machine (docs/status.png).
var transitions = map[TicketStatus][]TicketStatus{
	StatusNew:        {StatusInProgress, StatusClosed},
	StatusInProgress: {StatusResolved},
	StatusResolved:   {StatusInProgress, StatusClosed},
	StatusClosed:     {},
}

// AllowedTransitions returns statuses reachable from s in one step.
func (s TicketStatus) AllowedTransitions() []TicketStatus {
	return append([]TicketStatus{}, transitions[s]...)
}

func (s TicketStatus) CanTransitionTo(to TicketStatus) bool {
	for _, t := range transitions[s] {
		if t == to {
			return true
		}
	}
	return false
}

type Priority string

const (
	PriorityLow      Priority = "LOW"
	PriorityMedium   Priority = "MEDIUM"
	PriorityHigh     Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	}
	return false
}

type Ticket struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	Priority    Priority     `json:"priority"`
	CategoryID  string       `json:"category_id"`
	EquipmentID *string      `json:"equipment_id"`
	DueAt       time.Time    `json:"due_at"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// IsOverdue implements rule 4: the deadline has passed and the ticket is not resolved or closed.
func (t *Ticket) IsOverdue(now time.Time) bool {
	return t.DueAt.Before(now) && t.Status != StatusResolved && t.Status != StatusClosed
}

type CreateTicketInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	CategoryID  string   `json:"category_id"`
	EquipmentID *string  `json:"equipment_id"`
}

func (in *CreateTicketInput) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.CategoryID = strings.TrimSpace(in.CategoryID)
	in.EquipmentID = normalizeOptionalID(in.EquipmentID)
	if in.Priority == "" {
		in.Priority = PriorityMedium
	}

	details := map[string]string{}
	validateTicketFields(details, in.Title, in.Description, in.Priority, in.EquipmentID)
	if in.CategoryID == "" {
		details["category_id"] = "is required"
	} else if !IsValidID(in.CategoryID) {
		details["category_id"] = "must be a valid UUID"
	}
	if len(details) > 0 {
		return NewValidationError(details)
	}
	return nil
}

// UpdateTicketInput holds the editable fields. Category and status are not editable here:
// category is fixed at creation, status changes only through transitions.
type UpdateTicketInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	EquipmentID *string  `json:"equipment_id"`
}

func (in *UpdateTicketInput) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.EquipmentID = normalizeOptionalID(in.EquipmentID)

	details := map[string]string{}
	validateTicketFields(details, in.Title, in.Description, in.Priority, in.EquipmentID)
	if len(details) > 0 {
		return NewValidationError(details)
	}
	return nil
}

func validateTicketFields(details map[string]string, title, description string, priority Priority, equipmentID *string) {
	requireString(details, "title", title, 255)
	if len([]rune(description)) > 10000 {
		details["description"] = "is too long"
	}
	if !priority.Valid() {
		details["priority"] = "must be one of LOW, MEDIUM, HIGH, CRITICAL"
	}
	if equipmentID != nil && !IsValidID(*equipmentID) {
		details["equipment_id"] = "must be a valid UUID"
	}
}

func normalizeOptionalID(id *string) *string {
	if id == nil {
		return nil
	}
	v := strings.TrimSpace(*id)
	if v == "" {
		return nil
	}
	return &v
}

type TicketFilter struct {
	Status      TicketStatus
	Priority    Priority
	CategoryID  string
	EquipmentID string
	Overdue     *bool
	Now         time.Time
	Limit       int
	Offset      int
}
