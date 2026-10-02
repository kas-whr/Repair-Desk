package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const equipmentColumns = `id, name, inventory_number, location, status`

type EquipmentRepository struct {
	db *pgxpool.Pool
}

func NewEquipmentRepository(db *pgxpool.Pool) *EquipmentRepository {
	return &EquipmentRepository{db: db}
}

func scanEquipment(row pgx.Row) (domain.Equipment, error) {
	var e domain.Equipment
	err := row.Scan(&e.ID, &e.Name, &e.InventoryNumber, &e.Location, &e.Status)
	return e, err
}

func (r *EquipmentRepository) List(ctx context.Context, f domain.EquipmentFilter) ([]domain.Equipment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+equipmentColumns+` FROM equipment
		 WHERE ($1 = '' OR status = $1)
		 ORDER BY name, inventory_number`,
		string(f.Status))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Equipment, error) {
		return scanEquipment(row)
	})
}

func (r *EquipmentRepository) Get(ctx context.Context, id string) (domain.Equipment, error) {
	e, err := scanEquipment(r.db.QueryRow(ctx, `SELECT `+equipmentColumns+` FROM equipment WHERE id = $1`, id))
	return e, mapError(err, "equipment")
}

func (r *EquipmentRepository) Create(ctx context.Context, in domain.EquipmentInput) (domain.Equipment, error) {
	e, err := scanEquipment(r.db.QueryRow(ctx,
		`INSERT INTO equipment (name, inventory_number, location, status)
		 VALUES ($1, $2, $3, $4) RETURNING `+equipmentColumns,
		in.Name, in.InventoryNumber, in.Location, in.Status))
	return e, mapError(err, "equipment")
}

func (r *EquipmentRepository) Update(ctx context.Context, id string, in domain.EquipmentInput) (domain.Equipment, error) {
	e, err := scanEquipment(r.db.QueryRow(ctx,
		`UPDATE equipment SET name = $2, inventory_number = $3, location = $4, status = $5
		 WHERE id = $1 RETURNING `+equipmentColumns,
		id, in.Name, in.InventoryNumber, in.Location, in.Status))
	return e, mapError(err, "equipment")
}

func (r *EquipmentRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM equipment WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewNotFoundError("equipment")
	}
	return nil
}
