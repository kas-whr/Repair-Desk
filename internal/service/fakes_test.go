package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

// In-memory repositories for service tests. They implement only what the tests need.

type fakeCategories struct{ items map[string]domain.Category }

func (f *fakeCategories) List(context.Context) ([]domain.Category, error) { return nil, nil }
func (f *fakeCategories) Get(_ context.Context, id string) (domain.Category, error) {
	c, ok := f.items[id]
	if !ok {
		return c, domain.NewNotFoundError("category")
	}
	return c, nil
}
func (f *fakeCategories) Create(context.Context, domain.CategoryInput, time.Time) (domain.Category, error) {
	return domain.Category{}, nil
}
func (f *fakeCategories) Update(context.Context, string, domain.CategoryInput) (domain.Category, error) {
	return domain.Category{}, nil
}
func (f *fakeCategories) Delete(context.Context, string) error { return nil }

type fakeEquipment struct{ items map[string]domain.Equipment }

func (f *fakeEquipment) List(context.Context, domain.EquipmentFilter) ([]domain.Equipment, error) {
	return nil, nil
}
func (f *fakeEquipment) Get(_ context.Context, id string) (domain.Equipment, error) {
	e, ok := f.items[id]
	if !ok {
		return e, domain.NewNotFoundError("equipment")
	}
	return e, nil
}
func (f *fakeEquipment) Create(context.Context, domain.EquipmentInput) (domain.Equipment, error) {
	return domain.Equipment{}, nil
}
func (f *fakeEquipment) Update(context.Context, string, domain.EquipmentInput) (domain.Equipment, error) {
	return domain.Equipment{}, nil
}
func (f *fakeEquipment) Delete(context.Context, string) error { return nil }

type fakeTickets struct{ items map[string]domain.Ticket }

func (f *fakeTickets) List(context.Context, domain.TicketFilter) ([]domain.Ticket, int, error) {
	return nil, 0, nil
}
func (f *fakeTickets) Get(_ context.Context, id string) (domain.Ticket, error) {
	t, ok := f.items[id]
	if !ok {
		return t, domain.NewNotFoundError("ticket")
	}
	return t, nil
}
func (f *fakeTickets) Create(_ context.Context, t domain.Ticket) (domain.Ticket, error) {
	t.ID = uuid.NewString()
	f.items[t.ID] = t
	return t, nil
}
func (f *fakeTickets) Update(_ context.Context, id string, expected domain.TicketStatus, in domain.UpdateTicketInput, now time.Time) (domain.Ticket, error) {
	t, ok := f.items[id]
	if !ok || t.Status != expected {
		return t, domain.ErrConcurrentUpdate
	}
	t.Title, t.Description, t.Priority, t.EquipmentID, t.UpdatedAt = in.Title, in.Description, in.Priority, in.EquipmentID, now
	f.items[id] = t
	return t, nil
}
func (f *fakeTickets) UpdateStatus(_ context.Context, id string, from, to domain.TicketStatus, now time.Time) (domain.Ticket, error) {
	t, ok := f.items[id]
	if !ok || t.Status != from {
		return t, domain.ErrConcurrentUpdate
	}
	t.Status, t.UpdatedAt = to, now
	f.items[id] = t
	return t, nil
}
func (f *fakeTickets) Delete(_ context.Context, id string, expected domain.TicketStatus) error {
	t, ok := f.items[id]
	if !ok || t.Status != expected {
		return domain.ErrConcurrentUpdate
	}
	delete(f.items, id)
	return nil
}

type fakeComments struct{ created int }

func (f *fakeComments) ListByTicket(context.Context, string) ([]domain.Comment, error) {
	return nil, nil
}
func (f *fakeComments) Get(context.Context, string, string) (domain.Comment, error) {
	return domain.Comment{}, nil
}
func (f *fakeComments) Create(_ context.Context, ticketID, content string, now time.Time) (domain.Comment, error) {
	f.created++
	return domain.Comment{ID: uuid.NewString(), TicketID: ticketID, Content: content, CreatedAt: now}, nil
}
func (f *fakeComments) Update(context.Context, string, string, string) (domain.Comment, error) {
	return domain.Comment{}, nil
}
func (f *fakeComments) Delete(context.Context, string, string) error { return nil }
