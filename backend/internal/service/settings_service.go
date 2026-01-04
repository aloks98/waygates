package service

import (
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// SettingsService handles business logic for settings
type SettingsService struct {
	repo        *repository.SettingsRepository
	syncService *SyncService
	logger      *zap.Logger
}

// NewSettingsService creates a new settings service
func NewSettingsService(repo *repository.SettingsRepository, logger *zap.Logger) *SettingsService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SettingsService{
		repo:   repo,
		logger: logger.Named("settings-service"),
	}
}

// SetSyncService sets the sync service for updating catchall.conf
// This is called after construction to avoid circular dependencies
func (s *SettingsService) SetSyncService(syncService *SyncService) {
	s.syncService = syncService
}

// Get retrieves a setting by key
func (s *SettingsService) Get(key string) (string, error) {
	setting, err := s.repo.Get(key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// GetWithDefault retrieves a setting by key, returning default if not found
func (s *SettingsService) GetWithDefault(key, defaultValue string) string {
	return s.repo.GetValue(key, defaultValue)
}

// Set creates or updates a setting
func (s *SettingsService) Set(key, value string) error {
	s.logger.Info("Updating setting", zap.String("key", key))
	return s.repo.Set(key, value)
}

// GetAll retrieves all settings as a map
func (s *SettingsService) GetAll() (map[string]string, error) {
	return s.repo.GetAll()
}

// Delete deletes a setting by key
func (s *SettingsService) Delete(key string) error {
	return s.repo.Delete(key)
}

// GetNotFoundSettings retrieves the 404 page configuration
func (s *SettingsService) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	return s.repo.GetNotFoundSettings()
}

// SetNotFoundSettings updates the 404 page configuration
func (s *SettingsService) SetNotFoundSettings(settings *models.NotFoundSettings) error {
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
