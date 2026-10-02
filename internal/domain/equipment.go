package domain

import "strings"

type EquipmentStatus string

const (
	EquipmentActive  EquipmentStatus = "ACTIVE"
	EquipmentBroken  EquipmentStatus = "BROKEN"
	EquipmentRetired EquipmentStatus = "RETIRED"
)

func (s EquipmentStatus) Valid() bool {
	switch s {
	case EquipmentActive, EquipmentBroken, EquipmentRetired:
		return true
	}
	return false
}

type Equipment struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	InventoryNumber string          `json:"inventory_number"`
	Location        string          `json:"location"`
	Status          EquipmentStatus `json:"status"`
}

type EquipmentInput struct {
	Name            string          `json:"name"`
	InventoryNumber string          `json:"inventory_number"`
	Location        string          `json:"location"`
	Status          EquipmentStatus `json:"status"`
}

type EquipmentFilter struct {
	Status EquipmentStatus
}

func (in *EquipmentInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	in.InventoryNumber = strings.TrimSpace(in.InventoryNumber)
	in.Location = strings.TrimSpace(in.Location)
	if in.Status == "" {
		in.Status = EquipmentActive
	}

	details := map[string]string{}
	requireString(details, "name", in.Name, 255)
	requireString(details, "inventory_number", in.InventoryNumber, 100)
	requireString(details, "location", in.Location, 255)
	if !in.Status.Valid() {
		details["status"] = "must be one of ACTIVE, BROKEN, RETIRED"
	}
	if len(details) > 0 {
		return NewValidationError(details)
	}
	return nil
}

func requireString(details map[string]string, field, value string, maxLen int) {
	switch {
	case value == "":
		details[field] = "is required"
	case len([]rune(value)) > maxLen:
		details[field] = "is too long"
	}
}
