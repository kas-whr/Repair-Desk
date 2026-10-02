package service

import (
	"context"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

type EquipmentService struct {
	repo EquipmentRepository
}

func NewEquipmentService(repo EquipmentRepository) *EquipmentService {
	return &EquipmentService{repo: repo}
}

func (s *EquipmentService) List(ctx context.Context, f domain.EquipmentFilter) ([]domain.Equipment, error) {
	if f.Status != "" && !f.Status.Valid() {
		return nil, domain.NewValidationError(map[string]string{"status": "must be one of ACTIVE, BROKEN, RETIRED"})
	}
	return s.repo.List(ctx, f)
}

func (s *EquipmentService) Get(ctx context.Context, id string) (domain.Equipment, error) {
	return s.repo.Get(ctx, id)
}

func (s *EquipmentService) Create(ctx context.Context, in domain.EquipmentInput) (domain.Equipment, error) {
	if err := in.Validate(); err != nil {
		return domain.Equipment{}, err
	}
	return s.repo.Create(ctx, in)
}

func (s *EquipmentService) Update(ctx context.Context, id string, in domain.EquipmentInput) (domain.Equipment, error) {
	if err := in.Validate(); err != nil {
		return domain.Equipment{}, err
	}
	return s.repo.Update(ctx, id, in)
}

// Delete removes the equipment; its tickets keep existing with equipment_id = NULL (ON DELETE SET NULL).
func (s *EquipmentService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
