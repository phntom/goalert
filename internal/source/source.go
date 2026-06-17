// Package source ingests Pikud HaOref alerts from the live feeds (ynet, oref
// history, Tzeva Adom) and emits source-agnostic alert.Alert values. Network
// failures are logged and never fatal; main decides what to do on shutdown.
package source

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/metrics"
)

// Source ingests alerts and emits them on out until ctx is cancelled.
type Source interface {
	Name() string
	Run(ctx context.Context, out chan<- alert.Alert)
}

// poller performs conditional GETs against a single URL, reusing one HTTP
// client and remembering Last-Modified so unchanged feeds return cheaply.
type poller struct {
	name    string
	url     string
	referer string
	client  *http.Client
	metrics *metrics.Metrics
	lastMod string
}

func newPoller(name, url, referer string, m *metrics.Metrics) *poller {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableKeepAlives = false
	return &poller{
		name: name, url: url, referer: referer, metrics: m,
		client: &http.Client{Timeout: 3 * time.Second, Transport: t},
	}
}

// fetch returns the cleaned body and changed=true on a fresh 200, or
// (nil,false) on 304, error, or a non-200 status.
func (p *poller) fetch(ctx context.Context) (body []byte, changed bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		mlog.Error("source request build failed", mlog.String("source", p.name), mlog.Err(err))
		return nil, false
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", p.referer)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if p.lastMod != "" {
		req.Header.Set("If-Modified-Since", p.lastMod)
	}

	start := time.Now()
	res, err := p.client.Do(req)
	p.metrics.HTTPResponse.WithLabelValues(p.name).Observe(time.Since(start).Seconds())
	if err != nil {
		p.client.CloseIdleConnections()
		p.metrics.SourceFetchFail.WithLabelValues(p.name).Inc()
		mlog.Warn("source fetch failed", mlog.String("source", p.name), mlog.Err(err))
		return nil, false
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusNotModified:
		p.metrics.SourceFetchOK.WithLabelValues(p.name).Inc()
		return nil, false
	case http.StatusOK:
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			p.metrics.SourceFetchFail.WithLabelValues(p.name).Inc()
			return nil, false
		}
		if lm := res.Header.Get("Last-Modified"); lm != "" {
			p.lastMod = lm
		}
		p.metrics.SourceFetchOK.WithLabelValues(p.name).Inc()
		return clean(raw), true
	default:
		p.metrics.SourceFetchFail.WithLabelValues(p.name).Inc()
		mlog.Warn("source bad status", mlog.String("source", p.name), mlog.Int("status", res.StatusCode))
		return nil, false
	}
}

// clean strips the UTF-8 BOM and leading NULs that oref feeds sometimes serve.
func clean(b []byte) []byte {
	return bytes.TrimSpace(bytes.TrimPrefix(bytes.TrimLeft(b, "\x00"), []byte{0xEF, 0xBB, 0xBF}))
}
