package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// TrafficSampleRepository handles database operations for traffic samples.
type TrafficSampleRepository struct {
	db *gorm.DB
}

// NewTrafficSampleRepository creates a new TrafficSampleRepository.
func NewTrafficSampleRepository(db *gorm.DB) *TrafficSampleRepository {
	return &TrafficSampleRepository{db: db}
}

// Create inserts a new TrafficSample row.
func (r *TrafficSampleRepository) Create(sample *models.TrafficSample) error {
	return r.db.Create(sample).Error
}

// ListSince returns all samples with collected_at >= t, ordered by collected_at ASC.
func (r *TrafficSampleRepository) ListSince(t time.Time) ([]models.TrafficSample, error) {
	var samples []models.TrafficSample
	if err := r.db.
		Where("collected_at >= ?", t).
		Order("collected_at ASC").
		Find(&samples).Error; err != nil {
		return nil, err
	}
	return samples, nil
}

// DeleteOlderThan deletes all samples with collected_at < t and returns the
// number of rows deleted.
func (r *TrafficSampleRepository) DeleteOlderThan(t time.Time) (int64, error) {
	result := r.db.Where("collected_at < ?", t).Delete(&models.TrafficSample{})
	return result.RowsAffected, result.Error
}
