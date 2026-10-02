package service

import (
	"context"
	"time"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

type CommentService struct {
	comments CommentRepository
	tickets  TicketRepository
	now      func() time.Time
}

func NewCommentService(comments CommentRepository, tickets TicketRepository, now func() time.Time) *CommentService {
	return &CommentService{comments: comments, tickets: tickets, now: now}
}

func (s *CommentService) List(ctx context.Context, ticketID string) ([]domain.Comment, error) {
	if _, err := s.tickets.Get(ctx, ticketID); err != nil {
		return nil, err
	}
	return s.comments.ListByTicket(ctx, ticketID)
}

func (s *CommentService) Create(ctx context.Context, ticketID string, in domain.CommentInput) (domain.Comment, error) {
	if err := in.Validate(); err != nil {
		return domain.Comment{}, err
	}
	if err := s.ensureTicketOpen(ctx, ticketID); err != nil {
		return domain.Comment{}, err
	}
	return s.comments.Create(ctx, ticketID, in.Content, s.now().UTC())
}

func (s *CommentService) Update(ctx context.Context, ticketID, commentID string, in domain.CommentInput) (domain.Comment, error) {
	if err := in.Validate(); err != nil {
		return domain.Comment{}, err
	}
	if err := s.ensureTicketOpen(ctx, ticketID); err != nil {
		return domain.Comment{}, err
	}
	if _, err := s.comments.Get(ctx, ticketID, commentID); err != nil {
		return domain.Comment{}, err
	}
	return s.comments.Update(ctx, ticketID, commentID, in.Content)
}

func (s *CommentService) Delete(ctx context.Context, ticketID, commentID string) error {
	if err := s.ensureTicketOpen(ctx, ticketID); err != nil {
		return err
	}
	if _, err := s.comments.Get(ctx, ticketID, commentID); err != nil {
		return err
	}
	return s.comments.Delete(ctx, ticketID, commentID)
}

// ensureTicketOpen enforces that a CLOSED ticket is fully immutable, comments included.
func (s *CommentService) ensureTicketOpen(ctx context.Context, ticketID string) error {
	t, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		return err
	}
	if t.Status == domain.StatusClosed {
		return domain.ErrTicketClosed
	}
	return nil
}
