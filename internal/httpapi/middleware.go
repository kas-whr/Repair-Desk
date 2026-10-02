package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
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

		logger.InfoContext(r.Context(), "http request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
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
