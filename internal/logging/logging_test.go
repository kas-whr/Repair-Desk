package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNew_UsesCommonSchema(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelInfo, "repair-desk-api").Warn("something happened", "status", 409)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log line is not JSON: %v (%s)", err, buf.String())
	}
	want := map[string]any{
		"level":        "warn",
		"message":      "something happened",
		"service.name": "repair-desk-api",
		"status":       float64(409),
	}
	for k, v := range want {
		if rec[k] != v {
			t.Errorf("%s: got %v, want %v", k, rec[k], v)
		}
	}
	if _, ok := rec["timestamp"]; !ok {
		t.Errorf("timestamp is missing: %v", rec)
	}
	for _, old := range []string{"time", "msg"} {
		if _, ok := rec[old]; ok {
			t.Errorf("default key %q must be renamed: %v", old, rec)
		}
	}
}
