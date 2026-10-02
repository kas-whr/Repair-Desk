package domain

import (
	"errors"
	"fmt"
)

// Kind classifies an error so the transport layer can map it to a status code.
type Kind int

const (
	KindValidation Kind = iota + 1
	KindNotFound
	KindConflict
)

// Error is a business error with a stable machine-readable code.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Details map[string]string
}

func (e *Error) Error() string { return e.Message }

func NewValidationError(details map[string]string) *Error {
	return &Error{Kind: KindValidation, Code: "VALIDATION_ERROR", Message: "request validation failed", Details: details}
}

func NewNotFoundError(entity string) *Error {
	return &Error{Kind: KindNotFound, Code: "NOT_FOUND", Message: fmt.Sprintf("%s not found", entity)}
}

func NewConflictError(code, message string) *Error {
	return &Error{Kind: KindConflict, Code: code, Message: message}
}

// Errors returned by the business rules.
var (
	ErrEquipmentRetired = &Error{
		Kind:    KindValidation,
		Code:    "EQUIPMENT_RETIRED",
		Message: "cannot create or assign a ticket to retired equipment",
	}
	ErrTicketClosed = &Error{
		Kind:    KindConflict,
		Code:    "TICKET_CLOSED",
		Message: "closed ticket cannot be modified",
	}
	ErrTicketNotDeletable = &Error{
		Kind:    KindConflict,
		Code:    "TICKET_NOT_DELETABLE",
		Message: "only tickets in status NEW can be deleted",
	}
	ErrConcurrentUpdate = &Error{
		Kind:    KindConflict,
		Code:    "CONCURRENT_UPDATE",
		Message: "ticket was modified by another request, reload and try again",
	}
)

// AsError extracts a *domain.Error from an error chain.
func AsError(err error) (*Error, bool) {
	var de *Error
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}
