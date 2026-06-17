package source

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/metrics"
)

// DefaultOrefHistoryURL is the authoritative Pikud HaOref history feed. It is
// the source of truth for end-of-alert (category 13) and a dedup backstop.
const DefaultOrefHistoryURL = "https://alerts-history.oref.org.il/Shared/Ajax/GetAlarmsHistory.aspx?lang=he&mode=1"

const (
	orefInterval = 1500 * time.Millisecond
	orefMaxAge   = 10 * time.Minute
)

type orefRecord struct {
	Data         string `json:"data"`
	AlertDate    string `json:"alertDate"`
	Category     int    `json:"category"`
	CategoryDesc string `json:"category_desc"`
	Rid          int    `json:"rid"`
}

// OrefHistory polls the oref history feed. The first poll only primes the seen
// set (so we never replay the day's backlog); later polls emit new records.
type OrefHistory struct {
	poller *poller
	seen   map[int]bool
	primed bool
}

// NewOrefHistory builds the source. Pass "" to use DefaultOrefHistoryURL.
func NewOrefHistory(url string, m *metrics.Metrics) *OrefHistory {
	if url == "" {
		url = DefaultOrefHistoryURL
	}
	return &OrefHistory{
		poller: newPoller("oref", url, "https://www.oref.org.il/", m),
		seen:   make(map[int]bool),
	}
}

// Name implements Source.
func (o *OrefHistory) Name() string { return "oref" }

// Run implements Source.
func (o *OrefHistory) Run(ctx context.Context, out chan<- alert.Alert) {
	ticker := time.NewTicker(orefInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		body, changed := o.poller.fetch(ctx)
		if !changed {
			continue
		}
		for _, a := range o.parse(body, time.Now()) {
			select {
			case out <- a:
			case <-ctx.Done():
				return
			}
		}
	}
}

// parse groups new, recent records by category and time. It updates the seen
// set; the first call primes only and emits nothing.
func (o *OrefHistory) parse(body []byte, now time.Time) []alert.Alert {
	var recs []orefRecord
	if err := json.Unmarshal(body, &recs); err != nil {
		mlog.Error("oref unmarshal failed", mlog.Err(err))
		return nil
	}
	type key struct {
		cat  int
		date string
	}
	groups := make(map[key]*alert.Alert)
	var order []key
	for _, r := range recs {
		if r.Rid != 0 {
			if o.seen[r.Rid] {
				continue
			}
			o.seen[r.Rid] = true
		}
		if !o.primed {
			continue
		}
		at := parseOrefDate(r.AlertDate, now)
		if now.Sub(at) > orefMaxAge {
			continue
		}
		cat := alert.FromOrefHistory(r.Category)
		if cat == alert.CatUnknown {
			continue
		}
		k := key{r.Category, r.AlertDate}
		g, ok := groups[k]
		if !ok {
			g = &alert.Alert{Category: cat, Kind: alert.KindOf(cat), At: at, Source: "oref"}
			groups[k] = g
			order = append(order, k)
		}
		g.Areas = append(g.Areas, r.Data)
		if r.Rid != 0 {
			g.IDs = append(g.IDs, strconv.Itoa(r.Rid))
		}
	}
	o.primed = true

	out := make([]alert.Alert, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	return out
}

// parseOrefDate parses "2006-01-02T15:04:05" in Israel time, falling back to now.
func parseOrefDate(s string, now time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, loc)
	if err != nil {
		return now
	}
	return t
}
