package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// MockL4ProxyRepository is a mock implementation of L4ProxyRepositoryInterface for testing
type MockL4ProxyRepository struct {
	CreateFunc    func(proxy *models.L4Proxy) error
	GetByIDFunc   func(id int) (*models.L4Proxy, error)
	GetByIDsFunc  func(ids []int) ([]models.L4Proxy, error)
	ListFunc      func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error)
	UpdateFunc    func(proxy *models.L4Proxy) error
	DeleteFunc    func(id int) error
	GetByPortFunc func(port int, protocol string) (*models.L4Proxy, error)
}

func (m *MockL4ProxyRepository) Create(proxy *models.L4Proxy) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(proxy)
	}
	return nil
}

func (m *MockL4ProxyRepository) GetByID(id int) (*models.L4Proxy, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockL4ProxyRepository) GetByIDs(ids []int) ([]models.L4Proxy, error) {
	if m.GetByIDsFunc != nil {
		return m.GetByIDsFunc(ids)
	}
	return []models.L4Proxy{}, nil
}

func (m *MockL4ProxyRepository) List(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(params)
	}
	return []models.L4Proxy{}, 0, nil
}

func (m *MockL4ProxyRepository) Update(proxy *models.L4Proxy) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(proxy)
	}
	return nil
}

func (m *MockL4ProxyRepository) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockL4ProxyRepository) GetByPort(port int, protocol string) (*models.L4Proxy, error) {
	if m.GetByPortFunc != nil {
		return m.GetByPortFunc(port, protocol)
	}
	return nil, gorm.ErrRecordNotFound
}

// Helper function to create a valid test L4 proxy request
func newTestL4ProxyRequest() *CreateL4ProxyRequest {
	return &CreateL4ProxyRequest{
		Name:       "Test L4 Proxy",
		ListenPort: 8080,
		Protocol:   models.L4ProtocolTCP,
		IsActive:   true,
		Routes: []CreateL4RouteRequest{
			{
				Priority:            1,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				Upstreams: []L4UpstreamRequest{
					{Host: "127.0.0.1", Port: 9090},
				},
			},
		},
	}
}

// Helper function to create a valid test L4 proxy model
func newTestL4Proxy() *models.L4Proxy {
	return &models.L4Proxy{
		ID:         1,
		Name:       "Test L4 Proxy",
		ListenPort: 8080,
		Protocol:   models.L4ProtocolTCP,
		IsActive:   true,
		CreatedBy:  1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Routes: []models.L4Route{
			{
				ID:                  1,
				L4ProxyID:           1,
				Priority:            1,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				Upstreams: models.L4UpstreamSlice{
					{Host: "127.0.0.1", Port: 9090},
				},
			},
		},
	}
}

// Helper function to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// TestNewL4ProxyService tests service creation
func TestNewL4ProxyService(t *testing.T) {
	repo := &MockL4ProxyRepository{}

	t.Run("creates service with nil logger", func(t *testing.T) {
		svc := NewL4ProxyService(repo, nil)
		require.NotNil(t, svc)
		assert.Equal(t, repo, svc.repo)
		assert.NotNil(t, svc.logger, "Should use nop logger when nil provided")
	})

	t.Run("creates service with mock repo", func(t *testing.T) {
		svc := NewL4ProxyService(repo, nil)
		require.NotNil(t, svc)
		assert.Equal(t, repo, svc.repo)
	})
}

// TestL4ProxyService_Create tests creating an L4 proxy
func TestL4ProxyService_Create(t *testing.T) {
	tests := []struct {
		name          string
		req           *CreateL4ProxyRequest
		createdBy     int
		mockCreate    func(proxy *models.L4Proxy) error
		mockGetByPort func(port int, protocol string) (*models.L4Proxy, error)
		expectErr     error
		expectErrType interface{}
	}{
		{
			name:      "success",
			req:       newTestL4ProxyRequest(),
			createdBy: 1,
			mockCreate: func(proxy *models.L4Proxy) error {
				proxy.ID = 1
				return nil
			},
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
		},
		{
			name: "success - UDP proxy",
			req: &CreateL4ProxyRequest{
				Name:       "UDP Proxy",
				ListenPort: 5353,
				Protocol:   models.L4ProtocolUDP,
				IsActive:   true,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "8.8.8.8", Port: 53}},
					},
				},
			},
			createdBy: 1,
			mockCreate: func(proxy *models.L4Proxy) error {
				proxy.ID = 1
				return nil
			},
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
		},
		{
			name: "validation error - empty name",
			req: &CreateL4ProxyRequest{
				Name:       "",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			createdBy:     1,
			expectErrType: models.ErrL4ProxyNameRequired,
		},
		{
			name: "validation error - invalid protocol",
			req: &CreateL4ProxyRequest{
				Name:       "Test",
				ListenPort: 8080,
				Protocol:   "invalid",
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			createdBy:     1,
			expectErrType: models.ErrL4ProxyInvalidProtocol,
		},
		{
			name: "validation error - port zero",
			req: &CreateL4ProxyRequest{
				Name:       "Test",
				ListenPort: 0,
				Protocol:   models.L4ProtocolTCP,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			createdBy:     1,
			expectErrType: models.ErrL4ProxyInvalidPort,
		},
		{
			name: "validation error - port too high",
			req: &CreateL4ProxyRequest{
				Name:       "Test",
				ListenPort: 65536,
				Protocol:   models.L4ProtocolTCP,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			createdBy:     1,
			expectErrType: models.ErrL4ProxyInvalidPort,
		},
		{
			name:      "port conflict error",
			req:       newTestL4ProxyRequest(),
			createdBy: 1,
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return &models.L4Proxy{ID: 2, ListenPort: 8080, Protocol: models.L4ProtocolTCP}, nil
			},
			expectErr: ErrL4ProxyPortConflict,
		},
		{
			name:      "database error on port check",
			req:       newTestL4ProxyRequest(),
			createdBy: 1,
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, errors.New("database connection error")
			},
			expectErr: errors.New("failed to check port conflict"),
		},
		{
			name:      "database error on create",
			req:       newTestL4ProxyRequest(),
			createdBy: 1,
			mockCreate: func(_ *models.L4Proxy) error {
				return errors.New("database error")
			},
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
			expectErr: errors.New("failed to create l4 proxy"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				CreateFunc:    tc.mockCreate,
				GetByPortFunc: tc.mockGetByPort,
			}
			svc := NewL4ProxyService(repo, nil)

			result, err := svc.Create(tc.req, tc.createdBy)

			if tc.expectErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectErr, ErrL4ProxyPortConflict) {
					assert.ErrorIs(t, err, ErrL4ProxyPortConflict)
				} else {
					assert.Contains(t, err.Error(), tc.expectErr.Error())
				}
				return
			}

			if tc.expectErrType != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectErrType, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.req.Name, result.Name)
			assert.Equal(t, tc.req.ListenPort, result.ListenPort)
			assert.Equal(t, tc.req.Protocol, result.Protocol)
			assert.Equal(t, tc.createdBy, result.CreatedBy)
		})
	}
}

// TestL4ProxyService_Create_RouteValidation tests route validation during create
func TestL4ProxyService_Create_RouteValidation(t *testing.T) {
	regexPattern := "^SSH-.*"

	tests := []struct {
		name        string
		routes      []CreateL4RouteRequest
		expectedErr error
	}{
		{
			name: "valid route with TLS matcher and SNI hostnames",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherTLS,
					SNIHostnames:        []string{"example.com"},
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					TLSPassthrough:      true,
					Upstreams:           []L4UpstreamRequest{{Host: "backend", Port: 443}},
				},
			},
		},
		{
			name: "invalid route - missing upstreams",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherAny,
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					Upstreams:           []L4UpstreamRequest{},
				},
			},
			expectedErr: models.ErrL4RouteUpstreamsRequired,
		},
		{
			name: "invalid route - TLS terminate and passthrough both enabled",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherAny,
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					TLSTerminate:        true,
					TLSPassthrough:      true,
					Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 8080}},
				},
			},
			expectedErr: models.ErrL4RouteTLSConflict,
		},
		{
			name: "invalid route - regexp matcher without pattern",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherRegexp,
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 8080}},
				},
			},
			expectedErr: models.ErrL4RouteRegexRequired,
		},
		{
			name: "valid route - regexp matcher with pattern",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherRegexp,
					RegexPattern:        &regexPattern,
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 8080}},
				},
			},
		},
		{
			name: "invalid route - TLS matcher without SNI hostnames",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherTLS,
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					TLSPassthrough:      true,
					Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 8080}},
				},
			},
			expectedErr: models.ErrL4RouteSNIRequired,
		},
		{
			name: "invalid route - invalid matcher type",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         "invalid_matcher",
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 8080}},
				},
			},
			expectedErr: models.ErrL4RouteInvalidMatcher,
		},
		{
			name: "invalid route - invalid load balancing policy",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherAny,
					LoadBalancingPolicy: "invalid_policy",
					Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 8080}},
				},
			},
			expectedErr: models.ErrL4RouteInvalidLBPolicy,
		},
		{
			name: "invalid route - upstream with empty host",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherAny,
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					Upstreams:           []L4UpstreamRequest{{Host: "", Port: 8080}},
				},
			},
			expectedErr: models.ErrL4UpstreamHostRequired,
		},
		{
			name: "invalid route - upstream with invalid port",
			routes: []CreateL4RouteRequest{
				{
					Priority:            1,
					MatcherType:         models.L4MatcherAny,
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 0}},
				},
			},
			expectedErr: models.ErrL4UpstreamInvalidPort,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				GetByPortFunc: func(_ int, _ string) (*models.L4Proxy, error) {
					return nil, gorm.ErrRecordNotFound
				},
				CreateFunc: func(proxy *models.L4Proxy) error {
					proxy.ID = 1
					return nil
				},
			}
			svc := NewL4ProxyService(repo, nil)

			req := &CreateL4ProxyRequest{
				Name:       "Test Proxy",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				IsActive:   true,
				Routes:     tc.routes,
			}

			result, err := svc.Create(req, 1)

			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

// TestL4ProxyService_GetByID tests getting an L4 proxy by ID
func TestL4ProxyService_GetByID(t *testing.T) {
	tests := []struct {
		name        string
		id          int
		mockGetByID func(id int) (*models.L4Proxy, error)
		expectErr   error
	}{
		{
			name: "success - returns proxy with routes",
			id:   1,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return newTestL4Proxy(), nil
			},
		},
		{
			name: "not found",
			id:   999,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
			expectErr: ErrL4ProxyNotFound,
		},
		{
			name: "database error",
			id:   1,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, errors.New("database error")
			},
			expectErr: errors.New("failed to get l4 proxy"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				GetByIDFunc: tc.mockGetByID,
			}
			svc := NewL4ProxyService(repo, nil)

			result, err := svc.GetByID(tc.id)

			if tc.expectErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectErr, ErrL4ProxyNotFound) {
					assert.ErrorIs(t, err, ErrL4ProxyNotFound)
				} else {
					assert.Contains(t, err.Error(), tc.expectErr.Error())
				}
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.id, result.ID)
			assert.Len(t, result.Routes, 1)
		})
	}
}

// TestL4ProxyService_List tests listing L4 proxies with filters
func TestL4ProxyService_List(t *testing.T) {
	tests := []struct {
		name          string
		req           *ListL4ProxiesRequest
		mockList      func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error)
		expectErr     bool
		expectedPage  int
		expectedLimit int
		expectedTotal int64
		expectedCount int
		expectedPages int
	}{
		{
			name: "success with defaults",
			req:  &ListL4ProxiesRequest{Page: 0, Limit: 0},
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return []models.L4Proxy{*newTestL4Proxy()}, 1, nil
			},
			expectedPage:  1,
			expectedLimit: 20,
			expectedTotal: 1,
			expectedCount: 1,
			expectedPages: 1,
		},
		{
			name: "success with pagination",
			req:  &ListL4ProxiesRequest{Page: 2, Limit: 10},
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return []models.L4Proxy{}, 25, nil
			},
			expectedPage:  2,
			expectedLimit: 10,
			expectedTotal: 25,
			expectedCount: 0,
			expectedPages: 3,
		},
		{
			name: "empty result",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 10},
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return []models.L4Proxy{}, 0, nil
			},
			expectedPage:  1,
			expectedLimit: 10,
			expectedTotal: 0,
			expectedCount: 0,
			expectedPages: 0,
		},
		{
			name: "limit capped at 100",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 200},
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return []models.L4Proxy{}, 0, nil
			},
			expectedPage:  1,
			expectedLimit: 20,
			expectedTotal: 0,
			expectedCount: 0,
			expectedPages: 0,
		},
		{
			name: "negative page becomes 1",
			req:  &ListL4ProxiesRequest{Page: -1, Limit: 10},
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return []models.L4Proxy{}, 0, nil
			},
			expectedPage:  1,
			expectedLimit: 10,
			expectedTotal: 0,
			expectedCount: 0,
			expectedPages: 0,
		},
		{
			name: "with search filter",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 10, Search: "test"},
			mockList: func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				assert.Equal(t, "test", params.Search)
				return []models.L4Proxy{*newTestL4Proxy()}, 1, nil
			},
			expectedPage:  1,
			expectedLimit: 10,
			expectedTotal: 1,
			expectedCount: 1,
			expectedPages: 1,
		},
		{
			name: "with protocol filter",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 10, Protocol: "tcp"},
			mockList: func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				assert.Equal(t, "tcp", params.Protocol)
				return []models.L4Proxy{*newTestL4Proxy()}, 1, nil
			},
			expectedPage:  1,
			expectedLimit: 10,
			expectedTotal: 1,
			expectedCount: 1,
			expectedPages: 1,
		},
		{
			name: "with is_active filter true",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 10, IsActive: boolPtr(true)},
			mockList: func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				require.NotNil(t, params.IsActive)
				assert.True(t, *params.IsActive)
				return []models.L4Proxy{*newTestL4Proxy()}, 1, nil
			},
			expectedPage:  1,
			expectedLimit: 10,
			expectedTotal: 1,
			expectedCount: 1,
			expectedPages: 1,
		},
		{
			name: "with is_active filter false",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 10, IsActive: boolPtr(false)},
			mockList: func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				require.NotNil(t, params.IsActive)
				assert.False(t, *params.IsActive)
				proxy := newTestL4Proxy()
				proxy.IsActive = false
				return []models.L4Proxy{*proxy}, 1, nil
			},
			expectedPage:  1,
			expectedLimit: 10,
			expectedTotal: 1,
			expectedCount: 1,
			expectedPages: 1,
		},
		{
			name: "with sort and order",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 10, Sort: "name", Order: "asc"},
			mockList: func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				assert.Equal(t, "name", params.Sort)
				assert.Equal(t, "asc", params.Order)
				return []models.L4Proxy{*newTestL4Proxy()}, 1, nil
			},
			expectedPage:  1,
			expectedLimit: 10,
			expectedTotal: 1,
			expectedCount: 1,
			expectedPages: 1,
		},
		{
			name: "database error",
			req:  &ListL4ProxiesRequest{Page: 1, Limit: 10},
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return nil, 0, errors.New("database error")
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				ListFunc: tc.mockList,
			}
			svc := NewL4ProxyService(repo, nil)

			result, err := svc.List(tc.req)

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to list l4 proxies")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedPage, result.Page)
			assert.Equal(t, tc.expectedLimit, result.Limit)
			assert.Equal(t, tc.expectedTotal, result.Total)
			assert.Len(t, result.Items, tc.expectedCount)
			assert.Equal(t, tc.expectedPages, result.TotalPages)
		})
	}
}

// TestL4ProxyService_Update tests updating an L4 proxy
func TestL4ProxyService_Update(t *testing.T) {
	existingProxy := newTestL4Proxy()

	tests := []struct {
		name          string
		id            int
		req           *UpdateL4ProxyRequest
		mockGetByID   func(id int) (*models.L4Proxy, error)
		mockUpdate    func(proxy *models.L4Proxy) error
		mockGetByPort func(port int, protocol string) (*models.L4Proxy, error)
		expectErr     error
		expectErrType interface{}
	}{
		{
			name: "success - same port",
			id:   1,
			req: &UpdateL4ProxyRequest{
				Name:       "Updated Name",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				IsActive:   true,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return existingProxy, nil
			},
			mockUpdate: func(_ *models.L4Proxy) error {
				return nil
			},
		},
		{
			name: "success - different port",
			id:   1,
			req: &UpdateL4ProxyRequest{
				Name:       "Updated Name",
				ListenPort: 9090,
				Protocol:   models.L4ProtocolTCP,
				IsActive:   true,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 8080}},
					},
				},
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return existingProxy, nil
			},
			mockUpdate: func(_ *models.L4Proxy) error {
				return nil
			},
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
		},
		{
			name: "success - change protocol",
			id:   1,
			req: &UpdateL4ProxyRequest{
				Name:       "Updated Name",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolUDP,
				IsActive:   true,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return existingProxy, nil
			},
			mockUpdate: func(_ *models.L4Proxy) error {
				return nil
			},
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
		},
		{
			name: "not found",
			id:   999,
			req: &UpdateL4ProxyRequest{
				Name:       "Updated Name",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
			expectErr: ErrL4ProxyNotFound,
		},
		{
			name: "validation error - empty name",
			id:   1,
			req: &UpdateL4ProxyRequest{
				Name:       "",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return existingProxy, nil
			},
			expectErrType: models.ErrL4ProxyNameRequired,
		},
		{
			name: "port conflict when changing port",
			id:   1,
			req: &UpdateL4ProxyRequest{
				Name:       "Updated Name",
				ListenPort: 9090,
				Protocol:   models.L4ProtocolTCP,
				IsActive:   true,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 8080}},
					},
				},
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return existingProxy, nil
			},
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return &models.L4Proxy{ID: 2, ListenPort: 9090, Protocol: models.L4ProtocolTCP}, nil
			},
			expectErr: ErrL4ProxyPortConflict,
		},
		{
			name: "database error on get",
			id:   1,
			req: &UpdateL4ProxyRequest{
				Name:       "Updated Name",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, errors.New("database error")
			},
			expectErr: errors.New("failed to get l4 proxy"),
		},
		{
			name: "database error on update",
			id:   1,
			req: &UpdateL4ProxyRequest{
				Name:       "Updated Name",
				ListenPort: 8080,
				Protocol:   models.L4ProtocolTCP,
				IsActive:   true,
				Routes: []CreateL4RouteRequest{
					{
						Priority:            1,
						MatcherType:         models.L4MatcherAny,
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
						Upstreams:           []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					},
				},
			},
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return existingProxy, nil
			},
			mockUpdate: func(_ *models.L4Proxy) error {
				return errors.New("database error")
			},
			expectErr: errors.New("failed to update l4 proxy"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				GetByIDFunc:   tc.mockGetByID,
				UpdateFunc:    tc.mockUpdate,
				GetByPortFunc: tc.mockGetByPort,
			}
			svc := NewL4ProxyService(repo, nil)

			result, err := svc.Update(tc.id, tc.req)

			if tc.expectErr != nil {
				require.Error(t, err)
				switch {
				case errors.Is(tc.expectErr, ErrL4ProxyNotFound):
					assert.ErrorIs(t, err, ErrL4ProxyNotFound)
				case errors.Is(tc.expectErr, ErrL4ProxyPortConflict):
					assert.ErrorIs(t, err, ErrL4ProxyPortConflict)
				default:
					assert.Contains(t, err.Error(), tc.expectErr.Error())
				}
				assert.Nil(t, result)
				return
			}

			if tc.expectErrType != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectErrType, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

// TestL4ProxyService_Delete tests deleting an L4 proxy
func TestL4ProxyService_Delete(t *testing.T) {
	tests := []struct {
		name        string
		id          int
		mockGetByID func(id int) (*models.L4Proxy, error)
		mockDelete  func(id int) error
		expectErr   error
	}{
		{
			name: "success",
			id:   1,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return newTestL4Proxy(), nil
			},
			mockDelete: func(_ int) error {
				return nil
			},
		},
		{
			name: "not found",
			id:   999,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
			expectErr: ErrL4ProxyNotFound,
		},
		{
			name: "database error on get",
			id:   1,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, errors.New("database error")
			},
			expectErr: errors.New("failed to get l4 proxy"),
		},
		{
			name: "database error on delete",
			id:   1,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return newTestL4Proxy(), nil
			},
			mockDelete: func(_ int) error {
				return errors.New("database error")
			},
			expectErr: errors.New("failed to delete l4 proxy"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deleteCalled := false
			repo := &MockL4ProxyRepository{
				GetByIDFunc: tc.mockGetByID,
				DeleteFunc: func(id int) error {
					deleteCalled = true
					if tc.mockDelete != nil {
						return tc.mockDelete(id)
					}
					return nil
				},
			}
			svc := NewL4ProxyService(repo, nil)

			err := svc.Delete(tc.id)

			if tc.expectErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectErr, ErrL4ProxyNotFound) {
					assert.ErrorIs(t, err, ErrL4ProxyNotFound)
				} else {
					assert.Contains(t, err.Error(), tc.expectErr.Error())
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, deleteCalled, "Delete should have been called")
		})
	}
}

// TestL4ProxyService_ToggleActive tests toggling the active status
func TestL4ProxyService_ToggleActive(t *testing.T) {
	tests := []struct {
		name           string
		id             int
		initialActive  bool
		mockGetByID    func(id int) (*models.L4Proxy, error)
		mockUpdate     func(proxy *models.L4Proxy) error
		expectErr      error
		expectIsActive bool
	}{
		{
			name:          "success - active to inactive",
			id:            1,
			initialActive: true,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				proxy := newTestL4Proxy()
				proxy.IsActive = true
				return proxy, nil
			},
			mockUpdate: func(_ *models.L4Proxy) error {
				return nil
			},
			expectIsActive: false,
		},
		{
			name:          "success - inactive to active",
			id:            1,
			initialActive: false,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				proxy := newTestL4Proxy()
				proxy.IsActive = false
				return proxy, nil
			},
			mockUpdate: func(_ *models.L4Proxy) error {
				return nil
			},
			expectIsActive: true,
		},
		{
			name: "not found",
			id:   999,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
			expectErr: ErrL4ProxyNotFound,
		},
		{
			name: "database error on get",
			id:   1,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return nil, errors.New("database error")
			},
			expectErr: errors.New("failed to get l4 proxy"),
		},
		{
			name: "database error on update",
			id:   1,
			mockGetByID: func(_ int) (*models.L4Proxy, error) {
				return newTestL4Proxy(), nil
			},
			mockUpdate: func(_ *models.L4Proxy) error {
				return errors.New("database error")
			},
			expectErr: errors.New("failed to toggle l4 proxy status"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var updatedProxy *models.L4Proxy
			repo := &MockL4ProxyRepository{
				GetByIDFunc: tc.mockGetByID,
				UpdateFunc: func(proxy *models.L4Proxy) error {
					updatedProxy = proxy
					if tc.mockUpdate != nil {
						return tc.mockUpdate(proxy)
					}
					return nil
				},
			}
			svc := NewL4ProxyService(repo, nil)

			result, err := svc.ToggleActive(tc.id)

			if tc.expectErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectErr, ErrL4ProxyNotFound) {
					assert.ErrorIs(t, err, ErrL4ProxyNotFound)
				} else {
					assert.Contains(t, err.Error(), tc.expectErr.Error())
				}
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectIsActive, result.IsActive)
			assert.Equal(t, tc.expectIsActive, updatedProxy.IsActive)
		})
	}
}

// TestL4ProxyService_GetStats tests getting L4 proxy statistics
func TestL4ProxyService_GetStats(t *testing.T) {
	tests := []struct {
		name                   string
		mockList               func(params repository.L4ProxyListParams) ([]models.L4Proxy, int64, error)
		expectErr              bool
		expectedTotalProxies   int64
		expectedActiveProxies  int64
		expectedTCPProxies     int64
		expectedUDPProxies     int64
		expectedTotalRoutes    int64
		expectedTotalUpstreams int64
	}{
		{
			name: "success - mixed proxies",
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				proxy1 := newTestL4Proxy()
				proxy1.ID = 1
				proxy1.IsActive = true
				proxy1.Protocol = models.L4ProtocolTCP

				proxy2 := newTestL4Proxy()
				proxy2.ID = 2
				proxy2.IsActive = false
				proxy2.Protocol = models.L4ProtocolUDP
				proxy2.Routes = []models.L4Route{
					{
						ID:        2,
						L4ProxyID: 2,
						Upstreams: models.L4UpstreamSlice{
							{Host: "127.0.0.1", Port: 8080},
							{Host: "127.0.0.1", Port: 8081},
						},
					},
				}

				proxy3 := newTestL4Proxy()
				proxy3.ID = 3
				proxy3.IsActive = true
				proxy3.Protocol = models.L4ProtocolTCP
				proxy3.Routes = []models.L4Route{
					{ID: 3, L4ProxyID: 3, Upstreams: models.L4UpstreamSlice{{Host: "127.0.0.1", Port: 9000}}},
					{ID: 4, L4ProxyID: 3, Upstreams: models.L4UpstreamSlice{{Host: "127.0.0.1", Port: 9001}}},
				}

				return []models.L4Proxy{*proxy1, *proxy2, *proxy3}, 3, nil
			},
			expectedTotalProxies:   3,
			expectedActiveProxies:  2,
			expectedTCPProxies:     2,
			expectedUDPProxies:     1,
			expectedTotalRoutes:    4, // 1 + 1 + 2
			expectedTotalUpstreams: 5, // 1 + 2 + 1 + 1
		},
		{
			name: "success - empty",
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return []models.L4Proxy{}, 0, nil
			},
			expectedTotalProxies:   0,
			expectedActiveProxies:  0,
			expectedTCPProxies:     0,
			expectedUDPProxies:     0,
			expectedTotalRoutes:    0,
			expectedTotalUpstreams: 0,
		},
		{
			name: "success - all active TCP",
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				proxy1 := newTestL4Proxy()
				proxy1.IsActive = true
				proxy1.Protocol = models.L4ProtocolTCP

				proxy2 := newTestL4Proxy()
				proxy2.ID = 2
				proxy2.IsActive = true
				proxy2.Protocol = models.L4ProtocolTCP

				return []models.L4Proxy{*proxy1, *proxy2}, 2, nil
			},
			expectedTotalProxies:   2,
			expectedActiveProxies:  2,
			expectedTCPProxies:     2,
			expectedUDPProxies:     0,
			expectedTotalRoutes:    2,
			expectedTotalUpstreams: 2,
		},
		{
			name: "database error",
			mockList: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
				return nil, 0, errors.New("database error")
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				ListFunc: tc.mockList,
			}
			svc := NewL4ProxyService(repo, nil)

			result, err := svc.GetStats()

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get l4 proxy stats")
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedTotalProxies, result.TotalProxies)
			assert.Equal(t, tc.expectedActiveProxies, result.ActiveProxies)
			assert.Equal(t, tc.expectedTCPProxies, result.TCPProxies)
			assert.Equal(t, tc.expectedUDPProxies, result.UDPProxies)
			assert.Equal(t, tc.expectedTotalRoutes, result.TotalRoutes)
			assert.Equal(t, tc.expectedTotalUpstreams, result.TotalUpstreams)
		})
	}
}

// TestL4ProxyService_CheckPortConflict tests the port conflict checking logic
func TestL4ProxyService_CheckPortConflict(t *testing.T) {
	tests := []struct {
		name          string
		port          int
		protocol      string
		excludeID     int
		mockGetByPort func(port int, protocol string) (*models.L4Proxy, error)
		expectErr     error
	}{
		{
			name:      "no conflict - port not in use",
			port:      8080,
			protocol:  models.L4ProtocolTCP,
			excludeID: 0,
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, gorm.ErrRecordNotFound
			},
		},
		{
			name:      "no conflict - same proxy (update case)",
			port:      8080,
			protocol:  models.L4ProtocolTCP,
			excludeID: 1,
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return &models.L4Proxy{ID: 1, ListenPort: 8080, Protocol: models.L4ProtocolTCP}, nil
			},
		},
		{
			name:      "conflict - different proxy",
			port:      8080,
			protocol:  models.L4ProtocolTCP,
			excludeID: 1,
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return &models.L4Proxy{ID: 2, ListenPort: 8080, Protocol: models.L4ProtocolTCP}, nil
			},
			expectErr: ErrL4ProxyPortConflict,
		},
		{
			name:      "conflict - new proxy creation",
			port:      8080,
			protocol:  models.L4ProtocolTCP,
			excludeID: 0,
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return &models.L4Proxy{ID: 2, ListenPort: 8080, Protocol: models.L4ProtocolTCP}, nil
			},
			expectErr: ErrL4ProxyPortConflict,
		},
		{
			name:      "database error",
			port:      8080,
			protocol:  models.L4ProtocolTCP,
			excludeID: 0,
			mockGetByPort: func(_ int, _ string) (*models.L4Proxy, error) {
				return nil, errors.New("database error")
			},
			expectErr: errors.New("failed to check port conflict"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				GetByPortFunc: tc.mockGetByPort,
			}
			svc := NewL4ProxyService(repo, nil)

			err := svc.checkPortConflict(tc.port, tc.protocol, tc.excludeID)

			if tc.expectErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectErr, ErrL4ProxyPortConflict) {
					assert.ErrorIs(t, err, ErrL4ProxyPortConflict)
				} else {
					assert.Contains(t, err.Error(), tc.expectErr.Error())
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestL4ProxyService_RequestToModel tests the request to model conversion
func TestL4ProxyService_RequestToModel(t *testing.T) {
	svc := NewL4ProxyService(&MockL4ProxyRepository{}, nil)
	description := "Test description"
	weight := 10
	regexPattern := ".*"
	proxyProtocol := "v1"

	t.Run("basic conversion", func(t *testing.T) {
		req := &CreateL4ProxyRequest{
			Name:        "Test Proxy",
			Description: &description,
			ListenPort:  8080,
			Protocol:    models.L4ProtocolTCP,
			IsActive:    true,
			Routes: []CreateL4RouteRequest{
				{
					Priority:    1,
					MatcherType: models.L4MatcherAny,
					Upstreams: []L4UpstreamRequest{
						{Host: "127.0.0.1", Port: 9090, Weight: &weight},
					},
					LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				},
			},
		}

		result := svc.requestToModel(req)

		assert.Equal(t, "Test Proxy", result.Name)
		assert.Equal(t, &description, result.Description)
		assert.Equal(t, 8080, result.ListenPort)
		assert.Equal(t, models.L4ProtocolTCP, result.Protocol)
		assert.True(t, result.IsActive)
		require.Len(t, result.Routes, 1)
		assert.Equal(t, 1, result.Routes[0].Priority)
		assert.Equal(t, models.L4MatcherAny, result.Routes[0].MatcherType)
		require.Len(t, result.Routes[0].Upstreams, 1)
		assert.Equal(t, "127.0.0.1", result.Routes[0].Upstreams[0].Host)
		assert.Equal(t, 9090, result.Routes[0].Upstreams[0].Port)
		assert.Equal(t, &weight, result.Routes[0].Upstreams[0].Weight)
	})

	t.Run("conversion with all route fields", func(t *testing.T) {
		req := &CreateL4ProxyRequest{
			Name:       "Test Proxy",
			ListenPort: 8080,
			Protocol:   models.L4ProtocolTCP,
			Routes: []CreateL4RouteRequest{
				{
					Priority:             1,
					MatcherType:          models.L4MatcherRegexp,
					SNIHostnames:         []string{"example.com", "test.com"},
					AllowedIPRanges:      []string{"192.168.1.0/24"},
					RegexPattern:         &regexPattern,
					Upstreams:            []L4UpstreamRequest{{Host: "127.0.0.1", Port: 9090}},
					LoadBalancingPolicy:  models.L4LoadBalancingLeastConn,
					TLSTerminate:         true,
					TLSPassthrough:       false,
					ProxyProtocolVersion: &proxyProtocol,
				},
			},
		}

		result := svc.requestToModel(req)

		require.Len(t, result.Routes, 1)
		route := result.Routes[0]
		assert.Equal(t, models.L4MatcherRegexp, route.MatcherType)
		assert.Equal(t, []string{"example.com", "test.com"}, []string(route.SNIHostnames))
		assert.Equal(t, []string{"192.168.1.0/24"}, []string(route.AllowedIPRanges))
		assert.Equal(t, &regexPattern, route.RegexPattern)
		assert.Equal(t, models.L4LoadBalancingLeastConn, route.LoadBalancingPolicy)
		assert.True(t, route.TLSTerminate)
		assert.False(t, route.TLSPassthrough)
		assert.Equal(t, &proxyProtocol, route.ProxyProtocolVersion)
	})

	t.Run("conversion without routes", func(t *testing.T) {
		req := &CreateL4ProxyRequest{
			Name:       "Test Proxy",
			ListenPort: 8080,
			Protocol:   models.L4ProtocolTCP,
			IsActive:   true,
			Routes:     []CreateL4RouteRequest{},
		}

		result := svc.requestToModel(req)

		assert.Equal(t, "Test Proxy", result.Name)
		assert.Empty(t, result.Routes)
	})
}

// TestL4ProxyService_UpdateRequestToModel tests the update request to model conversion
func TestL4ProxyService_UpdateRequestToModel(t *testing.T) {
	svc := NewL4ProxyService(&MockL4ProxyRepository{}, nil)
	description := "Updated description"

	req := &UpdateL4ProxyRequest{
		Name:        "Updated Proxy",
		Description: &description,
		ListenPort:  9090,
		Protocol:    models.L4ProtocolUDP,
		IsActive:    false,
		Routes: []CreateL4RouteRequest{
			{
				Priority:            2,
				MatcherType:         models.L4MatcherSSH,
				LoadBalancingPolicy: models.L4LoadBalancingFirst,
				Upstreams:           []L4UpstreamRequest{{Host: "ssh-server", Port: 22}},
			},
		},
	}

	result := svc.updateRequestToModel(req)

	assert.Equal(t, "Updated Proxy", result.Name)
	assert.Equal(t, &description, result.Description)
	assert.Equal(t, 9090, result.ListenPort)
	assert.Equal(t, models.L4ProtocolUDP, result.Protocol)
	assert.False(t, result.IsActive)
	require.Len(t, result.Routes, 1)
	assert.Equal(t, 2, result.Routes[0].Priority)
	assert.Equal(t, models.L4MatcherSSH, result.Routes[0].MatcherType)
}

// TestL4ProxyServiceErrors tests that service errors are defined correctly
func TestL4ProxyServiceErrors(t *testing.T) {
	assert.Equal(t, "l4 proxy not found", ErrL4ProxyNotFound.Error())
	assert.Equal(t, "port and protocol combination already in use", ErrL4ProxyPortConflict.Error())
	assert.Equal(t, "l4 proxy is already active", ErrL4ProxyAlreadyActive.Error())
	assert.Equal(t, "l4 proxy is already inactive", ErrL4ProxyAlreadyInactive.Error())
}

// TestL4ProxyService_Pagination tests pagination calculation edge cases
func TestL4ProxyService_Pagination(t *testing.T) {
	tests := []struct {
		name               string
		total              int64
		limit              int
		expectedTotalPages int
	}{
		{"exact division", 100, 10, 10},
		{"with remainder", 101, 10, 11},
		{"single page", 5, 10, 1},
		{"zero items", 0, 10, 0},
		{"large dataset", 1000, 20, 50},
		{"single item", 1, 10, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockL4ProxyRepository{
				ListFunc: func(_ repository.L4ProxyListParams) ([]models.L4Proxy, int64, error) {
					return []models.L4Proxy{}, tc.total, nil
				},
			}
			svc := NewL4ProxyService(repo, nil)

			result, err := svc.List(&ListL4ProxiesRequest{Page: 1, Limit: tc.limit})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedTotalPages, result.TotalPages)
		})
	}
}

// TestL4ProxyService_Update_PreservesCreatedFields tests that update preserves created_by and created_at
func TestL4ProxyService_Update_PreservesCreatedFields(t *testing.T) {
	originalCreatedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	originalCreatedBy := 42

	existingProxy := &models.L4Proxy{
		ID:         1,
		Name:       "Original",
		ListenPort: 8080,
		Protocol:   models.L4ProtocolTCP,
		IsActive:   true,
		CreatedBy:  originalCreatedBy,
		CreatedAt:  originalCreatedAt,
		Routes: []models.L4Route{
			{
				ID:                  1,
				L4ProxyID:           1,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				Upstreams:           models.L4UpstreamSlice{{Host: "localhost", Port: 9000}},
			},
		},
	}

	var updatedProxy *models.L4Proxy

	repo := &MockL4ProxyRepository{
		GetByIDFunc: func(_ int) (*models.L4Proxy, error) {
			return existingProxy, nil
		},
		UpdateFunc: func(proxy *models.L4Proxy) error {
			updatedProxy = proxy
			return nil
		},
	}
	svc := NewL4ProxyService(repo, nil)

	req := &UpdateL4ProxyRequest{
		Name:       "Updated Name",
		ListenPort: 8080,
		Protocol:   models.L4ProtocolTCP,
		IsActive:   false,
		Routes: []CreateL4RouteRequest{
			{
				Priority:            1,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				Upstreams:           []L4UpstreamRequest{{Host: "localhost", Port: 9000}},
			},
		},
	}

	_, err := svc.Update(1, req)

	require.NoError(t, err)
	require.NotNil(t, updatedProxy)
	assert.Equal(t, originalCreatedBy, updatedProxy.CreatedBy, "CreatedBy should be preserved")
	assert.Equal(t, originalCreatedAt, updatedProxy.CreatedAt, "CreatedAt should be preserved")
	assert.Equal(t, 1, updatedProxy.ID, "ID should be preserved")
}
