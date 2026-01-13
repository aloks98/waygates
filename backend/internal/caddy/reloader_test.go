package caddy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Test NewReloader with default configuration
func TestNewReloader_Defaults(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{}

	r := NewReloader(cfg, logger)

	assert.Equal(t, "caddy", r.caddyBinary, "Expected default caddy binary 'caddy'")
	assert.Equal(t, DefaultAdminAPIURL, r.adminAPIURL, "Expected default admin API URL")
}

// Test NewReloader with custom configuration
func TestNewReloader_CustomConfig(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: "/usr/local/bin/caddy",
		AdminAPIURL: "http://localhost:3000",
	}

	r := NewReloader(cfg, logger)

	assert.Equal(t, "/usr/local/bin/caddy", r.caddyBinary, "Expected custom binary path")
	assert.Equal(t, "http://localhost:3000", r.adminAPIURL, "Expected custom admin API URL")
}

// Test GetStatus returns correct status information
func TestReloaderStatus(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: "/usr/bin/caddy",
		AdminAPIURL: "http://localhost:2019",
	}

	r := NewReloader(cfg, logger)
	status := r.GetStatus()

	assert.Equal(t, "/usr/bin/caddy", status.CaddyBinary, "Status CaddyBinary mismatch")
	assert.Equal(t, "http://localhost:2019", status.AdminAPIURL, "Status AdminAPIURL mismatch")
	assert.Equal(t, 0, status.ReloadCount, "Initial reload count should be 0")
	assert.True(t, status.LastReload.IsZero(), "Initial last reload should be zero time")
}

// Test ValidationError implements error interface correctly
func TestValidationError(t *testing.T) {
	t.Parallel()
	err := &ValidationError{
		Message: "invalid syntax",
		Output:  "line 5: unexpected token",
	}

	expected := "configuration validation failed: invalid syntax"
	assert.Equal(t, expected, err.Error(), "ValidationError.Error() mismatch")
}

// Test extractValidationError with various inputs
func TestExtractValidationError(t *testing.T) {
	t.Parallel()
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
			name:     "Empty output",
			input:    "",
			expected: "validation failed with unknown error",
		},
		{
			name:     "Only whitespace",
			input:    "   \n   \n   ",
			expected: "validation failed with unknown error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractValidationError(tc.input)
			assert.Equal(t, tc.expected, result, "extractValidationError result mismatch")
		})
	}
}

// Test ReloadResult fields
func TestReloadResult(t *testing.T) {
	t.Parallel()

	result := &ReloadResult{
		Success:     true,
		Message:     "Configuration reloaded successfully",
		ReloadCount: 5,
	}

	assert.True(t, result.Success, "Expected Success to be true")
	assert.Equal(t, "Configuration reloaded successfully", result.Message)
	assert.Equal(t, 5, result.ReloadCount)
}
