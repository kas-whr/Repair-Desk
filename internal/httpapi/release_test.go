package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kas-whr/Repair-Desk/internal/buildinfo"
	"github.com/kas-whr/Repair-Desk/internal/metrics"
	dto "github.com/prometheus/client_model/go"
)

type releaseTestPinger struct{ err error }

func (p releaseTestPinger) Ping(context.Context) error { return p.err }

func TestReleaseProbes(t *testing.T) {
	for _, dbErr := range []error{nil, errors.New("database unavailable")} {
		h := NewRouter(Deps{
			DB:     releaseTestPinger{err: dbErr},
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		})
		for _, path := range []string{"/healthz", "/readyz"} {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			want := http.StatusOK
			if path == "/readyz" && dbErr != nil {
				want = http.StatusServiceUnavailable
			}
			if rec.Code != want {
				t.Fatalf("%s with DB error %v: got %d, want %d", path, dbErr, rec.Code, want)
			}
		}
	}
}

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

func TestReleaseTrafficAudience(t *testing.T) {
	h := NewRouter(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	for _, tc := range []struct{ header, audience string }{
		{"", "default"}, {"beta", "beta"}, {"unbounded-user-value", "default"},
	} {
		counter := metrics.ReleaseRequests.WithLabelValues(buildinfo.Version, "GET", "200", tc.audience)
		before := &dto.Metric{}
		if err := counter.Write(before); err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		r.Header.Set("X-User-Group", tc.header)
		h.ServeHTTP(httptest.NewRecorder(), r)
		after := &dto.Metric{}
		if err := counter.Write(after); err != nil {
			t.Fatal(err)
		}
		if after.GetCounter().GetValue()-before.GetCounter().GetValue() != 1 {
			t.Fatalf("header %q was not counted in audience %q", tc.header, tc.audience)
		}
	}
}
