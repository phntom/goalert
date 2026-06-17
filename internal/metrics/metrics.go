// Package metrics holds the Prometheus instrumentation for the bot. Each
// Metrics value owns a private registry so it can be constructed repeatedly
// (e.g. in tests) without colliding on the global default registry.
package metrics

import (
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is the set of collectors the bot reports. Exposition metric names are
// kept stable so existing dashboards keep working.
type Metrics struct {
	reg *prometheus.Registry

	SourceFetchOK   *prometheus.CounterVec   // labels: source
	SourceFetchFail *prometheus.CounterVec   // labels: source
	HTTPResponse    *prometheus.HistogramVec // labels: source

	PostsOK   prometheus.Counter
	PatchesOK prometheus.Counter
	PatchFail prometheus.Counter

	AreasPerAlert   prometheus.Histogram
	RegionsPerAlert prometheus.Histogram
	HourOfDay       prometheus.Histogram
	DayOfWeek       prometheus.Histogram
}

// New constructs the collectors and registers them on a private registry.
func New() *Metrics {
	reg := prometheus.NewRegistry()
	f := promauto.With(reg)
	m := &Metrics{
		reg: reg,
		SourceFetchOK: f.NewCounterVec(prometheus.CounterOpts{
			Name: "successful_source_fetches", Help: "Number of successful source fetches.",
		}, []string{"source"}),
		SourceFetchFail: f.NewCounterVec(prometheus.CounterOpts{
			Name: "failed_source_fetches", Help: "Number of failed source fetches.",
		}, []string{"source"}),
		HTTPResponse: f.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_response_time_seconds", Help: "HTTP client response times per source.",
			Buckets: prometheus.DefBuckets,
		}, []string{"source"}),
		PostsOK: f.NewCounter(prometheus.CounterOpts{
			Name: "successful_posts", Help: "Number of successful post creations.",
		}),
		PatchesOK: f.NewCounter(prometheus.CounterOpts{
			Name: "successful_patches", Help: "Number of successful post patches.",
		}),
		PatchFail: f.NewCounter(prometheus.CounterOpts{
			Name: "failed_patches", Help: "Number of failed post patches.",
		}),
		AreasPerAlert: f.NewHistogram(prometheus.HistogramOpts{
			Name: "number_of_cities", Help: "Areas per published alert.",
			Buckets: prometheus.LinearBuckets(1, 1, 10),
		}),
		RegionsPerAlert: f.NewHistogram(prometheus.HistogramOpts{
			Name: "number_of_regions", Help: "Distinct regions per published alert.",
			Buckets: prometheus.LinearBuckets(1, 1, 10),
		}),
		HourOfDay: f.NewHistogram(prometheus.HistogramOpts{
			Name: "time_of_day", Help: "Hour of day of published alerts.",
			Buckets: prometheus.LinearBuckets(0, 1, 24),
		}),
		DayOfWeek: f.NewHistogram(prometheus.HistogramOpts{
			Name: "day_of_week", Help: "Day of week of published alerts.",
			Buckets: prometheus.LinearBuckets(0, 1, 7),
		}),
	}
	return m
}

// Serve starts the /metrics HTTP endpoint and blocks; run it in a goroutine.
func (m *Metrics) Serve(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{}))
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		mlog.Error("metrics server stopped", mlog.String("addr", addr), mlog.Err(err))
	}
}
