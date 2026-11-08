package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/kas-whr/Repair-Desk/internal/metrics"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// withLogging writes one structured log line per request to stdout (12-factor XI).
func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", requestID)

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		route := r.Pattern
		if route == "" {
			route = "unmatched" // 404/405: keep label cardinality bounded
		}
		elapsed := time.Since(start)
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, route, strconv.Itoa(status)).Observe(elapsed.Seconds())

		// r.Pattern is set by ServeMux on this same request, e.g. "GET /api/v1/tickets/{id}":
		// unlike path it has no IDs, so it is suitable for grouping.
		logger.Log(r.Context(), level, "http request",
			"request_id", requestID,
			"method", r.Method,
			"route", r.Pattern,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", float64(elapsed.Microseconds())/1000,
		)
	})
}

func withRecover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				logger.ErrorContext(r.Context(), "panic", "panic", rec, "stack", string(debug.Stack()))
				writeErrorPayload(w, http.StatusInternalServerError, ErrorPayload{Code: "INTERNAL_ERROR", Message: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// headerOnlyWriter captures the status and headers the stdlib mux would send for 404/405.
type headerOnlyWriter struct {
	header http.Header
	status int
}

func (w *headerOnlyWriter) Header() http.Header         { return w.header }
func (w *headerOnlyWriter) Write(b []byte) (int, error) { return len(b), nil }
func (w *headerOnlyWriter) WriteHeader(status int)      { w.status = status }

// withJSONFallback replaces the mux's plain-text 404 and 405 with the API error format.
func withJSONFallback(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}

		rec := &headerOnlyWriter{header: http.Header{}}
		h.ServeHTTP(rec, r)
		if rec.status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", rec.header.Get("Allow"))
			writeErrorPayload(w, http.StatusMethodNotAllowed, ErrorPayload{Code: "METHOD_NOT_ALLOWED", Message: "method not allowed"})
			return
		}
		writeErrorPayload(w, http.StatusNotFound, ErrorPayload{Code: "ROUTE_NOT_FOUND", Message: "route not found"})
	})
}
