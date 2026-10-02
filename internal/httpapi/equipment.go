package httpapi

import (
	"net/http"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

func (h *Handler) listEquipment(w http.ResponseWriter, r *http.Request) {
	f := domain.EquipmentFilter{Status: domain.EquipmentStatus(r.URL.Query().Get("status"))}
	items, err := h.equipment.List(r.Context(), f)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeList(w, items)
}

func (h *Handler) getEquipment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	e, err := h.equipment.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) createEquipment(w http.ResponseWriter, r *http.Request) {
	var in domain.EquipmentInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	e, err := h.equipment.Create(r.Context(), in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) updateEquipment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	var in domain.EquipmentInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	e, err := h.equipment.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) deleteEquipment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	if err := h.equipment.Delete(r.Context(), id); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
