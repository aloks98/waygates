package caddy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	// Find backup file in the backup directory (it has a timestamp suffix)
	backupDir := fm.GetBackupDir()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	var backupFound bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "caddy.json.") && strings.HasSuffix(entry.Name(), ".backup") {
			backupFound = true
			backupPath := filepath.Join(backupDir, entry.Name())
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

func TestFileManager_GetBackupDir(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)

	backupDir := fm.GetBackupDir()
	expected := filepath.Join(tempDir, "backup")

	if backupDir != expected {
		t.Errorf("Expected backup dir %s, got %s", expected, backupDir)
	}
}

func TestFileManager_ReadJSONConfig(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	configPath := fm.GetJSONConfigPath()
	expected := []byte(`{"test": true}`)

	// Write config
	err := fm.WriteJSONConfig(configPath, expected)
	if err != nil {
		t.Fatalf("WriteJSONConfig failed: %v", err)
	}

	// Read it back
	data, err := fm.ReadJSONConfig(configPath)
	if err != nil {
		t.Fatalf("ReadJSONConfig failed: %v", err)
	}

	if !bytes.Equal(data, expected) {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

func TestFileManager_ReadJSONConfig_Nonexistent(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)

	// Read a nonexistent file
	data, err := fm.ReadJSONConfig(filepath.Join(tempDir, "nonexistent.json"))
	if err != nil {
		t.Errorf("ReadJSONConfig should not error for nonexistent file: %v", err)
	}
	if data != nil {
		t.Error("Expected nil data for nonexistent file")
	}
}

func TestFileManager_ConfigChanged(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	configPath := fm.GetJSONConfigPath()
	original := []byte(`{"version": 1}`)

	// Write initial config
	err := fm.WriteJSONConfig(configPath, original)
	if err != nil {
		t.Fatalf("WriteJSONConfig failed: %v", err)
	}

	// Check with same content - should not be changed
	changed, err := fm.ConfigChanged(configPath, original)
	if err != nil {
		t.Fatalf("ConfigChanged failed: %v", err)
	}
	if changed {
		t.Error("ConfigChanged should return false for identical content")
	}

	// Check with different content - should be changed
	newContent := []byte(`{"version": 2}`)
	changed, err = fm.ConfigChanged(configPath, newContent)
	if err != nil {
		t.Fatalf("ConfigChanged failed: %v", err)
	}
	if !changed {
		t.Error("ConfigChanged should return true for different content")
	}
}

func TestFileManager_ConfigChanged_Nonexistent(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)

	// Check against nonexistent file - should be considered changed
	changed, err := fm.ConfigChanged(filepath.Join(tempDir, "nonexistent.json"), []byte(`{}`))
	if err != nil {
		t.Fatalf("ConfigChanged failed: %v", err)
	}
	if !changed {
		t.Error("ConfigChanged should return true when file doesn't exist")
	}
}

func TestFileManager_CleanupOldBackupsByAge(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	backupDir := fm.GetBackupDir()

	// Create 5 backups with different ages
	for i := 0; i < 5; i++ {
		filename := fmt.Sprintf("caddy.json.2026010%d_120000.backup", i)
		backupPath := filepath.Join(backupDir, filename)
		if err := os.WriteFile(backupPath, []byte("backup"), 0644); err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}
		// Set modification time: 0,1,2 days old (keep) and 10,11 days old (delete)
		var modTime time.Time
		if i < 3 {
			modTime = time.Now().Add(time.Duration(-i) * 24 * time.Hour)
		} else {
			modTime = time.Now().Add(time.Duration(-10-i) * 24 * time.Hour)
		}
		if err := os.Chtimes(backupPath, modTime, modTime); err != nil {
			t.Fatalf("Failed to set mtime: %v", err)
		}
	}

	// Clean up, keeping only backups from last 7 days
	err := fm.CleanupOldBackupsByAge(7)
	if err != nil {
		t.Fatalf("CleanupOldBackupsByAge failed: %v", err)
	}

	// Count remaining backups
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup dir: %v", err)
	}

	backupCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".backup") {
			backupCount++
		}
	}

	// Should have 3 recent backups remaining
	if backupCount != 3 {
		t.Errorf("Expected 3 backups after cleanup, got %d", backupCount)
	}
}

func TestFileManager_CleanupOldBackupsByAge_NothingToClean(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Cleanup with no backups - should not error
	err := fm.CleanupOldBackupsByAge(7)
	if err != nil {
		t.Errorf("CleanupOldBackupsByAge should not error when no backups exist: %v", err)
	}
}

func TestFileManager_CleanupOldBackupsByAge_DefaultRetention(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Pass 0 to use default retention
	err := fm.CleanupOldBackupsByAge(0)
	if err != nil {
		t.Errorf("CleanupOldBackupsByAge with 0 should use default: %v", err)
	}
}
