package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandler exercises the monitoring routes without calling Setup, which
// registers global promauto collectors and starts the metrics server.
func TestHandler(t *testing.T) {
	m := &Monitoring{}
	notReady := m.handler(func() bool { return false })
	ready := m.handler(func() bool { return true })

	t.Run("healthz is always ready", func(t *testing.T) {
		rec := httptest.NewRecorder()
		notReady.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("/healthz status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok\n" {
			t.Errorf("/healthz body = %q, want %q", rec.Body.String(), "ok\n")
		}
	})

	t.Run("readyz reports 503 while not ready", func(t *testing.T) {
		rec := httptest.NewRecorder()
		notReady.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("/readyz status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if rec.Body.String() != "not ready\n" {
			t.Errorf("/readyz body = %q, want %q", rec.Body.String(), "not ready\n")
		}
	})

	t.Run("readyz reports 200 when ready", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ready.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("/readyz status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok\n" {
			t.Errorf("/readyz body = %q, want %q", rec.Body.String(), "ok\n")
		}
	})

	t.Run("metrics serves the default registry", func(t *testing.T) {
		rec := httptest.NewRecorder()
		notReady.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("/metrics status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}
