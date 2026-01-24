package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// TestableSettingsService wraps SettingsService for testing with interface-based repo
type TestableSettingsService struct {
	repo        repository.SettingsRepositoryInterface
	syncService *SyncService
	logger      *zap.Logger
}

// NewTestableSettingsService creates a settings service for testing
func NewTestableSettingsService(repo repository.SettingsRepositoryInterface, logger *zap.Logger) *TestableSettingsService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TestableSettingsService{
		repo:   repo,
		logger: logger.Named("settings-service"),
	}
}

// SetSyncService sets the sync service
func (s *TestableSettingsService) SetSyncService(syncService *SyncService) {
	s.syncService = syncService
}

// Get retrieves a setting by key
func (s *TestableSettingsService) Get(key string) (string, error) {
	setting, err := s.repo.Get(key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// GetWithDefault retrieves a setting by key, returning default if not found
func (s *TestableSettingsService) GetWithDefault(key, defaultValue string) string {
	return s.repo.GetValue(key, defaultValue)
}

// Set creates or updates a setting
func (s *TestableSettingsService) Set(key, value string) error {
	s.logger.Info("Updating setting", zap.String("key", key))
	return s.repo.Set(key, value)
}

// GetAll retrieves all settings as a map
func (s *TestableSettingsService) GetAll() (map[string]string, error) {
	return s.repo.GetAll()
}

// Delete deletes a setting by key
func (s *TestableSettingsService) Delete(key string) error {
	return s.repo.Delete(key)
}

// GetNotFoundSettings retrieves the 404 page configuration
func (s *TestableSettingsService) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	return s.repo.GetNotFoundSettings()
}

// SetNotFoundSettings updates the 404 page configuration
func (s *TestableSettingsService) SetNotFoundSettings(settings *models.NotFoundSettings) error {
	s.logger.Info("Updating 404 settings",
		zap.String("mode", settings.Mode),
		zap.String("redirect_url", settings.RedirectURL))

	// Update in database
	if err := s.repo.SetNotFoundSettings(settings); err != nil {
		return err
	}

	// Update catchall.conf via sync service
	if s.syncService != nil {
		if err := s.syncService.UpdateCatchAll(); err != nil {
			s.logger.Error("Failed to update catchall.conf", zap.Error(err))
			return err
		}
	}

	return nil
}

// mockSettingsRepoInterface implements SettingsRepositoryInterface for testing
type mockSettingsRepoInterface struct {
	settings         map[string]*models.Setting
	notFoundSettings *models.NotFoundSettings
	getErr           error
	setErr           error
	deleteErr        error
	getAllErr        error
	notFoundGetErr   error
	notFoundSetErr   error
}

func newMockSettingsRepoInterface() *mockSettingsRepoInterface {
	return &mockSettingsRepoInterface{
		settings: make(map[string]*models.Setting),
		notFoundSettings: &models.NotFoundSettings{
			Mode:        "default",
			RedirectURL: "",
		},
	}
}

func (m *mockSettingsRepoInterface) Get(key string) (*models.Setting, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, errors.New("setting not found")
}

func (m *mockSettingsRepoInterface) GetValue(key, defaultValue string) string {
	if s, ok := m.settings[key]; ok {
		return s.Value
	}
	return defaultValue
}

func (m *mockSettingsRepoInterface) Set(key, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.settings[key] = &models.Setting{Key: key, Value: value}
	return nil
}

func (m *mockSettingsRepoInterface) GetAll() (map[string]string, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	result := make(map[string]string)
	for k, v := range m.settings {
		result[k] = v.Value
	}
	return result, nil
}

func (m *mockSettingsRepoInterface) Delete(key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.settings, key)
	return nil
}

func (m *mockSettingsRepoInterface) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	if m.notFoundGetErr != nil {
		return nil, m.notFoundGetErr
	}
	return m.notFoundSettings, nil
}

func (m *mockSettingsRepoInterface) SetNotFoundSettings(settings *models.NotFoundSettings) error {
	if m.notFoundSetErr != nil {
		return m.notFoundSetErr
	}
	m.notFoundSettings = settings
	return nil
}

// Verify interface implementation
var _ repository.SettingsRepositoryInterface = (*mockSettingsRepoInterface)(nil)

func TestNewSettingsService(t *testing.T) {
	t.Parallel()

	t.Run("with nil logger creates service with nop logger", func(t *testing.T) {
		t.Parallel()
		svc := NewSettingsService(nil, nil)
		require.NotNil(t, svc, "Expected service to be created")
		assert.NotNil(t, svc.logger, "Expected logger to be set (nop logger)")
	})

	t.Run("with actual logger uses provided logger", func(t *testing.T) {
		t.Parallel()
		logger := zap.NewNop()
		svc := NewSettingsService(nil, logger)
		require.NotNil(t, svc)
		assert.NotNil(t, svc.logger, "Expected logger to be set")
	})
}

func TestSettingsService_SetSyncService(t *testing.T) {
	t.Parallel()

	t.Run("initially nil", func(t *testing.T) {
		t.Parallel()
		svc := NewSettingsService(nil, nil)
		assert.Nil(t, svc.syncService, "Expected syncService to be nil initially")
	})

	t.Run("sets sync service", func(t *testing.T) {
		t.Parallel()
		svc := NewSettingsService(nil, nil)
		svc.SetSyncService(nil)
		assert.Nil(t, svc.syncService, "Expected syncService to still be nil when passed nil")
	})
}

func TestSettingsService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		setupRepo   func(*mockSettingsRepoInterface)
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "success - existing key",
			key:  "app_name",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["app_name"] = &models.Setting{Key: "app_name", Value: "MyApp"}
			},
			expected:    "MyApp",
			expectError: false,
		},
		{
			name:        "error - key not found",
			key:         "nonexistent",
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expectError: true,
			errorMsg:    "setting not found",
		},
		{
			name: "error - repository error",
			key:  "any_key",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.getErr = errors.New("database connection failed")
			},
			expectError: true,
			errorMsg:    "database connection failed",
		},
		{
			name: "success - empty value",
			key:  "empty_setting",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["empty_setting"] = &models.Setting{Key: "empty_setting", Value: ""}
			},
			expected:    "",
			expectError: false,
		},
		{
			name: "success - value with special characters",
			key:  "special",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["special"] = &models.Setting{Key: "special", Value: "value with spaces & special <chars>"}
			},
			expected:    "value with spaces & special <chars>",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockSettingsRepoInterface()
			tt.setupRepo(mock)
			svc := NewTestableSettingsService(mock, nil)

			result, err := svc.Get(tt.key)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSettingsService_GetWithDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		key          string
		defaultValue string
		setupRepo    func(*mockSettingsRepoInterface)
		expected     string
	}{
		{
			name:         "returns existing value",
			key:          "existing_key",
			defaultValue: "default",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["existing_key"] = &models.Setting{Key: "existing_key", Value: "existing_value"}
			},
			expected: "existing_value",
		},
		{
			name:         "returns default for non-existing key",
			key:          "missing_key",
			defaultValue: "default_value",
			setupRepo:    func(_ *mockSettingsRepoInterface) {},
			expected:     "default_value",
		},
		{
			name:         "returns empty default when specified",
			key:          "missing",
			defaultValue: "",
			setupRepo:    func(_ *mockSettingsRepoInterface) {},
			expected:     "",
		},
		{
			name:         "returns stored empty value over non-empty default",
			key:          "empty_key",
			defaultValue: "should_not_return",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["empty_key"] = &models.Setting{Key: "empty_key", Value: ""}
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockSettingsRepoInterface()
			tt.setupRepo(mock)
			svc := NewTestableSettingsService(mock, nil)

			result := svc.GetWithDefault(tt.key, tt.defaultValue)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettingsService_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		value       string
		setupRepo   func(*mockSettingsRepoInterface)
		expectError bool
		errorMsg    string
	}{
		{
			name:        "success - new setting",
			key:         "new_key",
			value:       "new_value",
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expectError: false,
		},
		{
			name:  "success - update existing setting",
			key:   "existing_key",
			value: "updated_value",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["existing_key"] = &models.Setting{Key: "existing_key", Value: "old_value"}
			},
			expectError: false,
		},
		{
			name:        "success - empty value",
			key:         "empty_key",
			value:       "",
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expectError: false,
		},
		{
			name:  "error - repository error",
			key:   "any_key",
			value: "any_value",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.setErr = errors.New("database write failed")
			},
			expectError: true,
			errorMsg:    "database write failed",
		},
		{
			name:        "success - value with unicode",
			key:         "unicode_key",
			value:       "Hello, 世界! 🌍",
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockSettingsRepoInterface()
			tt.setupRepo(mock)
			svc := NewTestableSettingsService(mock, nil)

			err := svc.Set(tt.key, tt.value)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				// Verify the value was actually set
				assert.Equal(t, tt.value, mock.settings[tt.key].Value)
			}
		})
	}
}

func TestSettingsService_GetAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupRepo   func(*mockSettingsRepoInterface)
		expected    map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "success - multiple settings",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["key1"] = &models.Setting{Key: "key1", Value: "value1"}
				m.settings["key2"] = &models.Setting{Key: "key2", Value: "value2"}
				m.settings["key3"] = &models.Setting{Key: "key3", Value: "value3"}
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			expectError: false,
		},
		{
			name:        "success - empty settings",
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name: "success - single setting",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["only_key"] = &models.Setting{Key: "only_key", Value: "only_value"}
			},
			expected: map[string]string{
				"only_key": "only_value",
			},
			expectError: false,
		},
		{
			name: "error - repository error",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.getAllErr = errors.New("database query failed")
			},
			expectError: true,
			errorMsg:    "database query failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockSettingsRepoInterface()
			tt.setupRepo(mock)
			svc := NewTestableSettingsService(mock, nil)

			result, err := svc.GetAll()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSettingsService_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		setupRepo   func(*mockSettingsRepoInterface)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success - delete existing",
			key:  "to_delete",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.settings["to_delete"] = &models.Setting{Key: "to_delete", Value: "value"}
			},
			expectError: false,
		},
		{
			name:        "success - delete non-existing (no-op)",
			key:         "nonexistent",
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expectError: false,
		},
		{
			name: "error - repository error",
			key:  "any_key",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.deleteErr = errors.New("database delete failed")
			},
			expectError: true,
			errorMsg:    "database delete failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockSettingsRepoInterface()
			tt.setupRepo(mock)
			svc := NewTestableSettingsService(mock, nil)

			err := svc.Delete(tt.key)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				// Verify the key is removed
				_, exists := mock.settings[tt.key]
				assert.False(t, exists, "Expected setting to be deleted")
			}
		})
	}
}

func TestSettingsService_GetNotFoundSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupRepo   func(*mockSettingsRepoInterface)
		expected    *models.NotFoundSettings
		expectError bool
		errorMsg    string
	}{
		{
			name: "success - default mode",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.notFoundSettings = &models.NotFoundSettings{
					Mode:        "default",
					RedirectURL: "",
				}
			},
			expected: &models.NotFoundSettings{
				Mode:        "default",
				RedirectURL: "",
			},
			expectError: false,
		},
		{
			name: "success - redirect mode",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.notFoundSettings = &models.NotFoundSettings{
					Mode:        "redirect",
					RedirectURL: "https://example.com/404",
				}
			},
			expected: &models.NotFoundSettings{
				Mode:        "redirect",
				RedirectURL: "https://example.com/404",
			},
			expectError: false,
		},
		{
			name: "error - repository error",
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.notFoundGetErr = errors.New("database error")
			},
			expectError: true,
			errorMsg:    "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockSettingsRepoInterface()
			tt.setupRepo(mock)
			svc := NewTestableSettingsService(mock, nil)

			result, err := svc.GetNotFoundSettings()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.Mode, result.Mode)
				assert.Equal(t, tt.expected.RedirectURL, result.RedirectURL)
			}
		})
	}
}

func TestSettingsService_SetNotFoundSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		settings    *models.NotFoundSettings
		setupRepo   func(*mockSettingsRepoInterface)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success - set redirect mode",
			settings: &models.NotFoundSettings{
				Mode:        "redirect",
				RedirectURL: "https://new.example.com/404",
			},
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expectError: false,
		},
		{
			name: "success - set default mode",
			settings: &models.NotFoundSettings{
				Mode:        "default",
				RedirectURL: "",
			},
			setupRepo:   func(_ *mockSettingsRepoInterface) {},
			expectError: false,
		},
		{
			name: "error - repository set error",
			settings: &models.NotFoundSettings{
				Mode:        "redirect",
				RedirectURL: "https://example.com",
			},
			setupRepo: func(m *mockSettingsRepoInterface) {
				m.notFoundSetErr = errors.New("database write failed")
			},
			expectError: true,
			errorMsg:    "database write failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := newMockSettingsRepoInterface()
			tt.setupRepo(mock)
			svc := NewTestableSettingsService(mock, nil)

			err := svc.SetNotFoundSettings(tt.settings)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				// Verify settings were updated
				assert.Equal(t, tt.settings.Mode, mock.notFoundSettings.Mode)
				assert.Equal(t, tt.settings.RedirectURL, mock.notFoundSettings.RedirectURL)
			}
		})
	}
}

func TestSettingsService_SetNotFoundSettings_WithSyncService(t *testing.T) {
	t.Parallel()

	t.Run("success - calls sync service UpdateCatchAll", func(t *testing.T) {
		t.Parallel()
		mock := newMockSettingsRepoInterface()
		proxyRepo := &MockProxyRepository{
			ListFunc: func(_ repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{}, 0, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "redirect", RedirectURL: "https://example.com"}, nil
			},
		}
		aclRepo := &SyncMockACLRepository{
			ListGroupsFunc: func(_ repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
				return []models.ACLGroup{}, 0, nil
			},
		}
		fileManager := &MockFileManager{
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/config.json"
			},
			BackupJSONConfigFunc: func(_ string) error {
				return nil
			},
			WriteJSONConfigFunc: func(_ string, _ []byte) error {
				return nil
			},
			FileExistsFunc: func(_ string) bool {
				return true
			},
		}
		reloader := &MockReloader{
			ValidateJSONFunc: func(_ string) error {
				return nil
			},
			ReloadJSONFunc: func(_ context.Context, _ string) (*caddy.ReloadResult, error) {
				return &caddy.ReloadResult{Success: true}, nil
			},
		}

		syncSvc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			ACLRepo:      aclRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
		})

		svc := NewTestableSettingsService(mock, nil)
		svc.SetSyncService(syncSvc)

		settings := &models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://example.com",
		}

		err := svc.SetNotFoundSettings(settings)
		require.NoError(t, err)
	})

	t.Run("error - sync service UpdateCatchAll fails on JSON write", func(t *testing.T) {
		t.Parallel()
		mock := newMockSettingsRepoInterface()
		proxyRepo := &MockProxyRepository{
			ListFunc: func(_ repository.ProxyListParams) ([]models.Proxy, int64, error) {
				return []models.Proxy{}, 0, nil
			},
		}
		settingsRepo := &MockSettingsRepository{
			GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
				return &models.NotFoundSettings{Mode: "default"}, nil
			},
		}
		aclRepo := &SyncMockACLRepository{
			ListGroupsFunc: func(_ repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
				return []models.ACLGroup{}, 0, nil
			},
		}
		fileManager := &MockFileManager{
			GetJSONConfigPathFunc: func() string {
				return "/etc/caddy/config.json"
			},
			FileExistsFunc: func(_ string) bool {
				return true
			},
			BackupJSONConfigFunc: func(_ string) error {
				return nil
			},
			WriteJSONConfigFunc: func(_ string, _ []byte) error {
				return errors.New("disk full")
			},
		}
		reloader := &MockReloader{
			ValidateJSONFunc: func(_ string) error {
				return nil
			},
		}

		syncSvc := NewSyncService(SyncServiceConfig{
			ProxyRepo:    proxyRepo,
			SettingsRepo: settingsRepo,
			ACLRepo:      aclRepo,
			FileManager:  fileManager,
			Reloader:     reloader,
		})

		svc := NewTestableSettingsService(mock, nil)
		svc.SetSyncService(syncSvc)

		settings := &models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://example.com",
		}

		err := svc.SetNotFoundSettings(settings)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "disk full")
	})
}

func TestSettingsService_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("Get with empty key", func(t *testing.T) {
		t.Parallel()
		mock := newMockSettingsRepoInterface()
		svc := NewTestableSettingsService(mock, nil)

		_, err := svc.Get("")
		require.Error(t, err)
	})

	t.Run("Set with empty key", func(t *testing.T) {
		t.Parallel()
		mock := newMockSettingsRepoInterface()
		svc := NewTestableSettingsService(mock, nil)

		err := svc.Set("", "value")
		require.NoError(t, err) // Repository allows empty keys
	})

	t.Run("GetWithDefault with empty key and default", func(t *testing.T) {
		t.Parallel()
		mock := newMockSettingsRepoInterface()
		svc := NewTestableSettingsService(mock, nil)

		result := svc.GetWithDefault("", "")
		assert.Equal(t, "", result)
	})

	t.Run("Delete with empty key", func(t *testing.T) {
		t.Parallel()
		mock := newMockSettingsRepoInterface()
		svc := NewTestableSettingsService(mock, nil)

		err := svc.Delete("")
		require.NoError(t, err) // Repository allows deleting empty keys
	})

	t.Run("Set very long value", func(t *testing.T) {
		t.Parallel()
		mock := newMockSettingsRepoInterface()
		svc := NewTestableSettingsService(mock, nil)

		longValue := string(make([]byte, 10000))
		err := svc.Set("long_key", longValue)
		require.NoError(t, err)
		assert.Equal(t, longValue, mock.settings["long_key"].Value)
	})
}
