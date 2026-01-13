package caddy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// DefaultAdminAPIURL is the default Caddy admin API endpoint
	DefaultAdminAPIURL = "http://localhost:2019"
)

// Reloader handles Caddy configuration validation and reloading
type Reloader struct {
	caddyBinary string
	adminAPIURL string
	httpClient  *http.Client
	logger      *zap.Logger
	mu          sync.Mutex
	lastReload  time.Time
	reloadCount int
}

// ReloaderConfig holds configuration for the Reloader
type ReloaderConfig struct {
	CaddyBinary string // Path to caddy binary (default: "caddy")
	AdminAPIURL string // Caddy admin API URL (default: "http://localhost:2019")
}

// NewReloader creates a new Reloader
func NewReloader(cfg ReloaderConfig, logger *zap.Logger) *Reloader {
	if cfg.CaddyBinary == "" {
		cfg.CaddyBinary = "caddy"
	}
	if cfg.AdminAPIURL == "" {
		cfg.AdminAPIURL = DefaultAdminAPIURL
	}

	return &Reloader{
		caddyBinary: cfg.CaddyBinary,
		adminAPIURL: cfg.AdminAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Message string
	Output  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("configuration validation failed: %s", e.Message)
}

// ReloadResult contains the result of a reload operation
type ReloadResult struct {
	Success     bool
	Message     string
	Duration    time.Duration
	ReloadCount int
}

// GetStatus returns the current reloader status
func (r *Reloader) GetStatus() ReloaderStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	return ReloaderStatus{
		CaddyBinary: r.caddyBinary,
		AdminAPIURL: r.adminAPIURL,
		LastReload:  r.lastReload,
		ReloadCount: r.reloadCount,
	}
}

// ReloaderStatus contains status information about the reloader
type ReloaderStatus struct {
	CaddyBinary string
	AdminAPIURL string
	LastReload  time.Time
	ReloadCount int
}

// extractValidationError extracts a clean error message from Caddy output
func extractValidationError(output string) string {
	// Look for common error patterns
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Error:") {
			return strings.TrimPrefix(line, "Error: ")
		}
		if strings.Contains(line, "error:") {
			return strings.TrimPrefix(line, "error: ")
		}
	}

	// If no specific error found, return the first non-empty line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return "validation failed with unknown error"
}

// TestConnection tests if Caddy is running and responding
func (r *Reloader) TestConnection(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, r.caddyBinary, "version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("caddy not available: %w", err)
	}

	r.logger.Debug("Caddy connection test successful",
		zap.String("version", strings.TrimSpace(stdout.String())))

	return nil
}

// ValidateJSON validates a JSON configuration file without reloading
func (r *Reloader) ValidateJSON(configPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Debug("Validating JSON configuration", zap.String("path", configPath))

	cmd := exec.Command(r.caddyBinary, "validate", "--config", configPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}

		r.logger.Error("JSON configuration validation failed",
			zap.String("path", configPath),
			zap.String("output", output),
			zap.Error(err))

		return &ValidationError{
			Message: extractValidationError(output),
			Output:  output,
		}
	}

	r.logger.Debug("JSON configuration validation successful", zap.String("path", configPath))
	return nil
}

// ReloadJSON reloads Caddy with a JSON configuration file using the admin API
func (r *Reloader) ReloadJSON(ctx context.Context, configPath string) (*ReloadResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()

	// Validate the JSON configuration first
	r.logger.Debug("Validating JSON configuration before reload", zap.String("path", configPath))

	validateCmd := exec.CommandContext(ctx, r.caddyBinary, "validate", "--config", configPath)
	var validateStdout, validateStderr bytes.Buffer
	validateCmd.Stdout = &validateStdout
	validateCmd.Stderr = &validateStderr

	if err := validateCmd.Run(); err != nil {
		output := strings.TrimSpace(validateStderr.String())
		if output == "" {
			output = strings.TrimSpace(validateStdout.String())
		}

		r.logger.Error("JSON configuration validation failed",
			zap.String("path", configPath),
			zap.String("output", output),
			zap.Error(err))

		return nil, &ValidationError{
			Message: extractValidationError(output),
			Output:  output,
		}
	}

	// Read the JSON configuration file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		r.logger.Error("Failed to read JSON configuration file",
			zap.String("path", configPath),
			zap.Error(err))
		return nil, fmt.Errorf("reading JSON config file: %w", err)
	}

	// Load configuration via Caddy admin API
	r.logger.Info("Reloading Caddy via admin API",
		zap.String("path", configPath),
		zap.String("admin_url", r.adminAPIURL))

	loadURL := fmt.Sprintf("%s/load", r.adminAPIURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loadURL, bytes.NewReader(configData))
	if err != nil {
		r.logger.Error("Failed to create admin API request", zap.Error(err))
		return nil, fmt.Errorf("creating admin API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.logger.Error("Failed to send request to Caddy admin API",
			zap.String("url", loadURL),
			zap.Error(err))
		return nil, fmt.Errorf("sending request to admin API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		r.logger.Error("Caddy admin API returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)))
		return nil, fmt.Errorf("admin API returned status %d: %s", resp.StatusCode, string(body))
	}

	r.lastReload = time.Now()
	r.reloadCount++
	duration := time.Since(start)

	r.logger.Info("Caddy JSON configuration reloaded successfully via admin API",
		zap.String("path", configPath),
		zap.Duration("duration", duration),
		zap.Int("reload_count", r.reloadCount))

	return &ReloadResult{
		Success:     true,
		Message:     "JSON configuration reloaded successfully via admin API",
		Duration:    duration,
		ReloadCount: r.reloadCount,
	}, nil
}
