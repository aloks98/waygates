package repository

import (
	"strings"

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

// GetMetricsPublishSettings retrieves the metrics publish endpoint configuration.
func (r *SettingsRepository) GetMetricsPublishSettings() (*models.MetricsPublishSettings, error) {
	enabledStr := r.GetValue(models.SettingMetricsPublishEnabled, "false")
	enabled := enabledStr == "true"

	cidrsStr := r.GetValue(models.SettingMetricsAllowedCIDRs, "")
	var cidrs []string
	if cidrsStr != "" {
		cidrs = splitCIDRs(cidrsStr)
	}

	settings := &models.MetricsPublishSettings{
		Enabled:       enabled,
		Host:          r.GetValue(models.SettingMetricsPublishHost, ""),
		Path:          r.GetValue(models.SettingMetricsPublishPath, "/metrics"),
		BasicAuthUser: r.GetValue(models.SettingMetricsBasicAuthUser, ""),
		BasicAuthHash: r.GetValue(models.SettingMetricsBasicAuthHash, ""),
		AllowedCIDRs:  cidrs,
	}
	return settings, nil
}

// SetMetricsPublishSettings updates the metrics publish endpoint configuration.
func (r *SettingsRepository) SetMetricsPublishSettings(settings *models.MetricsPublishSettings) error {
	enabledStr := "false"
	if settings.Enabled {
		enabledStr = "true"
	}
	if err := r.Set(models.SettingMetricsPublishEnabled, enabledStr); err != nil {
		return err
	}
	if err := r.Set(models.SettingMetricsPublishHost, settings.Host); err != nil {
		return err
	}
	if err := r.Set(models.SettingMetricsPublishPath, settings.Path); err != nil {
		return err
	}
	if err := r.Set(models.SettingMetricsBasicAuthUser, settings.BasicAuthUser); err != nil {
		return err
	}
	if err := r.Set(models.SettingMetricsBasicAuthHash, settings.BasicAuthHash); err != nil {
		return err
	}
	return r.Set(models.SettingMetricsAllowedCIDRs, joinCIDRs(settings.AllowedCIDRs))
}

// splitCIDRs splits a comma-separated CIDR string into a slice, trimming blanks
// and skipping empty elements.
func splitCIDRs(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// joinCIDRs joins a CIDR slice into a comma-separated string.
func joinCIDRs(cidrs []string) string {
	return strings.Join(cidrs, ",")
}
