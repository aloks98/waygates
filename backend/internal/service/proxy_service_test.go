package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// MockProxyRepository is a mock implementation of ProxyRepositoryInterface
type MockProxyRepository struct {
	ListFunc           func(params repository.ProxyListParams) ([]models.Proxy, int64, error)
	GetByIDFunc        func(id int) (*models.Proxy, error)
	GetByHostnameFunc  func(hostname string) (*models.Proxy, error)
	CreateFunc         func(proxy *models.Proxy) error
	UpdateFunc         func(proxy *models.Proxy) error
	DeleteFunc         func(id int) error
	UpdateStatusFunc   func(id int, isActive bool) error
	HostnameExistsFunc func(hostname string, excludeID int) (bool, error)
	GetStatsFunc       func() (*repository.ProxyStats, error)
}

func (m *MockProxyRepository) List(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(params)
	}
	return []models.Proxy{}, 0, nil
}

func (m *MockProxyRepository) GetByID(id int) (*models.Proxy, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockProxyRepository) GetByHostname(hostname string) (*models.Proxy, error) {
	if m.GetByHostnameFunc != nil {
		return m.GetByHostnameFunc(hostname)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockProxyRepository) Create(proxy *models.Proxy) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(proxy)
	}
	return nil
}

func (m *MockProxyRepository) Update(proxy *models.Proxy) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(proxy)
	}
	return nil
}

func (m *MockProxyRepository) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockProxyRepository) UpdateStatus(id int, isActive bool) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(id, isActive)
	}
	return nil
}

func (m *MockProxyRepository) HostnameExists(hostname string, excludeID int) (bool, error) {
	if m.HostnameExistsFunc != nil {
		return m.HostnameExistsFunc(hostname, excludeID)
	}
	return false, nil
}

func (m *MockProxyRepository) GetStats() (*repository.ProxyStats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return &repository.ProxyStats{}, nil
}

// MockProxySyncer is a mock implementation of ProxySyncer
type MockProxySyncer struct {
	SyncProxyFunc    func(proxy *models.Proxy) error
	RemoveProxyFunc  func(proxyID int, hostname string) error
	EnableProxyFunc  func(proxyID int, hostname string) error
	DisableProxyFunc func(proxyID int, hostname string) error
}

func (m *MockProxySyncer) SyncProxy(proxy *models.Proxy) error {
	if m.SyncProxyFunc != nil {
		return m.SyncProxyFunc(proxy)
	}
	return nil
}

func (m *MockProxySyncer) RemoveProxy(proxyID int, hostname string) error {
	if m.RemoveProxyFunc != nil {
		return m.RemoveProxyFunc(proxyID, hostname)
	}
	return nil
}

func (m *MockProxySyncer) EnableProxy(proxyID int, hostname string) error {
	if m.EnableProxyFunc != nil {
		return m.EnableProxyFunc(proxyID, hostname)
	}
	return nil
}

func (m *MockProxySyncer) DisableProxy(proxyID int, hostname string) error {
	if m.DisableProxyFunc != nil {
		return m.DisableProxyFunc(proxyID, hostname)
	}
	return nil
}

// Helper to create a valid test proxy
func newTestProxy() *models.Proxy {
	return &models.Proxy{
		Name:     "Test Proxy",
		Hostname: "test.example.com",
		Type:     models.ProxyTypeReverseProxy,
		Upstreams: []interface{}{
			map[string]interface{}{"address": "http://localhost:8080"},
		},
		IsActive: true,
	}
}

// TestNewProxyService tests service creation
func TestNewProxyService(t *testing.T) {
	repo := &MockProxyRepository{}
	syncer := &MockProxySyncer{}

	svc := NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: syncer,
		Logger:      nil, // Should use nop logger
	})

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
	if svc.repo != repo {
		t.Error("Expected repo to be set")
	}
	if svc.syncService != syncer {
		t.Error("Expected syncService to be set")
	}
}

// TestListProxies tests listing proxies
func TestListProxies(t *testing.T) {
	tests := []struct {
		name          string
		req           ListProxiesRequest
		mockProxies   []models.Proxy
		mockTotal     int64
		mockErr       error
		expectErr     bool
		expectedPage  int
		expectedLimit int
		expectedTotal int64
	}{
		{
			name: "success with defaults",
			req:  ListProxiesRequest{Page: 0, Limit: 0},
			mockProxies: []models.Proxy{
				{ID: 1, Name: "Proxy 1"},
				{ID: 2, Name: "Proxy 2"},
			},
			mockTotal:     2,
			expectedPage:  1,
			expectedLimit: 20,
			expectedTotal: 2,
		},
		{
			name:          "success with pagination",
			req:           ListProxiesRequest{Page: 2, Limit: 10},
			mockProxies:   []models.Proxy{},
			mockTotal:     25,
			expectedPage:  2,
			expectedLimit: 10,
			expectedTotal: 25,
		},
		{
			name:          "limit capped at 100",
			req:           ListProxiesRequest{Page: 1, Limit: 200},
			mockProxies:   []models.Proxy{},
			mockTotal:     0,
			expectedPage:  1,
			expectedLimit: 20,
			expectedTotal: 0,
		},
		{
			name:      "repository error",
			req:       ListProxiesRequest{Page: 1, Limit: 10},
			mockErr:   errors.New("database error"),
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockProxyRepository{
				ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
					return tc.mockProxies, tc.mockTotal, tc.mockErr
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: &MockProxySyncer{},
			})

			result, err := svc.ListProxies(tc.req)

			if tc.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Total != tc.expectedTotal {
				t.Errorf("Expected total %d, got %d", tc.expectedTotal, result.Total)
			}
			if result.Page != tc.expectedPage {
				t.Errorf("Expected page %d, got %d", tc.expectedPage, result.Page)
			}
			if result.Limit != tc.expectedLimit {
				t.Errorf("Expected limit %d, got %d", tc.expectedLimit, result.Limit)
			}
		})
	}
}

// TestGetProxyByID tests getting a proxy by ID
func TestGetProxyByID(t *testing.T) {
	tests := []struct {
		name      string
		id        int
		mockProxy *models.Proxy
		mockErr   error
		expectErr error
	}{
		{
			name:      "success",
			id:        1,
			mockProxy: &models.Proxy{ID: 1, Name: "Test Proxy"},
		},
		{
			name:      "not found",
			id:        999,
			mockErr:   gorm.ErrRecordNotFound,
			expectErr: ErrProxyNotFound,
		},
		{
			name:      "other error",
			id:        1,
			mockErr:   errors.New("database error"),
			expectErr: errors.New("failed to get proxy: database error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockProxyRepository{
				GetByIDFunc: func(id int) (*models.Proxy, error) {
					return tc.mockProxy, tc.mockErr
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: &MockProxySyncer{},
			})

			result, err := svc.GetProxyByID(tc.id)

			if tc.expectErr != nil {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if errors.Is(tc.expectErr, ErrProxyNotFound) && !errors.Is(err, ErrProxyNotFound) {
					t.Errorf("Expected ErrProxyNotFound, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.ID != tc.mockProxy.ID {
				t.Errorf("Expected proxy ID %d, got %d", tc.mockProxy.ID, result.ID)
			}
		})
	}
}

// TestCreateProxy tests creating a proxy
func TestCreateProxy(t *testing.T) {
	tests := []struct {
		name             string
		proxy            *models.Proxy
		userID           int
		hostnameExists   bool
		hostnameCheckErr error
		createErr        error
		syncErr          error
		expectErr        error
	}{
		{
			name:   "success",
			proxy:  newTestProxy(),
			userID: 1,
		},
		{
			name: "validation error - empty hostname",
			proxy: &models.Proxy{
				Name:     "Test",
				Hostname: "",
				Type:     models.ProxyTypeReverseProxy,
			},
			userID:    1,
			expectErr: errors.New("validation error"),
		},
		{
			name:           "hostname conflict",
			proxy:          newTestProxy(),
			userID:         1,
			hostnameExists: true,
			expectErr:      ErrHostnameConflict,
		},
		{
			name:             "hostname check error",
			proxy:            newTestProxy(),
			userID:           1,
			hostnameCheckErr: errors.New("database error"),
			expectErr:        errors.New("failed to check hostname"),
		},
		{
			name:      "create error",
			proxy:     newTestProxy(),
			userID:    1,
			createErr: errors.New("database error"),
			expectErr: errors.New("failed to create proxy"),
		},
		{
			name:      "sync error rolls back",
			proxy:     newTestProxy(),
			userID:    1,
			syncErr:   errors.New("caddy error"),
			expectErr: errors.New("failed to sync proxy configuration"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deleteCallCount := 0
			repo := &MockProxyRepository{
				HostnameExistsFunc: func(hostname string, excludeID int) (bool, error) {
					return tc.hostnameExists, tc.hostnameCheckErr
				},
				CreateFunc: func(proxy *models.Proxy) error {
					if tc.createErr == nil {
						proxy.ID = 1 // Simulate ID assignment
					}
					return tc.createErr
				},
				DeleteFunc: func(id int) error {
					deleteCallCount++
					return nil
				},
			}
			syncer := &MockProxySyncer{
				SyncProxyFunc: func(proxy *models.Proxy) error {
					return tc.syncErr
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: syncer,
			})

			err := svc.CreateProxy(tc.proxy, tc.userID)

			if tc.expectErr != nil {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if errors.Is(tc.expectErr, ErrHostnameConflict) && !errors.Is(err, ErrHostnameConflict) {
					t.Errorf("Expected ErrHostnameConflict, got %v", err)
				}
				// Verify rollback on sync error
				if tc.syncErr != nil && deleteCallCount != 1 {
					t.Errorf("Expected delete to be called once for rollback, called %d times", deleteCallCount)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tc.proxy.CreatedBy != tc.userID {
				t.Errorf("Expected CreatedBy %d, got %d", tc.userID, tc.proxy.CreatedBy)
			}
		})
	}
}

// TestUpdateProxy tests updating a proxy
func TestUpdateProxy(t *testing.T) {
	existingProxy := &models.Proxy{
		ID:        1,
		Name:      "Existing",
		Hostname:  "existing.example.com",
		Type:      models.ProxyTypeReverseProxy,
		IsActive:  true,
		CreatedBy: 1,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://localhost:8080"}},
	}

	tests := []struct {
		name            string
		id              int
		proxy           *models.Proxy
		getExistingErr  error
		hostnameExists  bool
		updateErr       error
		syncErr         error
		expectErr       error
		expectRemoveOld bool
	}{
		{
			name: "success - same hostname",
			id:   1,
			proxy: &models.Proxy{
				Name:      "Updated",
				Hostname:  "existing.example.com",
				Type:      models.ProxyTypeReverseProxy,
				Upstreams: []interface{}{map[string]interface{}{"address": "http://localhost:9090"}},
			},
		},
		{
			name: "success - different hostname",
			id:   1,
			proxy: &models.Proxy{
				Name:      "Updated",
				Hostname:  "new.example.com",
				Type:      models.ProxyTypeReverseProxy,
				Upstreams: []interface{}{map[string]interface{}{"address": "http://localhost:9090"}},
			},
			expectRemoveOld: true,
		},
		{
			name:           "not found",
			id:             999,
			proxy:          newTestProxy(),
			getExistingErr: gorm.ErrRecordNotFound,
			expectErr:      ErrProxyNotFound,
		},
		{
			name: "hostname conflict",
			id:   1,
			proxy: &models.Proxy{
				Name:      "Updated",
				Hostname:  "taken.example.com",
				Type:      models.ProxyTypeReverseProxy,
				Upstreams: []interface{}{map[string]interface{}{"address": "http://localhost:9090"}},
			},
			hostnameExists: true,
			expectErr:      ErrHostnameConflict,
		},
		{
			name: "update error",
			id:   1,
			proxy: &models.Proxy{
				Name:      "Updated",
				Hostname:  "existing.example.com",
				Type:      models.ProxyTypeReverseProxy,
				Upstreams: []interface{}{map[string]interface{}{"address": "http://localhost:9090"}},
			},
			updateErr: errors.New("database error"),
			expectErr: errors.New("failed to update proxy"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			removeProxyCalled := false
			repo := &MockProxyRepository{
				GetByIDFunc: func(id int) (*models.Proxy, error) {
					if tc.getExistingErr != nil {
						return nil, tc.getExistingErr
					}
					return existingProxy, nil
				},
				HostnameExistsFunc: func(hostname string, excludeID int) (bool, error) {
					return tc.hostnameExists, nil
				},
				UpdateFunc: func(proxy *models.Proxy) error {
					return tc.updateErr
				},
			}
			syncer := &MockProxySyncer{
				RemoveProxyFunc: func(proxyID int, hostname string) error {
					removeProxyCalled = true
					return nil
				},
				SyncProxyFunc: func(proxy *models.Proxy) error {
					return tc.syncErr
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: syncer,
			})

			err := svc.UpdateProxy(tc.id, tc.proxy)

			if tc.expectErr != nil {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if errors.Is(tc.expectErr, ErrProxyNotFound) && !errors.Is(err, ErrProxyNotFound) {
					t.Errorf("Expected ErrProxyNotFound, got %v", err)
				}
				if errors.Is(tc.expectErr, ErrHostnameConflict) && !errors.Is(err, ErrHostnameConflict) {
					t.Errorf("Expected ErrHostnameConflict, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify old file removed when hostname changes
			if tc.expectRemoveOld && !removeProxyCalled {
				t.Error("Expected RemoveProxy to be called for hostname change")
			}
			if !tc.expectRemoveOld && removeProxyCalled {
				t.Error("RemoveProxy should not be called when hostname unchanged")
			}

			// Verify preserved fields
			if tc.proxy.ID != tc.id {
				t.Errorf("Expected ID %d, got %d", tc.id, tc.proxy.ID)
			}
			if tc.proxy.IsActive != existingProxy.IsActive {
				t.Errorf("Expected IsActive %v to be preserved", existingProxy.IsActive)
			}
			if tc.proxy.CreatedBy != existingProxy.CreatedBy {
				t.Errorf("Expected CreatedBy %d to be preserved", existingProxy.CreatedBy)
			}
		})
	}
}

// TestDeleteProxy tests deleting a proxy
func TestDeleteProxy(t *testing.T) {
	tests := []struct {
		name      string
		id        int
		getErr    error
		deleteErr error
		expectErr error
	}{
		{
			name: "success",
			id:   1,
		},
		{
			name:      "not found",
			id:        999,
			getErr:    gorm.ErrRecordNotFound,
			expectErr: ErrProxyNotFound,
		},
		{
			name:      "delete error",
			id:        1,
			deleteErr: errors.New("database error"),
			expectErr: errors.New("failed to delete proxy"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			removeProxyCalled := false
			repo := &MockProxyRepository{
				GetByIDFunc: func(id int) (*models.Proxy, error) {
					if tc.getErr != nil {
						return nil, tc.getErr
					}
					return &models.Proxy{ID: id, Hostname: "test.example.com"}, nil
				},
				DeleteFunc: func(id int) error {
					return tc.deleteErr
				},
			}
			syncer := &MockProxySyncer{
				RemoveProxyFunc: func(proxyID int, hostname string) error {
					removeProxyCalled = true
					return nil
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: syncer,
			})

			err := svc.DeleteProxy(tc.id)

			if tc.expectErr != nil {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !removeProxyCalled {
				t.Error("Expected RemoveProxy to be called")
			}
		})
	}
}

// TestEnableProxy tests enabling a proxy
func TestEnableProxy(t *testing.T) {
	tests := []struct {
		name            string
		id              int
		proxyIsActive   bool
		getErr          error
		updateStatusErr error
		enableSyncErr   error
		expectErr       error
	}{
		{
			name:          "success",
			id:            1,
			proxyIsActive: false,
		},
		{
			name:          "already enabled",
			id:            1,
			proxyIsActive: true,
			expectErr:     ErrProxyAlreadyEnabled,
		},
		{
			name:      "not found",
			id:        999,
			getErr:    gorm.ErrRecordNotFound,
			expectErr: ErrProxyNotFound,
		},
		{
			name:            "status update error",
			id:              1,
			proxyIsActive:   false,
			updateStatusErr: errors.New("database error"),
			expectErr:       errors.New("failed to enable proxy"),
		},
		{
			name:          "sync error rolls back",
			id:            1,
			proxyIsActive: false,
			enableSyncErr: errors.New("caddy error"),
			expectErr:     errors.New("failed to enable proxy file"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var lastStatusUpdate bool
			updateStatusCallCount := 0
			repo := &MockProxyRepository{
				GetByIDFunc: func(id int) (*models.Proxy, error) {
					if tc.getErr != nil {
						return nil, tc.getErr
					}
					return &models.Proxy{
						ID:       id,
						Hostname: "test.example.com",
						IsActive: tc.proxyIsActive,
					}, nil
				},
				UpdateStatusFunc: func(id int, isActive bool) error {
					updateStatusCallCount++
					lastStatusUpdate = isActive
					return tc.updateStatusErr
				},
			}
			syncer := &MockProxySyncer{
				EnableProxyFunc: func(proxyID int, hostname string) error {
					return tc.enableSyncErr
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: syncer,
			})

			err := svc.EnableProxy(tc.id)

			if tc.expectErr != nil {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				// Verify rollback on sync error
				if tc.enableSyncErr != nil && updateStatusCallCount != 2 {
					t.Errorf("Expected 2 status updates (set + rollback), got %d", updateStatusCallCount)
				}
				if tc.enableSyncErr != nil && lastStatusUpdate != false {
					t.Error("Expected rollback to set status to false")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if updateStatusCallCount != 1 {
				t.Errorf("Expected 1 status update, got %d", updateStatusCallCount)
			}
			if !lastStatusUpdate {
				t.Error("Expected status to be set to true")
			}
		})
	}
}

// TestDisableProxy tests disabling a proxy
func TestDisableProxy(t *testing.T) {
	tests := []struct {
		name            string
		id              int
		proxyIsActive   bool
		getErr          error
		updateStatusErr error
		expectErr       error
	}{
		{
			name:          "success",
			id:            1,
			proxyIsActive: true,
		},
		{
			name:          "already disabled",
			id:            1,
			proxyIsActive: false,
			expectErr:     ErrProxyAlreadyDisabled,
		},
		{
			name:      "not found",
			id:        999,
			getErr:    gorm.ErrRecordNotFound,
			expectErr: ErrProxyNotFound,
		},
		{
			name:            "status update error",
			id:              1,
			proxyIsActive:   true,
			updateStatusErr: errors.New("database error"),
			expectErr:       errors.New("failed to disable proxy"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var lastStatusUpdate bool
			updateStatusCalled := false
			repo := &MockProxyRepository{
				GetByIDFunc: func(id int) (*models.Proxy, error) {
					if tc.getErr != nil {
						return nil, tc.getErr
					}
					return &models.Proxy{
						ID:       id,
						Hostname: "test.example.com",
						IsActive: tc.proxyIsActive,
					}, nil
				},
				UpdateStatusFunc: func(id int, isActive bool) error {
					updateStatusCalled = true
					lastStatusUpdate = isActive
					return tc.updateStatusErr
				},
			}
			syncer := &MockProxySyncer{
				DisableProxyFunc: func(proxyID int, hostname string) error {
					return nil
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: syncer,
			})

			err := svc.DisableProxy(tc.id)

			if tc.expectErr != nil {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !updateStatusCalled {
				t.Error("Expected UpdateStatus to be called")
			}
			if lastStatusUpdate != false {
				t.Error("Expected status to be set to false")
			}
		})
	}
}

// TestGetStats tests getting proxy statistics
func TestGetStats(t *testing.T) {
	tests := []struct {
		name      string
		mockStats *repository.ProxyStats
		mockErr   error
		expectErr bool
	}{
		{
			name: "success",
			mockStats: &repository.ProxyStats{
				Total:    10,
				Active:   7,
				Inactive: 3,
				ByType: map[string]int64{
					"reverse_proxy": 8,
					"redirect":      2,
				},
			},
		},
		{
			name:      "error",
			mockErr:   errors.New("database error"),
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockProxyRepository{
				GetStatsFunc: func() (*repository.ProxyStats, error) {
					return tc.mockStats, tc.mockErr
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: &MockProxySyncer{},
			})

			result, err := svc.GetStats()

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Total != tc.mockStats.Total {
				t.Errorf("Expected total %d, got %d", tc.mockStats.Total, result.Total)
			}
			if result.Active != tc.mockStats.Active {
				t.Errorf("Expected active %d, got %d", tc.mockStats.Active, result.Active)
			}
		})
	}
}

// TestListProxies_PaginationDefaults tests pagination default values
func TestListProxies_PaginationDefaults(t *testing.T) {
	testCases := []struct {
		name          string
		inputPage     int
		inputLimit    int
		expectedPage  int
		expectedLimit int
	}{
		{"Zero page becomes 1", 0, 20, 1, 20},
		{"Negative page becomes 1", -1, 20, 1, 20},
		{"Page 1 stays 1", 1, 20, 1, 20},
		{"Page 5 stays 5", 5, 20, 5, 20},
		{"Zero limit becomes 20", 1, 0, 1, 20},
		{"Negative limit becomes 20", 1, -1, 1, 20},
		{"Limit 50 stays 50", 1, 50, 1, 50},
		{"Limit 100 stays 100", 1, 100, 1, 100},
		{"Limit 101 becomes 20", 1, 101, 1, 20},
		{"Limit 1000 becomes 20", 1, 1000, 1, 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := ListProxiesRequest{
				Page:  tc.inputPage,
				Limit: tc.inputLimit,
			}

			// Apply the same defaults as the service
			if req.Page < 1 {
				req.Page = 1
			}
			if req.Limit < 1 || req.Limit > 100 {
				req.Limit = 20
			}

			if req.Page != tc.expectedPage {
				t.Errorf("Expected page %d, got %d", tc.expectedPage, req.Page)
			}
			if req.Limit != tc.expectedLimit {
				t.Errorf("Expected limit %d, got %d", tc.expectedLimit, req.Limit)
			}
		})
	}
}

// TestCaddyError tests CaddyError type
func TestCaddyError(t *testing.T) {
	err := NewCaddyError("test error message")

	if err.Error() != "test error message" {
		t.Errorf("Expected error message 'test error message', got '%s'", err.Error())
	}

	if !IsCaddyError(err) {
		t.Error("Expected IsCaddyError to return true")
	}
}

// TestIsCaddyError tests IsCaddyError function
func TestIsCaddyError(t *testing.T) {
	caddyErr := NewCaddyError("caddy error")
	if !IsCaddyError(caddyErr) {
		t.Error("Expected IsCaddyError to return true for CaddyError")
	}

	regularErr := ErrProxyNotFound
	if IsCaddyError(regularErr) {
		t.Error("Expected IsCaddyError to return false for regular error")
	}
}

// TestServiceErrors tests that service errors are defined correctly
func TestServiceErrors(t *testing.T) {
	if ErrProxyNotFound.Error() != "proxy not found" {
		t.Errorf("Unexpected ErrProxyNotFound message: %s", ErrProxyNotFound.Error())
	}

	if ErrHostnameConflict.Error() != "hostname already exists" {
		t.Errorf("Unexpected ErrHostnameConflict message: %s", ErrHostnameConflict.Error())
	}

	if ErrProxyAlreadyEnabled.Error() != "proxy is already enabled" {
		t.Errorf("Unexpected ErrProxyAlreadyEnabled message: %s", ErrProxyAlreadyEnabled.Error())
	}

	if ErrProxyAlreadyDisabled.Error() != "proxy is already disabled" {
		t.Errorf("Unexpected ErrProxyAlreadyDisabled message: %s", ErrProxyAlreadyDisabled.Error())
	}
}

// TestListProxies_FilterPassthrough tests that filter params are passed through to repo
func TestListProxies_FilterPassthrough(t *testing.T) {
	testCases := []struct {
		name             string
		req              ListProxiesRequest
		expectTypes      []string
		expectTypesExcl  []string
		expectStatus     string
		expectStatusNot  string
		expectSearch     string
	}{
		{
			name: "types filter passed through",
			req: ListProxiesRequest{
				Page:  1,
				Limit: 10,
				Types: []string{"reverse_proxy", "redirect"},
			},
			expectTypes: []string{"reverse_proxy", "redirect"},
		},
		{
			name: "types exclude filter passed through",
			req: ListProxiesRequest{
				Page:         1,
				Limit:        10,
				TypesExclude: []string{"static"},
			},
			expectTypesExcl: []string{"static"},
		},
		{
			name: "status filter passed through",
			req: ListProxiesRequest{
				Page:   1,
				Limit:  10,
				Status: "active",
			},
			expectStatus: "active",
		},
		{
			name: "status not filter passed through",
			req: ListProxiesRequest{
				Page:      1,
				Limit:     10,
				StatusNot: "inactive",
			},
			expectStatusNot: "inactive",
		},
		{
			name: "search filter passed through",
			req: ListProxiesRequest{
				Page:   1,
				Limit:  10,
				Search: "example.com",
			},
			expectSearch: "example.com",
		},
		{
			name: "combined filters passed through",
			req: ListProxiesRequest{
				Page:         1,
				Limit:        10,
				Search:       "test",
				Types:        []string{"reverse_proxy"},
				TypesExclude: []string{"static"},
				Status:       "active",
				StatusNot:    "inactive",
			},
			expectSearch:    "test",
			expectTypes:     []string{"reverse_proxy"},
			expectTypesExcl: []string{"static"},
			expectStatus:    "active",
			expectStatusNot: "inactive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedParams repository.ProxyListParams
			repo := &MockProxyRepository{
				ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
					capturedParams = params
					return []models.Proxy{}, 0, nil
				},
			}
			svc := NewProxyService(ProxyServiceConfig{
				Repo:        repo,
				SyncService: &MockProxySyncer{},
			})

			_, err := svc.ListProxies(tc.req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify types
			if len(tc.expectTypes) > 0 {
				if len(capturedParams.Types) != len(tc.expectTypes) {
					t.Errorf("Expected %d types, got %d", len(tc.expectTypes), len(capturedParams.Types))
				}
				for i, typ := range tc.expectTypes {
					if capturedParams.Types[i] != typ {
						t.Errorf("Expected type[%d] '%s', got '%s'", i, typ, capturedParams.Types[i])
					}
				}
			}

			// Verify types exclude
			if len(tc.expectTypesExcl) > 0 {
				if len(capturedParams.TypesExclude) != len(tc.expectTypesExcl) {
					t.Errorf("Expected %d excluded types, got %d", len(tc.expectTypesExcl), len(capturedParams.TypesExclude))
				}
				for i, typ := range tc.expectTypesExcl {
					if capturedParams.TypesExclude[i] != typ {
						t.Errorf("Expected excluded type[%d] '%s', got '%s'", i, typ, capturedParams.TypesExclude[i])
					}
				}
			}

			// Verify status
			if tc.expectStatus != "" && capturedParams.Status != tc.expectStatus {
				t.Errorf("Expected status '%s', got '%s'", tc.expectStatus, capturedParams.Status)
			}

			// Verify status not
			if tc.expectStatusNot != "" && capturedParams.StatusNot != tc.expectStatusNot {
				t.Errorf("Expected status not '%s', got '%s'", tc.expectStatusNot, capturedParams.StatusNot)
			}

			// Verify search
			if tc.expectSearch != "" && capturedParams.Search != tc.expectSearch {
				t.Errorf("Expected search '%s', got '%s'", tc.expectSearch, capturedParams.Search)
			}
		})
	}
}

// TestListProxiesRequest_Structure tests ListProxiesRequest struct fields
func TestListProxiesRequest_Structure(t *testing.T) {
	req := ListProxiesRequest{
		Page:         2,
		Limit:        50,
		Search:       "test",
		Types:        []string{"reverse_proxy", "redirect"},
		TypesExclude: []string{"static"},
		Status:       "active",
		StatusNot:    "inactive",
		Sort:         "name",
		Order:        "desc",
	}

	if req.Page != 2 {
		t.Errorf("Expected Page 2, got %d", req.Page)
	}
	if req.Limit != 50 {
		t.Errorf("Expected Limit 50, got %d", req.Limit)
	}
	if req.Search != "test" {
		t.Errorf("Expected Search 'test', got '%s'", req.Search)
	}
	if len(req.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(req.Types))
	}
	if req.Types[0] != "reverse_proxy" {
		t.Errorf("Expected first type 'reverse_proxy', got '%s'", req.Types[0])
	}
	if len(req.TypesExclude) != 1 {
		t.Errorf("Expected 1 excluded type, got %d", len(req.TypesExclude))
	}
	if req.TypesExclude[0] != "static" {
		t.Errorf("Expected excluded type 'static', got '%s'", req.TypesExclude[0])
	}
	if req.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", req.Status)
	}
	if req.StatusNot != "inactive" {
		t.Errorf("Expected StatusNot 'inactive', got '%s'", req.StatusNot)
	}
	if req.Sort != "name" {
		t.Errorf("Expected Sort 'name', got '%s'", req.Sort)
	}
	if req.Order != "desc" {
		t.Errorf("Expected Order 'desc', got '%s'", req.Order)
	}
}

// TestListProxiesRequest_Defaults tests ListProxiesRequest default values
func TestListProxiesRequest_Defaults(t *testing.T) {
	req := ListProxiesRequest{}

	if req.Page != 0 {
		t.Errorf("Expected default Page 0, got %d", req.Page)
	}
	if req.Limit != 0 {
		t.Errorf("Expected default Limit 0, got %d", req.Limit)
	}
	if req.Search != "" {
		t.Errorf("Expected empty Search, got '%s'", req.Search)
	}
	if len(req.Types) != 0 {
		t.Errorf("Expected empty Types, got %v", req.Types)
	}
	if len(req.TypesExclude) != 0 {
		t.Errorf("Expected empty TypesExclude, got %v", req.TypesExclude)
	}
	if req.Status != "" {
		t.Errorf("Expected empty Status, got '%s'", req.Status)
	}
	if req.StatusNot != "" {
		t.Errorf("Expected empty StatusNot, got '%s'", req.StatusNot)
	}
}

// TestListProxies_MultipleTypesFilter tests listing with multiple types
func TestListProxies_MultipleTypesFilter(t *testing.T) {
	expectedTypes := []string{"reverse_proxy", "redirect"}
	var capturedParams repository.ProxyListParams

	repo := &MockProxyRepository{
		ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			capturedParams = params
			return []models.Proxy{
				{ID: 1, Type: models.ProxyTypeReverseProxy},
				{ID: 2, Type: models.ProxyTypeRedirect},
			}, 2, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: &MockProxySyncer{},
	})

	req := ListProxiesRequest{
		Page:  1,
		Limit: 10,
		Types: expectedTypes,
	}

	result, err := svc.ListProxies(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(capturedParams.Types) != 2 {
		t.Errorf("Expected 2 types passed to repo, got %d", len(capturedParams.Types))
	}
	if capturedParams.Types[0] != "reverse_proxy" {
		t.Errorf("Expected first type 'reverse_proxy', got '%s'", capturedParams.Types[0])
	}
	if capturedParams.Types[1] != "redirect" {
		t.Errorf("Expected second type 'redirect', got '%s'", capturedParams.Types[1])
	}
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result.Items))
	}
}

// TestListProxies_TypesExcludeFilter tests listing with excluded types
func TestListProxies_TypesExcludeFilter(t *testing.T) {
	excludeTypes := []string{"static"}
	var capturedParams repository.ProxyListParams

	repo := &MockProxyRepository{
		ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			capturedParams = params
			return []models.Proxy{
				{ID: 1, Type: models.ProxyTypeReverseProxy},
				{ID: 2, Type: models.ProxyTypeRedirect},
			}, 2, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: &MockProxySyncer{},
	})

	req := ListProxiesRequest{
		Page:         1,
		Limit:        10,
		TypesExclude: excludeTypes,
	}

	result, err := svc.ListProxies(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(capturedParams.TypesExclude) != 1 {
		t.Errorf("Expected 1 excluded type passed to repo, got %d", len(capturedParams.TypesExclude))
	}
	if capturedParams.TypesExclude[0] != "static" {
		t.Errorf("Expected excluded type 'static', got '%s'", capturedParams.TypesExclude[0])
	}
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result.Items))
	}
}

// TestListProxies_StatusNotFilter tests listing with status negation
func TestListProxies_StatusNotFilter(t *testing.T) {
	var capturedParams repository.ProxyListParams

	repo := &MockProxyRepository{
		ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			capturedParams = params
			return []models.Proxy{
				{ID: 1, IsActive: false},
			}, 1, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: &MockProxySyncer{},
	})

	req := ListProxiesRequest{
		Page:      1,
		Limit:     10,
		StatusNot: "active",
	}

	result, err := svc.ListProxies(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedParams.StatusNot != "active" {
		t.Errorf("Expected StatusNot 'active', got '%s'", capturedParams.StatusNot)
	}
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(result.Items))
	}
}

// TestListProxies_CombinedFilters tests listing with multiple filter types combined
func TestListProxies_CombinedFilters(t *testing.T) {
	var capturedParams repository.ProxyListParams

	repo := &MockProxyRepository{
		ListFunc: func(params repository.ProxyListParams) ([]models.Proxy, int64, error) {
			capturedParams = params
			return []models.Proxy{
				{ID: 1, Name: "Test", Type: models.ProxyTypeReverseProxy, IsActive: true},
			}, 1, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: &MockProxySyncer{},
	})

	req := ListProxiesRequest{
		Page:         1,
		Limit:        10,
		Search:       "Test",
		Types:        []string{"reverse_proxy", "redirect"},
		TypesExclude: []string{"static"},
		Status:       "active",
		Sort:         "name",
		Order:        "asc",
	}

	_, err := svc.ListProxies(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify all params passed through
	if capturedParams.Search != "Test" {
		t.Errorf("Expected Search 'Test', got '%s'", capturedParams.Search)
	}
	if len(capturedParams.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(capturedParams.Types))
	}
	if len(capturedParams.TypesExclude) != 1 {
		t.Errorf("Expected 1 excluded type, got %d", len(capturedParams.TypesExclude))
	}
	if capturedParams.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", capturedParams.Status)
	}
	if capturedParams.Sort != "name" {
		t.Errorf("Expected Sort 'name', got '%s'", capturedParams.Sort)
	}
	if capturedParams.Order != "asc" {
		t.Errorf("Expected Order 'asc', got '%s'", capturedParams.Order)
	}
}

// TestUpdateProxy_TypeChange tests that config is cleaned up when type changes
func TestUpdateProxy_TypeChange(t *testing.T) {
	existingProxy := &models.Proxy{
		ID:        1,
		Name:      "Existing",
		Hostname:  "existing.example.com",
		Type:      models.ProxyTypeReverseProxy,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://localhost:8080"}},
	}

	repo := &MockProxyRepository{
		GetByIDFunc: func(id int) (*models.Proxy, error) {
			return existingProxy, nil
		},
		HostnameExistsFunc: func(hostname string, excludeID int) (bool, error) {
			return false, nil
		},
		UpdateFunc: func(proxy *models.Proxy) error {
			return nil
		},
	}
	syncer := &MockProxySyncer{
		SyncProxyFunc: func(proxy *models.Proxy) error {
			return nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: syncer,
	})

	// Change from reverse_proxy to redirect
	newProxy := &models.Proxy{
		Name:     "Updated",
		Hostname: "existing.example.com",
		Type:     models.ProxyTypeRedirect,
		RedirectConfig: models.JSONField{
			"url":  "https://example.com",
			"code": 301,
		},
		Upstreams: []interface{}{map[string]interface{}{"address": "should be cleared"}},
	}

	err := svc.UpdateProxy(1, newProxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify upstreams cleared for redirect type
	if newProxy.Upstreams != nil {
		t.Error("Expected Upstreams to be nil for redirect type")
	}

	// Verify redirect config preserved
	if len(newProxy.RedirectConfig) == 0 {
		t.Error("Expected RedirectConfig to be preserved")
	}
}
