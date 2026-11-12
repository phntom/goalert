package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in     string
		want   slog.Level
		wantOK bool
	}{
		{"", slog.LevelInfo, true},
		{"debug", slog.LevelDebug, true},
		{"DEBUG", slog.LevelDebug, true},
		{" warn ", slog.LevelWarn, true},
		{"info", slog.LevelInfo, true},
		{"error", slog.LevelError, true},
		{"verbose", slog.LevelInfo, false},
	}
	for _, tt := range tests {
		got, ok := ParseLevel(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("ParseLevel(%q) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestNewWritesOneJSONLinePerCall(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, slog.LevelInfo)

	logger.Info("hello", "key", "value")

	line := buf.String()
	if !strings.HasSuffix(line, "\n") || strings.Count(line, "\n") != 1 {
		t.Fatalf("expected exactly one line, got %q", line)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	for _, k := range []string{"time", "level", "msg", "key"} {
		if _, ok := m[k]; !ok {
			t.Errorf("output missing key %q: %v", k, m)
		}
	}
	if m["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", m["msg"])
	}
	if m["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", m["level"])
	}
	if m["key"] != "value" {
		t.Errorf("key = %v, want value", m["key"])
	}
}

func TestNewFiltersBelowLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, slog.LevelInfo)

	logger.Debug("hidden")

	if buf.Len() != 0 {
		t.Errorf("expected no output for Debug at info level, got %q", buf.String())
	}
}
