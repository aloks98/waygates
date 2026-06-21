package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

func TestExportProxies_ByIDs_MapsFieldsAndDropsServerManaged(t *testing.T) {
	desc := "a description"
	repo := &MockProxyRepository{
		GetByIDsFunc: func(ids []int) ([]models.Proxy, error) {
			return []models.Proxy{{
				ID:                    ids[0],
				Type:                  models.ProxyTypeReverseProxy,
				Name:                  "proxy-1",
				Hostname:              "one.example.com",
				Description:           &desc,
				SSLEnabled:            true,
				SSLForced:             true,
				IsActive:              false, // inactive should be preserved
				CreatedBy:             99,
				CreatedAt:             time.Now(),
				UpdatedAt:             time.Now(),
				Upstreams:             []interface{}{"localhost:8080"},
				BlockExploits:         true,
				TLSInsecureSkipVerify: true,
			}}, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: &MockProxySyncer{}})

	exports, err := svc.ExportProxies([]int{1}, ListProxiesRequest{})
	require.NoError(t, err)
	require.Len(t, exports, 1)

	e := exports[0]
	assert.Equal(t, models.ProxyTypeReverseProxy, e.Type)
	assert.Equal(t, "proxy-1", e.Name)
	assert.Equal(t, "one.example.com", e.Hostname)
	require.NotNil(t, e.Description)
	assert.Equal(t, "a description", *e.Description)
	assert.True(t, e.SSLEnabled)
	assert.False(t, e.IsActive, "is_active must be carried through for import round-trip")
	assert.True(t, e.BlockExploits)
	assert.True(t, e.TLSInsecureSkipVerify)
	assert.Equal(t, []interface{}{"localhost:8080"}, e.Upstreams)
}

func TestExportProxies_ByIDs_SkipsMissing(t *testing.T) {
	repo := &MockProxyRepository{
		// GetByIDs is a single WHERE id IN (...) query: missing rows are simply
		// absent from the result, so id 2 never comes back.
		GetByIDsFunc: func(ids []int) ([]models.Proxy, error) {
			out := make([]models.Proxy, 0, len(ids))
			for _, id := range ids {
				if id == 2 {
					continue // missing
				}
				out = append(out, models.Proxy{ID: id, Name: "p", Hostname: "h.example.com"})
			}
			return out, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: &MockProxySyncer{}})

	exports, err := svc.ExportProxies([]int{1, 2, 3}, ListProxiesRequest{})
	require.NoError(t, err)
	assert.Len(t, exports, 2, "missing id 2 should be skipped")
}

func TestExportProxies_AllByFilter(t *testing.T) {
	var capturedParams repository.ProxyListParams
	repo := &MockProxyRepository{
		ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			capturedParams = params
			return []models.Proxy{
				{ID: 1, Name: "a", Hostname: "a.example.com", Type: models.ProxyTypeRedirect},
				{ID: 2, Name: "b", Hostname: "b.example.com", Type: models.ProxyTypeRedirect},
			}, 2, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: &MockProxySyncer{}})

	exports, err := svc.ExportProxies(nil, ListProxiesRequest{Types: []string{models.ProxyTypeRedirect}})
	require.NoError(t, err)
	require.Len(t, exports, 2)
	assert.Equal(t, []string{models.ProxyTypeRedirect}, capturedParams.Types)
	assert.Equal(t, "a", exports[0].Name)
	assert.Equal(t, "b", exports[1].Name)
}

func TestExportProxies_OmitsEmptyCustomHeaders(t *testing.T) {
	repo := &MockProxyRepository{
		GetByIDsFunc: func(ids []int) ([]models.Proxy, error) {
			return []models.Proxy{{ID: ids[0], Name: "p", Hostname: "h.example.com"}}, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: &MockProxySyncer{}})

	exports, err := svc.ExportProxies([]int{1}, ListProxiesRequest{})
	require.NoError(t, err)
	require.Len(t, exports, 1)
	assert.Nil(t, exports[0].CustomHeaders, "empty custom headers should be omitted")
}
