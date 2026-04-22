package httpclient

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	log "github.com/sirupsen/logrus"
)

const maxLoggedBodyBytes = 8192

func Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return Do(req)
}

func Do(req *http.Request) (*http.Response, error) {
	requestBody, _ := snapshotRequestBody(req)
	logRequest(req, requestBody)

	startedAt := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(startedAt)
	if err != nil {
		log.WithFields(log.Fields{
			"component":   "provider_http",
			"method":      req.Method,
			"url":         req.URL.String(),
			"duration_ms": duration.Milliseconds(),
			"error":       err.Error(),
		}).Info("Provider HTTP error")
		return nil, err
	}

	responseBody, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		log.WithFields(log.Fields{
			"component":   "provider_http",
			"method":      req.Method,
			"url":         req.URL.String(),
			"duration_ms": duration.Milliseconds(),
			"status_code": resp.StatusCode,
			"error":       readErr.Error(),
		}).Info("Provider HTTP response read error")
		return nil, readErr
	}

	resp.Body = io.NopCloser(bytes.NewReader(responseBody))
	logResponse(req, resp, duration, responseBody)
	return resp, nil
}

func snapshotRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.GetBody == nil {
		return nil, nil
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	defer body.Close()

	return io.ReadAll(body)
}

func logRequest(req *http.Request, requestBody []byte) {
	if !shouldLogProviderHTTP() {
		return
	}

	log.WithFields(log.Fields{
		"component":      "provider_http",
		"direction":      "request",
		"method":         req.Method,
		"url":            req.URL.String(),
		"headers":        sanitizeHeaders(req.Header),
		"request_body":   truncateBody(requestBody),
		"request_bytes":  len(requestBody),
		"request_host":   req.URL.Host,
		"request_scheme": req.URL.Scheme,
	}).Info("Provider HTTP request")
}

func logResponse(req *http.Request, resp *http.Response, duration time.Duration, responseBody []byte) {
	if !shouldLogProviderHTTP() {
		return
	}

	log.WithFields(log.Fields{
		"component":       "provider_http",
		"direction":       "response",
		"method":          req.Method,
		"url":             req.URL.String(),
		"status":          resp.Status,
		"status_code":     resp.StatusCode,
		"duration_ms":     duration.Milliseconds(),
		"headers":         sanitizeHeaders(resp.Header),
		"response_body":   truncateBody(responseBody),
		"response_bytes":  len(responseBody),
		"response_length": resp.ContentLength,
	}).Info("Provider HTTP response")
}

func shouldLogProviderHTTP() bool {
	return config.IsProviderHTTPDebugEnabled() || log.StandardLogger().IsLevelEnabled(log.DebugLevel)
}

func sanitizeHeaders(headers http.Header) map[string][]string {
	if headers == nil {
		return map[string][]string{}
	}

	clone := make(map[string][]string, len(headers))
	for key, values := range headers {
		if key == "Authorization" || key == "Cookie" || key == "Set-Cookie" {
			clone[key] = []string{"[REDACTED]"}
			continue
		}
		copyValues := append([]string(nil), values...)
		clone[key] = copyValues
	}
	return clone
}

func truncateBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if len(body) <= maxLoggedBodyBytes {
		return string(body)
	}
	return string(body[:maxLoggedBodyBytes]) + "...[truncated]"
}
