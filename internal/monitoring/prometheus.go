package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sync"
)

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

func (m *Monitoring) Setup() {
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

		http.Handle("/metrics", promhttp.Handler())
		//goland:noinspection GoUnhandledErrorResult
		go http.ListenAndServe(":3000", nil) //nolint:errcheck
	})
}
