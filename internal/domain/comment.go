package domain

import (
	"strings"
	"time"
)

type Comment struct {
	ID        string    `json:"id"`
	TicketID  string    `json:"ticket_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CommentInput struct {
	Content string `json:"content"`
}

func (in *CommentInput) Validate() error {
	in.Content = strings.TrimSpace(in.Content)
	details := map[string]string{}
	requireString(details, "content", in.Content, 5000)
	if len(details) > 0 {
		return NewValidationError(details)
	}
	return nil
}
