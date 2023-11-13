package fetcher

import (
	"net/http"
	"time"
)

func CreateHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	// Customize HTTP headers with default values.
	defaultHeaders := map[string]string{
		"Accept":             "*/*",
		"User-Agent":         "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Accept-Language":    "en-GB,en-US;q=0.9,en;q=0.8",
		"Sec-Ch-Ua":          "\"Google Chrome\";v=\"119\", \"Chromium\";v=\"119\", \"Not?A_Brand\";v=\"24\"",
		"Sec-Ch-Ua-Mobile":   "?0",
		"Sec-Ch-Ua-Platform": "\"Linux\"",
		"Sec-Fetch-Dest":     "script",
		"Sec-Fetch-Mode":     "no-cors",
		"Sec-Fetch-Site":     "same-site",
	}

	// Apply default headers to the HTTP client's Transport.
	client.Transport = &headerTransport{
		base:           http.DefaultTransport,
		defaultHeaders: defaultHeaders,
	}

	return client
}

// Custom transport to set default headers.
type headerTransport struct {
	base           http.RoundTripper
	defaultHeaders map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Apply default headers to the request.
	for key, value := range t.defaultHeaders {
		req.Header.Set(key, value)
	}

	// Perform the actual HTTP request using the underlying transport.
	return t.base.RoundTrip(req)
}
