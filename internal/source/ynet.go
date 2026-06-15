package source

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/metrics"
)

// DefaultYnetURL is the AWS-direct origin that bypasses Akamai's cache, so new
// alerts appear without waiting for the CDN TTL to expire. Override with the
// YNET_URL env var (e.g. the Akamai URL) if needed.
const DefaultYnetURL = "https://source-alerts.ynet.co.il/alertsRss/YnetPicodeHaorefAlertFiles.js?callback=jsonCallback"

const ynetInterval = 250 * time.Millisecond

type ynetFeed struct {
	Alerts struct {
		Items []struct {
			Item struct {
				Guid        string `json:"guid"`
				Pubdate     string `json:"pubdate"`
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"item"`
		} `json:"items"`
	} `json:"alerts"`
}

// Ynet polls the ynet alert feed, the bot's fast primary path.
type Ynet struct {
	poller *poller
	seen   map[string]bool
}

// NewYnet builds the ynet source. Pass "" to use DefaultYnetURL.
func NewYnet(url string, m *metrics.Metrics) *Ynet {
	if url == "" {
		url = DefaultYnetURL
	}
	return &Ynet{
		poller: newPoller("ynet", url, "https://www.ynet.co.il/", m),
		seen:   make(map[string]bool),
	}
}

// Name implements Source.
func (y *Ynet) Name() string { return "ynet" }

// Run implements Source: poll every quarter second and emit grouped alerts.
func (y *Ynet) Run(ctx context.Context, out chan<- alert.Alert) {
	ticker := time.NewTicker(ynetInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		body, changed := y.poller.fetch(ctx)
		if !changed {
			continue
		}
		for _, a := range y.parse(body) {
			select {
			case out <- a:
			case <-ctx.Done():
				return
			}
		}
	}
}

// parse groups new feed items by category and publish time into alerts. It is
// pure aside from updating the seen-guid set.
func (y *Ynet) parse(body []byte) []alert.Alert {
	js := unwrapJSONP(body)
	if js == nil {
		return nil
	}
	var feed ynetFeed
	if err := json.Unmarshal(js, &feed); err != nil {
		mlog.Error("ynet unmarshal failed", mlog.Err(err), mlog.String("body", string(js)))
		return nil
	}
	if len(feed.Alerts.Items) == 0 {
		clear(y.seen) // feed cleared; forget guids so a repeat alert re-fires
		return nil
	}

	groups := make(map[string]*alert.Alert)
	var order []string
	for _, it := range feed.Alerts.Items {
		item := it.Item
		if y.seen[item.Guid] {
			continue
		}
		y.seen[item.Guid] = true
		if alert.IsHebrewDrill(item.Description) {
			continue
		}
		cat := alert.FromYnetDescription(item.Description)
		if cat == alert.CatUnknown {
			mlog.Warn("ynet unknown category", mlog.String("desc", item.Description), mlog.String("title", item.Title))
			continue
		}
		key := string(rune(cat)) + "|" + item.Pubdate
		g, ok := groups[key]
		if !ok {
			g = &alert.Alert{Category: cat, Kind: alert.KindAlert, At: parseClock(item.Pubdate), Source: "ynet"}
			groups[key] = g
			order = append(order, key)
		}
		g.Areas = append(g.Areas, item.Title)
		g.IDs = append(g.IDs, item.Guid)
	}

	out := make([]alert.Alert, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	return out
}

// unwrapJSONP extracts the JSON object from a jsonCallback(...) wrapper.
func unwrapJSONP(b []byte) []byte {
	i := bytes.IndexByte(b, '{')
	j := bytes.LastIndexByte(b, '}')
	if i < 0 || j < i {
		return nil
	}
	return b[i : j+1]
}

// parseClock turns a "15:04" publish time into today's timestamp in Israel,
// falling back to now on parse failure.
func parseClock(hhmm string) time.Time {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	now := time.Now().In(loc)
	t, err := time.ParseInLocation("15:04", strings.TrimSpace(hhmm), loc)
	if err != nil {
		return now
	}
	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, loc)
}
