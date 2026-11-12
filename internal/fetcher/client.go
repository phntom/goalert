package fetcher

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/phntom/goalert/internal/monitoring"
)

func CreateHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableKeepAlives = false
	client.Transport = t

	return client
}

func FetchSource(client *http.Client, url string, sourceName string, referrer string, monitor *monitoring.Monitoring) []byte {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	// Setting headers
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", referrer)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	var res *http.Response
	for i := range 5 {
		// Sending the request
		startTime := time.Now()
		res, err = client.Do(req)
		duration := time.Since(startTime).Seconds()
		monitor.HttpResponseTimeHistogram.WithLabelValues(sourceName).Observe(duration)

		if err != nil {
			slog.Error("failed to fetch - client error",
				"error", err,
				"source", sourceName,
				"attempt", i,
			)
			if res != nil {
				// Can be non-nil together with an error on redirect failures.
				_ = res.Body.Close()
				res = nil
			}
			client.CloseIdleConnections()
			monitor.FailedSourceFetches.WithLabelValues(sourceName).Inc()
			continue
		}
		break
	}

	if err != nil {
		slog.Error("Failed fetching "+sourceName, "error", err)
		return nil
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		slog.Warn("failed to fetch - wrong status code",
			"status", res.StatusCode,
			"source", sourceName,
		)
		monitor.FailedSourceFetches.WithLabelValues(sourceName).Inc()
		return nil
	}
	content, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("failed to fetch - io error",
			"error", err,
			"source", sourceName,
		)
		monitor.FailedSourceFetches.WithLabelValues(sourceName).Inc()
		return nil
	}
	monitor.SuccessfulSourceFetches.WithLabelValues(sourceName).Inc()
	return content
}
