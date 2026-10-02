package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const commentColumns = `id, ticket_id, content, created_at`

type CommentRepository struct {
	db *pgxpool.Pool
}

func NewCommentRepository(db *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{db: db}
}

func scanComment(row pgx.Row) (domain.Comment, error) {
	var c domain.Comment
	err := row.Scan(&c.ID, &c.TicketID, &c.Content, &c.CreatedAt)
	return c, err
}

func (r *CommentRepository) ListByTicket(ctx context.Context, ticketID string) ([]domain.Comment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+commentColumns+` FROM comments WHERE ticket_id = $1 ORDER BY created_at, id`, ticketID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Comment, error) {
		return scanComment(row)
	})
}

func (r *CommentRepository) Get(ctx context.Context, ticketID, commentID string) (domain.Comment, error) {
	c, err := scanComment(r.db.QueryRow(ctx,
		`SELECT `+commentColumns+` FROM comments WHERE id = $1 AND ticket_id = $2`, commentID, ticketID))
	return c, mapError(err, "comment")
}

// Create inserts the comment only if the ticket exists and is not CLOSED.
func (r *CommentRepository) Create(ctx context.Context, ticketID, content string, now time.Time) (domain.Comment, error) {
	c, err := scanComment(r.db.QueryRow(ctx,
		`INSERT INTO comments (ticket_id, content, created_at)
		 SELECT id, $2, $3 FROM tickets WHERE id = $1 AND status <> 'CLOSED'
		 RETURNING `+commentColumns,
		ticketID, content, now))
	return c, mapGuardedError(err, "comment")
}

func (r *CommentRepository) Update(ctx context.Context, ticketID, commentID, content string) (domain.Comment, error) {
	c, err := scanComment(r.db.QueryRow(ctx,
		`UPDATE comments c SET content = $3
		 FROM tickets t
		 WHERE c.id = $1 AND c.ticket_id = $2 AND t.id = c.ticket_id AND t.status <> 'CLOSED'
		 RETURNING c.id, c.ticket_id, c.content, c.created_at`,
		commentID, ticketID, content))
	return c, mapGuardedError(err, "comment")
}

func (r *CommentRepository) Delete(ctx context.Context, ticketID, commentID string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM comments c USING tickets t
		 WHERE c.id = $1 AND c.ticket_id = $2 AND t.id = c.ticket_id AND t.status <> 'CLOSED'`,
		commentID, ticketID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConcurrentUpdate
	}
	return nil
}
