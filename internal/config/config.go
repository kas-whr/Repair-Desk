package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is read only from environment variables (12-factor III).
type Config struct {
	HTTPPort    string
	MetricsPort string
	LogLevel    slog.Level
	DB          DBConfig
	// Fault injection for SLO demonstrations; disabled by default
	FaultErrorRate float64
	FaultDelay     time.Duration
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

func (c DBConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, c.Port),
		Path:     c.Name,
		RawQuery: url.Values{"sslmode": {c.SSLMode}}.Encode(),
	}
	return u.String()
}

func Load() (Config, error) {
	cfg := Config{
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		MetricsPort: getEnv("METRICS_PORT", "9090"),
		DB: DBConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			Name:     os.Getenv("POSTGRES_DB"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
	}

	if v := os.Getenv("FAULT_ERROR_RATE"); v != "" {
		rate, err := strconv.ParseFloat(v, 64)
		if err != nil || rate < 0 || rate > 1 {
			return Config{}, fmt.Errorf("FAULT_ERROR_RATE must be a number between 0 and 1, got %q", v)
		}
		cfg.FaultErrorRate = rate
	}
	if v := os.Getenv("FAULT_DELAY_MS"); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil || ms < 0 {
			return Config{}, fmt.Errorf("FAULT_DELAY_MS must be a non-negative integer, got %q", v)
		}
		cfg.FaultDelay = time.Duration(ms) * time.Millisecond
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(getEnv("LOG_LEVEL", "INFO"))); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
	}

	var missing []string
	for name, v := range map[string]string{
		"POSTGRES_DB":       cfg.DB.Name,
		"POSTGRES_USER":     cfg.DB.User,
		"POSTGRES_PASSWORD": cfg.DB.Password,
	} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("required environment variables are not set: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
