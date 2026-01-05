package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// mockAuditLogRepo implements a mock audit log repository for testing
type mockAuditLogRepo struct {
	logs      []*models.AuditLog
	createErr error
	getErr    error
	listErr   error
	statsErr  error
}

func newMockAuditLogRepo() *mockAuditLogRepo {
	return &mockAuditLogRepo{
		logs: make([]*models.AuditLog, 0),
	}
}

func (m *mockAuditLogRepo) Create(log *models.AuditLog) error {
	if m.createErr != nil {
		return m.createErr
	}
	log.ID = len(m.logs) + 1
	log.CreatedAt = time.Now()
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditLogRepo) GetByID(id int) (*models.AuditLog, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, log := range m.logs {
		if log.ID == id {
			return log, nil
		}
	}
	return nil, errors.New("audit log not found")
}

func (m *mockAuditLogRepo) List(params repository.AuditLogListParams) ([]models.AuditLog, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}

	// Simple filtering for tests
	result := make([]models.AuditLog, 0, len(m.logs))
	for _, log := range m.logs {
		if params.Action != "" && log.Action != params.Action {
			continue
		}
		if params.Status != "" && log.Status != params.Status {
			continue
		}
		result = append(result, *log)
	}

	total := int64(len(result))

	// Apply pagination
	start := (params.Page - 1) * params.Limit
	if start >= len(result) {
		return []models.AuditLog{}, total, nil
	}
	end := start + params.Limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], total, nil
}

func (m *mockAuditLogRepo) GetStats() (*models.AuditLogStats, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}

	stats := &models.AuditLogStats{
		TotalLogs:      int64(len(m.logs)),
		ByAction:       make(map[string]int64),
		ByStatus:       make(map[string]int64),
		ByResourceType: make(map[string]int64),
		RecentActivity: make([]models.AuditLog, 0),
	}

	for _, log := range m.logs {
		stats.ByAction[log.Action]++
		stats.ByStatus[log.Status]++
		if log.ResourceType != nil {
			stats.ByResourceType[*log.ResourceType]++
		}
	}

	// Add up to 10 recent logs
	count := len(m.logs)
	if count > 10 {
		count = 10
	}
	for i := len(m.logs) - 1; i >= len(m.logs)-count && i >= 0; i-- {
		stats.RecentActivity = append(stats.RecentActivity, *m.logs[i])
	}

	return stats, nil
}

func (m *mockAuditLogRepo) DeleteOlderThan(before time.Time) (int64, error) {
	var deleted int64
	var remaining []*models.AuditLog
	for _, log := range m.logs {
		if log.CreatedAt.Before(before) {
			deleted++
		} else {
			remaining = append(remaining, log)
		}
	}
	m.logs = remaining
	return deleted, nil
}

func (m *mockAuditLogRepo) CountByAction(action string) (int64, error) {
	var count int64
	for _, log := range m.logs {
		if log.Action == action {
			count++
		}
	}
	return count, nil
}

func (m *mockAuditLogRepo) CountByUserID(userID int) (int64, error) {
	var count int64
	for _, log := range m.logs {
		if log.UserID != nil && *log.UserID == userID {
			count++
		}
	}
	return count, nil
}

// mockSettingsService implements a mock settings service for testing
type mockSettingsService struct {
	settings map[string]string
	setErr   error
}

func newMockSettingsService() *mockSettingsService {
	return &mockSettingsService{
		settings: make(map[string]string),
	}
}

func (m *mockSettingsService) Get(key string) (string, error) {
	if v, ok := m.settings[key]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

func (m *mockSettingsService) GetWithDefault(key, defaultValue string) string {
	if v, ok := m.settings[key]; ok {
		return v
	}
	return defaultValue
}

func (m *mockSettingsService) Set(key, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.settings[key] = value
	return nil
}

func (m *mockSettingsService) GetAll() (map[string]string, error) {
	return m.settings, nil
}

func (m *mockSettingsService) Delete(key string) error {
	delete(m.settings, key)
	return nil
}

func (m *mockSettingsService) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	return &models.NotFoundSettings{Mode: "default"}, nil
}

func (m *mockSettingsService) SetNotFoundSettings(settings *models.NotFoundSettings) error {
	return nil
}

func TestNewAuditService(t *testing.T) {
	// Test with nil logger
	svc := NewAuditService(nil, nil, nil)
	if svc == nil {
		t.Fatal("Expected service to be created")
	}
	if svc.logger == nil {
		t.Error("Expected logger to be set (nop logger)")
	}

	// Test with actual logger
	logger := zap.NewNop()
	svc = NewAuditService(nil, nil, logger)
	if svc.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestAuditService_LogEvent(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	userID := 1
	resourceID := 5
	event := models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionProxyCreate,
		ResourceType: models.AuditResourceTypeProxy,
		ResourceID:   &resourceID,
		ResourceName: "Test Proxy",
		Details: map[string]interface{}{
			"hostname": "test.example.com",
		},
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		Status:    models.AuditStatusSuccess,
	}

	err := svc.LogEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(repo.logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(repo.logs))
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionProxyCreate {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionProxyCreate, log.Action)
	}
	if log.Status != models.AuditStatusSuccess {
		t.Errorf("Expected status '%s', got '%s'", models.AuditStatusSuccess, log.Status)
	}
	if log.UserID == nil || *log.UserID != 1 {
		t.Error("Expected UserID to be 1")
	}
	if log.ResourceType == nil || *log.ResourceType != models.AuditResourceTypeProxy {
		t.Error("Expected ResourceType to be 'proxy'")
	}
}

func TestAuditService_LogEvent_WithDetails(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	event := models.AuditEvent{
		Action: models.AuditActionSettingsUpdate,
		Details: map[string]interface{}{
			"key":       "theme",
			"old_value": "light",
			"new_value": "dark",
		},
		Status: models.AuditStatusSuccess,
	}

	err := svc.LogEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(repo.logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(repo.logs))
	}

	log := repo.logs[0]
	if log.Details == nil {
		t.Fatal("Expected Details to be set")
	}
	if log.Details["key"] != "theme" {
		t.Errorf("Expected Details['key'] to be 'theme', got '%v'", log.Details["key"])
	}
}

func TestAuditService_LogEvent_SystemAction(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// System actions have no user
	event := models.AuditEvent{
		Action:       models.AuditActionSystemStartup,
		ResourceType: models.AuditResourceTypeSystem,
		Status:       models.AuditStatusSuccess,
	}

	err := svc.LogEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.UserID != nil {
		t.Error("Expected UserID to be nil for system action")
	}
}

func TestAuditService_LogEvent_Error(t *testing.T) {
	repo := newMockAuditLogRepo()
	repo.createErr = errors.New("database error")
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	event := models.AuditEvent{
		Action: models.AuditActionProxyCreate,
		Status: models.AuditStatusSuccess,
	}

	err := svc.LogEvent(context.Background(), event)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestAuditService_LogEvent_CategoryDisabled(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()

	// Disable proxy_create event
	settingsSvc.settings[models.SettingAuditConfig] = `{"proxy_create":false,"proxy_update":true,"proxy_delete":true,"proxy_enable":true,"proxy_disable":true,"auth_login":true,"auth_logout":true,"auth_register":true,"auth_password_change":true,"auth_login_failed":true,"settings_update":true,"sync_started":true,"sync_completed":true,"sync_failed":true,"system_startup":true,"caddy_reload":true}`

	svc := NewAuditService(repo, settingsSvc, nil)

	event := models.AuditEvent{
		Action: models.AuditActionProxyCreate,
		Status: models.AuditStatusSuccess,
	}

	err := svc.LogEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not have logged anything
	if len(repo.logs) != 0 {
		t.Errorf("Expected 0 logs (proxy_create disabled), got %d", len(repo.logs))
	}
}

func TestAuditService_LogProxyCreate(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	proxy := &models.Proxy{
		ID:         1,
		Name:       "Test Proxy",
		Hostname:   "test.example.com",
		Type:       models.ProxyTypeReverseProxy,
		SSLEnabled: true,
	}

	err := svc.LogProxyCreate(context.Background(), 1, proxy, "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(repo.logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(repo.logs))
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionProxyCreate {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionProxyCreate, log.Action)
	}
	if *log.ResourceName != "Test Proxy" {
		t.Errorf("Expected resource name 'Test Proxy', got '%s'", *log.ResourceName)
	}
}

func TestAuditService_LogProxyUpdate(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	proxy := &models.Proxy{
		ID:       1,
		Name:     "Updated Proxy",
		Hostname: "updated.example.com",
		Type:     models.ProxyTypeReverseProxy,
	}

	changes := map[string]interface{}{
		"name": map[string]string{
			"old": "Old Name",
			"new": "Updated Proxy",
		},
	}

	err := svc.LogProxyUpdate(context.Background(), 1, proxy, changes, "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionProxyUpdate {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionProxyUpdate, log.Action)
	}
	if log.Details["changes"] == nil {
		t.Error("Expected changes in details")
	}
}

func TestAuditService_LogProxyDelete(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogProxyDelete(context.Background(), 1, 5, "Deleted Proxy", "deleted.example.com", "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionProxyDelete {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionProxyDelete, log.Action)
	}
	if *log.ResourceID != 5 {
		t.Errorf("Expected resource ID 5, got %d", *log.ResourceID)
	}
}

func TestAuditService_LogLogin_Success(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogLogin(context.Background(), 1, "testuser", "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionAuthLogin {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionAuthLogin, log.Action)
	}
	if log.Status != models.AuditStatusSuccess {
		t.Errorf("Expected status '%s', got '%s'", models.AuditStatusSuccess, log.Status)
	}
}

func TestAuditService_LogLogin_Failure(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogLoginFailed(context.Background(), "testuser", "192.168.1.1", "Mozilla/5.0", "invalid password")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionAuthLoginFailed {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionAuthLoginFailed, log.Action)
	}
	if log.Status != models.AuditStatusFailure {
		t.Errorf("Expected status '%s', got '%s'", models.AuditStatusFailure, log.Status)
	}
	if log.UserID != nil {
		t.Error("Expected UserID to be nil for failed login")
	}
	if log.ErrorMessage == nil || *log.ErrorMessage != "invalid password" {
		t.Error("Expected error message to be set")
	}
}

func TestAuditService_LogSettingsUpdate(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogSettingsUpdate(context.Background(), 1, "theme", "light", "dark", "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionSettingsUpdate {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionSettingsUpdate, log.Action)
	}
	if log.Details["old_value"] != "light" {
		t.Errorf("Expected old_value 'light', got '%v'", log.Details["old_value"])
	}
	if log.Details["new_value"] != "dark" {
		t.Errorf("Expected new_value 'dark', got '%v'", log.Details["new_value"])
	}
}

func TestAuditService_LogSyncEvent(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// Test sync started
	err := svc.LogSyncStarted(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test sync completed
	err = svc.LogSyncCompleted(context.Background(), 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test sync failed
	err = svc.LogSyncFailed(context.Background(), "connection failed")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(repo.logs) != 3 {
		t.Fatalf("Expected 3 logs, got %d", len(repo.logs))
	}

	// Check sync started
	if repo.logs[0].Action != models.AuditActionSyncStarted {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionSyncStarted, repo.logs[0].Action)
	}

	// Check sync completed
	if repo.logs[1].Action != models.AuditActionSyncCompleted {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionSyncCompleted, repo.logs[1].Action)
	}

	// Check sync failed
	if repo.logs[2].Action != models.AuditActionSyncFailed {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionSyncFailed, repo.logs[2].Action)
	}
	if repo.logs[2].Status != models.AuditStatusFailure {
		t.Errorf("Expected status '%s', got '%s'", models.AuditStatusFailure, repo.logs[2].Status)
	}
}

func TestAuditService_ListAuditLogs(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// Add some logs
	for i := 0; i < 25; i++ {
		repo.logs = append(repo.logs, &models.AuditLog{
			ID:        i + 1,
			Action:    models.AuditActionProxyCreate,
			Status:    models.AuditStatusSuccess,
			CreatedAt: time.Now(),
		})
	}

	params := repository.AuditLogListParams{
		Page:  1,
		Limit: 10,
	}

	result, err := svc.ListAuditLogs(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Total != 25 {
		t.Errorf("Expected Total 25, got %d", result.Total)
	}
	if len(result.Items) != 10 {
		t.Errorf("Expected 10 items, got %d", len(result.Items))
	}
	if result.TotalPages != 3 {
		t.Errorf("Expected 3 pages, got %d", result.TotalPages)
	}
}

func TestAuditService_GetAuditLogByID(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// Add a log
	repo.logs = append(repo.logs, &models.AuditLog{
		ID:        1,
		Action:    models.AuditActionProxyCreate,
		Status:    models.AuditStatusSuccess,
		CreatedAt: time.Now(),
	})

	log, err := svc.GetAuditLogByID(1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if log.ID != 1 {
		t.Errorf("Expected ID 1, got %d", log.ID)
	}
}

func TestAuditService_GetAuditLogByID_NotFound(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	_, err := svc.GetAuditLogByID(999)
	if err == nil {
		t.Error("Expected error for non-existent log")
	}
}

func TestAuditService_GetStats(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// Add some logs
	proxyType := models.AuditResourceTypeProxy
	repo.logs = append(repo.logs,
		&models.AuditLog{ID: 1, Action: models.AuditActionProxyCreate, Status: models.AuditStatusSuccess, ResourceType: &proxyType, CreatedAt: time.Now()},
		&models.AuditLog{ID: 2, Action: models.AuditActionProxyUpdate, Status: models.AuditStatusSuccess, ResourceType: &proxyType, CreatedAt: time.Now()},
		&models.AuditLog{ID: 3, Action: models.AuditActionAuthLogin, Status: models.AuditStatusSuccess, CreatedAt: time.Now()},
		&models.AuditLog{ID: 4, Action: models.AuditActionAuthLoginFailed, Status: models.AuditStatusFailure, CreatedAt: time.Now()},
	)

	stats, err := svc.GetStats()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stats.TotalLogs != 4 {
		t.Errorf("Expected TotalLogs 4, got %d", stats.TotalLogs)
	}
	if stats.ByAction[models.AuditActionProxyCreate] != 1 {
		t.Errorf("Expected 1 proxy.create, got %d", stats.ByAction[models.AuditActionProxyCreate])
	}
	if stats.ByStatus[models.AuditStatusSuccess] != 3 {
		t.Errorf("Expected 3 success, got %d", stats.ByStatus[models.AuditStatusSuccess])
	}
	if stats.ByStatus[models.AuditStatusFailure] != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.ByStatus[models.AuditStatusFailure])
	}
}

func TestAuditService_GetConfig(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// Test default config when no setting exists
	config, err := svc.GetConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !config.ProxyCreate {
		t.Error("Expected ProxyCreate to be true by default")
	}
	if !config.AuthLogin {
		t.Error("Expected AuthLogin to be true by default")
	}
}

func TestAuditService_SetConfig(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	config := &models.AuditConfig{
		ProxyCreate:        true,
		ProxyUpdate:        true,
		ProxyDelete:        true,
		ProxyEnable:        true,
		ProxyDisable:       true,
		AuthLogin:          false,
		AuthLogout:         false,
		AuthRegister:       false,
		AuthPasswordChange: false,
		AuthLoginFailed:    false,
		SettingsUpdate:     true,
		SyncStarted:        false,
		SyncCompleted:      false,
		SyncFailed:         false,
		SystemStartup:      true,
		CaddyReload:        true,
	}

	err := svc.SetConfig(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify config was saved
	savedConfig, err := svc.GetConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if savedConfig.AuthLogin != false {
		t.Error("Expected AuthLogin to be false")
	}
	if savedConfig.SyncStarted != false {
		t.Error("Expected SyncStarted to be false")
	}
}

func TestAuditService_isEventEnabled(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()

	// Set config with some events disabled
	settingsSvc.settings[models.SettingAuditConfig] = `{"proxy_create":true,"proxy_update":true,"proxy_delete":true,"proxy_enable":true,"proxy_disable":true,"auth_login":false,"auth_logout":false,"auth_register":false,"auth_password_change":false,"auth_login_failed":false,"settings_update":true,"sync_started":false,"sync_completed":false,"sync_failed":false,"system_startup":true,"caddy_reload":true}`

	svc := NewAuditService(repo, settingsSvc, nil)

	testCases := []struct {
		action   string
		expected bool
	}{
		{"proxy.create", true},
		{"proxy.update", true},
		{"proxy.delete", true},
		{"auth.login", false},
		{"auth.logout", false},
		{"auth.register", false},
		{"settings.update", true},
		{"sync.started", false},
		{"sync.completed", false},
		{"system.startup", true},
		{"caddy.reload", true},
		{"unknown.action", true}, // Unknown actions should be logged
	}

	for _, tc := range testCases {
		t.Run(tc.action, func(t *testing.T) {
			result := svc.isEventEnabled(tc.action)
			if result != tc.expected {
				t.Errorf("isEventEnabled(%s): expected %v, got %v", tc.action, tc.expected, result)
			}
		})
	}
}

func TestAuditService_InvalidateConfigCache(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// Set initial config
	config := &models.AuditConfig{
		ProxyCreate:        true,
		ProxyUpdate:        true,
		ProxyDelete:        true,
		ProxyEnable:        true,
		ProxyDisable:       true,
		AuthLogin:          true,
		AuthLogout:         true,
		AuthRegister:       true,
		AuthPasswordChange: true,
		AuthLoginFailed:    true,
		SettingsUpdate:     true,
		SyncStarted:        true,
		SyncCompleted:      true,
		SyncFailed:         true,
		SystemStartup:      true,
		CaddyReload:        true,
	}
	_ = svc.SetConfig(config)

	// Verify cache is set
	if svc.configCache == nil {
		t.Error("Expected configCache to be set")
	}

	// Invalidate cache
	svc.InvalidateConfigCache()

	// Verify cache is cleared
	if svc.configCache != nil {
		t.Error("Expected configCache to be nil after invalidation")
	}
}

func TestAuditService_LogProxyEnable(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	proxy := &models.Proxy{
		ID:       1,
		Name:     "Test Proxy",
		Hostname: "test.example.com",
	}

	err := svc.LogProxyEnable(context.Background(), 1, proxy, "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionProxyEnable {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionProxyEnable, log.Action)
	}
}

func TestAuditService_LogProxyDisable(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	proxy := &models.Proxy{
		ID:       1,
		Name:     "Test Proxy",
		Hostname: "test.example.com",
	}

	err := svc.LogProxyDisable(context.Background(), 1, proxy, "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionProxyDisable {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionProxyDisable, log.Action)
	}
}

func TestAuditService_LogLogout(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogLogout(context.Background(), 1, "testuser", "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionAuthLogout {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionAuthLogout, log.Action)
	}
}

func TestAuditService_LogRegister(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogRegister(context.Background(), 1, "newuser", "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionAuthRegister {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionAuthRegister, log.Action)
	}
}

func TestAuditService_LogPasswordChange(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogPasswordChange(context.Background(), 1, "testuser", "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionAuthPasswordChange {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionAuthPasswordChange, log.Action)
	}
}

func TestAuditService_LogSystemStartup(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	err := svc.LogSystemStartup(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionSystemStartup {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionSystemStartup, log.Action)
	}
	if log.UserID != nil {
		t.Error("Expected UserID to be nil for system startup")
	}
}

func TestAuditService_LogCaddyReload(t *testing.T) {
	repo := newMockAuditLogRepo()
	settingsSvc := newMockSettingsService()
	svc := NewAuditService(repo, settingsSvc, nil)

	// Test success
	err := svc.LogCaddyReload(context.Background(), true, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log := repo.logs[0]
	if log.Action != models.AuditActionCaddyReload {
		t.Errorf("Expected action '%s', got '%s'", models.AuditActionCaddyReload, log.Action)
	}
	if log.Status != models.AuditStatusSuccess {
		t.Errorf("Expected status '%s', got '%s'", models.AuditStatusSuccess, log.Status)
	}

	// Test failure
	err = svc.LogCaddyReload(context.Background(), false, "config error")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	log = repo.logs[1]
	if log.Status != models.AuditStatusFailure {
		t.Errorf("Expected status '%s', got '%s'", models.AuditStatusFailure, log.Status)
	}
}

func TestStringPtr(t *testing.T) {
	// Test empty string
	result := stringPtr("")
	if result != nil {
		t.Error("Expected nil for empty string")
	}

	// Test non-empty string
	result = stringPtr("test")
	if result == nil {
		t.Fatal("Expected non-nil for non-empty string")
	}
	if *result != "test" {
		t.Errorf("Expected 'test', got '%s'", *result)
	}
}
