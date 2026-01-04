package caddy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FileManager handles Caddyfile read/write operations
type FileManager struct {
	basePath      string // /etc/caddy
	sitesDir      string // /etc/caddy/sites
	backupDir     string // /etc/caddy/backup
	caddyfileName string // Caddyfile
	catchallName  string // catchall.conf
	logger        *zap.Logger
	mu            sync.Mutex
}

// NewFileManager creates a new FileManager
func NewFileManager(basePath string, logger *zap.Logger) *FileManager {
	return &FileManager{
		basePath:      basePath,
		sitesDir:      filepath.Join(basePath, "sites"),
		backupDir:     filepath.Join(basePath, "backup"),
		caddyfileName: "Caddyfile",
		catchallName:  "catchall.conf",
		logger:        logger,
	}
}

// EnsureDirectories creates required directories if they don't exist
func (m *FileManager) EnsureDirectories() error {
	dirs := []string{m.basePath, m.sitesDir, m.backupDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// GetCaddyfilePath returns the path to the main Caddyfile
func (m *FileManager) GetCaddyfilePath() string {
	return filepath.Join(m.basePath, m.caddyfileName)
}

// GetCatchAllPath returns the path to the catch-all config
func (m *FileManager) GetCatchAllPath() string {
	return filepath.Join(m.basePath, m.catchallName)
}

// GetSitesDir returns the path to the sites directory
func (m *FileManager) GetSitesDir() string {
	return m.sitesDir
}

// WriteMainCaddyfile writes the main Caddyfile atomically
func (m *FileManager) WriteMainCaddyfile(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.atomicWrite(m.GetCaddyfilePath(), content)
}

// WriteCatchAllFile writes the catch-all config atomically
func (m *FileManager) WriteCatchAllFile(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.atomicWrite(m.GetCatchAllPath(), content)
}

// WriteProxyFile writes a proxy config file atomically
func (m *FileManager) WriteProxyFile(filename, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := filepath.Join(m.sitesDir, filename)
	return m.atomicWrite(path, content)
}

// DeleteProxyFile deletes a proxy config file
func (m *FileManager) DeleteProxyFile(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try both enabled and disabled paths
	enabledPath := filepath.Join(m.sitesDir, filename)
	disabledPath := enabledPath + ".disabled"

	var lastErr error
	if err := os.Remove(enabledPath); err != nil && !os.IsNotExist(err) {
		lastErr = err
	}
	if err := os.Remove(disabledPath); err != nil && !os.IsNotExist(err) {
		lastErr = err
	}

	return lastErr
}

// EnableProxy enables a proxy by removing .disabled suffix
func (m *FileManager) EnableProxy(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	disabledPath := filepath.Join(m.sitesDir, filename+".disabled")
	enabledPath := filepath.Join(m.sitesDir, filename)

	// Check if disabled file exists
	if _, err := os.Stat(disabledPath); os.IsNotExist(err) {
		// Check if already enabled
		if _, err := os.Stat(enabledPath); err == nil {
			return nil // Already enabled
		}
		return fmt.Errorf("proxy config not found: %s", filename)
	}

	return os.Rename(disabledPath, enabledPath)
}

// DisableProxy disables a proxy by adding .disabled suffix
func (m *FileManager) DisableProxy(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	enabledPath := filepath.Join(m.sitesDir, filename)
	disabledPath := enabledPath + ".disabled"

	// Check if enabled file exists
	if _, err := os.Stat(enabledPath); os.IsNotExist(err) {
		// Check if already disabled
		if _, err := os.Stat(disabledPath); err == nil {
			return nil // Already disabled
		}
		return fmt.Errorf("proxy config not found: %s", filename)
	}

	return os.Rename(enabledPath, disabledPath)
}

// ListProxyFiles lists all proxy config files (both enabled and disabled)
func (m *FileManager) ListProxyFiles() (enabled []string, disabled []string, err error) {
	entries, err := os.ReadDir(m.sitesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read sites directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".conf.disabled") {
			disabled = append(disabled, strings.TrimSuffix(name, ".disabled"))
		} else if strings.HasSuffix(name, ".conf") {
			enabled = append(enabled, name)
		}
	}

	return enabled, disabled, nil
}

// ReadFile reads a file's contents
func (m *FileManager) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// Backup creates a timestamped backup of all config files
func (m *FileManager) Backup() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create backup subdirectory with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(m.backupDir, timestamp)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Backup main Caddyfile
	if err := m.copyFile(m.GetCaddyfilePath(), filepath.Join(backupPath, m.caddyfileName)); err != nil {
		m.logger.Warn("Failed to backup Caddyfile", zap.Error(err))
	}

	// Backup catch-all
	if err := m.copyFile(m.GetCatchAllPath(), filepath.Join(backupPath, m.catchallName)); err != nil {
		m.logger.Warn("Failed to backup catchall.conf", zap.Error(err))
	}

	// Backup sites directory
	sitesBackupDir := filepath.Join(backupPath, "sites")
	if err := os.MkdirAll(sitesBackupDir, 0755); err != nil {
		return backupPath, fmt.Errorf("failed to create sites backup directory: %w", err)
	}

	entries, err := os.ReadDir(m.sitesDir)
	if err != nil && !os.IsNotExist(err) {
		return backupPath, fmt.Errorf("failed to read sites directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(m.sitesDir, entry.Name())
		dst := filepath.Join(sitesBackupDir, entry.Name())
		if err := m.copyFile(src, dst); err != nil {
			m.logger.Warn("Failed to backup site config", zap.String("file", entry.Name()), zap.Error(err))
		}
	}

	m.logger.Info("Backup created", zap.String("path", backupPath))
	return backupPath, nil
}

// Restore restores from a backup directory
func (m *FileManager) Restore(backupPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupPath)
	}

	// Restore main Caddyfile
	caddyfileBackup := filepath.Join(backupPath, m.caddyfileName)
	if _, err := os.Stat(caddyfileBackup); err == nil {
		if err := m.copyFile(caddyfileBackup, m.GetCaddyfilePath()); err != nil {
			return fmt.Errorf("failed to restore Caddyfile: %w", err)
		}
	}

	// Restore catch-all
	catchallBackup := filepath.Join(backupPath, m.catchallName)
	if _, err := os.Stat(catchallBackup); err == nil {
		if err := m.copyFile(catchallBackup, m.GetCatchAllPath()); err != nil {
			return fmt.Errorf("failed to restore catchall.conf: %w", err)
		}
	}

	// Restore sites directory
	sitesBackupDir := filepath.Join(backupPath, "sites")
	if _, err := os.Stat(sitesBackupDir); err == nil {
		// Clear current sites
		entries, _ := os.ReadDir(m.sitesDir)
		for _, entry := range entries {
			_ = os.Remove(filepath.Join(m.sitesDir, entry.Name()))
		}

		// Copy backup files
		entries, err = os.ReadDir(sitesBackupDir)
		if err != nil {
			return fmt.Errorf("failed to read sites backup: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := filepath.Join(sitesBackupDir, entry.Name())
			dst := filepath.Join(m.sitesDir, entry.Name())
			if err := m.copyFile(src, dst); err != nil {
				return fmt.Errorf("failed to restore site %s: %w", entry.Name(), err)
			}
		}
	}

	m.logger.Info("Restored from backup", zap.String("path", backupPath))
	return nil
}

// GetLatestBackup returns the path to the most recent backup
func (m *FileManager) GetLatestBackup() (string, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no backups found")
		}
		return "", err
	}

	var latestDir string
	var latestTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestDir = entry.Name()
		}
	}

	if latestDir == "" {
		return "", fmt.Errorf("no backups found")
	}

	return filepath.Join(m.backupDir, latestDir), nil
}

// CleanOldBackups removes backups older than the specified duration
func (m *FileManager) CleanOldBackups(maxAge time.Duration) error {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(m.backupDir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				m.logger.Warn("Failed to remove old backup", zap.String("path", path), zap.Error(err))
			} else {
				m.logger.Debug("Removed old backup", zap.String("path", path))
			}
		}
	}

	return nil
}

// atomicWrite writes content to a file atomically using temp file + rename
func (m *FileManager) atomicWrite(path, content string) error {
	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in same directory for atomic rename
	tempFile, err := os.CreateTemp(dir, ".tmp_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on error
	defer func() {
		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
	}()

	// Write content
	if _, err := tempFile.WriteString(content); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Sync to disk
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(tempPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Clear temp path so defer doesn't try to remove it
	tempPath = ""

	return nil
}

// copyFile copies a file from src to dst
func (m *FileManager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

// ClearSites removes all files from the sites directory
func (m *FileManager) ClearSites() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(m.sitesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(m.sitesDir, entry.Name())
		if err := os.Remove(path); err != nil {
			m.logger.Warn("Failed to remove site file", zap.String("path", path), zap.Error(err))
		}
	}

	return nil
}

// FileExists checks if a file exists
func (m *FileManager) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetProxyFilePath returns the full path to a proxy config file
func (m *FileManager) GetProxyFilePath(filename string) string {
	return filepath.Join(m.sitesDir, filename)
}

// WriteIfChanged writes content to a file only if the content has changed
// Returns true if the file was written (content changed), false if unchanged
func (m *FileManager) WriteIfChanged(path, content string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read existing content
	existing, err := os.ReadFile(path)
	if err == nil {
		// File exists, compare content (ignore timestamp in header comments)
		if m.contentEqual(string(existing), content) {
			return false, nil // No change needed
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to read existing file: %w", err)
	}

	// Content is different or file doesn't exist, write it
	if err := m.atomicWrite(path, content); err != nil {
		return false, err
	}

	return true, nil
}

// contentEqual compares two config file contents, ignoring timestamp comments
func (m *FileManager) contentEqual(existing, new string) bool {
	// Remove lines starting with "# Generated:" or "# Updated:" since timestamps always change
	existingClean := m.removeTimestampLines(existing)
	newClean := m.removeTimestampLines(new)
	return existingClean == newClean
}

// removeTimestampLines removes timestamp comment lines from content
func (m *FileManager) removeTimestampLines(content string) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# Generated:") || strings.HasPrefix(trimmed, "# Updated:") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// ProxyFileExists checks if a proxy config file exists (enabled or disabled)
func (m *FileManager) ProxyFileExists(filename string) bool {
	enabledPath := filepath.Join(m.sitesDir, filename)
	disabledPath := enabledPath + ".disabled"

	return m.FileExists(enabledPath) || m.FileExists(disabledPath)
}
