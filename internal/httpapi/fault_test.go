package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithFault(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/categories", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	serve := func(f FaultConfig, path string) (*httptest.ResponseRecorder, *http.Request) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		withFault(f, mux, mux).ServeHTTP(rec, req)
		return rec, req
	}

	if rec, _ := serve(FaultConfig{}, "/api/v1/categories"); rec.Code != http.StatusOK {
		t.Fatalf("disabled: got %d, want 200", rec.Code)
	}

	rec, req := serve(FaultConfig{ErrorRate: 1}, "/api/v1/categories")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("error rate 1: got %d, want 500", rec.Code)
	}
	if req.Pattern != "GET /api/v1/categories" {
		t.Errorf("route must be set for metrics, got %q", req.Pattern)
	}

	if rec, _ := serve(FaultConfig{ErrorRate: 1}, "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("health checks must not be affected, got %d", rec.Code)
	}
}
