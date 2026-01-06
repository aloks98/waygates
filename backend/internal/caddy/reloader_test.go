package caddy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createMockCaddyScript creates a mock caddy script that can be configured
// to return specific outputs and exit codes for different commands
func createMockCaddyScript(t *testing.T, behavior map[string]struct {
	stdout   string
	stderr   string
	exitCode int
}) string {
	t.Helper()
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")

	// Build the script content based on behavior configuration
	var cases strings.Builder
	for cmd, b := range behavior {
		cases.WriteString("    \"" + cmd + "\")\n")
		if b.stdout != "" {
			cases.WriteString("        echo \"" + b.stdout + "\"\n")
		}
		if b.stderr != "" {
			cases.WriteString("        echo \"" + b.stderr + "\" >&2\n")
		}
		cases.WriteString("        exit " + string(rune('0'+b.exitCode)) + "\n")
		cases.WriteString("        ;;\n")
	}

	script := `#!/bin/bash
case "$1" in
` + cases.String() + `    *)
        echo "Unknown command: $1" >&2
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err, "Failed to create mock caddy script")

	return scriptPath
}

// createMockCaddyScriptWithExitCode creates a simple mock script with configurable exit code
func createMockCaddyScriptWithExitCode(t *testing.T, exitCode int, stdout, stderr string) string {
	t.Helper()
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")

	script := `#!/bin/bash
`
	if stdout != "" {
		script += `echo "` + stdout + `"
`
	}
	if stderr != "" {
		script += `echo "` + stderr + `" >&2
`
	}
	script += `exit ` + string(rune('0'+exitCode)) + `
`

	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err, "Failed to create mock caddy script")

	return scriptPath
}

// Test NewReloader with default configuration
func TestNewReloader_Defaults(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{}

	r := NewReloader(cfg, logger)

	assert.Equal(t, "caddy", r.caddyBinary, "Expected default caddy binary 'caddy'")
	assert.Equal(t, "/etc/caddy/Caddyfile", r.caddyfilePath, "Expected default path '/etc/caddy/Caddyfile'")
}

// Test NewReloader with custom configuration
func TestNewReloader_CustomConfig(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   "/usr/local/bin/caddy",
		CaddyfilePath: "/custom/path/Caddyfile",
	}

	r := NewReloader(cfg, logger)

	assert.Equal(t, "/usr/local/bin/caddy", r.caddyBinary, "Expected custom binary path")
	assert.Equal(t, "/custom/path/Caddyfile", r.caddyfilePath, "Expected custom config path")
}

// Test GetStatus returns correct status information
func TestReloaderStatus(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   "/usr/bin/caddy",
		CaddyfilePath: "/etc/caddy/Caddyfile",
	}

	r := NewReloader(cfg, logger)
	status := r.GetStatus()

	assert.Equal(t, "/usr/bin/caddy", status.CaddyBinary, "Status CaddyBinary mismatch")
	assert.Equal(t, "/etc/caddy/Caddyfile", status.CaddyfilePath, "Status CaddyfilePath mismatch")
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

	expected := "caddyfile validation failed: invalid syntax"
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
			name:     "Empty string",
			input:    "",
			expected: "validation failed with unknown error",
		},
		{
			name:     "Only whitespace",
			input:    "   \n  \n  ",
			expected: "validation failed with unknown error",
		},
		{
			name:     "Error in middle of output",
			input:    "INFO starting caddy\nError: adapter error\nINFO done",
			expected: "adapter error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractValidationError(tt.input)
			assert.Equal(t, tt.expected, result, "extractValidationError mismatch")
		})
	}
}

// Test ReloadResult struct
func TestReloadResult(t *testing.T) {
	t.Parallel()
	result := ReloadResult{
		Success:     true,
		Message:     "OK",
		Duration:    100 * time.Millisecond,
		ReloadCount: 5,
	}

	assert.True(t, result.Success, "Expected success to be true")
	assert.Equal(t, 5, result.ReloadCount, "Expected reload count 5")
	assert.Equal(t, "OK", result.Message, "Expected message 'OK'")
	assert.Equal(t, 100*time.Millisecond, result.Duration, "Expected duration 100ms")
}

// Test Validate with successful validation
func TestReloader_Validate_Success(t *testing.T) {
	t.Parallel()

	// Create a mock caddy script that returns success for validate
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        echo "Valid configuration"
        exit 0
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	// Create a mock Caddyfile
	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte(":8080 { respond \"Hello\" }"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err = r.Validate(ctx)
	assert.NoError(t, err, "Validate should succeed with valid config")
}

// Test Validate with validation failure
func TestReloader_Validate_Failure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        echo "Error: invalid directive on line 5" >&2
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("invalid config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err = r.Validate(ctx)

	require.Error(t, err, "Validate should fail with invalid config")

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr), "Error should be ValidationError")
	assert.Contains(t, validationErr.Message, "invalid directive", "Error message should contain directive info")
}

// Test Validate with stdout error output (when stderr is empty)
func TestReloader_Validate_StdoutError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        echo "Error: syntax error"
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("bad config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err = r.Validate(ctx)

	require.Error(t, err)
	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Contains(t, validationErr.Output, "syntax error")
}

// Test Validate with invalid binary path
func TestReloader_Validate_InvalidBinary(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   "/nonexistent/binary/caddy",
		CaddyfilePath: "/tmp/Caddyfile",
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err := r.Validate(ctx)

	require.Error(t, err, "Validate should fail with nonexistent binary")
}

// Test Validate with nonexistent Caddyfile path
func TestReloader_Validate_InvalidPath(t *testing.T) {
	t.Parallel()

	// Check if caddy is available for this test
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
	require.Error(t, err, "Validation should fail for nonexistent file")
}

// Test Validate with context cancellation
func TestReloader_Validate_ContextCancellation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	// Create a script that sleeps to allow cancellation
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
sleep 10
exit 0
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = r.Validate(ctx)
	require.Error(t, err, "Validate should fail with canceled context")
}

// Test Reload with successful reload
func TestReloader_Reload_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        echo "Valid configuration"
        exit 0
        ;;
    "reload")
        echo "Configuration reloaded"
        exit 0
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte(":8080 { respond \"Hello\" }"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	result, err := r.Reload(ctx)

	require.NoError(t, err, "Reload should succeed")
	require.NotNil(t, result, "Result should not be nil")
	assert.True(t, result.Success, "Result.Success should be true")
	assert.Equal(t, 1, result.ReloadCount, "ReloadCount should be 1")
	assert.Greater(t, result.Duration, time.Duration(0), "Duration should be positive")

	// Verify status is updated
	status := r.GetStatus()
	assert.Equal(t, 1, status.ReloadCount, "Status ReloadCount should be 1")
	assert.False(t, status.LastReload.IsZero(), "LastReload should be set")
}

// Test Reload increments reload count correctly
func TestReloader_Reload_IncrementCount(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
exit 0
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()

	// Reload multiple times
	for i := 1; i <= 3; i++ {
		result, err := r.Reload(ctx)
		require.NoError(t, err)
		assert.Equal(t, i, result.ReloadCount, "ReloadCount should increment")
	}

	status := r.GetStatus()
	assert.Equal(t, 3, status.ReloadCount, "Final ReloadCount should be 3")
}

// Test Reload with validation failure
func TestReloader_Reload_ValidationFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        echo "Error: invalid config" >&2
        exit 1
        ;;
    "reload")
        exit 0
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("bad config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	result, err := r.Reload(ctx)

	require.Error(t, err, "Reload should fail when validation fails")
	assert.Nil(t, result, "Result should be nil on failure")

	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr), "Error should be ValidationError")
}

// Test Reload with reload command failure
func TestReloader_Reload_ReloadFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        exit 0
        ;;
    "reload")
        echo "Error: could not reload config" >&2
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	result, err := r.Reload(ctx)

	require.Error(t, err, "Reload should fail when reload command fails")
	assert.Nil(t, result, "Result should be nil on failure")
	assert.Contains(t, err.Error(), "caddy reload failed", "Error should indicate reload failure")
}

// Test Reload with stdout error (stderr empty)
func TestReloader_Reload_StdoutError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        exit 0
        ;;
    "reload")
        echo "reload error from stdout"
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	result, err := r.Reload(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reload error from stdout")
}

// Test ForceReload with successful reload
func TestReloader_ForceReload_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "reload")
        echo "Force reload successful"
        exit 0
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	result, err := r.ForceReload(ctx)

	require.NoError(t, err, "ForceReload should succeed")
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 1, result.ReloadCount)
	assert.Contains(t, result.Message, "force reloaded")
}

// Test ForceReload with failure
func TestReloader_ForceReload_Failure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "reload")
        echo "Error: force reload failed" >&2
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	result, err := r.ForceReload(ctx)

	require.Error(t, err, "ForceReload should fail")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "caddy force reload failed")
}

// Test ForceReload with stdout error
func TestReloader_ForceReload_StdoutError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "reload")
        echo "stdout error message"
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	result, err := r.ForceReload(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "stdout error message")
}

// Test AdaptAndReload with successful execution
func TestReloader_AdaptAndReload_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "adapt")
        echo '{"apps":{"http":{}}}'
        exit 0
        ;;
    "reload")
        exit 0
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte(":8080 { respond \"Hello\" }"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	jsonConfig, err := r.AdaptAndReload(ctx)

	require.NoError(t, err, "AdaptAndReload should succeed")
	assert.Contains(t, jsonConfig, "apps", "JSON config should contain apps")

	// Verify reload count is incremented
	status := r.GetStatus()
	assert.Equal(t, 1, status.ReloadCount)
}

// Test AdaptAndReload with adapt failure
func TestReloader_AdaptAndReload_AdaptFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "adapt")
        echo "Error: failed to adapt config" >&2
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("bad config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	jsonConfig, err := r.AdaptAndReload(ctx)

	require.Error(t, err, "AdaptAndReload should fail when adapt fails")
	assert.Empty(t, jsonConfig)
	assert.Contains(t, err.Error(), "failed to adapt Caddyfile")
}

// Test AdaptAndReload with reload failure after successful adapt
func TestReloader_AdaptAndReload_ReloadFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "adapt")
        echo '{"apps":{}}'
        exit 0
        ;;
    "reload")
        echo "Error: reload failed after adapt" >&2
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	jsonConfig, err := r.AdaptAndReload(ctx)

	require.Error(t, err, "AdaptAndReload should fail when reload fails")
	// JSON config should still be returned even though reload failed
	assert.Contains(t, jsonConfig, "apps")
	assert.Contains(t, err.Error(), "failed to reload after adapt")
}

// Test TestConnection with successful connection
func TestReloader_TestConnection_Success(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "version")
        echo "v2.7.5"
        exit 0
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: scriptPath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err = r.TestConnection(ctx)

	assert.NoError(t, err, "TestConnection should succeed")
}

// Test TestConnection with invalid binary
func TestReloader_TestConnection_InvalidBinary(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: "/nonexistent/binary",
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err := r.TestConnection(ctx)

	require.Error(t, err, "TestConnection should fail for invalid binary")
	assert.Contains(t, err.Error(), "caddy not available")
}

// Test TestConnection with failing caddy command
func TestReloader_TestConnection_CommandFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
exit 1
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: scriptPath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	err = r.TestConnection(ctx)

	require.Error(t, err, "TestConnection should fail when command fails")
}

// Test concurrent access to Reloader (thread safety)
func TestReloader_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
exit 0
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 10

	// Run multiple concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			_ = r.Validate(ctx)
		}()

		go func() {
			defer wg.Done()
			_, _ = r.Reload(ctx)
		}()

		go func() {
			defer wg.Done()
			_ = r.GetStatus()
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	status := r.GetStatus()
	assert.GreaterOrEqual(t, status.ReloadCount, 0, "ReloadCount should be non-negative")
}

// Test context timeout during Validate
func TestReloader_Validate_ContextTimeout(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	// Script that sleeps longer than context timeout
	script := `#!/bin/bash
sleep 5
exit 0
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = r.Validate(ctx)
	require.Error(t, err, "Validate should fail with context timeout")
}

// Test context timeout during Reload
func TestReloader_Reload_ContextTimeout(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	// Script that completes validation but sleeps during reload
	script := `#!/bin/bash
case "$1" in
    "validate")
        exit 0
        ;;
    "reload")
        sleep 5
        exit 0
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := r.Reload(ctx)
	require.Error(t, err, "Reload should fail with context timeout")
	assert.Nil(t, result)
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
	err := os.WriteFile(caddyfilePath, []byte(validConfig), 0644)
	require.NoError(t, err, "Failed to write Caddyfile")

	cfg := ReloaderConfig{
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()

	// Test validation of valid config
	t.Run("Validate valid config", func(t *testing.T) {
		err := r.Validate(ctx)
		assert.NoError(t, err, "Valid config should pass validation")
	})

	// Test validation of invalid config
	t.Run("Validate invalid config", func(t *testing.T) {
		invalidConfig := `{
    invalid_directive
}
`
		err := os.WriteFile(caddyfilePath, []byte(invalidConfig), 0644)
		require.NoError(t, err, "Failed to write invalid Caddyfile")

		err = r.Validate(ctx)
		require.Error(t, err, "Invalid config should fail validation")

		// Check it's a ValidationError
		var validationErr *ValidationError
		assert.True(t, errors.As(err, &validationErr), "Expected ValidationError")
	})

	// Test TestConnection
	t.Run("Test connection", func(t *testing.T) {
		err := r.TestConnection(ctx)
		assert.NoError(t, err, "TestConnection should succeed when caddy is available")
	})
}

// Test Reload does not update state on failure
func TestReloader_Reload_NoStateUpdateOnFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "validate")
        exit 0
        ;;
    "reload")
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()

	// Get initial status
	initialStatus := r.GetStatus()

	// Attempt reload (will fail)
	_, err = r.Reload(ctx)
	require.Error(t, err)

	// Verify state was not updated
	finalStatus := r.GetStatus()
	assert.Equal(t, initialStatus.ReloadCount, finalStatus.ReloadCount, "ReloadCount should not change on failure")
	assert.Equal(t, initialStatus.LastReload, finalStatus.LastReload, "LastReload should not change on failure")
}

// Test ForceReload does not update state on failure
func TestReloader_ForceReload_NoStateUpdateOnFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
exit 1
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()

	// Get initial status
	initialStatus := r.GetStatus()

	// Attempt force reload (will fail)
	_, err = r.ForceReload(ctx)
	require.Error(t, err)

	// Verify state was not updated
	finalStatus := r.GetStatus()
	assert.Equal(t, initialStatus.ReloadCount, finalStatus.ReloadCount, "ReloadCount should not change on failure")
	assert.Equal(t, initialStatus.LastReload, finalStatus.LastReload, "LastReload should not change on failure")
}

// Test AdaptAndReload does not update state on adapt failure
func TestReloader_AdaptAndReload_NoStateUpdateOnAdaptFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "caddy")
	script := `#!/bin/bash
case "$1" in
    "adapt")
        exit 1
        ;;
    *)
        exit 1
        ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	caddyfilePath := filepath.Join(tempDir, "Caddyfile")
	err = os.WriteFile(caddyfilePath, []byte("config"), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary:   scriptPath,
		CaddyfilePath: caddyfilePath,
	}
	r := NewReloader(cfg, logger)

	ctx := context.Background()

	// Get initial status
	initialStatus := r.GetStatus()

	// Attempt adapt and reload (will fail)
	_, err = r.AdaptAndReload(ctx)
	require.Error(t, err)

	// Verify state was not updated
	finalStatus := r.GetStatus()
	assert.Equal(t, initialStatus.ReloadCount, finalStatus.ReloadCount, "ReloadCount should not change on failure")
}
