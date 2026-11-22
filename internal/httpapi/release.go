package httpapi

import (
	"net/http"
	"os"
	"strings"

	"github.com/kas-whr/Repair-Desk/internal/buildinfo"
)

func version(w http.ResponseWriter, _ *http.Request) {
	message := "Repair Desk stable API"
	if strings.HasPrefix(buildinfo.Version, "v2.") {
		message = "Repair Desk API: welcome to release 2"
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"version": buildinfo.Version,
		"pod":     os.Getenv("HOSTNAME"),
		"message": message,
	})
}

// Shadow instances provide a second defence against mirrored writes.
func withReadOnly(enabled bool, next http.Handler) http.Handler {
	if !enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			writeErrorPayload(w, http.StatusMethodNotAllowed, ErrorPayload{Code: "SHADOW_READ_ONLY", Message: "shadow instance does not accept writes"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
