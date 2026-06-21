package service

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// =============================================================================
// ExportL4Proxies — by-ids tests
// =============================================================================

func TestExportL4Proxies_ByIDs_MapsFieldsAndDropsServerManaged(t *testing.T) {
	t.Parallel()
	desc := "a description"
	ppv := "v2"
	weight := 10
	repo := &MockL4ProxyRepository{
		GetByIDsFunc: func(ids []int) ([]models.L4Proxy, error) {
			return []models.L4Proxy{{
				ID:          ids[0],
				Name:        "l4-proxy-1",
				Description: &desc,
				ListenPort:  5432,
				Protocol:    models.L4ProtocolTCP,
				IsActive:    false, // inactive should be preserved
				CreatedBy:   99,
				Routes: []models.L4Route{
					{
						ID:                   10,
						L4ProxyID:            ids[0],
						Priority:             1,
						MatcherType:          models.L4MatcherAny,
						SNIHostnames:         pq.StringArray{"example.com"},
						AllowedIPRanges:      pq.StringArray{"10.0.0.0/8"},
						RegexPattern:         nil,
						Upstreams:            models.L4UpstreamSlice{{Host: "db.internal", Port: 5432, Weight: &weight}},
						LoadBalancingPolicy:  models.L4LoadBalancingRoundRobin,
						TLSTerminate:         true,
						TLSPassthrough:       false,
						ProxyProtocolVersion: &ppv,
					},
				},
			}}, nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	exports, err := svc.ExportL4Proxies([]int{1}, ListL4ProxiesRequest{})
	require.NoError(t, err)
	require.Len(t, exports, 1)

	e := exports[0]
	// Top-level fields
	assert.Equal(t, "l4-proxy-1", e.Name)
	require.NotNil(t, e.Description)
	assert.Equal(t, "a description", *e.Description)
	assert.Equal(t, 5432, e.ListenPort)
	assert.Equal(t, models.L4ProtocolTCP, e.Protocol)
	assert.False(t, e.IsActive, "is_active=false must be preserved for round-trip import")

	// Route fields
	require.Len(t, e.Routes, 1)
	r := e.Routes[0]
	assert.Equal(t, 1, r.Priority)
	assert.Equal(t, models.L4MatcherAny, r.MatcherType)
	assert.Equal(t, []string{"example.com"}, r.SNIHostnames)
	assert.Equal(t, []string{"10.0.0.0/8"}, r.AllowedIPRanges)
	assert.Nil(t, r.RegexPattern)
	assert.Equal(t, models.L4LoadBalancingRoundRobin, r.LoadBalancingPolicy)
	assert.True(t, r.TLSTerminate)
	assert.False(t, r.TLSPassthrough)
	require.NotNil(t, r.ProxyProtocolVersion)
	assert.Equal(t, "v2", *r.ProxyProtocolVersion)

	// Upstream fields
	require.Len(t, r.Upstreams, 1)
	u := r.Upstreams[0]
	assert.Equal(t, "db.internal", u.Host)
	assert.Equal(t, 5432, u.Port)
	require.NotNil(t, u.Weight)
	assert.Equal(t, 10, *u.Weight)
}

func TestExportL4Proxies_ByIDs_SkipsMissing(t *testing.T) {
	t.Parallel()
	repo := &MockL4ProxyRepository{
		// GetByIDs is a single WHERE id IN (...) query: missing rows are simply
		// absent from the result, so id 2 never comes back.
		GetByIDsFunc: func(ids []int) ([]models.L4Proxy, error) {
			out := make([]models.L4Proxy, 0, len(ids))
			for _, id := range ids {
				if id == 2 {
					continue // missing
				}
				out = append(out, models.L4Proxy{
					ID:         id,
					Name:       "proxy",
					ListenPort: 8080,
					Protocol:   models.L4ProtocolTCP,
				})
			}
			return out, nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	exports, err := svc.ExportL4Proxies([]int{1, 2, 3}, ListL4ProxiesRequest{})
	require.NoError(t, err)
	assert.Len(t, exports, 2, "missing id 2 should be skipped")
}

func TestExportL4Proxies_ByIDs_IsActivePreserved(t *testing.T) {
	t.Parallel()
	repo := &MockL4ProxyRepository{
		GetByIDsFunc: func(ids []int) ([]models.L4Proxy, error) {
			return []models.L4Proxy{{
				ID:         ids[0],
				Name:       "proxy",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				IsActive:   true,
			}}, nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	exports, err := svc.ExportL4Proxies([]int{1}, ListL4ProxiesRequest{})
	require.NoError(t, err)
	require.Len(t, exports, 1)
	assert.True(t, exports[0].IsActive)
}

// =============================================================================
// ExportL4Proxies — by-filter tests
// =============================================================================

func TestExportL4Proxies_AllByFilter(t *testing.T) {
	t.Parallel()
	var capturedParams repository.L4ProxyListParams
	repo := &MockL4ProxyRepository{
		ListFunc: func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
			capturedParams = params
			return []models.L4Proxy{
				{ID: 1, Name: "a", ListenPort: 8080, Protocol: models.L4ProtocolTCP},
				{ID: 2, Name: "b", ListenPort: 9090, Protocol: models.L4ProtocolTCP},
			}, 2, nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	exports, err := svc.ExportL4Proxies(nil, ListL4ProxiesRequest{Protocol: "tcp"})
	require.NoError(t, err)
	require.Len(t, exports, 2)
	assert.Equal(t, "tcp", capturedParams.Protocol)
	assert.Equal(t, "a", exports[0].Name)
	assert.Equal(t, "b", exports[1].Name)
}

func TestExportL4Proxies_AllByFilter_Search(t *testing.T) {
	t.Parallel()
	var capturedParams repository.L4ProxyListParams
	repo := &MockL4ProxyRepository{
		ListFunc: func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
			capturedParams = params
			return []models.L4Proxy{
				{ID: 1, Name: "database-proxy", ListenPort: 5432, Protocol: models.L4ProtocolTCP},
			}, 1, nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	exports, err := svc.ExportL4Proxies(nil, ListL4ProxiesRequest{Search: "database"})
	require.NoError(t, err)
	require.Len(t, exports, 1)
	assert.Equal(t, "database", capturedParams.Search)
}

func TestExportL4Proxies_EmptySNIAndIPRangesOmitted(t *testing.T) {
	t.Parallel()
	repo := &MockL4ProxyRepository{
		GetByIDsFunc: func(ids []int) ([]models.L4Proxy, error) {
			return []models.L4Proxy{{
				ID:         ids[0],
				Name:       "proxy",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				Routes: []models.L4Route{
					{
						Priority:            0,
						MatcherType:         models.L4MatcherAny,
						SNIHostnames:        nil,
						AllowedIPRanges:     nil,
						Upstreams:           models.L4UpstreamSlice{{Host: "h", Port: 80}},
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					},
				},
			}}, nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	exports, err := svc.ExportL4Proxies([]int{1}, ListL4ProxiesRequest{})
	require.NoError(t, err)
	require.Len(t, exports, 1)
	r := exports[0].Routes[0]
	assert.Nil(t, r.SNIHostnames, "empty sni_hostnames should be nil/omitted")
	assert.Nil(t, r.AllowedIPRanges, "empty allowed_ip_ranges should be nil/omitted")
}
