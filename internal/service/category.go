package service

import (
	"context"
	"time"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

type CategoryService struct {
	repo CategoryRepository
	now  func() time.Time
}

func NewCategoryService(repo CategoryRepository, now func() time.Time) *CategoryService {
	return &CategoryService{repo: repo, now: now}
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}

func (s *CategoryService) Get(ctx context.Context, id string) (domain.Category, error) {
	return s.repo.Get(ctx, id)
}

func (s *CategoryService) Create(ctx context.Context, in domain.CategoryInput) (domain.Category, error) {
	if err := in.Validate(); err != nil {
		return domain.Category{}, err
	}
	return s.repo.Create(ctx, in, s.now().UTC())
}

// Update changes the category. Existing tickets keep their due_at: SLA is applied at creation time.
func (s *CategoryService) Update(ctx context.Context, id string, in domain.CategoryInput) (domain.Category, error) {
	if err := in.Validate(); err != nil {
		return domain.Category{}, err
	}
	return s.repo.Update(ctx, id, in)
}

// Delete removes the category together with its tickets (ON DELETE CASCADE).
func (s *CategoryService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
