package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// newBulkTestService builds a ProxyService whose GetByID returns an active proxy
// for any id NOT in failingIDs, and gorm.ErrRecordNotFound (→ ErrProxyNotFound)
// for ids in failingIDs. This lets us drive mixed success/failure through the
// real Enable/Disable/Delete service methods.
func newBulkTestService(failingIDs map[int]bool) *ProxyService {
	repo := &MockProxyRepository{
		GetByIDFunc: func(id int) (*models.Proxy, error) {
			if failingIDs[id] {
				return nil, gorm.ErrRecordNotFound
			}
			return &models.Proxy{ID: id, Hostname: fmt.Sprintf("host-%d.example.com", id), IsActive: true}, nil
		},
	}
	return NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: &MockProxySyncer{},
		Logger:      nil,
	})
}

func TestBulkSetActive_Disable_MixedResults(t *testing.T) {
	// id 2 fails (not found); ids 1 and 3 are active and disable cleanly.
	svc := newBulkTestService(map[int]bool{2: true})

	result := svc.BulkSetActive([]int{1, 2, 3}, false)

	require.Equal(t, 3, result.Requested)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Results, 3)

	assert.Equal(t, BulkItemResult{ID: 1, Status: bulkStatusOK}, result.Results[0])
	assert.Equal(t, 2, result.Results[1].ID)
	assert.Equal(t, bulkStatusError, result.Results[1].Status)
	assert.Contains(t, result.Results[1].Error, ErrProxyNotFound.Error())
	assert.Equal(t, BulkItemResult{ID: 3, Status: bulkStatusOK}, result.Results[2])
}

func TestBulkSetActive_Enable_AlreadyEnabledIsError(t *testing.T) {
	// Repo returns an already-active proxy, so EnableProxy returns
	// ErrProxyAlreadyEnabled for every id.
	repo := &MockProxyRepository{
		GetByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: id, Hostname: "h.example.com", IsActive: true}, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: &MockProxySyncer{}})

	result := svc.BulkSetActive([]int{1, 2}, true)

	assert.Equal(t, 2, result.Requested)
	assert.Equal(t, 0, result.Succeeded)
	assert.Equal(t, 2, result.Failed)
	for _, item := range result.Results {
		assert.Equal(t, bulkStatusError, item.Status)
		assert.Contains(t, item.Error, ErrProxyAlreadyEnabled.Error())
	}
}

func TestBulkDelete_MixedResults(t *testing.T) {
	svc := newBulkTestService(map[int]bool{5: true})

	result := svc.BulkDelete([]int{4, 5, 6})

	require.Equal(t, 3, result.Requested)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Results, 3)
	assert.Equal(t, bulkStatusOK, result.Results[0].Status)
	assert.Equal(t, bulkStatusError, result.Results[1].Status)
	assert.Equal(t, 5, result.Results[1].ID)
	assert.Equal(t, bulkStatusOK, result.Results[2].Status)
}

func TestBulkSetActive_Empty(t *testing.T) {
	svc := newBulkTestService(nil)

	result := svc.BulkSetActive([]int{}, true)

	assert.Equal(t, 0, result.Requested)
	assert.Equal(t, 0, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Results)
}
