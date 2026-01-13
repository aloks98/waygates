package caddy

import "context"

// FileManagerInterface defines the interface for file operations
type FileManagerInterface interface {
	EnsureDirectories() error
	FileExists(path string) bool

	// JSON configuration methods
	GetJSONConfigPath() string
	GetBackupDir() string
	ReadJSONConfig(path string) ([]byte, error)
	WriteJSONConfig(path string, data []byte) error
	ConfigChanged(path string, newData []byte) (bool, error)
	BackupJSONConfig(path string) error
	CleanupOldBackups(maxBackups int) error
}

// ReloaderInterface defines the interface for Caddy reload operations
type ReloaderInterface interface {
	TestConnection(ctx context.Context) error
	ValidateJSON(configPath string) error
	ReloadJSON(ctx context.Context, configPath string) (*ReloadResult, error)
}

// Ensure concrete types implement interfaces
var (
	_ FileManagerInterface = (*FileManager)(nil)
	_ ReloaderInterface    = (*Reloader)(nil)
)
