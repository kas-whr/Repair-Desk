package service

import (
	"context"
	"time"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

// Repositories return *domain.Error for "not found" and unique-constraint conflicts.
// Conditional writes (update/delete guarded by ticket status) return
// domain.ErrConcurrentUpdate when no row matched the condition.

type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	Get(ctx context.Context, id string) (domain.Category, error)
	Create(ctx context.Context, in domain.CategoryInput, now time.Time) (domain.Category, error)
	Update(ctx context.Context, id string, in domain.CategoryInput) (domain.Category, error)
	Delete(ctx context.Context, id string) error
}

type EquipmentRepository interface {
	List(ctx context.Context, f domain.EquipmentFilter) ([]domain.Equipment, error)
	Get(ctx context.Context, id string) (domain.Equipment, error)
	Create(ctx context.Context, in domain.EquipmentInput) (domain.Equipment, error)
	Update(ctx context.Context, id string, in domain.EquipmentInput) (domain.Equipment, error)
	Delete(ctx context.Context, id string) error
}

type TicketRepository interface {
	List(ctx context.Context, f domain.TicketFilter) ([]domain.Ticket, int, error)
	Get(ctx context.Context, id string) (domain.Ticket, error)
	Create(ctx context.Context, t domain.Ticket) (domain.Ticket, error)
	// Update applies in only if the ticket is still in status expected.
	Update(ctx context.Context, id string, expected domain.TicketStatus, in domain.UpdateTicketInput, now time.Time) (domain.Ticket, error)
	// UpdateStatus moves the ticket from -> to only if it is still in status from.
	UpdateStatus(ctx context.Context, id string, from, to domain.TicketStatus, now time.Time) (domain.Ticket, error)
	// Delete removes the ticket only if it is still in status expected.
	Delete(ctx context.Context, id string, expected domain.TicketStatus) error
}

// CommentRepository writes are guarded: they affect nothing if the ticket is CLOSED.
type CommentRepository interface {
	ListByTicket(ctx context.Context, ticketID string) ([]domain.Comment, error)
	Get(ctx context.Context, ticketID, commentID string) (domain.Comment, error)
	Create(ctx context.Context, ticketID, content string, now time.Time) (domain.Comment, error)
	Update(ctx context.Context, ticketID, commentID, content string) (domain.Comment, error)
	Delete(ctx context.Context, ticketID, commentID string) error
}
