package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// newL4BulkTestService builds an L4ProxyService whose GetByID returns an
// inactive proxy for any id NOT in failingIDs, and gorm.ErrRecordNotFound for
// ids in failingIDs. This drives mixed success/failure through SetActive/Delete.
func newL4BulkTestService(failingIDs map[int]bool) *L4ProxyService {
	repo := &MockL4ProxyRepository{
		GetByIDFunc: func(id int) (*models.L4Proxy, error) {
			if failingIDs[id] {
				return nil, gorm.ErrRecordNotFound
			}
			return &models.L4Proxy{
				ID:         id,
				Name:       fmt.Sprintf("proxy-%d", id),
				ListenPort: 8080 + id,
				Protocol:   models.L4ProtocolTCP,
				IsActive:   false,
			}, nil
		},
	}
	return NewL4ProxyService(repo, nil)
}

// =============================================================================
// SetActive Tests
// =============================================================================

func TestL4SetActive_SetsToTrue(t *testing.T) {
	t.Parallel()
	var updated *models.L4Proxy
	repo := &MockL4ProxyRepository{
		GetByIDFunc: func(id int) (*models.L4Proxy, error) {
			return &models.L4Proxy{ID: id, Name: "proxy", ListenPort: 8080, Protocol: models.L4ProtocolTCP, IsActive: false}, nil
		},
		UpdateFunc: func(proxy *models.L4Proxy) error {
			updated = proxy
			return nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	proxy, err := svc.SetActive(1, true)

	require.NoError(t, err)
	require.NotNil(t, proxy)
	assert.True(t, proxy.IsActive)
	require.NotNil(t, updated)
	assert.True(t, updated.IsActive, "repo.Update must be called with IsActive=true")
}

func TestL4SetActive_SetsToFalse(t *testing.T) {
	t.Parallel()
	var updated *models.L4Proxy
	repo := &MockL4ProxyRepository{
		GetByIDFunc: func(id int) (*models.L4Proxy, error) {
			return &models.L4Proxy{ID: id, Name: "proxy", ListenPort: 8080, Protocol: models.L4ProtocolTCP, IsActive: true}, nil
		},
		UpdateFunc: func(proxy *models.L4Proxy) error {
			updated = proxy
			return nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	proxy, err := svc.SetActive(1, false)

	require.NoError(t, err)
	require.NotNil(t, proxy)
	assert.False(t, proxy.IsActive)
	require.NotNil(t, updated)
	assert.False(t, updated.IsActive, "repo.Update must be called with IsActive=false")
}

func TestL4SetActive_NotFound(t *testing.T) {
	t.Parallel()
	repo := &MockL4ProxyRepository{
		GetByIDFunc: func(_ int) (*models.L4Proxy, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	svc := NewL4ProxyService(repo, nil)

	proxy, err := svc.SetActive(999, true)

	assert.Nil(t, proxy)
	assert.ErrorIs(t, err, ErrL4ProxyNotFound)
}

// =============================================================================
// BulkSetActive Tests
// =============================================================================

func TestL4BulkSetActive_Enable_MixedResults(t *testing.T) {
	t.Parallel()
	// id 2 fails (not found); ids 1 and 3 succeed.
	svc := newL4BulkTestService(map[int]bool{2: true})

	result := svc.BulkSetActive([]int{1, 2, 3}, true)

	require.Equal(t, 3, result.Requested)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Results, 3)

	assert.Equal(t, BulkItemResult{ID: 1, Status: bulkStatusOK}, result.Results[0])
	assert.Equal(t, 2, result.Results[1].ID)
	assert.Equal(t, bulkStatusError, result.Results[1].Status)
	assert.Contains(t, result.Results[1].Error, ErrL4ProxyNotFound.Error())
	assert.Equal(t, BulkItemResult{ID: 3, Status: bulkStatusOK}, result.Results[2])
}

func TestL4BulkSetActive_Disable_MixedResults(t *testing.T) {
	t.Parallel()
	// id 5 fails; ids 4 and 6 succeed.
	svc := newL4BulkTestService(map[int]bool{5: true})

	result := svc.BulkSetActive([]int{4, 5, 6}, false)

	require.Equal(t, 3, result.Requested)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Results, 3)
	assert.Equal(t, bulkStatusOK, result.Results[0].Status)
	assert.Equal(t, 5, result.Results[1].ID)
	assert.Equal(t, bulkStatusError, result.Results[1].Status)
	assert.Equal(t, bulkStatusOK, result.Results[2].Status)
}

func TestL4BulkSetActive_Empty(t *testing.T) {
	t.Parallel()
	svc := newL4BulkTestService(nil)

	result := svc.BulkSetActive([]int{}, true)

	assert.Equal(t, 0, result.Requested)
	assert.Equal(t, 0, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Results)
}

// =============================================================================
// BulkDelete Tests
// =============================================================================

func TestL4BulkDelete_MixedResults(t *testing.T) {
	t.Parallel()
	svc := newL4BulkTestService(map[int]bool{5: true})

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

func TestL4BulkDelete_AllSucceed(t *testing.T) {
	t.Parallel()
	svc := newL4BulkTestService(nil)

	result := svc.BulkDelete([]int{1, 2, 3})

	assert.Equal(t, 3, result.Requested)
	assert.Equal(t, 3, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	for _, item := range result.Results {
		assert.Equal(t, bulkStatusOK, item.Status)
	}
}
