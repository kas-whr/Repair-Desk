package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const categoryColumns = `id, name, sla_hours, created_at`

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func scanCategory(row pgx.Row) (domain.Category, error) {
	var c domain.Category
	err := row.Scan(&c.ID, &c.Name, &c.SLAHours, &c.CreatedAt)
	return c, err
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.Query(ctx, `SELECT `+categoryColumns+` FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Category, error) {
		return scanCategory(row)
	})
}

func (r *CategoryRepository) Get(ctx context.Context, id string) (domain.Category, error) {
	c, err := scanCategory(r.db.QueryRow(ctx, `SELECT `+categoryColumns+` FROM categories WHERE id = $1`, id))
	return c, mapError(err, "category")
}

func (r *CategoryRepository) Create(ctx context.Context, in domain.CategoryInput, now time.Time) (domain.Category, error) {
	c, err := scanCategory(r.db.QueryRow(ctx,
		`INSERT INTO categories (name, sla_hours, created_at) VALUES ($1, $2, $3) RETURNING `+categoryColumns,
		in.Name, in.SLAHours, now))
	return c, mapError(err, "category")
}

func (r *CategoryRepository) Update(ctx context.Context, id string, in domain.CategoryInput) (domain.Category, error) {
	c, err := scanCategory(r.db.QueryRow(ctx,
		`UPDATE categories SET name = $2, sla_hours = $3 WHERE id = $1 RETURNING `+categoryColumns,
		id, in.Name, in.SLAHours))
	return c, mapError(err, "category")
}

func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewNotFoundError("category")
	}
	return nil
}
