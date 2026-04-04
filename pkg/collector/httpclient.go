package collector

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// defaultClient is a shared HTTP client with sensible defaults.
var defaultClient = &http.Client{
	Timeout: 30 * time.Second,
}

// ErrRateLimited is returned when a server responds with 429 Too Many Requests.
// RetryAfter indicates how long to wait before retrying (from Retry-After header).
type ErrRateLimited struct {
	URL        string
	RetryAfter time.Duration
}

func (e *ErrRateLimited) Error() string {
	return fmt.Sprintf("%s: rate limited (retry after %s)", e.URL, e.RetryAfter)
}

// ErrEstimatedData is returned when a source returns estimated (non-direct) prices.
// The fallback chain should skip to the next source WITHOUT triggering the circuit breaker.
// Carries the parsed prices so they can be used as a last resort if all other sources fail.
type ErrEstimatedData struct {
	Zone   string
	Prices []models.HourlyPrice
}

func (e *ErrEstimatedData) Error() string {
	return fmt.Sprintf("zone %s: only estimated data available", e.Zone)
}

// parseRetryAfter parses the Retry-After header value.
// Supports both seconds (integer) and HTTP-date formats.
// Returns a default of 5 minutes if the header is missing or unparseable.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 5 * time.Minute
	}
	// Try as seconds first (most common for APIs)
	if secs, err := strconv.Atoi(header); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	// Try as HTTP-date
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 5 * time.Minute
}

// httpGet performs a GET request and returns the response body bytes.
// Returns ErrRateLimited on 429 responses with Retry-After duration.
func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		slog.Warn("rate limited", "url", url, "retry_after", retryAfter)
		return nil, &ErrRateLimited{URL: url, RetryAfter: retryAfter}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s returned %d: %s", url, resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}
	return data, nil
}

// httpPost performs a POST request with a body and returns the response body bytes.
// Returns ErrRateLimited on 429 responses with Retry-After duration.
func httpPost(ctx context.Context, url, contentType string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		slog.Warn("rate limited", "url", url, "retry_after", retryAfter)
		return nil, &ErrRateLimited{URL: url, RetryAfter: retryAfter}
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s returned %d: %s", url, resp.StatusCode, string(respBody))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}
	return data, nil
}
