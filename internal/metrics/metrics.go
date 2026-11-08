// Package metrics defines the Prometheus/OpenMetrics metrics of the application.
// Go runtime (go_*) and process (process_*) metrics are registered by client_golang itself.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "repairdesk"

// Business metrics.
var (
	TicketsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "tickets_created_total",
		Help:      "Tickets created, by category and priority.",
	}, []string{"category", "priority"})

	TicketStatusChanges = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ticket_status_changes_total",
		Help:      "Successful ticket status transitions.",
	}, []string{"from", "to"})

	TicketsResolved = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "tickets_resolved_total",
		Help:      "Tickets moved to RESOLVED, split by whether the SLA deadline (due_at) was met.",
	}, []string{"within_sla"})

	TicketResolutionSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "ticket_resolution_seconds",
		Help:      "Time from ticket creation to RESOLVED.",
		// 1m, 15m, 1h, 4h, 12h, 24h, 48h, 72h, 1w
		Buckets: []float64{60, 900, 3600, 4 * 3600, 12 * 3600, 24 * 3600, 48 * 3600, 72 * 3600, 168 * 3600},
	})

	BusinessRuleViolations = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "business_rule_violations_total",
		Help:      "Requests rejected by a business rule, by error code.",
	}, []string{"rule"})
)

// Technical metric: request count and latency per route (RED: rate, errors, duration).
var HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "http_request_duration_seconds",
	Help:      "HTTP request latency by method, route pattern and status code.",
	Buckets:   prometheus.DefBuckets,
}, []string{"method", "route", "status"})

// Error codes counted as business rule violations (see docs/api-docs.md).
var ruleCodes = []string{
	"INVALID_STATUS_TRANSITION",
	"TICKET_CLOSED",
	"TICKET_NOT_DELETABLE",
	"EQUIPMENT_RETIRED",
	"ALREADY_EXISTS",
}

func init() {
	// Export known series as 0 right away, so rate() and absent() work before the first event.
	for _, code := range ruleCodes {
		BusinessRuleViolations.WithLabelValues(code)
	}
	TicketsResolved.WithLabelValues("true")
	TicketsResolved.WithLabelValues("false")
}

// RecordRuleViolation counts the error code if it is a business rule violation.
func RecordRuleViolation(code string) {
	for _, c := range ruleCodes {
		if c == code {
			BusinessRuleViolations.WithLabelValues(code).Inc()
			return
		}
	}
}

// RecordResolved records a transition to RESOLVED.
func RecordResolved(createdAt, dueAt, now time.Time) {
	within := "false"
	if !now.After(dueAt) {
		within = "true"
	}
	TicketsResolved.WithLabelValues(within).Inc()
	TicketResolutionSeconds.Observe(now.Sub(createdAt).Seconds())
}

// Handler serves all registered metrics; OpenMetrics format is used when the scraper asks for it.
func Handler() http.Handler {
	return promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{EnableOpenMetrics: true})
}
