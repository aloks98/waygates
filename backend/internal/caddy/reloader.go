package caddy

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Reloader handles Caddy configuration validation and reloading
type Reloader struct {
	caddyBinary   string
	caddyfilePath string
	logger        *zap.Logger
	mu            sync.Mutex
	lastReload    time.Time
	reloadCount   int
}

// ReloaderConfig holds configuration for the Reloader
type ReloaderConfig struct {
	CaddyBinary   string // Path to caddy binary (default: "caddy")
	CaddyfilePath string // Path to Caddyfile (default: "/etc/caddy/Caddyfile")
}

// NewReloader creates a new Reloader
func NewReloader(cfg ReloaderConfig, logger *zap.Logger) *Reloader {
	if cfg.CaddyBinary == "" {
		cfg.CaddyBinary = "caddy"
	}
	if cfg.CaddyfilePath == "" {
		cfg.CaddyfilePath = "/etc/caddy/Caddyfile"
	}

	return &Reloader{
		caddyBinary:   cfg.CaddyBinary,
		caddyfilePath: cfg.CaddyfilePath,
		logger:        logger,
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
		LastReload:    r.lastReload,
		ReloadCount:   r.reloadCount,
	}
}

// ReloaderStatus contains status information about the reloader
type ReloaderStatus struct {
	CaddyBinary   string
	CaddyfilePath string
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
