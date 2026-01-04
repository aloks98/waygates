package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/caddy/caddyfile"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// MockSettingsRepository implements SettingsRepositoryInterface for testing
type MockSettingsRepository struct {
	GetFunc                 func(key string) (*models.Setting, error)
	GetValueFunc            func(key, defaultValue string) string
	SetFunc                 func(key, value string) error
	GetAllFunc              func() (map[string]string, error)
	DeleteFunc              func(key string) error
	GetNotFoundSettingsFunc func() (*models.NotFoundSettings, error)
	SetNotFoundSettingsFunc func(settings *models.NotFoundSettings) error
}

func (m *MockSettingsRepository) Get(key string) (*models.Setting, error) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return nil, errors.New("not found")
}

func (m *MockSettingsRepository) GetValue(key, defaultValue string) string {
	if m.GetValueFunc != nil {
		return m.GetValueFunc(key, defaultValue)
	}
	return defaultValue
}

func (m *MockSettingsRepository) Set(key, value string) error {
	if m.SetFunc != nil {
		return m.SetFunc(key, value)
	}
	return nil
}

func (m *MockSettingsRepository) GetAll() (map[string]string, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return map[string]string{}, nil
}

func (m *MockSettingsRepository) Delete(key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(key)
	}
	return nil
}

func (m *MockSettingsRepository) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	if m.GetNotFoundSettingsFunc != nil {
		return m.GetNotFoundSettingsFunc()
	}
	return &models.NotFoundSettings{Mode: "default"}, nil
}

func (m *MockSettingsRepository) SetNotFoundSettings(settings *models.NotFoundSettings) error {
	if m.SetNotFoundSettingsFunc != nil {
		return m.SetNotFoundSettingsFunc(settings)
	}
	return nil
}

// MockFileManager implements FileManagerInterface for testing
type MockFileManager struct {
	EnsureDirectoriesFunc  func() error
	GetCaddyfilePathFunc   func() string
	GetCatchAllPathFunc    func() string
	GetSitesDirFunc        func() string
	GetProxyFilePathFunc   func(filename string) string
	WriteMainCaddyfileFunc func(content string) error
	WriteCatchAllFileFunc  func(content string) error
	WriteProxyFileFunc     func(filename, content string) error
	WriteIfChangedFunc     func(filepath, content string) (bool, error)
	DeleteProxyFileFunc    func(filename string) error
	EnableProxyFunc        func(filename string) error
	DisableProxyFunc       func(filename string) error
	ListProxyFilesFunc     func() (enabled []string, disabled []string, err error)
	FileExistsFunc         func(path string) bool
	BackupFunc             func() (string, error)
	RestoreFunc            func(backupPath string) error
}

func (m *MockFileManager) EnsureDirectories() error {
	if m.EnsureDirectoriesFunc != nil {
		return m.EnsureDirectoriesFunc()
	}
	return nil
}

func (m *MockFileManager) GetCaddyfilePath() string {
	if m.GetCaddyfilePathFunc != nil {
		return m.GetCaddyfilePathFunc()
	}
	return "/etc/caddy/Caddyfile"
}

func (m *MockFileManager) GetCatchAllPath() string {
	if m.GetCatchAllPathFunc != nil {
		return m.GetCatchAllPathFunc()
	}
	return "/etc/caddy/catchall.conf"
}

func (m *MockFileManager) GetSitesDir() string {
	if m.GetSitesDirFunc != nil {
		return m.GetSitesDirFunc()
	}
	return "/etc/caddy/sites"
}

func (m *MockFileManager) GetProxyFilePath(filename string) string {
	if m.GetProxyFilePathFunc != nil {
		return m.GetProxyFilePathFunc(filename)
	}
	return "/etc/caddy/sites/" + filename
}

func (m *MockFileManager) WriteMainCaddyfile(content string) error {
	if m.WriteMainCaddyfileFunc != nil {
		return m.WriteMainCaddyfileFunc(content)
	}
	return nil
}

func (m *MockFileManager) WriteCatchAllFile(content string) error {
	if m.WriteCatchAllFileFunc != nil {
		return m.WriteCatchAllFileFunc(content)
	}
	return nil
}

func (m *MockFileManager) WriteProxyFile(filename, content string) error {
	if m.WriteProxyFileFunc != nil {
		return m.WriteProxyFileFunc(filename, content)
	}
	return nil
}

func (m *MockFileManager) WriteIfChanged(filepath, content string) (bool, error) {
	if m.WriteIfChangedFunc != nil {
		return m.WriteIfChangedFunc(filepath, content)
	}
	return false, nil
}

func (m *MockFileManager) DeleteProxyFile(filename string) error {
	if m.DeleteProxyFileFunc != nil {
		return m.DeleteProxyFileFunc(filename)
	}
	return nil
}

func (m *MockFileManager) EnableProxy(filename string) error {
	if m.EnableProxyFunc != nil {
		return m.EnableProxyFunc(filename)
	}
	return nil
}

func (m *MockFileManager) DisableProxy(filename string) error {
	if m.DisableProxyFunc != nil {
		return m.DisableProxyFunc(filename)
	}
	return nil
}

func (m *MockFileManager) ListProxyFiles() (enabled []string, disabled []string, err error) {
	if m.ListProxyFilesFunc != nil {
		return m.ListProxyFilesFunc()
	}
	return []string{}, []string{}, nil
}

func (m *MockFileManager) FileExists(path string) bool {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(path)
	}
	return true
}

func (m *MockFileManager) Backup() (string, error) {
	if m.BackupFunc != nil {
		return m.BackupFunc()
	}
	return "/tmp/backup", nil
}

func (m *MockFileManager) Restore(backupPath string) error {
	if m.RestoreFunc != nil {
		return m.RestoreFunc(backupPath)
	}
	return nil
}

// MockReloader implements ReloaderInterface for testing
type MockReloader struct {
	ValidateFunc       func(ctx context.Context) error
	ReloadFunc         func(ctx context.Context) (*caddy.ReloadResult, error)
	ForceReloadFunc    func(ctx context.Context) (*caddy.ReloadResult, error)
	AdaptAndReloadFunc func(ctx context.Context) (string, error)
	TestConnectionFunc func(ctx context.Context) error
}

func (m *MockReloader) Validate(ctx context.Context) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx)
	}
	return nil
}

func (m *MockReloader) Reload(ctx context.Context) (*caddy.ReloadResult, error) {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(ctx)
	}
	return &caddy.ReloadResult{Success: true, Duration: 100 * time.Millisecond}, nil
}

func (m *MockReloader) ForceReload(ctx context.Context) (*caddy.ReloadResult, error) {
	if m.ForceReloadFunc != nil {
		return m.ForceReloadFunc(ctx)
	}
	return &caddy.ReloadResult{Success: true}, nil
}

func (m *MockReloader) AdaptAndReload(ctx context.Context) (string, error) {
	if m.AdaptAndReloadFunc != nil {
		return m.AdaptAndReloadFunc(ctx)
	}
	return "{}", nil
}

func (m *MockReloader) TestConnection(ctx context.Context) error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx)
	}
	return nil
}

// MockBuilder implements BuilderInterface for testing
type MockBuilder struct {
	BuildMainCaddyfileFunc func(opts caddyfile.MainCaddyfileOptions) string
	BuildProxyFileFunc     func(proxy *models.Proxy) (string, error)
	BuildCatchAllFileFunc  func(settings *models.NotFoundSettings) string
	GetProxyFilenameFunc   func(proxy *models.Proxy) string
}

func (m *MockBuilder) BuildMainCaddyfile(opts caddyfile.MainCaddyfileOptions) string {
	if m.BuildMainCaddyfileFunc != nil {
		return m.BuildMainCaddyfileFunc(opts)
	}
	return "# Main Caddyfile"
}

func (m *MockBuilder) BuildProxyFile(proxy *models.Proxy) (string, error) {
	if m.BuildProxyFileFunc != nil {
		return m.BuildProxyFileFunc(proxy)
	}
	return "# Proxy config", nil
}

func (m *MockBuilder) BuildCatchAllFile(settings *models.NotFoundSettings) string {
	if m.BuildCatchAllFileFunc != nil {
		return m.BuildCatchAllFileFunc(settings)
	}
	return "# Catch-all config"
}

func (m *MockBuilder) GetProxyFilename(proxy *models.Proxy) string {
	if m.GetProxyFilenameFunc != nil {
		return m.GetProxyFilenameFunc(proxy)
	}
	return GetProxyFilename(proxy.ID, proxy.Hostname)
}

// Helper function to create a test service with mocks
func newTestSyncService() (*SyncService, *MockProxyRepository, *MockSettingsRepository, *MockFileManager, *MockReloader, *MockBuilder) {
	proxyRepo := &MockProxyRepository{}
	settingsRepo := &MockSettingsRepository{}
	fileManager := &MockFileManager{}
	reloader := &MockReloader{}
	builder := &MockBuilder{}

	svc := NewSyncService(SyncServiceConfig{
		ProxyRepo:    proxyRepo,
		SettingsRepo: settingsRepo,
		FileManager:  fileManager,
		Reloader:     reloader,
		Builder:      builder,
		Email:        "test@example.com",
		ACMEProvider: "off",
	})

	return svc, proxyRepo, settingsRepo, fileManager, reloader, builder
}

// TestNewSyncService tests service creation
func TestNewSyncService(t *testing.T) {
	svc, _, _, _, _, _ := newTestSyncService()

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
	if svc.email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", svc.email)
	}
	if svc.acmeProvider != "off" {
		t.Errorf("Expected acmeProvider 'off', got '%s'", svc.acmeProvider)
	}
}

// TestNewSyncService_NilLogger tests that nil logger is handled
func TestNewSyncService_NilLogger(t *testing.T) {
	svc := NewSyncService(SyncServiceConfig{
		Logger: nil,
	})

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
	if svc.logger == nil {
		t.Error("Expected logger to be set to nop logger")
	}
}

// TestGetStatus tests status retrieval
func TestGetStatus(t *testing.T) {
	svc, _, _, _, _, _ := newTestSyncService()

	status := svc.GetStatus()

	if status.IsSyncing {
		t.Error("Expected IsSyncing to be false initially")
	}
	if !status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be true initially")
	}
}

// TestFullSync_AlreadySyncing tests that concurrent syncs are prevented
func TestFullSync_AlreadySyncing(t *testing.T) {
	svc, _, _, _, _, _ := newTestSyncService()

	// Manually set syncing state
	svc.mu.Lock()
	svc.status.IsSyncing = true
	svc.mu.Unlock()

	err := svc.FullSync()

	if err == nil {
		t.Fatal("Expected error when already syncing")
	}
	if err.Error() != "sync already in progress" {
		t.Errorf("Expected 'sync already in progress' error, got: %v", err)
	}
}

// TestFullSync_Success tests a successful full sync
func TestFullSync_Success(t *testing.T) {
	svc, proxyRepo, settingsRepo, fileManager, reloader, builder := newTestSyncService()

	// Setup mocks
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true},
		}, 1, nil
	}

	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}

	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.WriteIfChangedFunc = func(filepath, content string) (bool, error) {
		return true, nil // Config changed
	}
	fileManager.ListProxyFilesFunc = func() ([]string, []string, error) {
		return []string{"1_example_com.conf"}, []string{}, nil
	}

	builder.GetProxyFilenameFunc = func(proxy *models.Proxy) string {
		return "1_example_com.conf"
	}

	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
	}

	err := svc.FullSync()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	status := svc.GetStatus()
	if !status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be true")
	}
	if status.SyncCount != 1 {
		t.Errorf("Expected SyncCount 1, got %d", status.SyncCount)
	}
}

// TestFullSync_ProxyListError tests error handling when listing proxies fails
func TestFullSync_ProxyListError(t *testing.T) {
	svc, proxyRepo, _, fileManager, _, _ := newTestSyncService()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return nil, 0, errors.New("database error")
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}

	err := svc.FullSync()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to list proxies") {
		t.Errorf("Expected 'failed to list proxies' error, got: %v", err)
	}

	status := svc.GetStatus()
	if status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be false after error")
	}
}

// TestFullSync_ReloadError tests error handling when Caddy reload fails
func TestFullSync_ReloadError(t *testing.T) {
	svc, proxyRepo, settingsRepo, fileManager, reloader, _ := newTestSyncService()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.WriteIfChangedFunc = func(filepath, content string) (bool, error) {
		return true, nil // Config changed - will trigger reload
	}
	fileManager.ListProxyFilesFunc = func() ([]string, []string, error) {
		return []string{}, []string{}, nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy not responding")
	}

	err := svc.FullSync()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy") {
		t.Errorf("Expected 'failed to reload Caddy' error, got: %v", err)
	}
}

// TestFullSync_NoChanges tests that reload is skipped when no changes
func TestFullSync_NoChanges(t *testing.T) {
	svc, proxyRepo, settingsRepo, fileManager, reloader, _ := newTestSyncService()

	reloadCalled := false
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.WriteIfChangedFunc = func(filepath, content string) (bool, error) {
		return false, nil // No changes
	}
	fileManager.ListProxyFilesFunc = func() ([]string, []string, error) {
		return []string{}, []string{}, nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		reloadCalled = true
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.FullSync()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if reloadCalled {
		t.Error("Reload should not be called when no changes")
	}
}

// TestSyncProxy_Success tests syncing a single proxy
func TestSyncProxy_Success(t *testing.T) {
	svc, _, _, fileManager, reloader, builder := newTestSyncService()

	writeProxyCalled := false
	fileManager.WriteProxyFileFunc = func(filename, content string) error {
		writeProxyCalled = true
		if filename != "1_example_com.conf" {
			t.Errorf("Expected filename '1_example_com.conf', got '%s'", filename)
		}
		return nil
	}
	builder.GetProxyFilenameFunc = func(proxy *models.Proxy) string {
		return "1_example_com.conf"
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !writeProxyCalled {
		t.Error("Expected WriteProxyFile to be called")
	}
}

// TestSyncProxy_InactiveProxy tests syncing an inactive proxy
func TestSyncProxy_InactiveProxy(t *testing.T) {
	svc, _, _, fileManager, reloader, builder := newTestSyncService()

	disableCalled := false
	fileManager.WriteProxyFileFunc = func(filename, content string) error {
		return nil
	}
	fileManager.DisableProxyFunc = func(filename string) error {
		disableCalled = true
		return nil
	}
	builder.GetProxyFilenameFunc = func(proxy *models.Proxy) string {
		return "1_example_com.conf"
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: false}
	err := svc.SyncProxy(proxy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !disableCalled {
		t.Error("Expected DisableProxy to be called for inactive proxy")
	}
}

// TestSyncProxy_BuildError tests error handling when building proxy config fails
func TestSyncProxy_BuildError(t *testing.T) {
	svc, _, _, _, _, builder := newTestSyncService()

	builder.BuildProxyFileFunc = func(proxy *models.Proxy) (string, error) {
		return "", errors.New("invalid proxy config")
	}
	builder.GetProxyFilenameFunc = func(proxy *models.Proxy) string {
		return "1_example_com.conf"
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to build proxy config") {
		t.Errorf("Expected 'failed to build proxy config' error, got: %v", err)
	}
}

// TestRemoveProxy_Success tests removing a proxy
func TestRemoveProxy_Success(t *testing.T) {
	svc, _, _, fileManager, reloader, _ := newTestSyncService()

	deleteCalled := false
	fileManager.DeleteProxyFileFunc = func(filename string) error {
		deleteCalled = true
		if filename != "1_example_com.conf" {
			t.Errorf("Expected filename '1_example_com.conf', got '%s'", filename)
		}
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.RemoveProxy(1, "example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("Expected DeleteProxyFile to be called")
	}
}

// TestRemoveProxy_DeleteError tests error handling when delete fails
func TestRemoveProxy_DeleteError(t *testing.T) {
	svc, _, _, fileManager, _, _ := newTestSyncService()

	fileManager.DeleteProxyFileFunc = func(filename string) error {
		return errors.New("file not found")
	}

	err := svc.RemoveProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to delete proxy file") {
		t.Errorf("Expected 'failed to delete proxy file' error, got: %v", err)
	}
}

// TestEnableProxy_Success tests enabling a proxy
func TestEnableProxy_Success(t *testing.T) {
	svc, _, _, fileManager, reloader, _ := newTestSyncService()

	enableCalled := false
	fileManager.EnableProxyFunc = func(filename string) error {
		enableCalled = true
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.EnableProxy(1, "example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !enableCalled {
		t.Error("Expected EnableProxy to be called")
	}
}

// TestEnableProxy_ReloadError tests error handling when reload fails after enable
func TestEnableProxy_ReloadError(t *testing.T) {
	svc, _, _, fileManager, reloader, _ := newTestSyncService()

	fileManager.EnableProxyFunc = func(filename string) error {
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy not responding")
	}

	err := svc.EnableProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy") {
		t.Errorf("Expected 'failed to reload Caddy' error, got: %v", err)
	}
}

// TestDisableProxy_Success tests disabling a proxy
func TestDisableProxy_Success(t *testing.T) {
	svc, _, _, fileManager, reloader, _ := newTestSyncService()

	disableCalled := false
	fileManager.DisableProxyFunc = func(filename string) error {
		disableCalled = true
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.DisableProxy(1, "example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !disableCalled {
		t.Error("Expected DisableProxy to be called")
	}
}

// TestDisableProxy_Error tests error handling when disable fails
func TestDisableProxy_Error(t *testing.T) {
	svc, _, _, fileManager, _, _ := newTestSyncService()

	fileManager.DisableProxyFunc = func(filename string) error {
		return errors.New("permission denied")
	}

	err := svc.DisableProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to disable proxy") {
		t.Errorf("Expected 'failed to disable proxy' error, got: %v", err)
	}
}

// TestUpdateCatchAll_Success tests updating catch-all config
func TestUpdateCatchAll_Success(t *testing.T) {
	svc, _, settingsRepo, fileManager, reloader, builder := newTestSyncService()

	writeCalled := false
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "redirect", RedirectURL: "https://example.com"}, nil
	}
	builder.BuildCatchAllFileFunc = func(settings *models.NotFoundSettings) string {
		return "# Redirect config"
	}
	fileManager.WriteCatchAllFileFunc = func(content string) error {
		writeCalled = true
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.UpdateCatchAll()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !writeCalled {
		t.Error("Expected WriteCatchAllFile to be called")
	}
}

// TestUpdateCatchAll_SettingsError tests error when getting settings fails
func TestUpdateCatchAll_SettingsError(t *testing.T) {
	svc, _, settingsRepo, _, _, _ := newTestSyncService()

	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return nil, errors.New("database error")
	}

	err := svc.UpdateCatchAll()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to get 404 settings") {
		t.Errorf("Expected 'failed to get 404 settings' error, got: %v", err)
	}
}

// TestUpdateCatchAll_WriteError tests error when writing catch-all fails
func TestUpdateCatchAll_WriteError(t *testing.T) {
	svc, _, settingsRepo, fileManager, _, _ := newTestSyncService()

	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	fileManager.WriteCatchAllFileFunc = func(content string) error {
		return errors.New("permission denied")
	}

	err := svc.UpdateCatchAll()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to write catch-all config") {
		t.Errorf("Expected 'failed to write catch-all config' error, got: %v", err)
	}
}

// TestSyncStatus_DefaultValues tests SyncStatus default values
func TestSyncStatus_DefaultValues(t *testing.T) {
	status := SyncStatus{}

	if status.IsSyncing {
		t.Error("Expected IsSyncing to be false by default")
	}
	if status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be false by default (zero value)")
	}
	if status.LastError != "" {
		t.Errorf("Expected LastError to be empty, got '%s'", status.LastError)
	}
	if !status.LastSyncTime.IsZero() {
		t.Error("Expected LastSyncTime to be zero")
	}
}

// TestSyncStatus_JSONSerialization tests JSON field names
func TestSyncStatus_JSONSerialization(t *testing.T) {
	now := time.Now()
	status := SyncStatus{
		LastSyncTime:    now,
		IsSyncing:       true,
		LastSyncSuccess: true,
		LastError:       "test error",
	}

	if !status.IsSyncing {
		t.Error("Expected IsSyncing to be true")
	}
	if !status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be true")
	}
	if status.LastError != "test error" {
		t.Errorf("Expected LastError 'test error', got '%s'", status.LastError)
	}
	if status.LastSyncTime != now {
		t.Error("LastSyncTime mismatch")
	}
}

// TestSanitizeFilename tests the sanitizeFilename function
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple hostname", "example.com", "example_com"},
		{"Subdomain", "api.example.com", "api_example_com"},
		{"Multiple subdomains", "www.api.example.com", "www_api_example_com"},
		{"Already safe", "example-site", "example-site"},
		{"With underscores", "my_site_name", "my_site_name"},
		{"Mixed characters", "test.site-123_name", "test_site-123_name"},
		{"Special characters", "test@site!name#123", "test_site_name_123"},
		{"Consecutive dots", "test..example...com", "test_example_com"},
		{"Leading/trailing dots", ".example.com.", "example_com"},
		{"Only dots", "...", ""},
		{"Empty string", "", ""},
		{"Uppercase preserved", "MyApp.Example.COM", "MyApp_Example_COM"},
		{"Numbers", "app123.site456.com", "app123_site456_com"},
		{"Hyphen and underscore mix", "my-app_v2.example.com", "my-app_v2_example_com"},
		{"Spaces become underscores", "my site name", "my_site_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetProxyFilename tests the GetProxyFilename function
func TestGetProxyFilename(t *testing.T) {
	tests := []struct {
		name     string
		proxyID  int
		hostname string
		expected string
	}{
		{"Simple case", 1, "example.com", "1_example_com.conf"},
		{"With subdomain", 42, "api.example.com", "42_api_example_com.conf"},
		{"Large ID", 99999, "test.site.com", "99999_test_site_com.conf"},
		{"Zero ID", 0, "example.com", "0_example_com.conf"},
		{"Complex hostname", 5, "my-app.sub.domain.co.uk", "5_my-app_sub_domain_co_uk.conf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProxyFilename(tt.proxyID, tt.hostname)
			if result != tt.expected {
				t.Errorf("GetProxyFilename(%d, %q) = %q, want %q", tt.proxyID, tt.hostname, result, tt.expected)
			}
		})
	}
}

// TestSyncStatus_AllFields tests all SyncStatus fields
func TestSyncStatus_AllFields(t *testing.T) {
	now := time.Now()
	status := SyncStatus{
		LastSyncTime:    now,
		IsSyncing:       true,
		LastSyncSuccess: true,
		LastError:       "",
		SyncCount:       5,
		LastReloadTime:  now,
		ReloadCount:     3,
		ConfigChanged:   true,
	}

	if status.SyncCount != 5 {
		t.Errorf("Expected SyncCount 5, got %d", status.SyncCount)
	}
	if status.ReloadCount != 3 {
		t.Errorf("Expected ReloadCount 3, got %d", status.ReloadCount)
	}
	if !status.ConfigChanged {
		t.Error("Expected ConfigChanged to be true")
	}
	if status.LastReloadTime != now {
		t.Error("LastReloadTime mismatch")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
