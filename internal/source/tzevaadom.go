package source

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/metrics"
)

// DefaultTzevaadomURL is the Tzeva Adom push WebSocket: the lowest-latency free
// source, and the origin of pre-alerts (earlier than the siren).
const DefaultTzevaadomURL = "wss://ws.tzevaadom.co.il/socket?platform=WEB"

const (
	tzevaadomOrigin = "https://www.tzevaadom.co.il"
	tzevaadomPing   = 30 * time.Second

	tzInstructionPreAlert = 0
	tzInstructionEnd      = 1
)

type tzMessage struct {
	Type string `json:"type"`
	Data struct {
		NotificationID  string          `json:"notificationId"`
		Time            json.RawMessage `json:"time"`
		Threat          int             `json:"threat"`
		IsDrill         bool            `json:"isDrill"`
		Cities          []string        `json:"cities"`
		CitiesIds       []int           `json:"citiesIds"`
		InstructionType *int            `json:"instructionType"`
	} `json:"data"`
}

// Tzevaadom is the Tzeva Adom WebSocket source.
type Tzevaadom struct {
	url     string
	areas   *area.Set
	metrics *metrics.Metrics
	seen    map[string]bool
}

// NewTzevaadom builds the source. Pass "" to use DefaultTzevaadomURL.
func NewTzevaadom(url string, areas *area.Set, m *metrics.Metrics) *Tzevaadom {
	if url == "" {
		url = DefaultTzevaadomURL
	}
	return &Tzevaadom{url: url, areas: areas, metrics: m, seen: make(map[string]bool)}
}

// Name implements Source.
func (t *Tzevaadom) Name() string { return "tzevaadom" }

// Run implements Source: connect, read until error, then reconnect with jitter.
func (t *Tzevaadom) Run(ctx context.Context, out chan<- alert.Alert) {
	for ctx.Err() == nil {
		if err := t.session(ctx, out); err != nil && ctx.Err() == nil {
			mlog.Warn("tzevaadom session ended", mlog.Err(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5*time.Second + time.Duration(rand.Int63n(int64(3*time.Second)))):
		}
	}
}

func (t *Tzevaadom) session(ctx context.Context, out chan<- alert.Alert) error {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	c, _, err := websocket.Dial(dialCtx, t.url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {tzevaadomOrigin}},
	})
	cancel()
	if err != nil {
		t.metrics.SourceFetchFail.WithLabelValues("tzevaadom").Inc()
		return err
	}
	defer func() { _ = c.Close(websocket.StatusNormalClosure, "") }()
	c.SetReadLimit(1 << 20)

	sctx, scancel := context.WithCancel(ctx)
	defer scancel()
	go ping(sctx, c)

	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return err
		}
		a, ok := t.parseMessage(data)
		if !ok {
			continue
		}
		t.metrics.SourceFetchOK.WithLabelValues("tzevaadom").Inc()
		select {
		case out <- a:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// parseMessage converts a WebSocket frame into an alert. ok is false for
// drills, unknown threats, duplicates, and non-alert frames.
func (t *Tzevaadom) parseMessage(b []byte) (alert.Alert, bool) {
	var m tzMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return alert.Alert{}, false
	}
	id := m.Type + "_" + m.Data.NotificationID
	if m.Data.NotificationID != "" && t.seen[id] {
		return alert.Alert{}, false
	}
	switch m.Type {
	case "ALERT":
		if m.Data.IsDrill {
			t.seen[id] = true
			return alert.Alert{}, false
		}
		cat, known := alert.FromTzevaadomThreat(m.Data.Threat)
		if !known || len(m.Data.Cities) == 0 {
			return alert.Alert{}, false
		}
		t.seen[id] = true
		return alert.Alert{
			Category: cat, Kind: alert.KindAlert, Areas: m.Data.Cities,
			At: parseEpoch(m.Data.Time), Source: "tzevaadom", IDs: []string{id},
		}, true
	case "SYSTEM_MESSAGE":
		cat, ok := t.systemCategory(m.Data.InstructionType)
		if !ok {
			return alert.Alert{}, false
		}
		names := t.resolveIDs(m.Data.CitiesIds)
		if len(names) == 0 {
			return alert.Alert{}, false
		}
		t.seen[id] = true
		return alert.Alert{
			Category: cat, Kind: alert.KindOf(cat), Areas: names,
			At: parseEpoch(m.Data.Time), Source: "tzevaadom", IDs: []string{id},
		}, true
	default:
		return alert.Alert{}, false
	}
}

func (t *Tzevaadom) systemCategory(instruction *int) (alert.Category, bool) {
	if instruction == nil {
		return alert.CatUnknown, false
	}
	switch *instruction {
	case tzInstructionPreAlert:
		return alert.CatPreAlert, true
	case tzInstructionEnd:
		return alert.CatEndAlert, true
	default:
		return alert.CatUnknown, false
	}
}

func (t *Tzevaadom) resolveIDs(ids []int) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if a, ok := t.areas.ByTzevaID(id); ok {
			names = append(names, a.Name("he"))
		}
	}
	return names
}

func ping(ctx context.Context, c *websocket.Conn) {
	ticker := time.NewTicker(tzevaadomPing)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Ping(ctx); err != nil {
				return
			}
		}
	}
}

// parseEpoch parses a unix timestamp that may arrive as a JSON number or string.
func parseEpoch(raw json.RawMessage) time.Time {
	s := strings.Trim(string(raw), `"`)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Now()
	}
	return time.Unix(n, 0)
}
