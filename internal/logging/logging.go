// Package logging builds the JSON logger used by the whole application.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// New returns a JSON logger whose records follow the common log schema:
// timestamp, level (lowercase), message, service.name, plus record attributes.
func New(w io.Writer, level slog.Level, service string) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	})
	return slog.New(h).With("service.name", service)
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		a.Key = "timestamp"
	case slog.MessageKey:
		a.Key = "message"
	case slog.LevelKey:
		a.Value = slog.StringValue(strings.ToLower(a.Value.String()))
	}
	return a
}
