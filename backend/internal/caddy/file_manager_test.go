package caddy

import (
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
		filepath.Join(tempDir, "sites"),
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

func TestFileManager_WriteMainCaddyfile(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	content := `{
    admin off
    email test@example.com
}

import sites/*.conf
import catchall.conf
`

	err := fm.WriteMainCaddyfile(content)
	if err != nil {
		t.Fatalf("WriteMainCaddyfile failed: %v", err)
	}

	// Verify file contents
	read, err := os.ReadFile(fm.GetCaddyfilePath())
	if err != nil {
		t.Fatalf("Failed to read Caddyfile: %v", err)
	}

	if string(read) != content {
		t.Errorf("Content mismatch.\nExpected:\n%s\nGot:\n%s", content, string(read))
	}
}

func TestFileManager_WriteProxyFile(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	filename := "1_api.conf"
	content := `api.example.com {
    reverse_proxy localhost:8080
}
`

	err := fm.WriteProxyFile(filename, content)
	if err != nil {
		t.Fatalf("WriteProxyFile failed: %v", err)
	}

	// Verify file contents
	path := filepath.Join(fm.GetSitesDir(), filename)
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read proxy file: %v", err)
	}

	if string(read) != content {
		t.Errorf("Content mismatch.\nExpected:\n%s\nGot:\n%s", content, string(read))
	}
}

func TestFileManager_DeleteProxyFile(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	filename := "1_api.conf"
	content := "test content"

	// Create enabled file
	_ = fm.WriteProxyFile(filename, content)

	// Delete it
	err := fm.DeleteProxyFile(filename)
	if err != nil {
		t.Fatalf("DeleteProxyFile failed: %v", err)
	}

	// Verify file is gone
	path := filepath.Join(fm.GetSitesDir(), filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("File should not exist after deletion")
	}
}

func TestFileManager_DeleteProxyFile_Disabled(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	filename := "1_api.conf"
	content := "test content"

	// Create and disable file
	_ = fm.WriteProxyFile(filename, content)
	_ = fm.DisableProxy(filename)

	// Delete it (should find the disabled version)
	err := fm.DeleteProxyFile(filename)
	if err != nil {
		t.Fatalf("DeleteProxyFile failed: %v", err)
	}

	// Verify both enabled and disabled are gone
	enabledPath := filepath.Join(fm.GetSitesDir(), filename)
	disabledPath := enabledPath + ".disabled"

	if _, err := os.Stat(enabledPath); !os.IsNotExist(err) {
		t.Error("Enabled file should not exist")
	}
	if _, err := os.Stat(disabledPath); !os.IsNotExist(err) {
		t.Error("Disabled file should not exist")
	}
}

func TestFileManager_EnableDisableProxy(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	filename := "1_api.conf"
	content := "test content"

	// Create enabled file
	_ = fm.WriteProxyFile(filename, content)

	enabledPath := filepath.Join(fm.GetSitesDir(), filename)
	disabledPath := enabledPath + ".disabled"

	// Disable it
	err := fm.DisableProxy(filename)
	if err != nil {
		t.Fatalf("DisableProxy failed: %v", err)
	}

	// Check disabled file exists, enabled doesn't
	if _, err := os.Stat(enabledPath); !os.IsNotExist(err) {
		t.Error("Enabled file should not exist after disable")
	}
	if _, err := os.Stat(disabledPath); err != nil {
		t.Error("Disabled file should exist after disable")
	}

	// Enable it again
	err = fm.EnableProxy(filename)
	if err != nil {
		t.Fatalf("EnableProxy failed: %v", err)
	}

	// Check enabled file exists, disabled doesn't
	if _, err := os.Stat(enabledPath); err != nil {
		t.Error("Enabled file should exist after enable")
	}
	if _, err := os.Stat(disabledPath); !os.IsNotExist(err) {
		t.Error("Disabled file should not exist after enable")
	}

	// Verify content is preserved
	read, _ := os.ReadFile(enabledPath)
	if string(read) != content {
		t.Error("Content should be preserved after enable/disable cycle")
	}
}

func TestFileManager_EnableProxy_AlreadyEnabled(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	filename := "1_api.conf"
	_ = fm.WriteProxyFile(filename, "test")

	// Enable when already enabled should not error
	err := fm.EnableProxy(filename)
	if err != nil {
		t.Errorf("EnableProxy on already enabled file should not error: %v", err)
	}
}

func TestFileManager_DisableProxy_AlreadyDisabled(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	filename := "1_api.conf"
	_ = fm.WriteProxyFile(filename, "test")
	_ = fm.DisableProxy(filename)

	// Disable when already disabled should not error
	err := fm.DisableProxy(filename)
	if err != nil {
		t.Errorf("DisableProxy on already disabled file should not error: %v", err)
	}
}

func TestFileManager_EnableProxy_NotFound(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	err := fm.EnableProxy("nonexistent.conf")
	if err == nil {
		t.Error("EnableProxy should error for nonexistent file")
	}
}

func TestFileManager_DisableProxy_NotFound(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	err := fm.DisableProxy("nonexistent.conf")
	if err == nil {
		t.Error("DisableProxy should error for nonexistent file")
	}
}

func TestFileManager_ListProxyFiles(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Create some files
	_ = fm.WriteProxyFile("1_api.conf", "test")
	_ = fm.WriteProxyFile("2_web.conf", "test")
	_ = fm.WriteProxyFile("3_app.conf", "test")

	// Disable one
	_ = fm.DisableProxy("2_web.conf")

	enabled, disabled, err := fm.ListProxyFiles()
	if err != nil {
		t.Fatalf("ListProxyFiles failed: %v", err)
	}

	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled files, got %d", len(enabled))
	}
	if len(disabled) != 1 {
		t.Errorf("Expected 1 disabled file, got %d", len(disabled))
	}

	// Check disabled list returns basename without .disabled suffix
	if len(disabled) > 0 && disabled[0] != "2_web.conf" {
		t.Errorf("Expected disabled[0] = '2_web.conf', got '%s'", disabled[0])
	}
}

func TestFileManager_ListProxyFiles_EmptyDir(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	enabled, disabled, err := fm.ListProxyFiles()
	if err != nil {
		t.Fatalf("ListProxyFiles failed: %v", err)
	}

	if len(enabled) != 0 || len(disabled) != 0 {
		t.Error("Expected empty lists for empty directory")
	}
}

func TestFileManager_Backup(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Create files to backup
	_ = fm.WriteMainCaddyfile("main content")
	_ = fm.WriteCatchAllFile("catchall content")
	_ = fm.WriteProxyFile("1_api.conf", "proxy content")

	backupPath, err := fm.Backup()
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Verify backup directory exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("Backup directory should exist: %v", err)
	}

	// Verify backup contains expected files
	files := []string{
		filepath.Join(backupPath, "Caddyfile"),
		filepath.Join(backupPath, "catchall.conf"),
		filepath.Join(backupPath, "sites", "1_api.conf"),
	}

	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("Backup should contain %s: %v", f, err)
		}
	}
}

func TestFileManager_Restore(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Create initial files
	_ = fm.WriteMainCaddyfile("original main")
	_ = fm.WriteCatchAllFile("original catchall")
	_ = fm.WriteProxyFile("1_api.conf", "original proxy")

	// Create backup
	backupPath, _ := fm.Backup()

	// Modify files
	_ = fm.WriteMainCaddyfile("modified main")
	_ = fm.WriteCatchAllFile("modified catchall")
	_ = fm.WriteProxyFile("1_api.conf", "modified proxy")
	_ = fm.WriteProxyFile("2_new.conf", "new proxy")

	// Restore from backup
	err := fm.Restore(backupPath)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify files are restored
	mainContent, _ := fm.ReadFile(fm.GetCaddyfilePath())
	if mainContent != "original main" {
		t.Error("Main Caddyfile not restored correctly")
	}

	catchallContent, _ := fm.ReadFile(fm.GetCatchAllPath())
	if catchallContent != "original catchall" {
		t.Error("Catchall not restored correctly")
	}

	proxyContent, _ := fm.ReadFile(filepath.Join(fm.GetSitesDir(), "1_api.conf"))
	if proxyContent != "original proxy" {
		t.Error("Proxy file not restored correctly")
	}

	// New file should be gone
	if _, err := os.Stat(filepath.Join(fm.GetSitesDir(), "2_new.conf")); !os.IsNotExist(err) {
		t.Error("New file should be removed after restore")
	}
}

func TestFileManager_Restore_NotFound(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	err := fm.Restore("/nonexistent/path")
	if err == nil {
		t.Error("Restore should fail for nonexistent backup")
	}
}

func TestFileManager_GetLatestBackup(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Create some files and backups
	_ = fm.WriteMainCaddyfile("v1")
	backup1, _ := fm.Backup()

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	_ = fm.WriteMainCaddyfile("v2")
	backup2, _ := fm.Backup()

	latest, err := fm.GetLatestBackup()
	if err != nil {
		t.Fatalf("GetLatestBackup failed: %v", err)
	}

	// Latest should be backup2
	if latest != backup2 {
		t.Errorf("Expected latest = %s, got %s (backup1 was %s)", backup2, latest, backup1)
	}
}

func TestFileManager_GetLatestBackup_NoBackups(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	_, err := fm.GetLatestBackup()
	if err == nil {
		t.Error("GetLatestBackup should fail when no backups exist")
	}
}

func TestFileManager_CleanOldBackups(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Create files and backup
	_ = fm.WriteMainCaddyfile("test")
	backupPath, _ := fm.Backup()

	// Set backup directory time to be old
	oldTime := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(backupPath, oldTime, oldTime)

	// Clean with 24h max age
	err := fm.CleanOldBackups(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanOldBackups failed: %v", err)
	}

	// Backup should be gone
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Old backup should be removed")
	}
}

func TestFileManager_ProxyFileExists(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	filename := "1_api.conf"

	// Should not exist initially
	if fm.ProxyFileExists(filename) {
		t.Error("File should not exist initially")
	}

	// Create enabled file
	_ = fm.WriteProxyFile(filename, "test")
	if !fm.ProxyFileExists(filename) {
		t.Error("Enabled file should exist")
	}

	// Disable it
	_ = fm.DisableProxy(filename)
	if !fm.ProxyFileExists(filename) {
		t.Error("Disabled file should still be found by ProxyFileExists")
	}

	// Delete it
	_ = fm.DeleteProxyFile(filename)
	if fm.ProxyFileExists(filename) {
		t.Error("Deleted file should not exist")
	}
}

func TestFileManager_AtomicWrite(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Write a file
	content := strings.Repeat("test content\n", 1000)
	err := fm.WriteMainCaddyfile(content)
	if err != nil {
		t.Fatalf("WriteMainCaddyfile failed: %v", err)
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(tempDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp_") {
			t.Errorf("Temp file left behind: %s", entry.Name())
		}
	}

	// Verify file has correct permissions
	info, _ := os.Stat(fm.GetCaddyfilePath())
	if info.Mode().Perm() != 0644 {
		t.Errorf("Expected permissions 0644, got %v", info.Mode().Perm())
	}
}

func TestFileManager_ClearSites(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	fm := NewFileManager(tempDir, logger)
	_ = fm.EnsureDirectories()

	// Create some files
	_ = fm.WriteProxyFile("1_api.conf", "test")
	_ = fm.WriteProxyFile("2_web.conf", "test")
	_ = fm.WriteProxyFile("3_app.conf", "test")
	_ = fm.DisableProxy("2_web.conf")

	// Clear sites
	err := fm.ClearSites()
	if err != nil {
		t.Fatalf("ClearSites failed: %v", err)
	}

	// Verify all files are gone
	enabled, disabled, _ := fm.ListProxyFiles()
	if len(enabled) != 0 || len(disabled) != 0 {
		t.Error("All files should be cleared")
	}
}
