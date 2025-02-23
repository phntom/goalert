package fetcher

import (
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/monitoring"
	"io"
	"net/http"
	"time"
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
	req, err := http.NewRequest("GET", url, nil)
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
	for i := 0; i < 5; i++ {
		// Sending the request
		startTime := time.Now()
		res, err = client.Do(req)
		duration := time.Since(startTime).Seconds()
		monitor.HttpResponseTimeHistogram.WithLabelValues(sourceName).Observe(duration)

		if err != nil {
			mlog.Error("failed to fetch - client error",
				mlog.Err(err),
				mlog.Any("source", sourceName),
				mlog.Any("attempt", i),
			)
			client.CloseIdleConnections()
			monitor.FailedSourceFetches.WithLabelValues(sourceName).Inc()
			continue
		}
		break
	}

	if err != nil {
		mlog.Error("Failed fetching "+sourceName, mlog.Err(err))
		return nil
	}

	if res.StatusCode != 200 {
		mlog.Warn("failed to fetch - wrong status code",
			mlog.Any("status", res.StatusCode),
			mlog.Any("res", res),
			mlog.Any("source", sourceName),
		)
		return nil
	}
	content, err := io.ReadAll(res.Body)
	if err != nil {
		mlog.Error("failed to fetch - io error",
			mlog.Err(err),
			mlog.Any("source", sourceName),
		)
		monitor.FailedSourceFetches.WithLabelValues(sourceName).Inc()
		return nil
	}
	monitor.SuccessfulSourceFetches.WithLabelValues(sourceName).Inc()
	return content
}
