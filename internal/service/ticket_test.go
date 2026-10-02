package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

const (
	networkID    = "11111111-1111-1111-1111-111111111111"
	laptopID     = "22222222-2222-2222-2222-222222222222"
	retiredID    = "33333333-3333-3333-3333-333333333333"
	missingID    = "44444444-4444-4444-4444-444444444444"
	newTicketID  = "55555555-5555-5555-5555-555555555555"
	closedTicket = "66666666-6666-6666-6666-666666666666"
)

var fixedNow = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

type testEnv struct {
	tickets  *fakeTickets
	comments *fakeComments
	svc      *TicketService
	commSvc  *CommentService
}

func newTestEnv() *testEnv {
	cats := &fakeCategories{items: map[string]domain.Category{
		networkID: {ID: networkID, Name: "Network", SLAHours: 4},
	}}
	eq := &fakeEquipment{items: map[string]domain.Equipment{
		laptopID:  {ID: laptopID, Status: domain.EquipmentActive},
		retiredID: {ID: retiredID, Status: domain.EquipmentRetired},
	}}
	tickets := &fakeTickets{items: map[string]domain.Ticket{
		newTicketID:  {ID: newTicketID, Title: "t", Status: domain.StatusNew, Priority: domain.PriorityLow, CategoryID: networkID},
		closedTicket: {ID: closedTicket, Title: "t", Status: domain.StatusClosed, Priority: domain.PriorityLow, CategoryID: networkID},
	}}
	comments := &fakeComments{}
	now := func() time.Time { return fixedNow }
	return &testEnv{
		tickets:  tickets,
		comments: comments,
		svc:      NewTicketService(tickets, cats, eq, now),
		commSvc:  NewCommentService(comments, tickets, now),
	}
}

func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	de, ok := domain.AsError(err)
	if !ok {
		t.Fatalf("expected domain error %s, got %v", code, err)
	}
	if de.Code != code {
		t.Fatalf("expected code %s, got %s (%s)", code, de.Code, de.Message)
	}
}

func TestCreateTicket_SetsNewStatusAndSLADeadline(t *testing.T) {
	env := newTestEnv()
	eqID := laptopID

	got, err := env.svc.Create(context.Background(), domain.CreateTicketInput{
		Title: "No internet", CategoryID: networkID, EquipmentID: &eqID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.StatusNew {
		t.Errorf("status: got %s, want NEW", got.Status)
	}
	if got.Priority != domain.PriorityMedium {
		t.Errorf("priority: got %s, want MEDIUM", got.Priority)
	}
	if want := fixedNow.Add(4 * time.Hour); !got.DueAt.Equal(want) {
		t.Errorf("due_at: got %s, want %s", got.DueAt, want)
	}
	if got.Overdue {
		t.Errorf("new ticket must not be overdue")
	}
}

func TestCreateTicket_RejectsRetiredEquipment(t *testing.T) {
	env := newTestEnv()
	eqID := retiredID
	_, err := env.svc.Create(context.Background(), domain.CreateTicketInput{
		Title: "Broken", CategoryID: networkID, EquipmentID: &eqID,
	})
	if !errors.Is(err, domain.ErrEquipmentRetired) {
		t.Fatalf("expected ErrEquipmentRetired, got %v", err)
	}
}

func TestCreateTicket_UnknownReferencesAreValidationErrors(t *testing.T) {
	env := newTestEnv()
	eqID := missingID

	_, err := env.svc.Create(context.Background(), domain.CreateTicketInput{Title: "x", CategoryID: missingID})
	assertCode(t, err, "VALIDATION_ERROR")

	_, err = env.svc.Create(context.Background(), domain.CreateTicketInput{Title: "x", CategoryID: networkID, EquipmentID: &eqID})
	assertCode(t, err, "VALIDATION_ERROR")
}

func TestChangeStatus_FollowsStateMachine(t *testing.T) {
	env := newTestEnv()
	ctx := context.Background()

	_, err := env.svc.ChangeStatus(ctx, newTicketID, domain.StatusResolved)
	assertCode(t, err, "INVALID_STATUS_TRANSITION")

	for _, to := range []domain.TicketStatus{domain.StatusInProgress, domain.StatusResolved, domain.StatusInProgress, domain.StatusResolved, domain.StatusClosed} {
		if _, err := env.svc.ChangeStatus(ctx, newTicketID, to); err != nil {
			t.Fatalf("transition to %s: %v", to, err)
		}
	}

	_, err = env.svc.ChangeStatus(ctx, newTicketID, domain.StatusInProgress)
	if !errors.Is(err, domain.ErrTicketClosed) {
		t.Fatalf("expected ErrTicketClosed, got %v", err)
	}
}

func TestChangeStatus_NewCanBeCancelled(t *testing.T) {
	env := newTestEnv()
	got, err := env.svc.ChangeStatus(context.Background(), newTicketID, domain.StatusClosed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.AllowedTransitions) != 0 {
		t.Errorf("closed ticket must have no transitions, got %v", got.AllowedTransitions)
	}
}

func TestUpdate_ClosedTicketIsImmutable(t *testing.T) {
	env := newTestEnv()
	_, err := env.svc.Update(context.Background(), closedTicket, domain.UpdateTicketInput{Title: "new", Priority: domain.PriorityHigh})
	if !errors.Is(err, domain.ErrTicketClosed) {
		t.Fatalf("expected ErrTicketClosed, got %v", err)
	}
}

func TestUpdate_ResolvedTicketIsEditable(t *testing.T) {
	env := newTestEnv()
	ctx := context.Background()
	env.svc.ChangeStatus(ctx, newTicketID, domain.StatusInProgress)
	env.svc.ChangeStatus(ctx, newTicketID, domain.StatusResolved)

	got, err := env.svc.Update(ctx, newTicketID, domain.UpdateTicketInput{Title: "edited", Priority: domain.PriorityCritical})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "edited" || got.Priority != domain.PriorityCritical {
		t.Errorf("ticket not updated: %+v", got.Ticket)
	}
}

func TestUpdate_KeepsAlreadyLinkedRetiredEquipment(t *testing.T) {
	env := newTestEnv()
	eqID := retiredID
	tk := env.tickets.items[newTicketID]
	tk.EquipmentID = &eqID
	env.tickets.items[newTicketID] = tk

	same := retiredID
	if _, err := env.svc.Update(context.Background(), newTicketID, domain.UpdateTicketInput{Title: "x", Priority: domain.PriorityLow, EquipmentID: &same}); err != nil {
		t.Fatalf("editing a ticket already linked to retired equipment must work: %v", err)
	}
}

func TestDelete_OnlyNewTickets(t *testing.T) {
	env := newTestEnv()
	ctx := context.Background()

	if err := env.svc.Delete(ctx, closedTicket); !errors.Is(err, domain.ErrTicketNotDeletable) {
		t.Fatalf("expected ErrTicketNotDeletable, got %v", err)
	}
	if err := env.svc.Delete(ctx, newTicketID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := env.svc.Get(ctx, newTicketID); err == nil {
		t.Fatalf("ticket must be deleted")
	}
}

func TestComments_ForbiddenOnClosedTicket(t *testing.T) {
	env := newTestEnv()
	ctx := context.Background()
	in := domain.CommentInput{Content: "hello"}

	if _, err := env.commSvc.Create(ctx, closedTicket, in); !errors.Is(err, domain.ErrTicketClosed) {
		t.Fatalf("expected ErrTicketClosed, got %v", err)
	}
	if _, err := env.commSvc.Create(ctx, newTicketID, in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.comments.created != 1 {
		t.Fatalf("expected exactly one comment created, got %d", env.comments.created)
	}
}
