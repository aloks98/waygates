package caddy

import "context"

// FileManagerInterface defines the interface for file operations
type FileManagerInterface interface {
	EnsureDirectories() error
	GetCaddyfilePath() string
	GetCatchAllPath() string
	GetSitesDir() string
	GetProxyFilePath(filename string) string
	WriteMainCaddyfile(content string) error
	WriteCatchAllFile(content string) error
	WriteProxyFile(filename, content string) error
	WriteIfChanged(filepath, content string) (bool, error)
	DeleteProxyFile(filename string) error
	EnableProxy(filename string) error
	DisableProxy(filename string) error
	ListProxyFiles() (enabled []string, disabled []string, err error)
	FileExists(path string) bool
	Backup() (string, error)
	Restore(backupPath string) error
}

// ReloaderInterface defines the interface for Caddy reload operations
type ReloaderInterface interface {
	Validate(ctx context.Context) error
	Reload(ctx context.Context) (*ReloadResult, error)
	ForceReload(ctx context.Context) (*ReloadResult, error)
	AdaptAndReload(ctx context.Context) (string, error)
	TestConnection(ctx context.Context) error
}

// Ensure concrete types implement interfaces
var (
	_ FileManagerInterface = (*FileManager)(nil)
	_ ReloaderInterface    = (*Reloader)(nil)
)
