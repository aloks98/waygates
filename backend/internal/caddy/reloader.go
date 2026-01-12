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
	caddyBinary   string
	caddyfilePath string
	adminAPIURL   string
	httpClient    *http.Client
	logger        *zap.Logger
	mu            sync.Mutex
	lastReload    time.Time
	reloadCount   int
}

// ReloaderConfig holds configuration for the Reloader
type ReloaderConfig struct {
	CaddyBinary   string // Path to caddy binary (default: "caddy")
	CaddyfilePath string // Path to Caddyfile (default: "/etc/caddy/Caddyfile")
	AdminAPIURL   string // Caddy admin API URL (default: "http://localhost:2019")
}

// NewReloader creates a new Reloader
func NewReloader(cfg ReloaderConfig, logger *zap.Logger) *Reloader {
	if cfg.CaddyBinary == "" {
		cfg.CaddyBinary = "caddy"
	}
	if cfg.CaddyfilePath == "" {
		cfg.CaddyfilePath = "/etc/caddy/Caddyfile"
	}
	if cfg.AdminAPIURL == "" {
		cfg.AdminAPIURL = DefaultAdminAPIURL
	}

	return &Reloader{
		caddyBinary:   cfg.CaddyBinary,
		caddyfilePath: cfg.CaddyfilePath,
		adminAPIURL:   cfg.AdminAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ValidationError represents a Caddyfile validation error
type ValidationError struct {
	Message string
	Output  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("caddyfile validation failed: %s", e.Message)
}

// ReloadResult contains the result of a reload operation
type ReloadResult struct {
	Success     bool
	Message     string
	Duration    time.Duration
	ReloadCount int
}

// Validate validates the Caddyfile without reloading
func (r *Reloader) Validate(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.validateInternal(ctx)
}

// validateInternal performs validation (must be called with lock held)
func (r *Reloader) validateInternal(ctx context.Context) error {
	r.logger.Debug("Validating Caddyfile", zap.String("path", r.caddyfilePath))

	cmd := exec.CommandContext(ctx, r.caddyBinary, "validate", "--config", r.caddyfilePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}

		r.logger.Error("Caddyfile validation failed",
			zap.String("path", r.caddyfilePath),
			zap.String("output", output),
			zap.Error(err))

		return &ValidationError{
			Message: extractValidationError(output),
			Output:  output,
		}
	}

	r.logger.Debug("Caddyfile validation successful")
	return nil
}

// Reload validates and reloads the Caddy configuration
func (r *Reloader) Reload(ctx context.Context) (*ReloadResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()

	// Validate first
	if err := r.validateInternal(ctx); err != nil {
		return nil, err
	}

	// Reload
	r.logger.Info("Reloading Caddy configuration", zap.String("path", r.caddyfilePath))

	cmd := exec.CommandContext(ctx, r.caddyBinary, "reload", "--config", r.caddyfilePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}

		r.logger.Error("Caddy reload failed",
			zap.String("path", r.caddyfilePath),
			zap.String("output", output),
			zap.Error(err))

		return nil, fmt.Errorf("caddy reload failed: %s", output)
	}

	r.lastReload = time.Now()
	r.reloadCount++
	duration := time.Since(start)

	r.logger.Info("Caddy configuration reloaded successfully",
		zap.Duration("duration", duration),
		zap.Int("reload_count", r.reloadCount))

	return &ReloadResult{
		Success:     true,
		Message:     "Configuration reloaded successfully",
		Duration:    duration,
		ReloadCount: r.reloadCount,
	}, nil
}

// ForceReload reloads without validation (use with caution)
func (r *Reloader) ForceReload(ctx context.Context) (*ReloadResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()

	r.logger.Warn("Force reloading Caddy configuration (skipping validation)")

	cmd := exec.CommandContext(ctx, r.caddyBinary, "reload", "--config", r.caddyfilePath, "--force")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}

		r.logger.Error("Caddy force reload failed",
			zap.String("output", output),
			zap.Error(err))

		return nil, fmt.Errorf("caddy force reload failed: %s", output)
	}

	r.lastReload = time.Now()
	r.reloadCount++
	duration := time.Since(start)

	r.logger.Info("Caddy configuration force reloaded",
		zap.Duration("duration", duration))

	return &ReloadResult{
		Success:     true,
		Message:     "Configuration force reloaded",
		Duration:    duration,
		ReloadCount: r.reloadCount,
	}, nil
}

// GetStatus returns the current reloader status
func (r *Reloader) GetStatus() ReloaderStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	return ReloaderStatus{
		CaddyBinary:   r.caddyBinary,
		CaddyfilePath: r.caddyfilePath,
		AdminAPIURL:   r.adminAPIURL,
		LastReload:    r.lastReload,
		ReloadCount:   r.reloadCount,
	}
}

// ReloaderStatus contains status information about the reloader
type ReloaderStatus struct {
	CaddyBinary   string
	CaddyfilePath string
	AdminAPIURL   string
	LastReload    time.Time
	ReloadCount   int
}

// AdaptAndReload adapts a Caddyfile to JSON format and reloads
// This is useful for debugging or when you need the JSON output
func (r *Reloader) AdaptAndReload(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Adapt Caddyfile to JSON
	r.logger.Debug("Adapting Caddyfile to JSON")

	cmd := exec.CommandContext(ctx, r.caddyBinary, "adapt", "--config", r.caddyfilePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		output := strings.TrimSpace(stderr.String())
		return "", fmt.Errorf("failed to adapt Caddyfile: %s", output)
	}

	jsonConfig := strings.TrimSpace(stdout.String())

	// Now reload
	reloadCmd := exec.CommandContext(ctx, r.caddyBinary, "reload", "--config", r.caddyfilePath)
	reloadCmd.Stdout = &bytes.Buffer{}
	reloadCmd.Stderr = &stderr

	if err := reloadCmd.Run(); err != nil {
		output := strings.TrimSpace(stderr.String())
		return jsonConfig, fmt.Errorf("failed to reload after adapt: %s", output)
	}

	r.lastReload = time.Now()
	r.reloadCount++

	return jsonConfig, nil
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
	defer resp.Body.Close()

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
