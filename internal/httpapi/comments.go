package httpapi

import (
	"net/http"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	ticketID, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	items, err := h.comments.List(r.Context(), ticketID)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeList(w, items)
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	ticketID, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	var in domain.CommentInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	c, err := h.comments.Create(r.Context(), ticketID, in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) updateComment(w http.ResponseWriter, r *http.Request) {
	ticketID, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	commentID, err := pathID(r, "commentId")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	var in domain.CommentInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	c, err := h.comments.Update(r.Context(), ticketID, commentID, in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	ticketID, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	commentID, err := pathID(r, "commentId")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	if err := h.comments.Delete(r.Context(), ticketID, commentID); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
