package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// mapError converts driver errors into domain errors where it makes sense.
func mapError(err error, entity string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NewNotFoundError(entity)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case codeUniqueViolation:
			return domain.NewConflictError("ALREADY_EXISTS", fmt.Sprintf("%s with the same %s already exists", entity, uniqueField(pgErr.ConstraintName)))
		case codeForeignKeyViolation:
			return domain.NewValidationError(map[string]string{foreignKeyField(pgErr.ConstraintName): "referenced entity does not exist"})
		}
	}
	return err
}

func uniqueField(constraint string) string {
	switch constraint {
	case "categories_name_key":
		return "name"
	case "equipment_inventory_number_key":
		return "inventory_number"
	}
	return "key"
}

func foreignKeyField(constraint string) string {
	switch constraint {
	case "tickets_category_id_fkey":
		return "category_id"
	case "tickets_equipment_id_fkey":
		return "equipment_id"
	case "comments_ticket_id_fkey":
		return "ticket_id"
	}
	return "reference"
}

// mapGuardedError is used for writes guarded by a WHERE condition on ticket status:
// no rows means the ticket changed between the read and the write.
func mapGuardedError(err error, entity string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrConcurrentUpdate
	}
	return mapError(err, entity)
}
