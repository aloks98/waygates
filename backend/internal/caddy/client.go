package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with Caddy Admin API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Caddy API client
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
	resp, err := c.httpClient.Get(c.baseURL + "/config/")
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
	resp, err := c.httpClient.Get(c.baseURL + "/config/")
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
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+path,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// PATCH sends a PATCH request to Caddy Admin API
func (c *Client) PATCH(path string, body interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, c.baseURL+path, bytes.NewBuffer(jsonData))
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// PUT sends a PUT request to Caddy Admin API
func (c *Client) PUT(path string, body interface{}) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, bytes.NewBuffer(jsonData))
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// DELETE sends a DELETE request to Caddy Admin API
func (c *Client) DELETE(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
