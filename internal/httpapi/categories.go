package httpapi

import (
	"net/http"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	items, err := h.categories.List(r.Context())
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeList(w, items)
}

func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	c, err := h.categories.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var in domain.CategoryInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	c, err := h.categories.Create(r.Context(), in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	var in domain.CategoryInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	c, err := h.categories.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	if err := h.categories.Delete(r.Context(), id); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
