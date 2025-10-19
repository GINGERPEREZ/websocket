package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config captures the minimal settings needed to configure a slog logger.
type Config struct {
	// Level represents the textual log level (debug, info, warn, error).
	Level string
	// Format controls the output encoding (json or text).
	Format string
	// AddSource toggles slog's source attribution.
	AddSource bool
}

// ParseLevel converts textual levels into slog levels, defaulting to info.
func ParseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "dbg":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "err":
		return slog.LevelError
	case "trace":
		return slog.LevelDebug - 2
	default:
		return slog.LevelInfo
	}
}

// New builds a slog.Logger for the provided writer using the supplied configuration.
func New(w io.Writer, cfg Config) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	handlerOpts := &slog.HandlerOptions{Level: ParseLevel(cfg.Level), AddSource: cfg.AddSource}
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "json":
		return slog.New(slog.NewJSONHandler(w, handlerOpts))
	default:
		return slog.New(slog.NewTextHandler(w, handlerOpts))
	}
}
