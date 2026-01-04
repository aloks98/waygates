package caddy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewReloader_Defaults(t *testing.T) {
	logger := zap.NewNop()
	cfg := ReloaderConfig{}

	r := NewReloader(cfg, logger)

	if r.caddyBinary != "caddy" {
		t.Errorf("Expected default caddy binary 'caddy', got '%s'", r.caddyBinary)
	}
	if r.caddyfilePath != "/etc/caddy/Caddyfile" {
		t.Errorf("Expected default path '/etc/caddy/Caddyfile', got '%s'", r.caddyfilePath)
	}
}

func TestNewReloader_CustomConfig(t *testing.T) {
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   "/usr/local/bin/caddy",
		CaddyfilePath: "/custom/path/Caddyfile",
	}

	r := NewReloader(cfg, logger)

	if r.caddyBinary != "/usr/local/bin/caddy" {
		t.Errorf("Expected custom binary path, got '%s'", r.caddyBinary)
	}
	if r.caddyfilePath != "/custom/path/Caddyfile" {
		t.Errorf("Expected custom config path, got '%s'", r.caddyfilePath)
	}
}

func TestReloaderStatus(t *testing.T) {
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   "/usr/bin/caddy",
		CaddyfilePath: "/etc/caddy/Caddyfile",
	}

	r := NewReloader(cfg, logger)
	status := r.GetStatus()

	if status.CaddyBinary != "/usr/bin/caddy" {
		t.Errorf("Status CaddyBinary mismatch")
	}
	if status.CaddyfilePath != "/etc/caddy/Caddyfile" {
		t.Errorf("Status CaddyfilePath mismatch")
	}
	if status.ReloadCount != 0 {
		t.Errorf("Initial reload count should be 0")
	}
	if !status.LastReload.IsZero() {
		t.Errorf("Initial last reload should be zero time")
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Message: "invalid syntax",
		Output:  "line 5: unexpected token",
	}

	expected := "caddyfile validation failed: invalid syntax"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestExtractValidationError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Error with prefix",
			input:    "Error: invalid hostname",
			expected: "invalid hostname",
		},
		{
			name:     "error lowercase",
			input:    "error: missing closing brace",
			expected: "missing closing brace",
		},
		{
			name:     "Multi-line with error",
			input:    "Loading config...\nError: syntax error on line 10\nDone",
			expected: "syntax error on line 10",
		},
		{
			name:     "No error prefix",
			input:    "unexpected token at position 5",
			expected: "unexpected token at position 5",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "validation failed with unknown error",
		},
		{
			name:     "Only whitespace",
			input:    "   \n  \n  ",
			expected: "validation failed with unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValidationError(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestReloadResult(t *testing.T) {
	result := ReloadResult{
		Success:     true,
		Message:     "OK",
		Duration:    100 * time.Millisecond,
		ReloadCount: 5,
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}
	if result.ReloadCount != 5 {
		t.Errorf("Expected reload count 5, got %d", result.ReloadCount)
	}
}

// Integration test - only runs if caddy binary is available
func TestReloader_Integration(t *testing.T) {
	// Check if caddy is available
	if _, err := exec.LookPath("caddy"); err != nil {
		t.Skip("Caddy binary not available, skipping integration test")
	}

	logger := zap.NewNop()
	tempDir := t.TempDir()

	// Create a valid Caddyfile
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	validConfig := `{
    admin off
}

:8080 {
    respond "Hello"
}
`
	if err := os.WriteFile(caddyfilePath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write Caddyfile: %v", err)
	}

	cfg := ReloaderConfig{
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()

	// Test validation of valid config
	t.Run("Validate valid config", func(t *testing.T) {
		err := r.Validate(ctx)
		if err != nil {
			t.Errorf("Valid config should pass validation: %v", err)
		}
	})

	// Test validation of invalid config
	t.Run("Validate invalid config", func(t *testing.T) {
		invalidConfig := `{
    invalid_directive
}
`
		if err := os.WriteFile(caddyfilePath, []byte(invalidConfig), 0644); err != nil {
			t.Fatalf("Failed to write invalid Caddyfile: %v", err)
		}

		err := r.Validate(ctx)
		if err == nil {
			t.Error("Invalid config should fail validation")
		}

		// Check it's a ValidationError
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	// Test TestConnection
	t.Run("Test connection", func(t *testing.T) {
		err := r.TestConnection(ctx)
		if err != nil {
			t.Errorf("TestConnection should succeed when caddy is available: %v", err)
		}
	})
}

func TestReloader_Validate_InvalidPath(t *testing.T) {
	// Check if caddy is available
	if _, err := exec.LookPath("caddy"); err != nil {
		t.Skip("Caddy binary not available, skipping test")
	}

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyfilePath: "/nonexistent/path/Caddyfile",
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err := r.Validate(ctx)
	if err == nil {
		t.Error("Validation should fail for nonexistent file")
	}
}

func TestReloader_TestConnection_InvalidBinary(t *testing.T) {
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: "/nonexistent/binary",
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err := r.TestConnection(ctx)
	if err == nil {
		t.Error("TestConnection should fail for invalid binary")
	}
}

func TestReloader_ContextCancellation(t *testing.T) {
	// Check if caddy is available
	if _, err := exec.LookPath("caddy"); err != nil {
		t.Skip("Caddy binary not available, skipping test")
	}

	logger := zap.NewNop()
	tempDir := t.TempDir()

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	validConfig := `{
    admin off
}
`
	if err := os.WriteFile(caddyfilePath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write Caddyfile: %v", err)
	}

	cfg := ReloaderConfig{
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations with canceled context should fail
	err := r.Validate(ctx)
	if err == nil {
		t.Error("Validate should fail with canceled context")
	}
}
