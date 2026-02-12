// Package ha provides Home Assistant Supervisor and Core API clients.
package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// SupervisorClient provides access to the HA Supervisor API.
type SupervisorClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewSupervisorClient creates a new Supervisor API client.
func NewSupervisorClient(token string) *SupervisorClient {
	return &SupervisorClient{
		baseURL: "http://supervisor",
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CoreInfo holds basic HA information.
type CoreInfo struct {
	Version         string `json:"version"`
	VersionLatest   string `json:"version_latest"`
	UpdateAvailable bool   `json:"update_available"`
	Arch            string `json:"arch"`
	Machine         string `json:"machine"`
}

// AddonInfo holds addon status information.
type AddonInfo struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Version string `json:"version"`
	State   string `json:"state"`
}

// transientStatusCodes are worth retrying.
var transientStatusCodes = map[int]bool{502: true, 503: true, 504: true}

// requestWithRetry makes an HTTP request with retry logic for transient failures.
func (c *SupervisorClient) requestWithRetry(ctx context.Context, method, endpoint string, body io.Reader) (json.RawMessage, error) {
	maxRetries := 3
	backoff := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		url := c.baseURL + endpoint
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				slog.Warn("supervisor request failed, retrying", "endpoint", endpoint, "attempt", attempt+1, "error", err)
				continue
			}
			return nil, fmt.Errorf("request %s: %w", endpoint, err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Supervisor API wraps responses in {"result": "ok", "data": ...}
			var wrapper struct {
				Result string          `json:"result"`
				Data   json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(data, &wrapper); err != nil {
				// Some endpoints return raw data
				return data, nil
			}
			if wrapper.Result == "ok" {
				return wrapper.Data, nil
			}
			return data, nil
		}

		if transientStatusCodes[resp.StatusCode] && attempt < maxRetries {
			slog.Warn("supervisor transient error, retrying", "endpoint", endpoint, "status", resp.StatusCode, "attempt", attempt+1)
			continue
		}

		return nil, fmt.Errorf("supervisor API %s returned %d: %s", endpoint, resp.StatusCode, string(data))
	}

	return nil, fmt.Errorf("supervisor API %s: max retries exceeded", endpoint)
}

// GetCoreInfo returns basic HA information.
func (c *SupervisorClient) GetCoreInfo(ctx context.Context) (*CoreInfo, error) {
	data, err := c.requestWithRetry(ctx, "GET", "/core/info", nil)
	if err != nil {
		return nil, err
	}
	var info CoreInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse core info: %w", err)
	}
	return &info, nil
}

// GetState gets a single entity state via Core API proxy.
func (c *SupervisorClient) GetState(ctx context.Context, entityID string) (map[string]any, error) {
	data, err := c.requestWithRetry(ctx, "GET", "/core/api/states/"+entityID, nil)
	if err != nil {
		return nil, err
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return state, nil
}

// PostState creates or updates an entity state via Core API proxy.
// This is the primary method for publishing sensor values to HA.
func (c *SupervisorClient) PostState(ctx context.Context, entityID string, state string, attrs map[string]any) error {
	payload := map[string]any{
		"state":      state,
		"attributes": attrs,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	_, err = c.requestWithRetry(ctx, "POST", "/core/api/states/"+entityID, strings.NewReader(string(body)))
	return err
}

// GetConfig gets HA configuration.
func (c *SupervisorClient) GetConfig(ctx context.Context) (map[string]any, error) {
	data, err := c.requestWithRetry(ctx, "GET", "/core/api/config", nil)
	if err != nil {
		return nil, err
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return config, nil
}

// ListAddons returns the list of installed addons (used for MQTT broker detection).
func (c *SupervisorClient) ListAddons(ctx context.Context) ([]AddonInfo, error) {
	data, err := c.requestWithRetry(ctx, "GET", "/addons", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Addons []AddonInfo `json:"addons"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse addons: %w", err)
	}
	return result.Addons, nil
}

// CallService calls an HA service.
func (c *SupervisorClient) CallService(ctx context.Context, domain, service string, serviceData map[string]any) error {
	body, err := json.Marshal(serviceData)
	if err != nil {
		return fmt.Errorf("marshal service data: %w", err)
	}
	endpoint := fmt.Sprintf("/core/api/services/%s/%s", domain, service)
	_, err = c.requestWithRetry(ctx, "POST", endpoint, strings.NewReader(string(body)))
	return err
}

// GetAddonOptions reads the current addon options from Supervisor.
// Options are nested inside the /addons/self/info response.
func (c *SupervisorClient) GetAddonOptions(ctx context.Context) (map[string]any, error) {
	data, err := c.requestWithRetry(ctx, "GET", "/addons/self/info", nil)
	if err != nil {
		return nil, err
	}
	var info struct {
		Options map[string]any `json:"options"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse addon info: %w", err)
	}
	if info.Options == nil {
		return make(map[string]any), nil
	}
	return info.Options, nil
}

// SetAddonOptions writes addon options via Supervisor API.
// This updates /data/options.json inside the addon container.
func (c *SupervisorClient) SetAddonOptions(ctx context.Context, options map[string]any) error {
	payload := map[string]any{"options": options}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal addon options: %w", err)
	}
	_, err = c.requestWithRetry(ctx, "POST", "/addons/self/options", strings.NewReader(string(body)))
	return err
}

// CreateNotification creates a persistent HA notification.
func (c *SupervisorClient) CreateNotification(ctx context.Context, title, message, notifID string) error {
	payload := map[string]string{
		"title":   title,
		"message": message,
	}
	if notifID != "" {
		payload["notification_id"] = notifID
	}
	body, _ := json.Marshal(payload)
	_, err := c.requestWithRetry(ctx, "POST", "/core/api/services/persistent_notification/create", strings.NewReader(string(body)))
	return err
}
