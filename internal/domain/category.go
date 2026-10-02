package domain

import (
	"strings"
	"time"
)

const (
	categoryNameMaxLen = 100
	slaHoursMax        = 24 * 365
)

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SLAHours  int       `json:"sla_hours"`
	CreatedAt time.Time `json:"created_at"`
}

type CategoryInput struct {
	Name     string `json:"name"`
	SLAHours int    `json:"sla_hours"`
}

func (in *CategoryInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	details := map[string]string{}
	switch {
	case in.Name == "":
		details["name"] = "is required"
	case len([]rune(in.Name)) > categoryNameMaxLen:
		details["name"] = "must be at most 100 characters"
	}
	if in.SLAHours <= 0 || in.SLAHours > slaHoursMax {
		details["sla_hours"] = "must be between 1 and 8760"
	}
	if len(details) > 0 {
		return NewValidationError(details)
	}
	return nil
}
