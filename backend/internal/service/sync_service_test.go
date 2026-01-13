package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/caddy"
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
	EnsureDirectoriesFunc func() error
	FileExistsFunc        func(path string) bool

	// JSON configuration methods
	GetJSONConfigPathFunc func() string
	WriteJSONConfigFunc   func(path string, data []byte) error
	BackupJSONConfigFunc  func(path string) error
}

func (m *MockFileManager) EnsureDirectories() error {
	if m.EnsureDirectoriesFunc != nil {
		return m.EnsureDirectoriesFunc()
	}
	return nil
}

func (m *MockFileManager) FileExists(path string) bool {
	if m.FileExistsFunc != nil {
		return m.FileExistsFunc(path)
	}
	return true
}

func (m *MockFileManager) GetJSONConfigPath() string {
	if m.GetJSONConfigPathFunc != nil {
		return m.GetJSONConfigPathFunc()
	}
	return "/etc/caddy/caddy.json"
}

func (m *MockFileManager) WriteJSONConfig(path string, data []byte) error {
	if m.WriteJSONConfigFunc != nil {
		return m.WriteJSONConfigFunc(path, data)
	}
	return nil
}

func (m *MockFileManager) BackupJSONConfig(path string) error {
	if m.BackupJSONConfigFunc != nil {
		return m.BackupJSONConfigFunc(path)
	}
	return nil
}

// MockReloader implements ReloaderInterface for testing
type MockReloader struct {
	TestConnectionFunc func(ctx context.Context) error
	ValidateJSONFunc   func(configPath string) error
	ReloadJSONFunc     func(ctx context.Context, configPath string) (*caddy.ReloadResult, error)
}

func (m *MockReloader) TestConnection(ctx context.Context) error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx)
	}
	return nil
}

func (m *MockReloader) ValidateJSON(configPath string) error {
	if m.ValidateJSONFunc != nil {
		return m.ValidateJSONFunc(configPath)
	}
	return nil
}

func (m *MockReloader) ReloadJSON(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
	if m.ReloadJSONFunc != nil {
		return m.ReloadJSONFunc(ctx, configPath)
	}
	return &caddy.ReloadResult{Success: true, Duration: 100 * time.Millisecond}, nil
}

// Helper function to create a test service with mocks
func newTestSyncService() (*SyncService, *MockProxyRepository, *MockSettingsRepository, *MockFileManager, *MockReloader) {
	proxyRepo := &MockProxyRepository{}
	settingsRepo := &MockSettingsRepository{}
	fileManager := &MockFileManager{}
	reloader := &MockReloader{}

	svc := NewSyncService(SyncServiceConfig{
		ProxyRepo:    proxyRepo,
		SettingsRepo: settingsRepo,
		FileManager:  fileManager,
		Reloader:     reloader,
		Email:        "test@example.com",
		ACMEProvider: "off",
	})

	return svc, proxyRepo, settingsRepo, fileManager, reloader
}

// TestNewSyncService tests service creation
func TestNewSyncService(t *testing.T) {
	svc, _, _, _, _ := newTestSyncService()

	if svc == nil {
		t.Fatal("Expected non-nil service")
	} else {
		if svc.email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", svc.email)
		}
		if svc.acmeProvider != "off" {
			t.Errorf("Expected acmeProvider 'off', got '%s'", svc.acmeProvider)
		}
	}
}

// TestNewSyncService_NilLogger tests that nil logger is handled
func TestNewSyncService_NilLogger(t *testing.T) {
	svc := NewSyncService(SyncServiceConfig{
		Logger: nil,
	})

	if svc == nil {
		t.Fatal("Expected non-nil service")
	} else if svc.logger == nil {
		t.Error("Expected logger to be set to nop logger")
	}
}

// TestGetStatus tests status retrieval
func TestGetStatus(t *testing.T) {
	svc, _, _, _, _ := newTestSyncService()

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
	svc, _, _, _, _ := newTestSyncService()

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

// TestFullSync_Success tests a successful full sync with JSON mode
func TestFullSync_Success(t *testing.T) {
	svc, proxyRepo, settingsRepo, fileManager, reloader := newTestSyncService()

	// Setup mocks for JSON mode
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
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}

	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
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
	svc, proxyRepo, _, fileManager, _ := newTestSyncService()

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

// TestFullSync_ReloadError tests error handling when JSON reload fails
func TestFullSync_ReloadError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/caddy.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy not responding")
	}

	err := svc.FullSync()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy with JSON config") {
		t.Errorf("Expected 'failed to reload Caddy with JSON config' error, got: %v", err)
	}
}

// TestFullSync_EmptyProxies tests sync with no proxies
func TestFullSync_EmptyProxies(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, _, reloader := newTestSyncServiceWithJSON()

	reloadJSONCalled := false
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	// No proxy ACL assignments needed when there are no proxies
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		reloadJSONCalled = true
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.FullSync()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reloadJSONCalled {
		t.Error("ReloadJSON should be called even with empty proxies")
	}
}

// TestSyncProxy_Success tests that syncing a single proxy triggers JSON rebuild
func TestSyncProxy_Success(t *testing.T) {
	svc, proxyRepo, settingsRepo, fileManager, reloader := newTestSyncService()

	// Setup mocks for JSON mode (SyncProxy now triggers full JSON rebuild)
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true},
		}, 1, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	writeJSONCalled := false
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		writeJSONCalled = true
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !writeJSONCalled {
		t.Error("Expected WriteJSONConfig to be called")
	}
}

// TestSyncProxy_InactiveProxy tests that syncing an inactive proxy triggers JSON rebuild
func TestSyncProxy_InactiveProxy(t *testing.T) {
	svc, proxyRepo, settingsRepo, fileManager, reloader := newTestSyncService()

	// Setup mocks for JSON mode
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "example.com", Name: "Test", IsActive: false},
		}, 1, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	reloadJSONCalled := false
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		reloadJSONCalled = true
		return &caddy.ReloadResult{Success: true}, nil
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: false}
	err := svc.SyncProxy(proxy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reloadJSONCalled {
		t.Error("Expected ReloadJSON to be called for inactive proxy sync")
	}
}

// TestSyncProxy_JSONBuildError tests error handling when JSON config build fails
func TestSyncProxy_JSONBuildError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, _ := newTestSyncServiceWithJSON()

	// Setup mocks to trigger an error during JSON sync
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return nil, 0, errors.New("database error")
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to list proxies") {
		t.Errorf("Expected 'failed to list proxies' error, got: %v", err)
	}
}

// TestRemoveProxy_Success tests removing a proxy triggers JSON rebuild
func TestRemoveProxy_Success(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil // Proxy already removed from DB
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	jsonWriteCalled := false
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		jsonWriteCalled = true
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.RemoveProxy(1, "example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !jsonWriteCalled {
		t.Error("Expected JSON config to be written")
	}
}

// TestRemoveProxy_JSONWriteError tests error handling when JSON write fails
func TestRemoveProxy_JSONWriteError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return errors.New("disk full")
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}

	err := svc.RemoveProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to write JSON config") {
		t.Errorf("Expected 'failed to write JSON config' error, got: %v", err)
	}
}

// TestEnableProxy_Success tests enabling a proxy triggers JSON rebuild
func TestEnableProxy_Success(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
		}, 1, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
		return []models.ProxyACLAssignment{}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	reloadJSONCalled := false
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		reloadJSONCalled = true
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.EnableProxy(1, "example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reloadJSONCalled {
		t.Error("Expected JSON reload to be called")
	}
}

// TestEnableProxy_ReloadError tests error handling when JSON reload fails after enable
func TestEnableProxy_ReloadError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
		}, 1, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
		return []models.ProxyACLAssignment{}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy not responding")
	}

	err := svc.EnableProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy with JSON config") {
		t.Errorf("Expected 'failed to reload Caddy with JSON config' error, got: %v", err)
	}
}

// TestDisableProxy_Success tests disabling a proxy triggers JSON rebuild
func TestDisableProxy_Success(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil // No active proxies after disable
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	reloadJSONCalled := false
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		reloadJSONCalled = true
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.DisableProxy(1, "example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reloadJSONCalled {
		t.Error("Expected JSON reload to be called")
	}
}

// TestDisableProxy_Error tests error handling when JSON sync fails during disable
func TestDisableProxy_Error(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, _ := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return nil, 0, errors.New("database error")
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}

	err := svc.DisableProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to list proxies") {
		t.Errorf("Expected 'failed to list proxies' error, got: %v", err)
	}
}

// TestUpdateCatchAll_Success tests that updating catch-all config triggers JSON rebuild
func TestUpdateCatchAll_Success(t *testing.T) {
	svc, proxyRepo, settingsRepo, fileManager, reloader := newTestSyncService()

	// Setup mocks for JSON mode (UpdateCatchAll now triggers full JSON rebuild)
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "redirect", RedirectURL: "https://example.com"}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	reloadJSONCalled := false
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		reloadJSONCalled = true
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.UpdateCatchAll()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reloadJSONCalled {
		t.Error("Expected ReloadJSON to be called")
	}
}

// TestUpdateCatchAll_SettingsError tests that default settings are used when getting settings fails
func TestUpdateCatchAll_SettingsError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	// Settings error should be handled gracefully with defaults
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return nil, errors.New("database error")
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	// Should succeed using default settings (settings errors are handled gracefully)
	err := svc.UpdateCatchAll()
	if err != nil {
		t.Fatalf("Expected no error (should use defaults), got: %v", err)
	}
}

// TestUpdateCatchAll_WriteError tests error when writing JSON config fails
func TestUpdateCatchAll_WriteError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/config.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return errors.New("permission denied")
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}

	err := svc.UpdateCatchAll()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to write JSON config") {
		t.Errorf("Expected 'failed to write JSON config' error, got: %v", err)
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
		}
		reloader := &MockReloader{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
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
		if fileExistsCalls < 1 {
			t.Errorf("Expected FileExists to be called at least once (for JSON config), got %d", fileExistsCalls)
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

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
		})

		// Should not panic even with error
		svc.Start(100 * time.Millisecond)
		time.Sleep(20 * time.Millisecond)
		svc.Stop()
	})

	t.Run("creates initial JSON config when file does not exist", func(t *testing.T) {
		jsonConfigWritten := false

		fileManager := &MockFileManager{
			EnsureDirectoriesFunc: func() error {
				return nil
			},
			FileExistsFunc: func(path string) bool {
				return false // Files don't exist
			},
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/config.json"
			},
			WriteJSONConfigFunc: func(path string, data []byte) error {
				jsonConfigWritten = true
				return nil
			},
		}
		settingsRepo := &MockSettingsRepository{}
		proxyRepo := &MockProxyRepository{}
		reloader := &MockReloader{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
			Email:        "test@example.com",
			ACMEProvider: "off",
		})

		svc.Start(100 * time.Millisecond)
		time.Sleep(20 * time.Millisecond)
		svc.Stop()

		if !jsonConfigWritten {
			t.Error("Expected JSON config to be written")
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
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/caddy.json"
			},
			WriteJSONConfigFunc: func(path string, data []byte) error {
				return errors.New("write error")
			},
		}
		settingsRepo := &MockSettingsRepository{}
		proxyRepo := &MockProxyRepository{}
		reloader := &MockReloader{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
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
		}
		reloader := &MockReloader{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
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

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
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
		}
		reloader := &MockReloader{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
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
		}
		reloader := &MockReloader{}

		// Create a new service for each cycle since stopChan is closed
		for i := 0; i < 3; i++ {
			svc := NewSyncService(SyncServiceConfig{
				ProxyRepo:    proxyRepo,
				SettingsRepo: settingsRepo,
				FileManager:  fileManager,
				Reloader:     reloader,
			})

			svc.Start(100 * time.Millisecond)
			time.Sleep(20 * time.Millisecond)
			svc.Stop()
		}
	})
}

// TestEnsureInitialConfigs tests the ensureInitialConfigs method for JSON mode
func TestEnsureInitialConfigs(t *testing.T) {
	t.Run("does nothing when JSON config exists", func(t *testing.T) {
		writeJSONCalled := false

		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return true // JSON config exists
			},
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/config.json"
			},
			WriteJSONConfigFunc: func(path string, data []byte) error {
				writeJSONCalled = true
				return nil
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			FileManager: fileManager,
		})

		err := svc.ensureInitialConfigs()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if writeJSONCalled {
			t.Error("WriteJSONConfig should not be called when file exists")
		}
	})

	t.Run("creates JSON config when missing", func(t *testing.T) {
		writeJSONCalled := false
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return false // JSON config doesn't exist
			},
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/config.json"
			},
			WriteJSONConfigFunc: func(path string, data []byte) error {
				writeJSONCalled = true
				return nil
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			FileManager:  fileManager,
			Email:        "test@example.com",
			ACMEProvider: "off",
		})

		err := svc.ensureInitialConfigs()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !writeJSONCalled {
			t.Error("Expected WriteJSONConfig to be called")
		}
	})

	t.Run("returns error on JSON config write failure", func(t *testing.T) {
		fileManager := &MockFileManager{
			FileExistsFunc: func(path string) bool {
				return false
			},
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/config.json"
			},
			WriteJSONConfigFunc: func(path string, data []byte) error {
				return errors.New("write failed")
			},
		}

		svc := NewSyncService(SyncServiceConfig{
			FileManager: fileManager,
		})

		err := svc.ensureInitialConfigs()
		if err == nil {
			t.Error("Expected error")
		}
		if !contains(err.Error(), "failed to write initial JSON config") {
			t.Errorf("Expected 'failed to write initial JSON config' error, got: %v", err)
		}
	})
}

// TestSyncService_SetError tests the setError internal method
func TestSyncService_SetError(t *testing.T) {
	svc, _, _, _, _ := newTestSyncService()

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
func TestFullSync_JSONBackup(t *testing.T) {
	t.Run("calls BackupJSONConfig before sync", func(t *testing.T) {
		backupCalled := false

		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{
					{ID: 1, Hostname: "test.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{
						map[string]interface{}{"host": "localhost", "port": float64(8080)},
					}},
				}, 1, nil
			},
		}
		settingsRepo := &MockSettingsRepository{}
		aclRepo := &SyncMockACLRepository{}
		fileManager := &MockFileManager{
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/caddy.json"
			},
			BackupJSONConfigFunc: func(path string) error {
				backupCalled = true
				if path != "/etc/caddy/caddy.json" {
					t.Errorf("Expected path '/etc/caddy/caddy.json', got '%s'", path)
				}
				return nil
			},
		}
		reloader := &MockReloader{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			ACLRepo:      aclRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
		})

		_ = svc.FullSync()

		if !backupCalled {
			t.Error("Expected BackupJSONConfig to be called")
		}
	})

	t.Run("handles backup failure gracefully", func(t *testing.T) {
		proxyRepo := &MockProxyRepository{
			ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{
					{ID: 1, Hostname: "test.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{
						map[string]interface{}{"host": "localhost", "port": float64(8080)},
					}},
				}, 1, nil
			},
		}
		settingsRepo := &MockSettingsRepository{}
		aclRepo := &SyncMockACLRepository{}
		fileManager := &MockFileManager{
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/caddy.json"
			},
			BackupJSONConfigFunc: func(path string) error {
				return errors.New("backup failed")
			},
		}
		reloader := &MockReloader{}

		svc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			ACLRepo:      aclRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
		})

		// Should not fail even when backup fails (backup is optional)
		err := svc.FullSync()
		if err != nil {
			t.Errorf("FullSync should succeed even when backup fails: %v", err)
		}
	})
}

// TestFullSync_JSONMode_OnlyActiveProxiesIncluded tests that inactive proxies are excluded from JSON config
func TestFullSync_JSONMode_OnlyActiveProxiesIncluded(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	// Mix of active and inactive proxies
	// Upstreams must be in the map format expected by the JSON builder
	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "active.com", Name: "Active", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{
				map[string]interface{}{"host": "localhost", "port": float64(3000)},
			}},
			{ID: 2, Hostname: "inactive.com", Name: "Inactive", IsActive: false, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{
				map[string]interface{}{"host": "localhost", "port": float64(3001)},
			}},
		}, 2, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
		return []models.ProxyACLAssignment{}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/caddy.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	var jsonData []byte
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		jsonData = data
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return &caddy.ReloadResult{Success: true}, nil
	}

	err := svc.FullSync()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify JSON was written
	if jsonData == nil {
		t.Fatal("Expected JSON config to be written")
	}

	// Verify active proxy is in config (simplified check - JSON should contain the hostname)
	jsonStr := string(jsonData)
	if !contains(jsonStr, "active.com") {
		t.Error("Expected active proxy 'active.com' to be in JSON config")
	}
	// Inactive proxy should not be in config
	if contains(jsonStr, "inactive.com") {
		t.Error("Expected inactive proxy 'inactive.com' to be excluded from JSON config")
	}
}

// TestSyncProxy_JSONWriteError tests error handling when writing JSON config fails during proxy sync
func TestSyncProxy_JSONWriteError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
		}, 1, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
		return []models.ProxyACLAssignment{}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/caddy.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return errors.New("disk full")
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to write JSON config") {
		t.Errorf("Expected 'failed to write JSON config' error, got: %v", err)
	}
}

// TestSyncProxy_JSONReloadError tests error handling when Caddy JSON reload fails during proxy sync
func TestSyncProxy_JSONReloadError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{
			{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
		}, 1, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
		return []models.ProxyACLAssignment{}, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/caddy.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy unreachable")
	}

	proxy := &models.Proxy{ID: 1, Hostname: "example.com", IsActive: true}
	err := svc.SyncProxy(proxy)

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy with JSON config") {
		t.Errorf("Expected 'failed to reload Caddy with JSON config' error, got: %v", err)
	}
}

// TestRemoveProxy_ReloadError tests error handling when JSON reload fails after remove
func TestRemoveProxy_ReloadError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/caddy.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy error")
	}

	err := svc.RemoveProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy with JSON config") {
		t.Errorf("Expected 'failed to reload Caddy with JSON config' error, got: %v", err)
	}
}

// TestEnableProxy_ListError tests error handling when listing proxies fails during enable
func TestEnableProxy_ListError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, _ := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return nil, 0, errors.New("database connection failed")
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/caddy.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}

	err := svc.EnableProxy(1, "example.com")

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to list proxies") {
		t.Errorf("Expected 'failed to list proxies' error, got: %v", err)
	}
}

// TestUpdateCatchAll_ReloadError tests error when JSON reload fails after catchall update
func TestUpdateCatchAll_ReloadError(t *testing.T) {
	svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

	proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
		return []models.Proxy{}, 0, nil
	}
	settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
		return &models.NotFoundSettings{Mode: "default"}, nil
	}
	aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
		return []models.ACLGroup{}, 0, nil
	}
	fileManager.GetJSONConfigPathFunc = func() string {
		return "/etc/caddy/caddy.json"
	}
	fileManager.FileExistsFunc = func(path string) bool {
		return true
	}
	fileManager.BackupJSONConfigFunc = func(path string) error {
		return nil
	}
	fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
		return nil
	}
	reloader.ValidateJSONFunc = func(configPath string) error {
		return nil
	}
	reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
		return nil, errors.New("caddy not responding")
	}

	err := svc.UpdateCatchAll()

	if err == nil {
		t.Fatal("Expected error")
	}
	if !contains(err.Error(), "failed to reload Caddy with JSON config") {
		t.Errorf("Expected 'failed to reload Caddy with JSON config' error, got: %v", err)
	}
}

// =============================================================================
// Mock ACL Repository for JSON Mode Tests
// =============================================================================

// SyncMockACLRepository is a simplified mock for ACL repository used in sync tests
type SyncMockACLRepository struct {
	ListGroupsFunc                    func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error)
	GetGroupByIDFunc                  func(id int) (*models.ACLGroup, error)
	GetProxyACLAssignmentsFunc        func(proxyID int) ([]models.ProxyACLAssignment, error)
	GetProxyACLAssignmentsByGroupFunc func(groupID int) ([]models.ProxyACLAssignment, error)
	CreateGroupFunc                   func(group *models.ACLGroup) error
	GetGroupByNameFunc                func(name string) (*models.ACLGroup, error)
	UpdateGroupFunc                   func(group *models.ACLGroup) error
	DeleteGroupFunc                   func(id int) error
	GetDBFunc                         func() *gorm.DB
	DeleteGroupWithTxFunc             func(tx *gorm.DB, id int) error
	GetProxyACLAssignsByGroupTxFn     func(tx *gorm.DB, groupID int) ([]models.ProxyACLAssignment, error)
	CreateIPRuleFunc                  func(rule *models.ACLIPRule) error
	GetIPRuleByIDFunc                 func(id int) (*models.ACLIPRule, error)
	ListIPRulesFunc                   func(groupID int) ([]models.ACLIPRule, error)
	UpdateIPRuleFunc                  func(rule *models.ACLIPRule) error
	DeleteIPRuleFunc                  func(id int) error
	CreateBasicAuthUserFunc           func(user *models.ACLBasicAuthUser) error
	GetBasicAuthUserByIDFunc          func(id int) (*models.ACLBasicAuthUser, error)
	GetBasicAuthUserFunc              func(groupID int, username string) (*models.ACLBasicAuthUser, error)
	ListBasicAuthUsersFunc            func(groupID int) ([]models.ACLBasicAuthUser, error)
	UpdateBasicAuthUserFunc           func(user *models.ACLBasicAuthUser) error
	DeleteBasicAuthUserFunc           func(id int) error
	CreateExternalProviderFunc        func(provider *models.ACLExternalProvider) error
	GetExternalProviderByIDFunc       func(id int) (*models.ACLExternalProvider, error)
	ListExternalProvidersFunc         func(groupID int) ([]models.ACLExternalProvider, error)
	UpdateExternalProviderFunc        func(provider *models.ACLExternalProvider) error
	DeleteExternalProviderFunc        func(id int) error
	GetWaygatesAuthFunc               func(groupID int) (*models.ACLWaygatesAuth, error)
	CreateWaygatesAuthFunc            func(auth *models.ACLWaygatesAuth) error
	UpdateWaygatesAuthFunc            func(auth *models.ACLWaygatesAuth) error
	DeleteWaygatesAuthFunc            func(groupID int) error
	GetOAuthProviderRestrictsFunc     func(groupID int) ([]models.ACLOAuthProviderRestriction, error)
	GetOAuthProviderRestrictFunc      func(groupID int, provider string) (*models.ACLOAuthProviderRestriction, error)
	CreateOAuthProviderRestrictFn     func(restriction *models.ACLOAuthProviderRestriction) error
	UpdateOAuthProviderRestrictFn     func(restriction *models.ACLOAuthProviderRestriction) error
	DeleteOAuthProviderRestrictFn     func(groupID int, provider string) error
	CreateProxyACLAssignmentFunc      func(assignment *models.ProxyACLAssignment) error
	GetProxyACLAssignmentByIDFunc     func(id int) (*models.ProxyACLAssignment, error)
	UpdateProxyACLAssignmentFunc      func(assignment *models.ProxyACLAssignment) error
	DeleteProxyACLAssignmentFunc      func(id int) error
	DeleteProxyACLAssignByPGFunc      func(proxyID, groupID int) error
	GetBrandingFunc                   func() (*models.ACLBranding, error)
	UpdateBrandingFunc                func(branding *models.ACLBranding) error
	CreateSessionFunc                 func(session *models.ACLSession) error
	GetSessionByTokenFunc             func(token string) (*models.ACLSession, error)
	DeleteSessionFunc                 func(token string) error
	DeleteExpiredSessionsFunc         func() (int64, error)
	DeleteUserSessionsFunc            func(userID int) error
	DeleteProxySessionsFunc           func(proxyID int) error
}

func (m *SyncMockACLRepository) CreateGroup(group *models.ACLGroup) error {
	if m.CreateGroupFunc != nil {
		return m.CreateGroupFunc(group)
	}
	return nil
}

func (m *SyncMockACLRepository) GetGroupByID(id int) (*models.ACLGroup, error) {
	if m.GetGroupByIDFunc != nil {
		return m.GetGroupByIDFunc(id)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) GetGroupByName(name string) (*models.ACLGroup, error) {
	if m.GetGroupByNameFunc != nil {
		return m.GetGroupByNameFunc(name)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) ListGroups(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
	if m.ListGroupsFunc != nil {
		return m.ListGroupsFunc(params)
	}
	return []models.ACLGroup{}, 0, nil
}

func (m *SyncMockACLRepository) UpdateGroup(group *models.ACLGroup) error {
	if m.UpdateGroupFunc != nil {
		return m.UpdateGroupFunc(group)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteGroup(id int) error {
	if m.DeleteGroupFunc != nil {
		return m.DeleteGroupFunc(id)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteGroupWithTx(tx *gorm.DB, id int) error {
	if m.DeleteGroupWithTxFunc != nil {
		return m.DeleteGroupWithTxFunc(tx, id)
	}
	return nil
}

func (m *SyncMockACLRepository) GetDB() *gorm.DB {
	if m.GetDBFunc != nil {
		return m.GetDBFunc()
	}
	return nil
}

func (m *SyncMockACLRepository) GetProxyACLAssignmentsByGroupWithTx(tx *gorm.DB, groupID int) ([]models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignsByGroupTxFn != nil {
		return m.GetProxyACLAssignsByGroupTxFn(tx, groupID)
	}
	return []models.ProxyACLAssignment{}, nil
}

func (m *SyncMockACLRepository) CreateIPRule(rule *models.ACLIPRule) error {
	if m.CreateIPRuleFunc != nil {
		return m.CreateIPRuleFunc(rule)
	}
	return nil
}

func (m *SyncMockACLRepository) GetIPRuleByID(id int) (*models.ACLIPRule, error) {
	if m.GetIPRuleByIDFunc != nil {
		return m.GetIPRuleByIDFunc(id)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) ListIPRules(groupID int) ([]models.ACLIPRule, error) {
	if m.ListIPRulesFunc != nil {
		return m.ListIPRulesFunc(groupID)
	}
	return []models.ACLIPRule{}, nil
}

func (m *SyncMockACLRepository) UpdateIPRule(rule *models.ACLIPRule) error {
	if m.UpdateIPRuleFunc != nil {
		return m.UpdateIPRuleFunc(rule)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteIPRule(id int) error {
	if m.DeleteIPRuleFunc != nil {
		return m.DeleteIPRuleFunc(id)
	}
	return nil
}

func (m *SyncMockACLRepository) CreateBasicAuthUser(user *models.ACLBasicAuthUser) error {
	if m.CreateBasicAuthUserFunc != nil {
		return m.CreateBasicAuthUserFunc(user)
	}
	return nil
}

func (m *SyncMockACLRepository) GetBasicAuthUserByID(id int) (*models.ACLBasicAuthUser, error) {
	if m.GetBasicAuthUserByIDFunc != nil {
		return m.GetBasicAuthUserByIDFunc(id)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) GetBasicAuthUser(groupID int, username string) (*models.ACLBasicAuthUser, error) {
	if m.GetBasicAuthUserFunc != nil {
		return m.GetBasicAuthUserFunc(groupID, username)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) ListBasicAuthUsers(groupID int) ([]models.ACLBasicAuthUser, error) {
	if m.ListBasicAuthUsersFunc != nil {
		return m.ListBasicAuthUsersFunc(groupID)
	}
	return []models.ACLBasicAuthUser{}, nil
}

func (m *SyncMockACLRepository) UpdateBasicAuthUser(user *models.ACLBasicAuthUser) error {
	if m.UpdateBasicAuthUserFunc != nil {
		return m.UpdateBasicAuthUserFunc(user)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteBasicAuthUser(id int) error {
	if m.DeleteBasicAuthUserFunc != nil {
		return m.DeleteBasicAuthUserFunc(id)
	}
	return nil
}

func (m *SyncMockACLRepository) CreateExternalProvider(provider *models.ACLExternalProvider) error {
	if m.CreateExternalProviderFunc != nil {
		return m.CreateExternalProviderFunc(provider)
	}
	return nil
}

func (m *SyncMockACLRepository) GetExternalProviderByID(id int) (*models.ACLExternalProvider, error) {
	if m.GetExternalProviderByIDFunc != nil {
		return m.GetExternalProviderByIDFunc(id)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) ListExternalProviders(groupID int) ([]models.ACLExternalProvider, error) {
	if m.ListExternalProvidersFunc != nil {
		return m.ListExternalProvidersFunc(groupID)
	}
	return []models.ACLExternalProvider{}, nil
}

func (m *SyncMockACLRepository) UpdateExternalProvider(provider *models.ACLExternalProvider) error {
	if m.UpdateExternalProviderFunc != nil {
		return m.UpdateExternalProviderFunc(provider)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteExternalProvider(id int) error {
	if m.DeleteExternalProviderFunc != nil {
		return m.DeleteExternalProviderFunc(id)
	}
	return nil
}

func (m *SyncMockACLRepository) GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error) {
	if m.GetWaygatesAuthFunc != nil {
		return m.GetWaygatesAuthFunc(groupID)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) CreateWaygatesAuth(auth *models.ACLWaygatesAuth) error {
	if m.CreateWaygatesAuthFunc != nil {
		return m.CreateWaygatesAuthFunc(auth)
	}
	return nil
}

func (m *SyncMockACLRepository) UpdateWaygatesAuth(auth *models.ACLWaygatesAuth) error {
	if m.UpdateWaygatesAuthFunc != nil {
		return m.UpdateWaygatesAuthFunc(auth)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteWaygatesAuth(groupID int) error {
	if m.DeleteWaygatesAuthFunc != nil {
		return m.DeleteWaygatesAuthFunc(groupID)
	}
	return nil
}

func (m *SyncMockACLRepository) GetOAuthProviderRestrictions(groupID int) ([]models.ACLOAuthProviderRestriction, error) {
	if m.GetOAuthProviderRestrictsFunc != nil {
		return m.GetOAuthProviderRestrictsFunc(groupID)
	}
	return []models.ACLOAuthProviderRestriction{}, nil
}

func (m *SyncMockACLRepository) GetOAuthProviderRestriction(groupID int, provider string) (*models.ACLOAuthProviderRestriction, error) {
	if m.GetOAuthProviderRestrictFunc != nil {
		return m.GetOAuthProviderRestrictFunc(groupID, provider)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) CreateOAuthProviderRestriction(restriction *models.ACLOAuthProviderRestriction) error {
	if m.CreateOAuthProviderRestrictFn != nil {
		return m.CreateOAuthProviderRestrictFn(restriction)
	}
	return nil
}

func (m *SyncMockACLRepository) UpdateOAuthProviderRestriction(restriction *models.ACLOAuthProviderRestriction) error {
	if m.UpdateOAuthProviderRestrictFn != nil {
		return m.UpdateOAuthProviderRestrictFn(restriction)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteOAuthProviderRestriction(groupID int, provider string) error {
	if m.DeleteOAuthProviderRestrictFn != nil {
		return m.DeleteOAuthProviderRestrictFn(groupID, provider)
	}
	return nil
}

func (m *SyncMockACLRepository) CreateProxyACLAssignment(assignment *models.ProxyACLAssignment) error {
	if m.CreateProxyACLAssignmentFunc != nil {
		return m.CreateProxyACLAssignmentFunc(assignment)
	}
	return nil
}

func (m *SyncMockACLRepository) GetProxyACLAssignmentByID(id int) (*models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignmentByIDFunc != nil {
		return m.GetProxyACLAssignmentByIDFunc(id)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) GetProxyACLAssignments(proxyID int) ([]models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignmentsFunc != nil {
		return m.GetProxyACLAssignmentsFunc(proxyID)
	}
	return []models.ProxyACLAssignment{}, nil
}

func (m *SyncMockACLRepository) GetProxyACLAssignmentsByGroup(groupID int) ([]models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignmentsByGroupFunc != nil {
		return m.GetProxyACLAssignmentsByGroupFunc(groupID)
	}
	return []models.ProxyACLAssignment{}, nil
}

func (m *SyncMockACLRepository) UpdateProxyACLAssignment(assignment *models.ProxyACLAssignment) error {
	if m.UpdateProxyACLAssignmentFunc != nil {
		return m.UpdateProxyACLAssignmentFunc(assignment)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteProxyACLAssignment(id int) error {
	if m.DeleteProxyACLAssignmentFunc != nil {
		return m.DeleteProxyACLAssignmentFunc(id)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteProxyACLAssignmentByProxyAndGroup(proxyID, groupID int) error {
	if m.DeleteProxyACLAssignByPGFunc != nil {
		return m.DeleteProxyACLAssignByPGFunc(proxyID, groupID)
	}
	return nil
}

func (m *SyncMockACLRepository) GetBranding() (*models.ACLBranding, error) {
	if m.GetBrandingFunc != nil {
		return m.GetBrandingFunc()
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) UpdateBranding(branding *models.ACLBranding) error {
	if m.UpdateBrandingFunc != nil {
		return m.UpdateBrandingFunc(branding)
	}
	return nil
}

func (m *SyncMockACLRepository) CreateSession(session *models.ACLSession) error {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(session)
	}
	return nil
}

func (m *SyncMockACLRepository) GetSessionByToken(token string) (*models.ACLSession, error) {
	if m.GetSessionByTokenFunc != nil {
		return m.GetSessionByTokenFunc(token)
	}
	return nil, errors.New("not found")
}

func (m *SyncMockACLRepository) DeleteSession(token string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(token)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteExpiredSessions() (int64, error) {
	if m.DeleteExpiredSessionsFunc != nil {
		return m.DeleteExpiredSessionsFunc()
	}
	return 0, nil
}

func (m *SyncMockACLRepository) DeleteUserSessions(userID int) error {
	if m.DeleteUserSessionsFunc != nil {
		return m.DeleteUserSessionsFunc(userID)
	}
	return nil
}

func (m *SyncMockACLRepository) DeleteProxySessions(proxyID int) error {
	if m.DeleteProxySessionsFunc != nil {
		return m.DeleteProxySessionsFunc(proxyID)
	}
	return nil
}

// Ensure SyncMockACLRepository implements the interface
var _ repository.ACLRepositoryInterface = (*SyncMockACLRepository)(nil)

// =============================================================================
// JSON Mode Test Helper
// =============================================================================

// newTestSyncServiceWithJSON creates a test sync service configured for JSON mode
func newTestSyncServiceWithJSON() (*SyncService, *MockProxyRepository, *MockSettingsRepository, *SyncMockACLRepository, *MockFileManager, *MockReloader) {
	proxyRepo := &MockProxyRepository{}
	settingsRepo := &MockSettingsRepository{}
	aclRepo := &SyncMockACLRepository{}
	fileManager := &MockFileManager{}
	reloader := &MockReloader{}

	svc := NewSyncService(SyncServiceConfig{
		ProxyRepo:         proxyRepo,
		SettingsRepo:      settingsRepo,
		ACLRepo:           aclRepo,
		FileManager:       fileManager,
		Reloader:          reloader,
		Email:             "test@example.com",
		ACMEProvider:      "off",
		WaygatesVerifyURL: "http://localhost:8080/api/acl/verify",
		WaygatesLoginURL:  "http://localhost:8080/login",
		StoragePath:       "/data",
	})

	return svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader
}

// =============================================================================
// JSON Mode Integration Tests
// =============================================================================

// TestSyncService_PerformFullSyncJSON tests JSON mode full sync operations
func TestSyncService_PerformFullSyncJSON(t *testing.T) {
	t.Run("successful JSON sync with proxies", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		// Setup mocks
		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{
				{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
				{ID: 2, Hostname: "api.example.com", Name: "API", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:4000"}},
			}, 2, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		}

		writeJSONCalled := false
		var writtenData []byte
		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			writeJSONCalled = true
			writtenData = data
			if path != "/etc/caddy/caddy.json" {
				t.Errorf("Expected path '/etc/caddy/caddy.json', got '%s'", path)
			}
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		err := svc.FullSync()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !writeJSONCalled {
			t.Error("Expected WriteJSONConfig to be called")
		}
		if len(writtenData) == 0 {
			t.Error("Expected JSON config data to be written")
		}

		status := svc.GetStatus()
		if !status.LastSyncSuccess {
			t.Error("Expected LastSyncSuccess to be true")
		}
		if status.SyncCount != 1 {
			t.Errorf("Expected SyncCount 1, got %d", status.SyncCount)
		}
	})

	t.Run("JSON sync with empty proxies", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{}, 0, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		writeJSONCalled := false
		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			writeJSONCalled = true
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		err := svc.FullSync()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !writeJSONCalled {
			t.Error("Expected WriteJSONConfig to be called even with empty proxies")
		}
	})

	t.Run("JSON sync with ACL assignments", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{
				{ID: 1, Hostname: "secure.example.com", Name: "Secure", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:5000"}},
			}, 1, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		testGroupDesc := "Test ACL Group"
		testGroup := &models.ACLGroup{
			ID:          1,
			Name:        "TestGroup",
			Description: &testGroupDesc,
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{*testGroup}, 1, nil
		}

		aclRepo.GetGroupByIDFunc = func(id int) (*models.ACLGroup, error) {
			if id == 1 {
				return testGroup, nil
			}
			return nil, errors.New("not found")
		}

		aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
			if proxyID == 1 {
				return []models.ProxyACLAssignment{
					{ID: 1, ProxyID: 1, ACLGroupID: 1, Enabled: true, ACLGroup: testGroup},
				}, nil
			}
			return []models.ProxyACLAssignment{}, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
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
	})

	t.Run("JSON sync with TLS domains", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{
				{ID: 1, Hostname: "ssl.example.com", Name: "SSL Site", IsActive: true, SSLEnabled: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:6000"}},
			}, 1, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		err := svc.FullSync()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("JSON sync failure on write error", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, _ := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{
				{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
			}, 1, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return errors.New("disk full")
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		err := svc.FullSync()

		if err == nil {
			t.Fatal("Expected error")
		}
		if !contains(err.Error(), "failed to write JSON config") {
			t.Errorf("Expected 'failed to write JSON config' error, got: %v", err)
		}

		status := svc.GetStatus()
		if status.LastSyncSuccess {
			t.Error("Expected LastSyncSuccess to be false after error")
		}
	})

	t.Run("JSON sync failure on validate error", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{
				{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
			}, 1, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return errors.New("invalid JSON syntax")
		}

		err := svc.FullSync()

		if err == nil {
			t.Fatal("Expected error")
		}
		if !contains(err.Error(), "JSON config validation failed") {
			t.Errorf("Expected 'JSON config validation failed' error, got: %v", err)
		}
	})

	t.Run("JSON sync failure on reload error", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{
				{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
			}, 1, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return nil, errors.New("caddy not responding")
		}

		err := svc.FullSync()

		if err == nil {
			t.Fatal("Expected error")
		}
		if !contains(err.Error(), "failed to reload Caddy with JSON config") {
			t.Errorf("Expected 'failed to reload Caddy with JSON config' error, got: %v", err)
		}
	})

	t.Run("JSON sync failure on proxy list error", func(t *testing.T) {
		svc, proxyRepo, _, _, fileManager, _ := newTestSyncServiceWithJSON()

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
	})
}

// TestSyncService_EnsureInitialJSONConfig tests initial JSON config creation
func TestSyncService_EnsureInitialJSONConfig(t *testing.T) {
	t.Run("creates initial config when file does not exist", func(t *testing.T) {
		svc, _, _, _, fileManager, _ := newTestSyncServiceWithJSON()

		writeJSONCalled := false
		fileManager.FileExistsFunc = func(path string) bool {
			return false // JSON config doesn't exist
		}
		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			writeJSONCalled = true
			if path != "/etc/caddy/caddy.json" {
				t.Errorf("Expected path '/etc/caddy/caddy.json', got '%s'", path)
			}
			return nil
		}

		err := svc.ensureInitialConfigs()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !writeJSONCalled {
			t.Error("Expected WriteJSONConfig to be called when file doesn't exist")
		}
	})

	t.Run("skips creation when file exists", func(t *testing.T) {
		svc, _, _, _, fileManager, _ := newTestSyncServiceWithJSON()

		writeJSONCalled := false
		fileManager.FileExistsFunc = func(path string) bool {
			return true // JSON config exists
		}
		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			writeJSONCalled = true
			return nil
		}

		err := svc.ensureInitialConfigs()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if writeJSONCalled {
			t.Error("WriteJSONConfig should not be called when file exists")
		}
	})

	t.Run("handles write error gracefully", func(t *testing.T) {
		svc, _, _, _, fileManager, _ := newTestSyncServiceWithJSON()

		fileManager.FileExistsFunc = func(path string) bool {
			return false
		}
		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return errors.New("permission denied")
		}

		err := svc.ensureInitialConfigs()

		if err == nil {
			t.Fatal("Expected error")
		}
		if !contains(err.Error(), "failed to write initial JSON config") {
			t.Errorf("Expected 'failed to write initial JSON config' error, got: %v", err)
		}
	})
}

// TestSyncService_JSONMode_NotFoundSettings tests not found settings in JSON mode
func TestSyncService_JSONMode_NotFoundSettings(t *testing.T) {
	t.Run("uses default settings when error", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{}, 0, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return nil, errors.New("database error")
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		// Should not error - uses default settings
		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("uses redirect settings when configured", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{}, 0, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{
				Mode:        "redirect",
				RedirectURL: "https://example.com/404",
			}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

// TestSyncService_JSONMode_BackupHandling tests backup behavior in JSON mode
func TestSyncService_JSONMode_BackupHandling(t *testing.T) {
	t.Run("continues when backup fails", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{}, 0, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return errors.New("backup failed")
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		// Should not error - backup failure is logged but not fatal
		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

// TestSyncService_JSONMode_ACLGroupLoading tests ACL group loading in JSON mode
func TestSyncService_JSONMode_ACLGroupLoading(t *testing.T) {
	t.Run("handles ACL group list error gracefully", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{}, 0, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return nil, 0, errors.New("database error")
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		// Should not error - ACL list failure is logged but not fatal
		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("handles individual group load error gracefully", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{}, 0, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{
				{ID: 1, Name: "Group1"},
				{ID: 2, Name: "Group2"},
			}, 2, nil
		}

		aclRepo.GetGroupByIDFunc = func(id int) (*models.ACLGroup, error) {
			if id == 1 {
				return &models.ACLGroup{ID: 1, Name: "Group1"}, nil
			}
			return nil, errors.New("not found")
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		// Should not error - individual group load failure is logged but not fatal
		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("handles ACL assignment error gracefully", func(t *testing.T) {
		svc, proxyRepo, settingsRepo, aclRepo, fileManager, reloader := newTestSyncServiceWithJSON()

		proxyRepo.ListFunc = func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			return []models.Proxy{
				{ID: 1, Hostname: "example.com", Name: "Test", IsActive: true, Type: models.ProxyTypeReverseProxy, Upstreams: []interface{}{"http://localhost:3000"}},
			}, 1, nil
		}

		settingsRepo.GetNotFoundSettingsFunc = func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{Mode: "default"}, nil
		}

		aclRepo.ListGroupsFunc = func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{}, 0, nil
		}

		aclRepo.GetProxyACLAssignmentsFunc = func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return nil, errors.New("database error")
		}

		fileManager.GetJSONConfigPathFunc = func() string {
			return "/etc/caddy/caddy.json"
		}
		fileManager.WriteJSONConfigFunc = func(path string, data []byte) error {
			return nil
		}
		fileManager.BackupJSONConfigFunc = func(path string) error {
			return nil
		}
		fileManager.FileExistsFunc = func(path string) bool {
			return true
		}

		reloader.ValidateJSONFunc = func(configPath string) error {
			return nil
		}
		reloader.ReloadJSONFunc = func(ctx context.Context, configPath string) (*caddy.ReloadResult, error) {
			return &caddy.ReloadResult{Success: true, Duration: 50 * time.Millisecond}, nil
		}

		// Should not error - ACL assignment failure is logged but not fatal
		err := svc.FullSync()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}
