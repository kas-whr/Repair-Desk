package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/kas-whr/Repair-Desk/internal/service"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	categories *service.CategoryService
	equipment  *service.EquipmentService
	tickets    *service.TicketService
	comments   *service.CommentService
	db         Pinger
	logger     *slog.Logger
}

type Deps struct {
	Categories *service.CategoryService
	Equipment  *service.EquipmentService
	Tickets    *service.TicketService
	Comments   *service.CommentService
	DB         Pinger
	Logger     *slog.Logger
}

func NewRouter(d Deps) http.Handler {
	h := &Handler{
		categories: d.Categories,
		equipment:  d.Equipment,
		tickets:    d.Tickets,
		comments:   d.Comments,
		db:         d.DB,
		logger:     d.Logger,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)

	mux.HandleFunc("GET /api/v1/categories", h.listCategories)
	mux.HandleFunc("POST /api/v1/categories", h.createCategory)
	mux.HandleFunc("GET /api/v1/categories/{id}", h.getCategory)
	mux.HandleFunc("PUT /api/v1/categories/{id}", h.updateCategory)
	mux.HandleFunc("DELETE /api/v1/categories/{id}", h.deleteCategory)

	mux.HandleFunc("GET /api/v1/equipment", h.listEquipment)
	mux.HandleFunc("POST /api/v1/equipment", h.createEquipment)
	mux.HandleFunc("GET /api/v1/equipment/{id}", h.getEquipment)
	mux.HandleFunc("PUT /api/v1/equipment/{id}", h.updateEquipment)
	mux.HandleFunc("DELETE /api/v1/equipment/{id}", h.deleteEquipment)

	mux.HandleFunc("GET /api/v1/tickets", h.listTickets)
	mux.HandleFunc("POST /api/v1/tickets", h.createTicket)
	mux.HandleFunc("GET /api/v1/tickets/{id}", h.getTicket)
	mux.HandleFunc("PUT /api/v1/tickets/{id}", h.updateTicket)
	mux.HandleFunc("PATCH /api/v1/tickets/{id}/status", h.changeTicketStatus)
	mux.HandleFunc("DELETE /api/v1/tickets/{id}", h.deleteTicket)

	mux.HandleFunc("GET /api/v1/tickets/{id}/comments", h.listComments)
	mux.HandleFunc("POST /api/v1/tickets/{id}/comments", h.createComment)
	mux.HandleFunc("PUT /api/v1/tickets/{id}/comments/{commentId}", h.updateComment)
	mux.HandleFunc("DELETE /api/v1/tickets/{id}/comments/{commentId}", h.deleteComment)

	return withRecover(d.Logger, withLogging(d.Logger, withJSONFallback(mux)))
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		h.logger.WarnContext(r.Context(), "readiness check failed", "error", err)
		writeErrorPayload(w, http.StatusServiceUnavailable, ErrorPayload{Code: "NOT_READY", Message: "database is unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
