package domain

import (
	"testing"
	"time"
)

func TestStatusTransitions(t *testing.T) {
	allowed := map[[2]TicketStatus]bool{
		{StatusNew, StatusInProgress}:      true,
		{StatusNew, StatusClosed}:          true,
		{StatusInProgress, StatusResolved}: true,
		{StatusResolved, StatusInProgress}: true,
		{StatusResolved, StatusClosed}:     true,
	}
	all := []TicketStatus{StatusNew, StatusInProgress, StatusResolved, StatusClosed}

	for _, from := range all {
		for _, to := range all {
			want := allowed[[2]TicketStatus{from, to}]
			if got := from.CanTransitionTo(to); got != want {
				t.Errorf("%s -> %s: got %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestIsOverdue(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		status TicketStatus
		due    time.Time
		want   bool
	}{
		{StatusNew, past, true},
		{StatusInProgress, past, true},
		{StatusResolved, past, false},
		{StatusClosed, past, false},
		{StatusNew, future, false},
	}
	for _, tt := range tests {
		ticket := Ticket{Status: tt.status, DueAt: tt.due}
		if got := ticket.IsOverdue(now); got != tt.want {
			t.Errorf("status=%s due=%s: got %v, want %v", tt.status, tt.due, got, tt.want)
		}
	}
}

func TestCreateTicketInputDefaultsAndValidation(t *testing.T) {
	in := CreateTicketInput{Title: "  Printer jam ", CategoryID: "8b0b3c1e-7a4f-4a52-9a43-3d6b8e2f6a10", EquipmentID: ptr("  ")}
	if err := in.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.Priority != PriorityMedium {
		t.Errorf("default priority: got %s, want MEDIUM", in.Priority)
	}
	if in.Title != "Printer jam" {
		t.Errorf("title not trimmed: %q", in.Title)
	}
	if in.EquipmentID != nil {
		t.Errorf("blank equipment_id must become nil")
	}

	bad := CreateTicketInput{Priority: "URGENT", CategoryID: "not-a-uuid"}
	err := bad.Validate()
	de, ok := AsError(err)
	if !ok || de.Kind != KindValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
	for _, field := range []string{"title", "priority", "category_id"} {
		if _, ok := de.Details[field]; !ok {
			t.Errorf("expected detail for %s, got %v", field, de.Details)
		}
	}
}

func ptr[T any](v T) *T { return &v }
