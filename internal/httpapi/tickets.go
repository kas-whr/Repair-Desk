package httpapi

import (
	"net/http"
	"strconv"

	"github.com/kas-whr/Repair-Desk/internal/domain"
)

func parseTicketFilter(r *http.Request) (domain.TicketFilter, error) {
	q := r.URL.Query()
	f := domain.TicketFilter{
		Status:      domain.TicketStatus(q.Get("status")),
		Priority:    domain.Priority(q.Get("priority")),
		CategoryID:  q.Get("category_id"),
		EquipmentID: q.Get("equipment_id"),
	}
	details := map[string]string{}

	if v := q.Get("overdue"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			details["overdue"] = "must be true or false"
		} else {
			f.Overdue = &b
		}
	}
	for name, dst := range map[string]*int{"limit": &f.Limit, "offset": &f.Offset} {
		if v := q.Get(name); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				details[name] = "must be an integer"
			} else {
				*dst = n
			}
		}
	}
	if len(details) > 0 {
		return f, domain.NewValidationError(details)
	}
	return f, nil
}

func (h *Handler) listTickets(w http.ResponseWriter, r *http.Request) {
	f, err := parseTicketFilter(r)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	page, err := h.tickets.List(r.Context(), f)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) getTicket(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	t, err := h.tickets.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) createTicket(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateTicketInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	t, err := h.tickets.Create(r.Context(), in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTicket(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	var in domain.UpdateTicketInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	t, err := h.tickets.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type changeStatusRequest struct {
	Status domain.TicketStatus `json:"status"`
}

func (h *Handler) changeTicketStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	var req changeStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	t, err := h.tickets.ChangeStatus(r.Context(), id, req.Status)
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTicket(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	if err := h.tickets.Delete(r.Context(), id); err != nil {
		writeError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
