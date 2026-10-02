package domain

import "github.com/google/uuid"

// IsValidID accepts only the canonical 36-char UUID form, the one PostgreSQL returns.
func IsValidID(id string) bool {
	if len(id) != 36 {
		return false
	}
	_, err := uuid.Parse(id)
	return err == nil
}
