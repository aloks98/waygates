package caddy

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestFileManager_EnsureDirectories(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	err := fm.EnsureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories failed: %v", err)
	}

	// Check all directories exist
	dirs := []string{
		tempDir,
		filepath.Join(tempDir, "backup"),
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory %s does not exist: %v", dir, err)
		} else if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestFileManager_GetJSONConfigPath(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	expected := filepath.Join(tempDir, "caddy.json")
	result := fm.GetJSONConfigPath()

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFileManager_WriteJSONConfig(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	configPath := fm.GetJSONConfigPath()
	configData := []byte(`{"admin": {"listen": "localhost:2019"}}`)

	err := fm.WriteJSONConfig(configPath, configData)
	if err != nil {
		t.Fatalf("WriteJSONConfig failed: %v", err)
	}

	// Verify file contents
	read, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	if !bytes.Equal(read, configData) {
		t.Errorf("Content mismatch.\nExpected:\n%s\nGot:\n%s", configData, read)
	}
}

func TestFileManager_WriteJSONConfig_CustomPath(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	customPath := filepath.Join(tempDir, "custom", "config.json")
	configData := []byte(`{"test": true}`)

	err := fm.WriteJSONConfig(customPath, configData)
	if err != nil {
		t.Fatalf("WriteJSONConfig failed: %v", err)
	}

	// Verify parent directory was created and content is correct
	read, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	if !bytes.Equal(read, configData) {
		t.Errorf("Content mismatch")
	}
}

func TestFileManager_BackupJSONConfig(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	configPath := fm.GetJSONConfigPath()
	originalData := []byte(`{"original": true}`)

	// Create original config
	err := fm.WriteJSONConfig(configPath, originalData)
	if err != nil {
		t.Fatalf("WriteJSONConfig failed: %v", err)
	}

	// Create backup
	err = fm.BackupJSONConfig(configPath)
	if err != nil {
		t.Fatalf("BackupJSONConfig failed: %v", err)
	}

	// Find backup file (it has a timestamp suffix)
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	var backupFound bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "caddy.json.") && strings.HasSuffix(entry.Name(), ".backup") {
			backupFound = true
			backupPath := filepath.Join(tempDir, entry.Name())
			backupData, _ := os.ReadFile(backupPath)
			if !bytes.Equal(backupData, originalData) {
				t.Error("Backup content doesn't match original")
			}
			break
		}
	}

	if !backupFound {
		t.Error("Backup file not found")
	}
}

func TestFileManager_BackupJSONConfig_NonexistentFile(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Backup a file that doesn't exist - should not error
	err := fm.BackupJSONConfig(filepath.Join(tempDir, "nonexistent.json"))
	if err != nil {
		t.Errorf("BackupJSONConfig should not error for nonexistent file: %v", err)
	}
}

func TestFileManager_FileExists(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	path := filepath.Join(tempDir, "test.txt")

	if fm.FileExists(path) {
		t.Error("FileExists should return false for nonexistent file")
	}

	_ = os.WriteFile(path, []byte("test"), 0644)

	if !fm.FileExists(path) {
		t.Error("FileExists should return true for existing file")
	}
}

func TestFileManager_AtomicWrite(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Write a large file to test atomicity
	configPath := fm.GetJSONConfigPath()
	content := []byte(strings.Repeat(`{"test": "data"}`, 1000))
	err := fm.WriteJSONConfig(configPath, content)
	if err != nil {
		t.Fatalf("WriteJSONConfig failed: %v", err)
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(tempDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp_") {
			t.Errorf("Temp file left behind: %s", entry.Name())
		}
	}

	// Verify file has correct permissions
	info, _ := os.Stat(configPath)
	if info.Mode().Perm() != 0644 {
		t.Errorf("Expected permissions 0644, got %v", info.Mode().Perm())
	}
}

func TestFileManager_WriteJSONConfig_OverwriteExisting(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	configPath := fm.GetJSONConfigPath()

	// Write initial content
	initial := []byte(`{"version": 1}`)
	err := fm.WriteJSONConfig(configPath, initial)
	if err != nil {
		t.Fatalf("First WriteJSONConfig failed: %v", err)
	}

	// Overwrite with new content
	updated := []byte(`{"version": 2}`)
	err = fm.WriteJSONConfig(configPath, updated)
	if err != nil {
		t.Fatalf("Second WriteJSONConfig failed: %v", err)
	}

	// Verify new content
	read, _ := os.ReadFile(configPath)
	if !bytes.Equal(read, updated) {
		t.Errorf("Expected updated content, got: %s", read)
	}
}
