package httpapi

import (
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// FaultConfig injects failures into /api/* requests to demonstrate SLI/SLO changes
// (chaos testing). Zero values (the default) disable injection completely.
type FaultConfig struct {
	ErrorRate float64       // share of requests answered with 500, 0..1
	Delay     time.Duration // added to every request
}

func (f FaultConfig) enabled() bool { return f.ErrorRate > 0 || f.Delay > 0 }

func withFault(f FaultConfig, mux *http.ServeMux, next http.Handler) http.Handler {
	if !f.enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		// Set the route before short-circuiting, so logs and metrics still group by route
		r.Pattern = pattern

		if f.Delay > 0 {
			select {
			case <-time.After(f.Delay):
			case <-r.Context().Done():
				return
			}
		}
		if f.ErrorRate > 0 && rand.Float64() < f.ErrorRate {
			writeErrorPayload(w, http.StatusInternalServerError, ErrorPayload{
				Code:    "INJECTED_FAULT",
				Message: "injected fault (FAULT_ERROR_RATE)",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
