//go:build smoke

// Live smoke tests against the real Pikud HaOref / Tzeva Adom endpoints.
// They are excluded from the default build; run with: go test -tags smoke ./internal/source/
package source

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/metrics"
)

func TestSmokeYnetLive(t *testing.T) {
	y := NewYnet("", metrics.New())
	body, changed := y.poller.fetch(context.Background())
	if !changed || len(body) == 0 {
		t.Fatalf("ynet fetch returned no body (changed=%v)", changed)
	}
	alerts := y.parse(body)
	t.Logf("ynet: %d bytes, %d active alert groups", len(body), len(alerts))
}

func TestSmokeOrefHistoryLive(t *testing.T) {
	o := NewOrefHistory("", metrics.New())
	body, changed := o.poller.fetch(context.Background())
	if !changed || len(body) == 0 {
		t.Fatalf("oref fetch returned no body (changed=%v)", changed)
	}
	// First parse primes the seen set and emits nothing.
	o.parse(body, time.Now())
	if len(o.seen) == 0 {
		t.Fatal("oref history parsed zero records; feed shape may have changed")
	}
	t.Logf("oref: %d bytes, primed %d records", len(body), len(o.seen))
}

func TestSmokeTzevaadomLive(t *testing.T) {
	_ = area.Default() // ensure embedded data loads
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tz := NewTzevaadom("", area.Default(), metrics.New())
	c, _, err := websocket.Dial(ctx, tz.url, &websocket.DialOptions{
		HTTPHeader: map[string][]string{"Origin": {tzevaadomOrigin}},
	})
	if err != nil {
		t.Fatalf("tzevaadom dial failed: %v", err)
	}
	_ = c.Close(websocket.StatusNormalClosure, "")
	t.Log("tzevaadom: WebSocket connected")
}
