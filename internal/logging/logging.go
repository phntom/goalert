// Package logging configures the process-wide structured logger.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// ParseLevel converts "debug", "info", "warn", "error" (case-insensitive, surrounding
// space ignored) to a slog.Level. The empty string means info. ok is false for any
// other value, in which case the returned level is info.
func ParseLevel(s string) (level slog.Level, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return slog.LevelInfo, true
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}

// New returns a logger that writes one JSON object per line to w at the given level.
func New(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}

// Setup installs New(os.Stderr, <parsed level>) as slog's default logger. If the value
// is not recognized it uses info and logs one warning (through the new logger) naming
// the rejected value.
func Setup(level string) {
	lvl, ok := ParseLevel(level)
	logger := New(os.Stderr, lvl)
	if !ok {
		logger.Warn("unknown LOG_LEVEL value, using info", "value", level)
	}
	slog.SetDefault(logger)
}
