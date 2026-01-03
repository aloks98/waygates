package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/aloks98/waygates/backend/internal/models"
)

// SettingsRepository handles database operations for settings
type SettingsRepository struct {
	db *gorm.DB
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting by key
func (r *SettingsRepository) Get(key string) (*models.Setting, error) {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetValue retrieves just the value for a setting key, returns default if not found
func (r *SettingsRepository) GetValue(key, defaultValue string) string {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return defaultValue
	}
	return setting.Value
}

// Set creates or updates a setting
func (r *SettingsRepository) Set(key, value string) error {
	setting := models.Setting{
		Key:   key,
		Value: value,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&setting).Error
}

// GetAll retrieves all settings as a map
func (r *SettingsRepository) GetAll() (map[string]string, error) {
	var settings []models.Setting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

// Delete deletes a setting by key
func (r *SettingsRepository) Delete(key string) error {
	return r.db.Where("key = ?", key).Delete(&models.Setting{}).Error
}

// GetNotFoundSettings retrieves the 404 page configuration
func (r *SettingsRepository) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	settings := &models.NotFoundSettings{
		Mode:        r.GetValue(models.SettingNotFoundMode, "default"),
		RedirectURL: r.GetValue(models.SettingNotFoundRedirectURL, ""),
	}
	return settings, nil
}

// SetNotFoundSettings updates the 404 page configuration
func (r *SettingsRepository) SetNotFoundSettings(settings *models.NotFoundSettings) error {
	if err := r.Set(models.SettingNotFoundMode, settings.Mode); err != nil {
		return err
	}
	return r.Set(models.SettingNotFoundRedirectURL, settings.RedirectURL)
}
