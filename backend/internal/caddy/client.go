package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Common Caddy API paths
const (
	// RoutesPath is the path to the routes array in Caddy config
	RoutesPath = "/config/apps/http/servers/srv0/routes"
)

// Client handles communication with Caddy Admin API.
//
// Thread Safety:
// Client is safe for concurrent use by multiple goroutines. The underlying
// http.Client is thread-safe, and Client itself contains no mutable state
// after construction. All methods can be called concurrently without
// additional synchronization.
//
// However, callers should be aware that Caddy Admin API operations are not
// atomic. When multiple goroutines modify routes concurrently, they should
// coordinate externally to avoid race conditions (e.g., one goroutine's
// changes overwriting another's). The ProxyService handles this by reading
// current routes and applying updates atomically.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Caddy API client.
// The returned client is safe for concurrent use.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// HealthCheck checks if Caddy Admin API is reachable
func (c *Client) HealthCheck() error {
	return c.HealthCheckWithContext(context.Background())
}

// HealthCheckWithContext checks if Caddy Admin API is reachable with context
func (c *Client) HealthCheckWithContext(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/config/", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Caddy Admin API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Caddy Admin API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetConfig retrieves the current Caddy configuration
func (c *Client) GetConfig() (map[string]interface{}, error) {
	return c.GetConfigWithContext(context.Background())
}

// GetConfigWithContext retrieves the current Caddy configuration with context
func (c *Client) GetConfigWithContext(ctx context.Context) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/config/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get config: status %d", resp.StatusCode)
	}

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

// POST sends a POST request to Caddy Admin API
func (c *Client) POST(path string, body interface{}) error {
	return c.POSTWithContext(context.Background(), path, body)
}

// POSTWithContext sends a POST request to Caddy Admin API with context
func (c *Client) POSTWithContext(ctx context.Context, path string, body interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("caddy API error (status %d): failed to read response body: %v", resp.StatusCode, readErr)
		}
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// PATCH sends a PATCH request to Caddy Admin API
func (c *Client) PATCH(path string, body interface{}) error {
	return c.PATCHWithContext(context.Background(), path, body)
}

// PATCHWithContext sends a PATCH request to Caddy Admin API with context
func (c *Client) PATCHWithContext(ctx context.Context, path string, body interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("caddy API error (status %d): failed to read response body: %v", resp.StatusCode, readErr)
		}
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// PUT sends a PUT request to Caddy Admin API
func (c *Client) PUT(path string, body interface{}) error {
	return c.PUTWithContext(context.Background(), path, body)
}

// PUTWithContext sends a PUT request to Caddy Admin API with context
func (c *Client) PUTWithContext(ctx context.Context, path string, body interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("caddy API error (status %d): failed to read response body: %v", resp.StatusCode, readErr)
		}
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// DELETE sends a DELETE request to Caddy Admin API
func (c *Client) DELETE(path string) error {
	return c.DELETEWithContext(context.Background(), path)
}

// DELETEWithContext sends a DELETE request to Caddy Admin API with context
func (c *Client) DELETEWithContext(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("caddy API error (status %d): failed to read response body: %v", resp.StatusCode, readErr)
		}
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
