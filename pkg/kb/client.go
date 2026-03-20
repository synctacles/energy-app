// Package kb provides a lightweight client for Synctacles KB search,
// filtered to energy-relevant articles.
package kb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/synctacles/energy-app/pkg/platform"
)

var defaultBaseURL = platform.APIBaseURL

// Client searches the Synctacles KB API with product=energy pre-set.
type Client struct {
	baseURL    string
	installID  string
	httpClient *http.Client
}

// NewClient creates a KB client. If baseURL is empty, the default is used.
func NewClient(baseURL, installID string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:   baseURL,
		installID: installID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Article represents a single KB search result.
type Article struct {
	ID               int     `json:"id"`
	Title            string  `json:"problem_title"`
	Description      string  `json:"problem_description,omitempty"`
	Solution         string  `json:"solution_text,omitempty"`
	Category         string  `json:"problem_category"`
	Component        string  `json:"problem_component,omitempty"`
	SourceURL        string  `json:"source_url,omitempty"`
	ConfidenceScore  float64 `json:"confidence_score,omitempty"`
	ProductRelevance string  `json:"product_relevance,omitempty"`
}

// SearchResult holds the response from a KB search.
type SearchResult struct {
	Query   string    `json:"query"`
	Total   int       `json:"total"`
	Results []Article `json:"results"`
}

// Search queries the KB API with product=energy pre-set.
func (c *Client) Search(ctx context.Context, query string, limit int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	u, err := url.Parse(c.baseURL + "/api/v1/kb/search")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	q.Set("product", "energy")
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.installID != "" {
		req.Header.Set("X-Install-ID", c.installID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
