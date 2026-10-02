package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const ticketColumns = `id, title, description, status, priority, category_id, equipment_id, due_at, created_at, updated_at`

type TicketRepository struct {
	db *pgxpool.Pool
}

func NewTicketRepository(db *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{db: db}
}

func scanTicket(row pgx.Row) (domain.Ticket, error) {
	var t domain.Ticket
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.CategoryID, &t.EquipmentID, &t.DueAt, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// ticketFilterSQL keeps the overdue condition identical to domain.Ticket.IsOverdue.
const ticketFilterSQL = `
	WHERE ($1 = '' OR status = $1)
	  AND ($2 = '' OR priority = $2)
	  AND ($3::uuid IS NULL OR category_id = $3)
	  AND ($4::uuid IS NULL OR equipment_id = $4)
	  AND ($5::boolean IS NULL OR
	       $5 = (due_at < $6 AND status NOT IN ('RESOLVED', 'CLOSED')))`

func (r *TicketRepository) List(ctx context.Context, f domain.TicketFilter) ([]domain.Ticket, int, error) {
	args := []any{string(f.Status), string(f.Priority), nullIfEmpty(f.CategoryID), nullIfEmpty(f.EquipmentID), f.Overdue, f.Now}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM tickets`+ticketFilterSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+ticketColumns+` FROM tickets`+ticketFilterSQL+`
		 ORDER BY created_at DESC, id
		 LIMIT $7 OFFSET $8`,
		append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	tickets, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Ticket, error) {
		return scanTicket(row)
	})
	return tickets, total, err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *TicketRepository) Get(ctx context.Context, id string) (domain.Ticket, error) {
	t, err := scanTicket(r.db.QueryRow(ctx, `SELECT `+ticketColumns+` FROM tickets WHERE id = $1`, id))
	return t, mapError(err, "ticket")
}

func (r *TicketRepository) Create(ctx context.Context, t domain.Ticket) (domain.Ticket, error) {
	created, err := scanTicket(r.db.QueryRow(ctx,
		`INSERT INTO tickets (title, description, status, priority, category_id, equipment_id, due_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING `+ticketColumns,
		t.Title, t.Description, t.Status, t.Priority, t.CategoryID, t.EquipmentID, t.DueAt, t.CreatedAt, t.UpdatedAt))
	return created, mapError(err, "ticket")
}

func (r *TicketRepository) Update(ctx context.Context, id string, expected domain.TicketStatus, in domain.UpdateTicketInput, now time.Time) (domain.Ticket, error) {
	t, err := scanTicket(r.db.QueryRow(ctx,
		`UPDATE tickets
		 SET title = $3, description = $4, priority = $5, equipment_id = $6, updated_at = $7
		 WHERE id = $1 AND status = $2
		 RETURNING `+ticketColumns,
		id, expected, in.Title, in.Description, in.Priority, in.EquipmentID, now))
	return t, mapGuardedError(err, "ticket")
}

func (r *TicketRepository) UpdateStatus(ctx context.Context, id string, from, to domain.TicketStatus, now time.Time) (domain.Ticket, error) {
	t, err := scanTicket(r.db.QueryRow(ctx,
		`UPDATE tickets SET status = $3, updated_at = $4
		 WHERE id = $1 AND status = $2
		 RETURNING `+ticketColumns,
		id, from, to, now))
	return t, mapGuardedError(err, "ticket")
}

func (r *TicketRepository) Delete(ctx context.Context, id string, expected domain.TicketStatus) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tickets WHERE id = $1 AND status = $2`, id, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConcurrentUpdate
	}
	return nil
}
