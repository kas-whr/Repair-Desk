package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandler_ExposesBusinessAndRuntimeMetrics(t *testing.T) {
	TicketsCreated.WithLabelValues("Network", "HIGH").Inc()
	RecordRuleViolation("TICKET_CLOSED")
	RecordRuleViolation("NOT_FOUND") // not a business rule, must be ignored
	now := time.Now()
	RecordResolved(now.Add(-2*time.Hour), now.Add(time.Hour), now)

	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`repairdesk_tickets_created_total{category="Network",priority="HIGH"} 1`,
		`repairdesk_business_rule_violations_total{rule="TICKET_CLOSED"} 1`,
		`repairdesk_business_rule_violations_total{rule="EQUIPMENT_RETIRED"} 0`,
		`repairdesk_tickets_resolved_total{within_sla="true"} 1`,
		`repairdesk_ticket_resolution_seconds_count 1`,
		"go_goroutines",
		"process_resident_memory_bytes",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output has no %q", want)
		}
	}
	if strings.Contains(body, `rule="NOT_FOUND"`) {
		t.Error("NOT_FOUND must not be counted as a rule violation")
	}
}
