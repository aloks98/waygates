package service

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// SetActive sets the active status of an L4 proxy to the given value. Unlike
// ToggleActive, it sets unconditionally to the target value rather than flipping.
// The repo.Update path correctly persists false (no GORM default-drop issue).
func (s *L4ProxyService) SetActive(id int, enable bool) (*models.L4Proxy, error) {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrL4ProxyNotFound
		}
		return nil, fmt.Errorf("failed to get l4 proxy: %w", err)
	}

	proxy.IsActive = enable

	if err := s.repo.Update(proxy); err != nil {
		return nil, fmt.Errorf("failed to set l4 proxy active status: %w", err)
	}

	s.logger.Info("l4 proxy active status set",
		zap.Int("id", id),
		zap.String("name", proxy.Name),
		zap.Bool("is_active", proxy.IsActive),
	)

	return proxy, nil
}

// BulkSetActive enables or disables each L4 proxy in ids. It is best-effort: a
// failure on one id never aborts the batch. Each id contributes one entry to
// Results with its individual outcome.
func (s *L4ProxyService) BulkSetActive(ids []int, enable bool) BulkResult {
	return s.runL4Bulk(ids, func(id int) error {
		_, err := s.SetActive(id, enable)
		return err
	})
}

// BulkDelete deletes each L4 proxy in ids. It is best-effort: a failure on one
// id never aborts the batch.
func (s *L4ProxyService) BulkDelete(ids []int) BulkResult {
	return s.runL4Bulk(ids, s.Delete)
}

// runL4Bulk applies op to every id, collecting a per-id result and aggregate
// counts.
func (s *L4ProxyService) runL4Bulk(ids []int, op func(id int) error) BulkResult {
	result := BulkResult{
		Requested: len(ids),
		Results:   make([]BulkItemResult, 0, len(ids)),
	}

	for _, id := range ids {
		if err := op(id); err != nil {
			result.Failed++
			result.Results = append(result.Results, BulkItemResult{
				ID:     id,
				Status: bulkStatusError,
				Error:  err.Error(),
			})
			continue
		}
		result.Succeeded++
		result.Results = append(result.Results, BulkItemResult{
			ID:     id,
			Status: bulkStatusOK,
		})
	}

	return result
}
