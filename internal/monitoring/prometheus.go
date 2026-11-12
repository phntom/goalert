package monitoring

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registerMetricsOnce guards promauto registration so Setup can be called
// more than once (per process) without panicking on duplicate collectors.
var registerMetricsOnce sync.Once

type Monitoring struct {
	SuccessfulSourceFetches   *prometheus.CounterVec
	FailedSourceFetches       *prometheus.CounterVec
	HttpResponseTimeHistogram *prometheus.HistogramVec
	SuccessfulPosts           prometheus.Counter
	SuccessfulPatches         prometheus.Counter
	FailedPatches             prometheus.Counter
	CitiesHistogram           prometheus.Histogram
	RegionsHistogram          prometheus.Histogram
	TimeOfDayHistogram        prometheus.Histogram
	DayOfWeekHistogram        prometheus.Histogram
}

// handler builds the HTTP routes served on the monitoring port: Prometheus
// metrics plus the liveness and readiness probes. ready reports whether the
// bot is connected to Mattermost.
func (m *Monitoring) handler(ready func() bool) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready\n"))
			return
		}
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}

func (m *Monitoring) Setup(ready func() bool) {
	registerMetricsOnce.Do(func() {
		m.SuccessfulSourceFetches = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "successful_source_fetches",
				Help: "Number of successful source fetches.",
			},
			[]string{"source"},
		)
		m.FailedSourceFetches = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "failed_source_fetches",
				Help: "Number of failed source fetches.",
			},
			[]string{"source"},
		)
		m.HttpResponseTimeHistogram = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_time_seconds",
				Help:    "Histogram of response times for HTTP client per source.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"source"},
		)
		m.SuccessfulPosts = promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "successful_posts",
				Help: "Number of successful POST requests.",
			},
		)
		m.SuccessfulPatches = promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "successful_patches",
				Help: "Number of successful PATCH requests.",
			},
		)
		m.FailedPatches = promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "failed_patches",
				Help: "Number of failed PATCH requests.",
			},
		)
		m.CitiesHistogram = promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "number_of_cities",
				Help:    "Histogram for the number of cities.",
				Buckets: prometheus.LinearBuckets(1, 1, 10), // Adjust buckets as needed
			},
		)
		m.RegionsHistogram = promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "number_of_regions",
				Help:    "Histogram for the number of regions.",
				Buckets: prometheus.LinearBuckets(1, 1, 10), // Adjust buckets as needed
			},
		)
		m.TimeOfDayHistogram = promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "time_of_day",
				Help:    "Histogram for time of day (hour).",
				Buckets: prometheus.LinearBuckets(0, 1, 24), // 24 hours
			},
		)
		m.DayOfWeekHistogram = promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "day_of_week",
				Help:    "Histogram for day of the week.",
				Buckets: prometheus.LinearBuckets(1, 1, 7), // 7 days
			},
		)
	})

	server := &http.Server{
		Addr:              ":3000",
		Handler:           m.handler(ready),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server stopped", "error", err)
		}
	}()
}
