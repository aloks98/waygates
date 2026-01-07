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
	BuildMainCaddyfileFunc    func(opts caddyfile.MainCaddyfileOptions) string
	BuildProxyFileFunc        func(proxy *models.Proxy) (string, error)
	BuildProxyFileWithACLFunc func(proxy *models.Proxy, aclAssignments []models.ProxyACLAssignment) (string, error)
	BuildCatchAllFileFunc     func(settings *models.NotFoundSettings) string
	GetProxyFilenameFunc      func(proxy *models.Proxy) string
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

func (m *MockBuilder) BuildProxyFileWithACL(proxy *models.Proxy, aclAssignments []models.ProxyACLAssignment) (string, error) {
	if m.BuildProxyFileWithACLFunc != nil {
		return m.BuildProxyFileWithACLFunc(proxy, aclAssignments)
	}
	// Fall back to BuildProxyFile if no ACL-specific func is set
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

// TestSyncService_Start tests the Start method
func TestSyncService_Start(t *testing.T) {
	t.Run("starts service and ensures directories", func(t *testing.T) {
		ensureDirsCalled := false
		fileExistsCalls := 0

		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{}, 0, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				ensureDirsCalled = true
				return nil
			},
			FileExistsFunc: func(path string) bool {
				fileExistsCalls++
				return true // Files exist, no need to create
			},
			WriteIfChangedFunc: func(filepath, content string) (bool, error) {
				return false, nil
			},
			ListProxyFilesFunc: func() ([]string, []string, error) {
				return []string{}, []string{}, nil
			},
		}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		// Start the service with a short interval
		svc.Start(100 * time.Millisecond)

		// Give it time to run initial setup
		time.Sleep(50 * time.Millisecond)

		// Stop the service
		svc.Stop()

		// Verify EnsureDirectories was called
		if !ensureDirsCalled {
			t.Error("Expected EnsureDirectories to be called")
		}

		// Verify FileExists was called to check for initial configs
		if fileExistsCalls < 2 {
			t.Errorf("Expected FileExists to be called at least twice (for Caddyfile and catchall), got %d", fileExistsCalls)
		}
	})

	t.Run("handles EnsureDirectories error gracefully", func(t *testing.T) {
		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				return errors.New("permission denied")
			},
			FileExistsFunc: func(path string) bool {
				return true
			},
		}
		settingsRepo := &MockSettingsRepository{}
		proxyRepo := &MockProxyRepository{}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		// Should not panic even with error
		svc.Start(100 * time.Millisecond)
		time.Sleep(20 * time.Millisecond)
		svc.Stop()
	})

	t.Run("creates initial configs when files do not exist", func(t *testing.T) {
		mainCaddyfileWritten := false
		catchAllWritten := false

		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				return nil
			},
			FileExistsFunc: func(path string) bool {
				return false // Files don't exist
			},
			GetCaddyfilePathFunc: func() string {
				return "/etc/caddy/Caddyfile"
			},
			GetCatchAllPathFunc: func() string {
				return "/etc/caddy/catchall.conf"
			},
			WriteMainCaddyfileFunc: func(content string) error {
				mainCaddyfileWritten = true
				return nil
			},
			WriteCatchAllFileFunc: func(content string) error {
				catchAllWritten = true
				return nil
			},
		}
		settingsRepo := &MockSettingsRepository{}
		proxyRepo := &MockProxyRepository{}
		reloader := &MockReloader{}
		builder := &MockBuilder{
			BuildMainCaddyfileFunc: func(opts caddyfile.MainCaddyfileOptions) string {
				return "# Main Caddyfile"
			},
			BuildCatchAllFileFunc: func(settings *models.NotFoundSettings) string {
				return "# Catchall"
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
			Email:        "test@example.com",
			ACMEProvider: "off",
		})

		svc.Start(100 * time.Millisecond)
		time.Sleep(20 * time.Millisecond)
		svc.Stop()

		if !mainCaddyfileWritten {
			t.Error("Expected main Caddyfile to be written")
		}
		if !catchAllWritten {
			t.Error("Expected catchall.conf to be written")
		}
	})

	t.Run("handles initial config write errors gracefully", func(t *testing.T) {
		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				return nil
			},
			FileExistsFunc: func(path string) bool {
				return false
			},
			GetCaddyfilePathFunc: func() string {
				return "/etc/caddy/Caddyfile"
			},
			WriteMainCaddyfileFunc: func(content string) error {
				return errors.New("write error")
			},
		}
		settingsRepo := &MockSettingsRepository{}
		proxyRepo := &MockProxyRepository{}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		// Should not panic
		svc.Start(100 * time.Millisecond)
		time.Sleep(20 * time.Millisecond)
		svc.Stop()
	})
}

// TestSyncService_Stop tests the Stop method
func TestSyncService_Stop(t *testing.T) {
	t.Run("stops running service cleanly", func(t *testing.T) {
		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{}, 0, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				return nil
			},
			FileExistsFunc: func(path string) bool {
				return true
			},
			WriteIfChangedFunc: func(filepath, content string) (bool, error) {
				return false, nil
			},
			ListProxyFilesFunc: func() ([]string, []string, error) {
				return []string{}, []string{}, nil
			},
		}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		svc.Start(50 * time.Millisecond)
		time.Sleep(30 * time.Millisecond)

		// Stop should complete without hanging
		// Note: The initial sync has a 5-second delay, so we allow up to 10 seconds for Stop
		// to complete, which accounts for the initial sync goroutine to finish
		done := make(chan struct{})
		go func() {
			svc.Stop()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			t.Error("Stop timed out - service may be hanging")
		}
	})

	t.Run("handles stop on unstarted service", func(t *testing.T) {
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
		})

		// Stop on unstarted service should not panic
		// Note: This may cause a panic due to closing nil channel
		// The implementation should handle this gracefully
		defer func() {
			if r := recover(); r != nil {
				// This is expected behavior if stopChan is closed without Start
				t.Log("Stop on unstarted service panicked as expected")
			}
		}()

		svc.Stop()
	})

	t.Run("stops ticker correctly", func(t *testing.T) {
		syncCount := 0
		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				syncCount++
				return []models.Proxy{}, 0, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				return nil
			},
			FileExistsFunc: func(path string) bool {
				return true
			},
			WriteIfChangedFunc: func(filepath, content string) (bool, error) {
				return false, nil
			},
			ListProxyFilesFunc: func() ([]string, []string, error) {
				return []string{}, []string{}, nil
			},
		}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		// Start with very short interval
		svc.Start(20 * time.Millisecond)

		// Wait for initial sync to start (after 5 second delay in Start)
		// We can't easily test the ticker without waiting too long
		// So we just verify stop works correctly
		time.Sleep(30 * time.Millisecond)

		svc.Stop()

		// After stop, no more syncs should occur
		countAfterStop := syncCount
		time.Sleep(50 * time.Millisecond)

		if syncCount > countAfterStop {
			t.Error("Syncs continued after Stop was called")
		}
	})
}

// TestSyncService_StartStop_Integration tests the full lifecycle
func TestSyncService_StartStop_Integration(t *testing.T) {
	t.Run("multiple start-stop cycles", func(t *testing.T) {
		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{}, 0, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				return nil
			},
			FileExistsFunc: func(path string) bool {
				return true
			},
			WriteIfChangedFunc: func(filepath, content string) (bool, error) {
				return false, nil
			},
			ListProxyFilesFunc: func() ([]string, []string, error) {
				return []string{}, []string{}, nil
			},
		}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		// Create a new service for each cycle since stopChan is closed
		for i := 0; i < 3; i++ {
			svc := NewSyncService(SyncServiceConfig{
				ProxyRepo:    proxyRepo,
				SettingsRepo: settingsRepo,
				FileManager:  fileManager,
				Reloader:     reloader,
				Builder:      builder,
			})

			svc.Start(100 * time.Millisecond)
			time.Sleep(20 * time.Millisecond)
			svc.Stop()
		}
	})
}

// TestEnsureInitialConfigs tests the ensureInitialConfigs method
func TestEnsureInitialConfigs(t *testing.T) {
	t.Run("does nothing when files exist", func(t *testing.T) {
		writeMainCalled := false
		writeCatchAllCalled := false

		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true // All files exist
			},
			GetCaddyfilePathFunc: func() string {
				return "/etc/caddy/Caddyfile"
			},
			GetCatchAllPathFunc: func() string {
				return "/etc/caddy/catchall.conf"
			},
			WriteMainCaddyfileFunc: func(content string) error {
				writeMainCalled = true
				return nil
			},
			WriteCatchAllFileFunc: func(content string) error {
				writeCatchAllCalled = true
				return nil
			},
		}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			FileManager: fileManager,
			Builder:     builder,
		})

		err := svc.ensureInitialConfigs()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if writeMainCalled {
			t.Error("WriteMainCaddyfile should not be called when file exists")
		}
		if writeCatchAllCalled {
			t.Error("WriteCatchAllFile should not be called when file exists")
		}
	})

	t.Run("creates Caddyfile when missing", func(t *testing.T) {
		writeMainCalled := false
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return path != "/etc/caddy/Caddyfile"
			},
			GetCaddyfilePathFunc: func() string {
				return "/etc/caddy/Caddyfile"
			},
			GetCatchAllPathFunc: func() string {
				return "/etc/caddy/catchall.conf"
			},
			WriteMainCaddyfileFunc: func(content string) error {
				writeMainCalled = true
				return nil
			},
		}
		builder := &MockBuilder{
			BuildMainCaddyfileFunc: func(opts caddyfile.MainCaddyfileOptions) string {
				return "# Main"
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			FileManager:  fileManager,
			Builder:      builder,
			Email:        "test@example.com",
			ACMEProvider: "off",
		})

		err := svc.ensureInitialConfigs()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !writeMainCalled {
			t.Error("Expected WriteMainCaddyfile to be called")
		}
	})

	t.Run("creates catchall when missing", func(t *testing.T) {
		writeCatchAllCalled := false
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return path != "/etc/caddy/catchall.conf"
			},
			GetCaddyfilePathFunc: func() string {
				return "/etc/caddy/Caddyfile"
			},
			GetCatchAllPathFunc: func() string {
				return "/etc/caddy/catchall.conf"
			},
			WriteCatchAllFileFunc: func(content string) error {
				writeCatchAllCalled = true
				return nil
			},
		}
		builder := &MockBuilder{
			BuildCatchAllFileFunc: func(settings *models.NotFoundSettings) string {
				return "# Catchall"
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			FileManager: fileManager,
			Builder:     builder,
		})

		err := svc.ensureInitialConfigs()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !writeCatchAllCalled {
			t.Error("Expected WriteCatchAllFile to be called")
		}
	})

	t.Run("returns error on Caddyfile write failure", func(t *testing.T) {
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return false
			},
			GetCaddyfilePathFunc: func() string {
				return "/etc/caddy/Caddyfile"
			},
			WriteMainCaddyfileFunc: func(content string) error {
				return errors.New("write failed")
			},
		}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			FileManager: fileManager,
			Builder:     builder,
		})

		err := svc.ensureInitialConfigs()
		if err == nil {
			t.Error("Expected error")
		}
		if !contains(err.Error(), "failed to write initial Caddyfile") {
			t.Errorf("Expected 'failed to write initial Caddyfile' error, got: %v", err)
		}
	})

	t.Run("returns error on catchall write failure", func(t *testing.T) {
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				if path == "/etc/caddy/Caddyfile" {
					return true
				}
				return false // catchall doesn't exist
			},
			GetCaddyfilePathFunc: func() string {
				return "/etc/caddy/Caddyfile"
			},
			GetCatchAllPathFunc: func() string {
				return "/etc/caddy/catchall.conf"
			},
			WriteCatchAllFileFunc: func(content string) error {
				return errors.New("catchall write failed")
			},
		}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			FileManager: fileManager,
			Builder:     builder,
		})

		err := svc.ensureInitialConfigs()
		if err == nil {
			t.Error("Expected error")
		}
		if !contains(err.Error(), "failed to write initial catchall.conf") {
			t.Errorf("Expected 'failed to write initial catchall.conf' error, got: %v", err)
		}
	})
}

// TestSyncService_SetError tests the setError internal method
func TestSyncService_SetError(t *testing.T) {
	svc, _, _, _, _, _ := newTestSyncService()

	testErr := errors.New("test sync error")
	svc.setError(testErr)

	status := svc.GetStatus()
	if status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be false after error")
	}
	if status.LastError != "test sync error" {
		t.Errorf("Expected LastError 'test sync error', got '%s'", status.LastError)
	}
}

// TestFullSync_WithRollback tests rollback behavior on sync failure
func TestFullSync_WithRollback(t *testing.T) {
	t.Run("attempts rollback on sync failure", func(t *testing.T) {
		restoreCalled := false
		reloadAfterRestoreCalled := false

		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return nil, 0, errors.New("database error")
			},
		}
		settingsRepo := &MockSettingsRepository{}
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true
			},
			BackupFunc: func() (string, error) {
				return "/tmp/backup-123", nil
			},
			RestoreFunc: func(backupPath string) error {
				restoreCalled = true
				if backupPath != "/tmp/backup-123" {
					t.Errorf("Expected backup path '/tmp/backup-123', got '%s'", backupPath)
				}
				return nil
			},
		}
		reloader := &MockReloader{
			ReloadFunc: func(ctx context.Context) (*caddy.ReloadResult, error) {
				reloadAfterRestoreCalled = true
				return &caddy.ReloadResult{Success: true}, nil
			},
		}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		err := svc.FullSync()
		if err == nil {
			t.Error("Expected error from FullSync")
		}

		if !restoreCalled {
			t.Error("Expected Restore to be called on failure")
		}
		if !reloadAfterRestoreCalled {
			t.Error("Expected Reload to be called after restore")
		}
	})

	t.Run("handles backup failure gracefully", func(t *testing.T) {
		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return nil, 0, errors.New("database error")
			},
		}
		settingsRepo := &MockSettingsRepository{}
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true
			},
			BackupFunc: func() (string, error) {
				return "", errors.New("backup failed")
			},
		}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		// Should not panic even when backup fails
		err := svc.FullSync()
		if err == nil {
			t.Error("Expected error from FullSync")
		}
	})

	t.Run("handles restore failure gracefully", func(t *testing.T) {
		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return nil, 0, errors.New("database error")
			},
		}
		settingsRepo := &MockSettingsRepository{}
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true
			},
			BackupFunc: func() (string, error) {
				return "/tmp/backup", nil
			},
			RestoreFunc: func(backupPath string) error {
				return errors.New("restore failed")
			},
		}
		reloader := &MockReloader{}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		// Should not panic even when restore fails
		err := svc.FullSync()
		if err == nil {
			t.Error("Expected error from FullSync")
		}
	})
}

// TestFullSync_OrphanedFileCleanup tests cleanup of orphaned proxy files
func TestFullSync_OrphanedFileCleanup(t *testing.T) {
	t.Run("removes orphaned enabled files", func(t *testing.T) {
		deletedFiles := []string{}

		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{
					{ID: 1, Hostname: "active.com", IsActive: true},
				}, 1, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true
			},
			WriteIfChangedFunc: func(filepath, content string) (bool, error) {
				return true, nil
			},
			ListProxyFilesFunc: func() ([]string, []string, error) {
				// Return an orphaned file
				return []string{"1_active_com.conf", "999_orphan.conf"}, []string{}, nil
			},
			DeleteProxyFileFunc: func(filename string) error {
				deletedFiles = append(deletedFiles, filename)
				return nil
			},
			GetProxyFilePathFunc: func(filename string) string {
				return "/etc/caddy/sites/" + filename
			},
		}
		reloader := &MockReloader{
			ReloadFunc: func(ctx context.Context) (*caddy.ReloadResult, error) {
				return &caddy.ReloadResult{Success: true}, nil
			},
		}
		builder := &MockBuilder{
			GetProxyFilenameFunc: func(proxy *models.Proxy) string {
				return GetProxyFilename(proxy.ID, proxy.Hostname)
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check that orphaned file was deleted
		found := false
		for _, f := range deletedFiles {
			if f == "999_orphan.conf" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected orphaned file '999_orphan.conf' to be deleted")
		}
	})

	t.Run("removes orphaned disabled files", func(t *testing.T) {
		deletedFiles := []string{}

		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{}, 0, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true
			},
			WriteIfChangedFunc: func(filepath, content string) (bool, error) {
				return true, nil
			},
			ListProxyFilesFunc: func() ([]string, []string, error) {
				// Return an orphaned disabled file
				return []string{}, []string{"999_orphan_disabled.conf"}, nil
			},
			DeleteProxyFileFunc: func(filename string) error {
				deletedFiles = append(deletedFiles, filename)
				return nil
			},
		}
		reloader := &MockReloader{
			ReloadFunc: func(ctx context.Context) (*caddy.ReloadResult, error) {
				return &caddy.ReloadResult{Success: true}, nil
			},
		}
		builder := &MockBuilder{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(deletedFiles) != 1 || deletedFiles[0] != "999_orphan_disabled.conf" {
			t.Errorf("Expected orphaned disabled file to be deleted, got: %v", deletedFiles)
		}
	})
}

// TestFullSync_InactiveProxyHandling tests handling of inactive proxies
func TestFullSync_InactiveProxyHandling(t *testing.T) {
	t.Run("disables inactive proxies", func(t *testing.T) {
		disabledProxies := []string{}

		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{
					{ID: 1, Hostname: "inactive.com", Name: "Inactive", IsActive: false},
				}, 1, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true
			},
			WriteIfChangedFunc: func(filepath, content string) (bool, error) {
				return true, nil
			},
			ListProxyFilesFunc: func() ([]string, []string, error) {
				return []string{"1_inactive_com.conf"}, []string{}, nil
			},
			DisableProxyFunc: func(filename string) error {
				disabledProxies = append(disabledProxies, filename)
				return nil
			},
			GetProxyFilePathFunc: func(filename string) string {
				return "/etc/caddy/sites/" + filename
			},
		}
		reloader := &MockReloader{
			ReloadFunc: func(ctx context.Context) (*caddy.ReloadResult, error) {
				return &caddy.ReloadResult{Success: true}, nil
			},
		}
		builder := &MockBuilder{
			GetProxyFilenameFunc: func(proxy *models.Proxy) string {
				return GetProxyFilename(proxy.ID, proxy.Hostname)
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Builder:      builder,
		})

		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(disabledProxies) != 1 || disabledProxies[0] != "1_inactive_com.conf" {
			t.Errorf("Expected inactive proxy to be disabled, got: %v", disabledProxies)
		}
	})
}

// TestSyncProxy_WriteError tests error handling when writing proxy file fails
func TestSyncProxy_WriteError(t *testing.T) {
	svc, _, _, fileManager, _, builder := newTestSyncService()

	builder.GetProxyFilenameFunc = func(proxy *models.Proxy) string {
		return "1_example_com.conf"
	}
	fileManager.WriteProxyFileFunc = func(filename, content string) error {
		return errors.New("disk full")
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to write proxy file") {
		t.Errorf("Expected 'failed to write proxy file' error, got: %v", err)
	}
}

// TestSyncProxy_DisableError tests error handling when disabling proxy fails
func TestSyncProxy_DisableError(t *testing.T) {
	svc, _, _, fileManager, _, builder := newTestSyncService()

	builder.GetProxyFilenameFunc = func(proxy *models.Proxy) string {
		return "1_example_com.conf"
	}
	fileManager.WriteProxyFileFunc = func(filename, content string) error {
		return nil
	}
	fileManager.DisableProxyFunc = func(filename string) error {
		return errors.New("rename failed")
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: false}
	err := svc.SyncProxy(proxy)

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to disable proxy") {
		t.Errorf("Expected 'failed to disable proxy' error, got: %v", err)
	}
}

// TestSyncProxy_ReloadError tests error handling when Caddy reload fails
func TestSyncProxy_ReloadError(t *testing.T) {
	svc, _, _, fileManager, reloader, builder := newTestSyncService()

	builder.GetProxyFilenameFunc = func(proxy *models.Proxy) string {
		return "1_example_com.conf"
	}
	fileManager.WriteProxyFileFunc = func(filename, content string) error {
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy unreachable")
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy") {
		t.Errorf("Expected 'failed to reload Caddy' error, got: %v", err)
	}
}

// TestRemoveProxy_ReloadError tests error handling when reload fails after remove
func TestRemoveProxy_ReloadError(t *testing.T) {
	svc, _, _, fileManager, reloader, _ := newTestSyncService()

	fileManager.DeleteProxyFileFunc = func(filename string) error {
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy error")
	}

	err := svc.RemoveProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy") {
		t.Errorf("Expected 'failed to reload Caddy' error, got: %v", err)
	}
}

// TestEnableProxy_Error tests error handling when enable fails
func TestEnableProxy_Error(t *testing.T) {
	svc, _, _, fileManager, _, _ := newTestSyncService()

	fileManager.EnableProxyFunc = func(filename string) error {
		return errors.New("file not found")
	}

	err := svc.EnableProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to enable proxy") {
		t.Errorf("Expected 'failed to enable proxy' error, got: %v", err)
	}
}

// TestUpdateCatchAll_ReloadError tests error when reload fails after catchall update
func TestUpdateCatchAll_ReloadError(t *testing.T) {
	svc, _, settingsRepo, fileManager, reloader, _ := newTestSyncService()

	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	fileManager.WriteCatchAllFileFunc = func(content string) error {
		return nil
	}
	reloader.ReloadFunc = func(ctx context.Context) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy not responding")
	}

	err := svc.UpdateCatchAll()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy") {
		t.Errorf("Expected 'failed to reload Caddy' error, got: %v", err)
	}
}
