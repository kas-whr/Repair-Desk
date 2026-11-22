package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kas-whr/Repair-Desk/internal/buildinfo"
)

func TestReleaseIdentityAndShadowGuard(t *testing.T) {
	old := buildinfo.Version
	defer func() { buildinfo.Version = old }()
	for _, v := range []string{"v1.0.0", "v2.0.0"} {
		buildinfo.Version = v
		t.Setenv("SHADOW_READ_ONLY", "true")
		h := NewRouter(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if rec.Code != 200 || body["version"] != v || rec.Header().Get("X-App-Version") != v {
			t.Fatalf("bad release response: %s", rec.Body.String())
		}
		blocked := httptest.NewRecorder()
		h.ServeHTTP(blocked, httptest.NewRequest(http.MethodPost, "/api/v1/tickets", nil))
		if blocked.Code != 405 {
			t.Fatalf("shadow write reached handler: %d", blocked.Code)
		}
		if body["message"] == "" {
			t.Fatal("missing release message")
		}
	}
}
