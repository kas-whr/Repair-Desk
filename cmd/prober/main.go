// Prober is a synthetic user: every PROBE_INTERVAL it runs the "open tickets page" scenario
// against the frontend (nginx -> api -> postgres) and exports the result as metrics.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kas-whr/Repair-Desk/internal/logging"
)

const serviceName = "repair-desk-prober"

var (
	probeSuccess = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "repairdesk_probe_success",
		Help: "1 if the last probe scenario succeeded, 0 otherwise.",
	})
	probeDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "repairdesk_probe_duration_seconds",
		Help: "Duration of the last probe scenario.",
	})
	probeRuns = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "repairdesk_probe_runs_total",
		Help: "Probe scenario runs by result.",
	}, []string{"result"})
)

// The requests the browser makes when a user opens the tickets page.
var scenario = []string{
	"/api/v1/categories",
	"/api/v1/equipment",
	"/api/v1/tickets?limit=20",
}

func main() {
	logger := logging.New(os.Stdout, slog.LevelInfo, serviceName)
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("prober stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	target := getEnv("PROBE_TARGET", "http://frontend.repair-desk.svc.cluster.local")
	interval, err := time.ParseDuration(getEnv("PROBE_INTERVAL", "10s"))
	if err != nil {
		return fmt.Errorf("PROBE_INTERVAL: %w", err)
	}
	timeout, err := time.ParseDuration(getEnv("PROBE_TIMEOUT", "2s"))
	if err != nil {
		return fmt.Errorf("PROBE_TIMEOUT: %w", err)
	}
	probeRuns.WithLabelValues("success")
	probeRuns.WithLabelValues("failure")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	srv := &http.Server{Addr: ":" + getEnv("METRICS_PORT", "9090"), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server failed", "error", err)
			stop()
		}
	}()
	logger.Info("prober started", "target", target, "interval", interval.String(), "timeout", timeout.String())

	client := &http.Client{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		probe(ctx, logger, client, target, timeout)
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		case <-ticker.C:
		}
	}
}

func probe(ctx context.Context, logger *slog.Logger, client *http.Client, target string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := runScenario(ctx, client, target)
	elapsed := time.Since(start)

	probeDuration.Set(elapsed.Seconds())
	if err != nil {
		probeSuccess.Set(0)
		probeRuns.WithLabelValues("failure").Inc()
		logger.Warn("probe failed", "error", err.Error(), "duration_ms", float64(elapsed.Microseconds())/1000)
		return
	}
	probeSuccess.Set(1)
	probeRuns.WithLabelValues("success").Inc()
	logger.Debug("probe succeeded", "duration_ms", float64(elapsed.Microseconds())/1000)
}

func runScenario(ctx context.Context, client *http.Client, target string) error {
	for _, path := range scenario {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", serviceName)
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("GET %s: %w", path, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
