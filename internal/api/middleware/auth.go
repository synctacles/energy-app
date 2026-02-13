package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// AuthValidation represents the response from the central auth service.
type AuthValidation struct {
	Valid          bool       `json:"valid"`
	Authorized     bool       `json:"authorized"`
	UserID         string     `json:"user_id"`
	Email          string     `json:"email"`
	ProductTier    string     `json:"product_tier"`
	Error          string     `json:"error,omitempty"`
	Message        string     `json:"message,omitempty"`
	UpgradeURL     string     `json:"upgrade_url,omitempty"`
	RateLimit      *RateLimit `json:"rate_limit,omitempty"`
	FeaturesEnabled []string  `json:"features_enabled,omitempty"`
}

// RateLimit represents rate limit information.
type RateLimit struct {
	DailyLimit int       `json:"daily_limit"`
	UsedToday  int       `json:"used_today"`
	Remaining  int       `json:"remaining"`
	ResetsAt   time.Time `json:"resets_at"`
}

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	// ContextKeyAuth stores the auth validation in the request context.
	ContextKeyAuth contextKey = "auth_validation"
)

// CentralAuth creates a middleware that validates API keys using the central auth service.
// It enforces tier-based access control and rate limiting.
func CentralAuth(authServiceURL, productID string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				logger.Warn("Auth failed: missing X-API-Key header",
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeJSONError(w, http.StatusUnauthorized, "api_key_required", "Include X-API-Key header with your request")
				return
			}

			// Determine required tier based on endpoint
			requiredTier := getRequiredTier(r)

			// Debug logging
			logger.Debug("Tier check",
				"path", r.URL.Path,
				"required_tier", requiredTier,
			)

			// Validate with central auth service
			validation, err := callAuthService(authServiceURL, apiKey, productID, requiredTier, r, logger)
			if err != nil {
				logger.Error("Auth service error",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeJSONError(w, http.StatusServiceUnavailable, "auth_service_unavailable", "Authentication service is temporarily unavailable")
				return
			}

			// Check if API key is valid
			if !validation.Valid {
				logger.Warn("Auth failed: invalid API key",
					"method", r.Method,
					"path", r.URL.Path,
					"error", validation.Error,
				)
				writeJSONError(w, http.StatusUnauthorized, validation.Error, validation.Message)
				return
			}

			// Check if user is authorized (tier check)
			if !validation.Authorized {
				logger.Warn("Auth failed: insufficient tier",
					"method", r.Method,
					"path", r.URL.Path,
					"user_tier", validation.ProductTier,
					"required_tier", requiredTier,
				)
				writeJSONErrorWithData(w, http.StatusForbidden, validation.Error, validation.Message, map[string]interface{}{
					"current_tier":  validation.ProductTier,
					"required_tier": requiredTier,
					"upgrade_url":   validation.UpgradeURL,
				})
				return
			}

			// Check rate limit
			if validation.RateLimit != nil && validation.RateLimit.Remaining <= 0 {
				logger.Warn("Rate limit exceeded",
					"method", r.Method,
					"path", r.URL.Path,
					"user_id", validation.UserID,
					"used", validation.RateLimit.UsedToday,
					"limit", validation.RateLimit.DailyLimit,
				)

				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", validation.RateLimit.DailyLimit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", validation.RateLimit.ResetsAt.Unix()))

				writeJSONErrorWithData(w, http.StatusTooManyRequests, "rate_limit_exceeded",
					fmt.Sprintf("Daily rate limit exceeded (%d/%d)", validation.RateLimit.UsedToday, validation.RateLimit.DailyLimit),
					map[string]interface{}{
						"limit":     validation.RateLimit.DailyLimit,
						"used":      validation.RateLimit.UsedToday,
						"resets_at": validation.RateLimit.ResetsAt,
					})
				return
			}

			// Store validation in context for downstream handlers
			ctx := context.WithValue(r.Context(), ContextKeyAuth, validation)

			// Add rate limit headers to response
			if validation.RateLimit != nil {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", validation.RateLimit.DailyLimit))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", validation.RateLimit.Remaining))
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", validation.RateLimit.ResetsAt.Unix()))
			}

			logger.Debug("Auth success",
				"method", r.Method,
				"path", r.URL.Path,
				"user_id", validation.UserID,
				"tier", validation.ProductTier,
			)

			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// callAuthService validates an API key with the central auth service.
func callAuthService(authServiceURL, apiKey, productID, requiredTier string, r *http.Request, logger *slog.Logger) (*AuthValidation, error) {
	reqBody := map[string]interface{}{
		"api_key":       apiKey,
		"product":       productID,
		"required_tier": requiredTier,
		"endpoint":      fmt.Sprintf("%s %s", r.Method, r.URL.Path),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal auth request: %w", err)
	}

	url := fmt.Sprintf("%s/auth/validate", authServiceURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call auth service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}

	var validation AuthValidation
	if err := json.NewDecoder(resp.Body).Decode(&validation); err != nil {
		return nil, fmt.Errorf("decode auth response: %w", err)
	}

	return &validation, nil
}

// getRequiredTier determines the required tier based on the endpoint path.
func getRequiredTier(r *http.Request) string {
	// Use the actual request path
	path := r.URL.Path

	// Pro tier endpoints
	proEndpoints := map[string]bool{
		"/api/v1/dashboard":     true,
		"/api/v1/best-window":   true,
		"/api/v1/energy-action": true,
		"/api/v1/tomorrow":      true,
		"/api/v1/balance":       true,
	}

	if proEndpoints[path] {
		return "pro"
	}

	// Free tier endpoints (default)
	return "free"
}

// GetAuthValidation retrieves the auth validation from the request context.
func GetAuthValidation(r *http.Request) *AuthValidation {
	val := r.Context().Value(ContextKeyAuth)
	if val == nil {
		return nil
	}
	validation, ok := val.(*AuthValidation)
	if !ok {
		return nil
	}
	return validation
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, error, message string) {
	writeJSONErrorWithData(w, status, error, message, nil)
}

// writeJSONErrorWithData writes a JSON error response with additional data.
func writeJSONErrorWithData(w http.ResponseWriter, status int, errorCode, message string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]interface{}{
		"detail":  errorCode,
		"message": message,
	}

	if data != nil {
		for k, v := range data {
			response[k] = v
		}
	}

	json.NewEncoder(w).Encode(response)
}
