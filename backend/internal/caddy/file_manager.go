package caddy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DefaultMaxBackups is the default number of backups to retain
const DefaultMaxBackups = 10

// FileManager handles Caddy JSON configuration file operations
type FileManager struct {
	basePath       string // /etc/caddy
	backupDir      string // /etc/caddy/backup
	jsonConfigName string // caddy.json
	logger         *zap.Logger
	mu             sync.Mutex
}

// NewFileManager creates a new FileManager
func NewFileManager(basePath string, logger *zap.Logger) *FileManager {
	return &FileManager{
		basePath:       basePath,
		backupDir:      filepath.Join(basePath, "backup"),
		jsonConfigName: "caddy.json",
		logger:         logger,
	}
}

// EnsureDirectories creates required directories if they don't exist
func (m *FileManager) EnsureDirectories() error {
	dirs := []string{m.basePath, m.backupDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// GetJSONConfigPath returns the path to the JSON config file
func (m *FileManager) GetJSONConfigPath() string {
	return filepath.Join(m.basePath, m.jsonConfigName)
}

// WriteJSONConfig writes JSON configuration data atomically
// Uses temp file + rename pattern for atomic writes
func (m *FileManager) WriteJSONConfig(path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.atomicWriteBytes(path, data)
}

// GetBackupDir returns the backup directory path
func (m *FileManager) GetBackupDir() string {
	return m.backupDir
}

// ReadJSONConfig reads the current JSON config file contents
func (m *FileManager) ReadJSONConfig(path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read JSON config: %w", err)
	}
	return data, nil
}

// ConfigChanged compares new config data with existing file
// Returns true if config has changed or file doesn't exist
func (m *FileManager) ConfigChanged(path string, newData []byte) (bool, error) {
	currentData, err := m.ReadJSONConfig(path)
	if err != nil {
		return false, err
	}

	// If no current config, consider it changed
	if currentData == nil {
		return true, nil
	}

	// Compare contents
	return !bytes.Equal(currentData, newData), nil
}

// BackupJSONConfig creates a backup of a JSON config file before overwriting
// The backup is stored in the dedicated backup directory with timestamp
func (m *FileManager) BackupJSONConfig(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No file to backup
		return nil
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup path with timestamp in backup directory
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Base(path)
	backupPath := filepath.Join(m.backupDir, filename+"."+timestamp+".backup")

	if err := m.copyFile(path, backupPath); err != nil {
		return fmt.Errorf("failed to backup JSON config %s: %w", path, err)
	}

	m.logger.Info("JSON config backed up",
		zap.String("original", path),
		zap.String("backup", backupPath),
	)

	return nil
}

// CleanupOldBackups removes old backups, keeping only the most recent maxBackups
func (m *FileManager) CleanupOldBackups(maxBackups int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if maxBackups <= 0 {
		maxBackups = DefaultMaxBackups
	}

	// List all backup files
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No backup directory yet
		}
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter for backup files and collect with modification times
	type backupFile struct {
		name    string
		modTime time.Time
	}
	var backups []backupFile

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".backup") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, backupFile{
			name:    entry.Name(),
			modTime: info.ModTime(),
		})
	}

	// If we have fewer backups than max, nothing to clean up
	if len(backups) <= maxBackups {
		return nil
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	// Delete backups beyond maxBackups
	deletedCount := 0
	for i := maxBackups; i < len(backups); i++ {
		backupPath := filepath.Join(m.backupDir, backups[i].name)
		if err := os.Remove(backupPath); err != nil {
			m.logger.Warn("Failed to delete old backup",
				zap.String("path", backupPath),
				zap.Error(err))
			continue
		}
		deletedCount++
	}

	if deletedCount > 0 {
		m.logger.Info("Cleaned up old backups",
			zap.Int("deleted", deletedCount),
			zap.Int("remaining", maxBackups))
	}

	return nil
}

// atomicWriteBytes writes byte data to a file atomically using temp file + rename
func (m *FileManager) atomicWriteBytes(path string, data []byte) error {
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
	if _, err := tempFile.Write(data); err != nil {
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

// FileExists checks if a file exists
func (m *FileManager) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
