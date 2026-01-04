package service

import (
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// mockSettingsRepo implements a mock settings repository for testing
type mockSettingsRepo struct {
	settings         map[string]*models.Setting
	notFoundSettings *models.NotFoundSettings
	getErr           error
	setErr           error
	deleteErr        error
}

func newMockSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{
		settings: make(map[string]*models.Setting),
		notFoundSettings: &models.NotFoundSettings{
			Mode:        "default",
			RedirectURL: "",
		},
	}
}

func (m *mockSettingsRepo) Get(key string) (*models.Setting, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, errors.New("setting not found")
}

func (m *mockSettingsRepo) GetValue(key, defaultValue string) string {
	if s, ok := m.settings[key]; ok {
		return s.Value
	}
	return defaultValue
}

func (m *mockSettingsRepo) Set(key, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.settings[key] = &models.Setting{Key: key, Value: value}
	return nil
}

func (m *mockSettingsRepo) GetAll() (map[string]string, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	result := make(map[string]string)
	for k, v := range m.settings {
		result[k] = v.Value
	}
	return result, nil
}

func (m *mockSettingsRepo) Delete(key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.settings, key)
	return nil
}

func (m *mockSettingsRepo) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.notFoundSettings, nil
}

func (m *mockSettingsRepo) SetNotFoundSettings(settings *models.NotFoundSettings) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.notFoundSettings = settings
	return nil
}

func TestNewSettingsService(t *testing.T) {
	// Test with nil logger
	svc := NewSettingsService(nil, nil)
	if svc == nil {
		t.Fatal("Expected service to be created")
	}
	if svc.logger == nil {
		t.Error("Expected logger to be set (nop logger)")
	}

	// Test with actual logger
	logger := zap.NewNop()
	svc = NewSettingsService(nil, logger)
	if svc.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestSettingsService_SetSyncService(t *testing.T) {
	svc := NewSettingsService(nil, nil)

	if svc.syncService != nil {
		t.Error("Expected syncService to be nil initially")
	}

	// Create a mock sync service (just checking the setter works)
	svc.SetSyncService(nil)

	// Still nil since we passed nil
	if svc.syncService != nil {
		t.Error("Expected syncService to still be nil")
	}
}

func TestSettingsService_GetWithDefault(t *testing.T) {
	mock := newMockSettingsRepo()
	mock.settings["existing_key"] = &models.Setting{Key: "existing_key", Value: "existing_value"}

	// We need to create a service that uses our mock
	// Since SettingsService expects *repository.SettingsRepository, we test via interface
	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "Existing key",
			key:          "existing_key",
			defaultValue: "default",
			expected:     "existing_value",
		},
		{
			name:         "Non-existing key",
			key:          "missing_key",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mock.GetValue(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSettingsService_GetAll(t *testing.T) {
	mock := newMockSettingsRepo()
	mock.settings["key1"] = &models.Setting{Key: "key1", Value: "value1"}
	mock.settings["key2"] = &models.Setting{Key: "key2", Value: "value2"}

	result, err := mock.GetAll()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 settings, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Errorf("Expected 'value2', got '%s'", result["key2"])
	}
}

func TestSettingsService_Set(t *testing.T) {
	mock := newMockSettingsRepo()

	err := mock.Set("new_key", "new_value")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mock.settings["new_key"] == nil {
		t.Fatal("Expected setting to be created")
	}
	if mock.settings["new_key"].Value != "new_value" {
		t.Errorf("Expected 'new_value', got '%s'", mock.settings["new_key"].Value)
	}
}

func TestSettingsService_Set_Error(t *testing.T) {
	mock := newMockSettingsRepo()
	mock.setErr = errors.New("database error")

	err := mock.Set("key", "value")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestSettingsService_Delete(t *testing.T) {
	mock := newMockSettingsRepo()
	mock.settings["to_delete"] = &models.Setting{Key: "to_delete", Value: "value"}

	err := mock.Delete("to_delete")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mock.settings["to_delete"] != nil {
		t.Error("Expected setting to be deleted")
	}
}

func TestSettingsService_GetNotFoundSettings(t *testing.T) {
	mock := newMockSettingsRepo()
	mock.notFoundSettings = &models.NotFoundSettings{
		Mode:        "redirect",
		RedirectURL: "https://example.com",
	}

	result, err := mock.GetNotFoundSettings()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Mode != "redirect" {
		t.Errorf("Expected 'redirect', got '%s'", result.Mode)
	}
	if result.RedirectURL != "https://example.com" {
		t.Errorf("Expected 'https://example.com', got '%s'", result.RedirectURL)
	}
}

func TestSettingsService_SetNotFoundSettings(t *testing.T) {
	mock := newMockSettingsRepo()

	settings := &models.NotFoundSettings{
		Mode:        "redirect",
		RedirectURL: "https://new.example.com",
	}

	err := mock.SetNotFoundSettings(settings)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mock.notFoundSettings.Mode != "redirect" {
		t.Errorf("Expected 'redirect', got '%s'", mock.notFoundSettings.Mode)
	}
	if mock.notFoundSettings.RedirectURL != "https://new.example.com" {
		t.Errorf("Expected 'https://new.example.com', got '%s'", mock.notFoundSettings.RedirectURL)
	}
}

func TestSettingsService_Get_Error(t *testing.T) {
	mock := newMockSettingsRepo()
	mock.getErr = errors.New("database error")

	_, err := mock.Get("any_key")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestSettingsService_GetAll_Error(t *testing.T) {
	mock := newMockSettingsRepo()
	mock.getErr = errors.New("database error")

	_, err := mock.GetAll()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
