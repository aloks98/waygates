package repository

import (
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// =============================================================================
// L4ProxyRepository Integration Tests
// =============================================================================

// CreateTestL4Proxy creates a test L4 proxy with the given parameters
func CreateTestL4Proxy(t *testing.T, db *gorm.DB, userID int, name string, port int, protocol string) *models.L4Proxy {
	t.Helper()
	proxy := &models.L4Proxy{
		Name:       name,
		ListenPort: port,
		Protocol:   protocol,
		IsActive:   true,
		CreatedBy:  userID,
	}
	if err := db.Create(proxy).Error; err != nil {
		t.Fatalf("Failed to create test L4 proxy: %v", err)
	}
	return proxy
}

// CreateTestL4ProxyWithRoutes creates a test L4 proxy with routes
func CreateTestL4ProxyWithRoutes(t *testing.T, db *gorm.DB, userID int, name string, port int, protocol string, routes []models.L4Route) *models.L4Proxy {
	t.Helper()
	proxy := &models.L4Proxy{
		Name:       name,
		ListenPort: port,
		Protocol:   protocol,
		IsActive:   true,
		CreatedBy:  userID,
		Routes:     routes,
	}
	if err := db.Create(proxy).Error; err != nil {
		t.Fatalf("Failed to create test L4 proxy with routes: %v", err)
	}
	return proxy
}

func TestL4ProxyRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewL4ProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success_TCPProxy", func(t *testing.T) {
		proxy := &models.L4Proxy{
			Name:       "Test TCP Proxy",
			ListenPort: 5432,
			Protocol:   models.L4ProtocolTCP,
			IsActive:   true,
			CreatedBy:  user.ID,
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotZero(t, proxy.ID)
		assert.NotZero(t, proxy.CreatedAt)
		assert.NotZero(t, proxy.UpdatedAt)
	})

	t.Run("Success_UDPProxy", func(t *testing.T) {
		proxy := &models.L4Proxy{
			Name:       "Test UDP Proxy",
			ListenPort: 5353,
			Protocol:   models.L4ProtocolUDP,
			IsActive:   true,
			CreatedBy:  user.ID,
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotZero(t, proxy.ID)
		assert.Equal(t, models.L4ProtocolUDP, proxy.Protocol)
	})

	t.Run("Success_WithDescription", func(t *testing.T) {
		description := "PostgreSQL database proxy"
		proxy := &models.L4Proxy{
			Name:        "Database Proxy",
			Description: &description,
			ListenPort:  5433,
			Protocol:    models.L4ProtocolTCP,
			IsActive:    true,
			CreatedBy:   user.ID,
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotNil(t, proxy.Description)
		assert.Equal(t, description, *proxy.Description)
	})

	t.Run("Success_WithRoutes", func(t *testing.T) {
		weight := 100
		routes := []models.L4Route{
			{
				Priority:            0,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams: models.L4UpstreamSlice{
					{Host: "backend1.local", Port: 5432, Weight: &weight},
					{Host: "backend2.local", Port: 5432, Weight: &weight},
				},
			},
		}

		proxy := &models.L4Proxy{
			Name:       "Proxy With Routes",
			ListenPort: 5434,
			Protocol:   models.L4ProtocolTCP,
			IsActive:   true,
			CreatedBy:  user.ID,
			Routes:     routes,
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotZero(t, proxy.ID)
		assert.Len(t, proxy.Routes, 1)
		assert.NotZero(t, proxy.Routes[0].ID)
		assert.Equal(t, proxy.ID, proxy.Routes[0].L4ProxyID)
	})

	t.Run("Success_WithMultipleRoutes", func(t *testing.T) {
		weight := 100
		routes := []models.L4Route{
			{
				Priority:            0,
				MatcherType:         models.L4MatcherTLS,
				SNIHostnames:        pq.StringArray{"db1.example.com"},
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams: models.L4UpstreamSlice{
					{Host: "db1.local", Port: 5432, Weight: &weight},
				},
			},
			{
				Priority:            10,
				MatcherType:         models.L4MatcherTLS,
				SNIHostnames:        pq.StringArray{"db2.example.com"},
				LoadBalancingPolicy: models.L4LoadBalancingLeastConn,
				TLSPassthrough:      true,
				Upstreams: models.L4UpstreamSlice{
					{Host: "db2.local", Port: 5432, Weight: &weight},
				},
			},
		}

		proxy := &models.L4Proxy{
			Name:       "Multi Route Proxy",
			ListenPort: 5435,
			Protocol:   models.L4ProtocolTCP,
			IsActive:   true,
			CreatedBy:  user.ID,
			Routes:     routes,
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.Len(t, proxy.Routes, 2)
	})

	t.Run("Error_DuplicatePortAndProtocol", func(t *testing.T) {
		proxy1 := &models.L4Proxy{
			Name:       "First TCP Proxy",
			ListenPort: 6000,
			Protocol:   models.L4ProtocolTCP,
			IsActive:   true,
			CreatedBy:  user.ID,
		}
		err := repo.Create(proxy1)
		require.NoError(t, err)

		proxy2 := &models.L4Proxy{
			Name:       "Second TCP Proxy",
			ListenPort: 6000, // Same port
			Protocol:   models.L4ProtocolTCP,
			IsActive:   true,
			CreatedBy:  user.ID,
		}
		err = repo.Create(proxy2)
		require.Error(t, err, "Should fail with duplicate port+protocol")
	})

	t.Run("Success_SamePortDifferentProtocol", func(t *testing.T) {
		proxy1 := &models.L4Proxy{
			Name:       "TCP on 7000",
			ListenPort: 7000,
			Protocol:   models.L4ProtocolTCP,
			IsActive:   true,
			CreatedBy:  user.ID,
		}
		err := repo.Create(proxy1)
		require.NoError(t, err)

		proxy2 := &models.L4Proxy{
			Name:       "UDP on 7000",
			ListenPort: 7000, // Same port but different protocol
			Protocol:   models.L4ProtocolUDP,
			IsActive:   true,
			CreatedBy:  user.ID,
		}
		err = repo.Create(proxy2)
		require.NoError(t, err, "Should succeed with same port but different protocol")
	})
}

func TestL4ProxyRepository_GetByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewL4ProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "GetByID Test", 8000, models.L4ProtocolTCP)

		fetched, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		assert.Equal(t, proxy.ID, fetched.ID)
		assert.Equal(t, "GetByID Test", fetched.Name)
		assert.Equal(t, 8000, fetched.ListenPort)
		assert.Equal(t, models.L4ProtocolTCP, fetched.Protocol)
	})

	t.Run("Success_PreloadsCreator", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Creator Preload Test", 8001, models.L4ProtocolTCP)

		fetched, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		assert.NotNil(t, fetched.Creator)
		assert.Equal(t, user.ID, fetched.Creator.ID)
	})

	t.Run("Success_PreloadsRoutesOrderedByPriority", func(t *testing.T) {
		weight := 100
		routes := []models.L4Route{
			{
				Priority:            20, // Higher priority number = lower precedence
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams:           models.L4UpstreamSlice{{Host: "third.local", Port: 80, Weight: &weight}},
			},
			{
				Priority:            0, // Highest precedence
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams:           models.L4UpstreamSlice{{Host: "first.local", Port: 80, Weight: &weight}},
			},
			{
				Priority:            10,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams:           models.L4UpstreamSlice{{Host: "second.local", Port: 80, Weight: &weight}},
			},
		}

		proxy := CreateTestL4ProxyWithRoutes(t, tdb.DB, user.ID, "Routes Order Test", 8002, models.L4ProtocolTCP, routes)

		fetched, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		require.Len(t, fetched.Routes, 3)

		// Routes should be ordered by priority ASC
		assert.Equal(t, 0, fetched.Routes[0].Priority)
		assert.Equal(t, "first.local", fetched.Routes[0].Upstreams[0].Host)
		assert.Equal(t, 10, fetched.Routes[1].Priority)
		assert.Equal(t, "second.local", fetched.Routes[1].Upstreams[0].Host)
		assert.Equal(t, 20, fetched.Routes[2].Priority)
		assert.Equal(t, "third.local", fetched.Routes[2].Upstreams[0].Host)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(999999)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Error_ZeroID", func(t *testing.T) {
		_, err := repo.GetByID(0)
		require.Error(t, err)
	})

	t.Run("Error_NegativeID", func(t *testing.T) {
		_, err := repo.GetByID(-1)
		require.Error(t, err)
	})
}

func TestL4ProxyRepository_GetByIDs(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewL4ProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)
	weight := 1

	routes := []models.L4Route{
		{
			Priority:            10,
			MatcherType:         models.L4MatcherAny,
			LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
			TLSPassthrough:      true,
			Upstreams:           models.L4UpstreamSlice{{Host: "b.local", Port: 80, Weight: &weight}},
		},
		{
			Priority:            0,
			MatcherType:         models.L4MatcherAny,
			LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
			TLSPassthrough:      true,
			Upstreams:           models.L4UpstreamSlice{{Host: "a.local", Port: 80, Weight: &weight}},
		},
	}
	p1 := CreateTestL4ProxyWithRoutes(t, tdb.DB, user.ID, "Batch 1", 9100, models.L4ProtocolTCP, routes)
	p2 := CreateTestL4Proxy(t, tdb.DB, user.ID, "Batch 2", 9101, models.L4ProtocolTCP)
	p3 := CreateTestL4Proxy(t, tdb.DB, user.ID, "Batch 3", 9102, models.L4ProtocolUDP)

	t.Run("ReturnsRequestedWithRoutesInOneQuery", func(t *testing.T) {
		got, err := repo.GetByIDs([]int{p1.ID, p3.ID})
		require.NoError(t, err)
		require.Len(t, got, 2)

		byID := map[int]models.L4Proxy{}
		for _, p := range got {
			byID[p.ID] = p
		}
		require.Contains(t, byID, p1.ID)
		require.Contains(t, byID, p3.ID)
		assert.NotContains(t, byID, p2.ID, "p2 was not requested")

		// Routes are preloaded and ordered by priority ASC.
		fetched1 := byID[p1.ID]
		require.Len(t, fetched1.Routes, 2)
		assert.Equal(t, 0, fetched1.Routes[0].Priority)
		assert.Equal(t, "a.local", fetched1.Routes[0].Upstreams[0].Host)
		assert.Equal(t, 10, fetched1.Routes[1].Priority)
	})

	t.Run("SkipsMissing", func(t *testing.T) {
		got, err := repo.GetByIDs([]int{p1.ID, 999999, p2.ID})
		require.NoError(t, err)
		assert.Len(t, got, 2, "missing id is simply absent, not an error")
	})

	t.Run("EmptyInput", func(t *testing.T) {
		got, err := repo.GetByIDs([]int{})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestL4ProxyRepository_GetByPort(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewL4ProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success_TCP", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Port Test TCP", 9000, models.L4ProtocolTCP)

		fetched, err := repo.GetByPort(9000, models.L4ProtocolTCP)
		require.NoError(t, err)
		assert.Equal(t, proxy.ID, fetched.ID)
		assert.Equal(t, 9000, fetched.ListenPort)
	})

	t.Run("Success_UDP", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Port Test UDP", 9001, models.L4ProtocolUDP)

		fetched, err := repo.GetByPort(9001, models.L4ProtocolUDP)
		require.NoError(t, err)
		assert.Equal(t, proxy.ID, fetched.ID)
		assert.Equal(t, models.L4ProtocolUDP, fetched.Protocol)
	})

	t.Run("Success_SamePortDifferentProtocol", func(t *testing.T) {
		CreateTestL4Proxy(t, tdb.DB, user.ID, "TCP 9002", 9002, models.L4ProtocolTCP)
		CreateTestL4Proxy(t, tdb.DB, user.ID, "UDP 9002", 9002, models.L4ProtocolUDP)

		tcpProxy, err := repo.GetByPort(9002, models.L4ProtocolTCP)
		require.NoError(t, err)
		assert.Equal(t, "TCP 9002", tcpProxy.Name)

		udpProxy, err := repo.GetByPort(9002, models.L4ProtocolUDP)
		require.NoError(t, err)
		assert.Equal(t, "UDP 9002", udpProxy.Name)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		_, err := repo.GetByPort(99999, models.L4ProtocolTCP)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Error_WrongProtocol", func(t *testing.T) {
		CreateTestL4Proxy(t, tdb.DB, user.ID, "Only TCP", 9003, models.L4ProtocolTCP)

		_, err := repo.GetByPort(9003, models.L4ProtocolUDP)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Success_CaseInsensitiveProtocol", func(t *testing.T) {
		CreateTestL4Proxy(t, tdb.DB, user.ID, "Case Test", 9004, models.L4ProtocolTCP)

		// Should match with uppercase protocol
		fetched, err := repo.GetByPort(9004, "TCP")
		require.NoError(t, err)
		assert.Equal(t, "Case Test", fetched.Name)
	})
}

func TestL4ProxyRepository_List(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewL4ProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	// Create test data
	CreateTestL4Proxy(t, tdb.DB, user.ID, "Alpha Database", 10001, models.L4ProtocolTCP)
	CreateTestL4Proxy(t, tdb.DB, user.ID, "Beta Cache", 10002, models.L4ProtocolTCP)
	CreateTestL4Proxy(t, tdb.DB, user.ID, "Gamma DNS", 10003, models.L4ProtocolUDP)

	inactiveProxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Delta Inactive", 10004, models.L4ProtocolTCP)
	inactiveProxy.IsActive = false
	_ = tdb.DB.Save(inactiveProxy)

	t.Run("Success_BasicPagination", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.GreaterOrEqual(t, len(proxies), 4)
	})

	t.Run("Success_Pagination_Page1", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.Equal(t, 2, len(proxies))
	})

	t.Run("Success_Pagination_Page2", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:  2,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.GreaterOrEqual(t, len(proxies), 2)
	})

	t.Run("Success_FilterByProtocolTCP", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:     1,
			Limit:    10,
			Protocol: models.L4ProtocolTCP,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		for _, p := range proxies {
			assert.Equal(t, models.L4ProtocolTCP, p.Protocol)
		}
	})

	t.Run("Success_FilterByProtocolUDP", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:     1,
			Limit:    10,
			Protocol: models.L4ProtocolUDP,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		for _, p := range proxies {
			assert.Equal(t, models.L4ProtocolUDP, p.Protocol)
		}
	})

	t.Run("Success_FilterByIsActiveTrue", func(t *testing.T) {
		isActive := true
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:     1,
			Limit:    10,
			IsActive: &isActive,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		for _, p := range proxies {
			assert.True(t, p.IsActive)
		}
	})

	t.Run("Success_FilterByIsActiveFalse", func(t *testing.T) {
		isActive := false
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:     1,
			Limit:    10,
			IsActive: &isActive,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		for _, p := range proxies {
			assert.False(t, p.IsActive)
		}
	})

	t.Run("Success_Search", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "Alpha",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		for _, p := range proxies {
			assert.True(t, strings.Contains(strings.ToLower(p.Name), "alpha") ||
				(p.Description != nil && strings.Contains(strings.ToLower(*p.Description), "alpha")))
		}
	})

	t.Run("Success_SearchByDescription", func(t *testing.T) {
		description := "Unique description for search"
		proxy := &models.L4Proxy{
			Name:        "Searchable Proxy",
			Description: &description,
			ListenPort:  10005,
			Protocol:    models.L4ProtocolTCP,
			IsActive:    true,
			CreatedBy:   user.ID,
		}
		_ = repo.Create(proxy)

		_, total, err := repo.List(L4ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "Unique description",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
	})

	t.Run("Success_SortByNameASC", func(t *testing.T) {
		proxies, _, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "name",
			Order: "asc",
		})
		require.NoError(t, err)
		if len(proxies) >= 2 {
			assert.LessOrEqual(t, proxies[0].Name, proxies[1].Name)
		}
	})

	t.Run("Success_SortByNameDESC", func(t *testing.T) {
		proxies, _, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "name",
			Order: "desc",
		})
		require.NoError(t, err)
		if len(proxies) >= 2 {
			assert.GreaterOrEqual(t, proxies[0].Name, proxies[1].Name)
		}
	})

	t.Run("Success_SortByListenPort", func(t *testing.T) {
		proxies, _, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "listen_port",
			Order: "asc",
		})
		require.NoError(t, err)
		if len(proxies) >= 2 {
			assert.LessOrEqual(t, proxies[0].ListenPort, proxies[1].ListenPort)
		}
	})

	t.Run("Success_InvalidSortUsesDefault", func(t *testing.T) {
		proxies, _, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "invalid_field",
			Order: "asc",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, proxies)
	})

	t.Run("Success_InvalidOrderUsesDefault", func(t *testing.T) {
		proxies, _, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "name",
			Order: "INVALID",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, proxies)
	})

	t.Run("Success_CombinedFilters", func(t *testing.T) {
		isActive := true
		proxies, _, err := repo.List(L4ProxyListParams{
			Page:     1,
			Limit:    10,
			Protocol: models.L4ProtocolTCP,
			IsActive: &isActive,
			Search:   "Database",
			Sort:     "name",
			Order:    "asc",
		})
		require.NoError(t, err)
		for _, p := range proxies {
			assert.Equal(t, models.L4ProtocolTCP, p.Protocol)
			assert.True(t, p.IsActive)
		}
	})

	t.Run("Success_EmptyResult", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "NonExistentSearchTerm12345",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, proxies)
	})

	t.Run("Success_LargePageNumber", func(t *testing.T) {
		proxies, total, err := repo.List(L4ProxyListParams{
			Page:  999,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.Empty(t, proxies) // Page too high
	})

	t.Run("Success_PreloadsRoutes", func(t *testing.T) {
		weight := 100
		routes := []models.L4Route{
			{
				Priority:            0,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams:           models.L4UpstreamSlice{{Host: "backend.local", Port: 80, Weight: &weight}},
			},
		}
		CreateTestL4ProxyWithRoutes(t, tdb.DB, user.ID, "List Routes Test", 10006, models.L4ProtocolTCP, routes)

		proxies, _, err := repo.List(L4ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "List Routes Test",
		})
		require.NoError(t, err)
		require.Len(t, proxies, 1)
		assert.Len(t, proxies[0].Routes, 1)
	})

	t.Run("Success_PreloadsCreator", func(t *testing.T) {
		proxies, _, err := repo.List(L4ProxyListParams{
			Page:  1,
			Limit: 10,
		})
		require.NoError(t, err)
		require.NotEmpty(t, proxies)
		assert.NotNil(t, proxies[0].Creator)
	})
}

func TestL4ProxyRepository_Update(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewL4ProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success_UpdateName", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Original Name", 11001, models.L4ProtocolTCP)

		proxy.Name = "Updated Name"
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.Equal(t, "Updated Name", fetched.Name)
	})

	t.Run("Success_UpdateDescription", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Description Update Test", 11002, models.L4ProtocolTCP)

		description := "Updated description"
		proxy.Description = &description
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.NotNil(t, fetched.Description)
		assert.Equal(t, "Updated description", *fetched.Description)
	})

	t.Run("Success_UpdatePort", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Port Update Test", 11003, models.L4ProtocolTCP)

		proxy.ListenPort = 11099
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.Equal(t, 11099, fetched.ListenPort)
	})

	t.Run("Success_UpdateProtocol", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Protocol Update Test", 11004, models.L4ProtocolTCP)

		proxy.Protocol = models.L4ProtocolUDP
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.Equal(t, models.L4ProtocolUDP, fetched.Protocol)
	})

	t.Run("Success_ToggleActive", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Toggle Test", 11005, models.L4ProtocolTCP)
		assert.True(t, proxy.IsActive)

		proxy.IsActive = false
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.False(t, fetched.IsActive)

		proxy.IsActive = true
		err = repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ = repo.GetByID(proxy.ID)
		assert.True(t, fetched.IsActive)
	})

	t.Run("Success_UpdateMultipleFields", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Multi Field Update", 11006, models.L4ProtocolTCP)

		proxy.Name = "All Updated"
		description := "All fields updated"
		proxy.Description = &description
		proxy.IsActive = false
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.Equal(t, "All Updated", fetched.Name)
		assert.Equal(t, "All fields updated", *fetched.Description)
		assert.False(t, fetched.IsActive)
	})

	t.Run("Error_DuplicatePortProtocolOnUpdate", func(t *testing.T) {
		// Create first proxy
		CreateTestL4Proxy(t, tdb.DB, user.ID, "Existing Proxy", 11007, models.L4ProtocolTCP)

		// Create second proxy and try to update to same port+protocol
		proxy2 := CreateTestL4Proxy(t, tdb.DB, user.ID, "Second Proxy", 11008, models.L4ProtocolTCP)
		proxy2.ListenPort = 11007 // Try to use the same port

		err := repo.Update(proxy2)
		require.Error(t, err, "Should fail due to unique constraint")
	})
}

func TestL4ProxyRepository_Delete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewL4ProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Delete Test", 12001, models.L4ProtocolTCP)

		err := repo.Delete(proxy.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(proxy.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Success_CascadeDeleteRoutes", func(t *testing.T) {
		weight := 100
		routes := []models.L4Route{
			{
				Priority:            0,
				MatcherType:         models.L4MatcherAny,
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams:           models.L4UpstreamSlice{{Host: "backend.local", Port: 80, Weight: &weight}},
			},
			{
				Priority:            10,
				MatcherType:         models.L4MatcherTLS,
				SNIHostnames:        pq.StringArray{"example.com"},
				LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
				TLSPassthrough:      true,
				Upstreams:           models.L4UpstreamSlice{{Host: "backend2.local", Port: 443, Weight: &weight}},
			},
		}

		proxy := CreateTestL4ProxyWithRoutes(t, tdb.DB, user.ID, "Cascade Delete Test", 12002, models.L4ProtocolTCP, routes)
		proxyID := proxy.ID

		// Verify routes exist before deletion
		var routeCount int64
		tdb.DB.Model(&models.L4Route{}).Where("l4_proxy_id = ?", proxyID).Count(&routeCount)
		assert.Equal(t, int64(2), routeCount)

		// Delete proxy
		err := repo.Delete(proxyID)
		require.NoError(t, err)

		// Verify routes are also deleted
		tdb.DB.Model(&models.L4Route{}).Where("l4_proxy_id = ?", proxyID).Count(&routeCount)
		assert.Equal(t, int64(0), routeCount)
	})

	t.Run("Success_NonExistent", func(t *testing.T) {
		// GORM delete doesn't error on non-existent records
		err := repo.Delete(999999)
		require.NoError(t, err)
	})

	t.Run("Success_AlreadyDeleted", func(t *testing.T) {
		proxy := CreateTestL4Proxy(t, tdb.DB, user.ID, "Double Delete", 12003, models.L4ProtocolTCP)

		_ = repo.Delete(proxy.ID)
		err := repo.Delete(proxy.ID) // Delete again
		require.NoError(t, err)
	})
}

// =============================================================================
// SQL Injection Prevention Tests
// =============================================================================

func TestL4ProxyList_SQLInjectionPrevented(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		sortInput     string
		expectedField string
		orderInput    string
		expectedOrder string
	}{
		{
			name:          "Valid sort field",
			sortInput:     "name",
			expectedField: "name",
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in sort - DROP TABLE",
			sortInput:     "name; DROP TABLE l4_proxies;--",
			expectedField: "created_at", // Should fallback to default
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in sort - UNION SELECT",
			sortInput:     "name UNION SELECT * FROM users--",
			expectedField: "created_at", // Should fallback to default
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in order - malicious order",
			sortInput:     "name",
			expectedField: "name",
			orderInput:    "ASC; DELETE FROM l4_proxies;--",
			expectedOrder: "DESC", // Should fallback to default
		},
		{
			name:          "Case insensitive sort field",
			sortInput:     "NAME",
			expectedField: "name",
			orderInput:    "DESC",
			expectedOrder: "DESC",
		},
		{
			name:          "Case insensitive order",
			sortInput:     "listen_port",
			expectedField: "listen_port",
			orderInput:    "ASC",
			expectedOrder: "ASC",
		},
		{
			name:          "Invalid sort field - fallback to default",
			sortInput:     "password", // Not in whitelist
			expectedField: "created_at",
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "Empty sort - use default",
			sortInput:     "",
			expectedField: "created_at",
			orderInput:    "",
			expectedOrder: "DESC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test sort field validation
			sortField := "created_at" // default
			if tc.sortInput != "" {
				if validField, ok := l4ProxyAllowedSortFields[strings.ToLower(tc.sortInput)]; ok {
					sortField = validField
				}
			}

			if sortField != tc.expectedField {
				t.Errorf("Sort field: expected '%s', got '%s'", tc.expectedField, sortField)
			}

			// Test sort order validation
			sortOrder := "DESC" // default
			if tc.orderInput != "" {
				if validOrder, ok := allowedSortOrders[strings.ToLower(tc.orderInput)]; ok {
					sortOrder = validOrder
				}
			}

			if sortOrder != tc.expectedOrder {
				t.Errorf("Sort order: expected '%s', got '%s'", tc.expectedOrder, sortOrder)
			}
		})
	}
}

func TestL4ProxyAllowedSortFields(t *testing.T) {
	t.Parallel()

	expectedFields := []string{"id", "name", "listen_port", "protocol", "is_active", "created_at", "updated_at"}

	for _, field := range expectedFields {
		if _, ok := l4ProxyAllowedSortFields[field]; !ok {
			t.Errorf("Expected field '%s' to be in l4ProxyAllowedSortFields", field)
		}
	}

	// Verify dangerous fields are NOT in the whitelist
	dangerousFields := []string{"password", "created_by", "secret", "token"}
	for _, field := range dangerousFields {
		if _, ok := l4ProxyAllowedSortFields[field]; ok {
			t.Errorf("Dangerous field '%s' should NOT be in l4ProxyAllowedSortFields", field)
		}
	}
}
