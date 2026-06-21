package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// =============================================================================
// ProxyRepository Integration Tests
// =============================================================================

func TestProxyRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success_ReverseProxy", func(t *testing.T) {
		proxy := &models.Proxy{
			Type:      models.ProxyTypeReverseProxy,
			Name:      "Test Reverse Proxy",
			Hostname:  "rp.example.com",
			IsActive:  true,
			CreatedBy: user.ID,
			Upstreams: []interface{}{
				map[string]interface{}{
					"host":   "backend",
					"port":   8080,
					"scheme": "http",
				},
			},
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotZero(t, proxy.ID)
		assert.NotZero(t, proxy.CreatedAt)
		assert.NotZero(t, proxy.UpdatedAt)
	})

	t.Run("Success_Redirect", func(t *testing.T) {
		proxy := &models.Proxy{
			Type:      models.ProxyTypeRedirect,
			Name:      "Test Redirect",
			Hostname:  "redirect.example.com",
			IsActive:  true,
			CreatedBy: user.ID,
			RedirectConfig: models.JSONField{
				"target":      "https://new.example.com",
				"status_code": 301,
			},
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotZero(t, proxy.ID)
	})

	t.Run("Success_Static", func(t *testing.T) {
		proxy := &models.Proxy{
			Type:      models.ProxyTypeStatic,
			Name:      "Test Static",
			Hostname:  "static.example.com",
			IsActive:  true,
			CreatedBy: user.ID,
			StaticConfig: models.JSONField{
				"root_path":  "/var/www",
				"index_file": "index.html",
			},
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotZero(t, proxy.ID)
	})

	t.Run("Success_WithAllFields", func(t *testing.T) {
		description := "Full proxy with all fields"
		proxy := &models.Proxy{
			Type:                  models.ProxyTypeReverseProxy,
			Name:                  "Full Proxy",
			Hostname:              "full.example.com",
			Description:           &description,
			SSLEnabled:            true,
			SSLForced:             true,
			IsActive:              true,
			CreatedBy:             user.ID,
			BlockExploits:         true,
			TLSInsecureSkipVerify: false,
			Upstreams: []interface{}{
				map[string]interface{}{
					"host":   "backend1",
					"port":   8080,
					"scheme": "http",
				},
				map[string]interface{}{
					"host":   "backend2",
					"port":   8080,
					"scheme": "http",
				},
			},
			LoadBalancing: models.JSONField{
				"strategy": "round_robin",
			},
			CustomHeaders: models.CustomHeaders{
				Request: map[string]string{"X-Custom-Header": "value"},
			},
		}

		err := repo.Create(proxy)
		require.NoError(t, err)
		assert.NotZero(t, proxy.ID)
	})

	t.Run("Error_DuplicateHostname", func(t *testing.T) {
		proxy1 := &models.Proxy{
			Type:      models.ProxyTypeReverseProxy,
			Name:      "First",
			Hostname:  "duplicate.example.com",
			CreatedBy: user.ID,
		}
		err := repo.Create(proxy1)
		require.NoError(t, err)

		proxy2 := &models.Proxy{
			Type:      models.ProxyTypeReverseProxy,
			Name:      "Second",
			Hostname:  "duplicate.example.com",
			CreatedBy: user.ID,
		}
		err = repo.Create(proxy2)
		require.Error(t, err, "Should fail with duplicate hostname")
	})
}

func TestProxyRepository_GetByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "GetByID Test", "getbyid.example.com", models.ProxyTypeReverseProxy)

		fetched, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		assert.Equal(t, proxy.ID, fetched.ID)
		assert.Equal(t, "GetByID Test", fetched.Name)
		assert.Equal(t, "getbyid.example.com", fetched.Hostname)
		assert.Equal(t, models.ProxyTypeReverseProxy, fetched.Type)
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

func TestProxyRepository_GetByIDs(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	p1 := CreateTestProxy(t, tdb.DB, user.ID, "Batch 1", "batch1.example.com", models.ProxyTypeReverseProxy)
	p2 := CreateTestProxy(t, tdb.DB, user.ID, "Batch 2", "batch2.example.com", models.ProxyTypeReverseProxy)
	p3 := CreateTestProxy(t, tdb.DB, user.ID, "Batch 3", "batch3.example.com", models.ProxyTypeReverseProxy)

	t.Run("ReturnsRequestedInOneQuery", func(t *testing.T) {
		got, err := repo.GetByIDs([]int{p1.ID, p3.ID})
		require.NoError(t, err)
		require.Len(t, got, 2)
		ids := map[int]bool{got[0].ID: true, got[1].ID: true}
		assert.True(t, ids[p1.ID])
		assert.True(t, ids[p3.ID])
		assert.False(t, ids[p2.ID], "p2 was not requested")
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

func TestProxyRepository_GetByHostname(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Hostname Test", "hostname-test.example.com", models.ProxyTypeReverseProxy)

		fetched, err := repo.GetByHostname("hostname-test.example.com")
		require.NoError(t, err)
		assert.Equal(t, proxy.ID, fetched.ID)
		assert.Equal(t, "Hostname Test", fetched.Name)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		_, err := repo.GetByHostname("nonexistent.example.com")
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Error_EmptyHostname", func(t *testing.T) {
		_, err := repo.GetByHostname("")
		require.Error(t, err)
	})

	t.Run("Success_CaseSensitive", func(t *testing.T) {
		// Create proxy with lowercase hostname
		CreateTestProxy(t, tdb.DB, user.ID, "Case Test", "case-test.example.com", models.ProxyTypeReverseProxy)

		// PostgreSQL is case-sensitive by default
		fetched, err := repo.GetByHostname("case-test.example.com")
		require.NoError(t, err)
		assert.Equal(t, "case-test.example.com", fetched.Hostname)
	})
}

func TestProxyRepository_Update(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success_UpdateName", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Original Name", "update-name.example.com", models.ProxyTypeReverseProxy)

		proxy.Name = "Updated Name"
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.Equal(t, "Updated Name", fetched.Name)
	})

	t.Run("Success_UpdateMultipleFields", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Multi Update", "multi-update.example.com", models.ProxyTypeReverseProxy)

		description := "Updated description"
		proxy.Description = &description
		proxy.BlockExploits = true
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.Equal(t, "Updated description", *fetched.Description)
		assert.True(t, fetched.BlockExploits)
	})

	t.Run("Success_UpdateUpstreams", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Upstreams Update", "upstreams-update.example.com", models.ProxyTypeReverseProxy)

		proxy.Upstreams = []interface{}{
			map[string]interface{}{
				"host":   "newbackend",
				"port":   9090,
				"scheme": "https",
			},
		}
		err := repo.Update(proxy)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.NotNil(t, fetched.Upstreams)
	})

	t.Run("Success_DisableSSL", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "SSL Toggle", "ssl-toggle.example.com", models.ProxyTypeReverseProxy)

		// Ensure SSL is enabled in the database to start.
		require.NoError(t, tdb.DB.Model(&models.Proxy{}).Where("id = ?", proxy.ID).Update("ssl_enabled", true).Error)

		// Disabling SSL must persist. With db.Updates(struct), the false zero
		// value is silently dropped and SSL stays enabled.
		proxy.SSLEnabled = false
		require.NoError(t, repo.Update(proxy))

		fetched, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		assert.False(t, fetched.SSLEnabled, "disabling SSL should persist")
	})
}

func TestProxyRepository_Delete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Delete Test", "delete-test.example.com", models.ProxyTypeReverseProxy)

		err := repo.Delete(proxy.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(proxy.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Success_NonExistent", func(t *testing.T) {
		// GORM soft delete doesn't error on non-existent records
		err := repo.Delete(999999)
		require.NoError(t, err)
	})

	t.Run("Success_AlreadyDeleted", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Double Delete", "double-delete.example.com", models.ProxyTypeReverseProxy)

		_ = repo.Delete(proxy.ID)
		err := repo.Delete(proxy.ID) // Delete again
		require.NoError(t, err)
	})
}

func TestProxyRepository_UpdateStatus(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success_Deactivate", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Deactivate Test", "deactivate.example.com", models.ProxyTypeReverseProxy)
		assert.True(t, proxy.IsActive)

		err := repo.UpdateStatus(proxy.ID, false)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.False(t, fetched.IsActive)
	})

	t.Run("Success_Activate", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Activate Test", "activate.example.com", models.ProxyTypeReverseProxy)

		// First deactivate
		_ = repo.UpdateStatus(proxy.ID, false)

		// Then activate
		err := repo.UpdateStatus(proxy.ID, true)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.True(t, fetched.IsActive)
	})

	t.Run("Success_NoChange", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "No Change Test", "nochange.example.com", models.ProxyTypeReverseProxy)

		// Set same status
		err := repo.UpdateStatus(proxy.ID, true)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(proxy.ID)
		assert.True(t, fetched.IsActive)
	})
}

func TestProxyRepository_HostnameExists(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Exists_WithoutExclude", func(t *testing.T) {
		CreateTestProxy(t, tdb.DB, user.ID, "Exists Test", "exists.example.com", models.ProxyTypeReverseProxy)

		exists, err := repo.HostnameExists("exists.example.com", 0)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NotExists", func(t *testing.T) {
		exists, err := repo.HostnameExists("notexists.example.com", 0)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Exists_WithExclude_OwnID", func(t *testing.T) {
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "Exclude Test", "exclude.example.com", models.ProxyTypeReverseProxy)

		// Should return false when excluding own ID
		exists, err := repo.HostnameExists("exclude.example.com", proxy.ID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Exists_WithExclude_OtherID", func(t *testing.T) {
		CreateTestProxy(t, tdb.DB, user.ID, "Other Test", "other.example.com", models.ProxyTypeReverseProxy)

		// Should return true when excluding different ID
		exists, err := repo.HostnameExists("other.example.com", 999999)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NotExists_EmptyHostname", func(t *testing.T) {
		exists, err := repo.HostnameExists("", 0)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestProxyRepository_GetStats(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	_ = CreateTestUser(t, tdb.DB) // Create initial user

	t.Run("Success_EmptyDB", func(t *testing.T) {
		tdb.CleanTables(t)
		CreateTestUser(t, tdb.DB) // Recreate user after clean

		stats, err := repo.GetStats()
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Total)
		assert.Equal(t, int64(0), stats.Active)
		assert.Equal(t, int64(0), stats.Inactive)
		assert.Empty(t, stats.ByType)
	})

	t.Run("Success_WithData", func(t *testing.T) {
		// Create a new user for this test since we cleaned tables
		statsUser := CreateTestUser(t, tdb.DB)

		// Create various proxies
		CreateTestProxy(t, tdb.DB, statsUser.ID, "RP1", "rp1-stats.example.com", models.ProxyTypeReverseProxy)
		CreateTestProxy(t, tdb.DB, statsUser.ID, "RP2", "rp2-stats.example.com", models.ProxyTypeReverseProxy)
		CreateTestProxy(t, tdb.DB, statsUser.ID, "Redirect1", "redirect-stats.example.com", models.ProxyTypeRedirect)

		proxy := CreateTestProxy(t, tdb.DB, statsUser.ID, "Static1", "static-stats.example.com", models.ProxyTypeStatic)
		_ = repo.UpdateStatus(proxy.ID, false) // Make one inactive

		stats, err := repo.GetStats()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, stats.Total, int64(4))
		assert.GreaterOrEqual(t, stats.Active, int64(3))
		assert.GreaterOrEqual(t, stats.Inactive, int64(1))
		assert.GreaterOrEqual(t, stats.ByType[models.ProxyTypeReverseProxy], int64(2))
		assert.GreaterOrEqual(t, stats.ByType[models.ProxyTypeRedirect], int64(1))
		assert.GreaterOrEqual(t, stats.ByType[models.ProxyTypeStatic], int64(1))
	})
}

func TestProxyRepository_List(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	// Create test data
	CreateTestProxy(t, tdb.DB, user.ID, "Alpha API", "alpha-api.example.com", models.ProxyTypeReverseProxy)
	CreateTestProxy(t, tdb.DB, user.ID, "Beta API", "beta-api.example.com", models.ProxyTypeReverseProxy)
	CreateTestProxy(t, tdb.DB, user.ID, "Gamma Redirect", "gamma-redirect.example.com", models.ProxyTypeRedirect)

	staticProxy := CreateTestProxy(t, tdb.DB, user.ID, "Delta Static", "delta-static.example.com", models.ProxyTypeStatic)
	_ = repo.UpdateStatus(staticProxy.ID, false)

	t.Run("Success_BasicPagination", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:  1,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.GreaterOrEqual(t, len(proxies), 4)
	})

	t.Run("Success_Pagination_Page1", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:  1,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.Equal(t, 2, len(proxies))
	})

	t.Run("Success_Pagination_Page2", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:  2,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.GreaterOrEqual(t, len(proxies), 2)
	})

	t.Run("Success_FilterByType", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:  1,
			Limit: 10,
			Types: []string{models.ProxyTypeReverseProxy},
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(2))
		for _, p := range proxies {
			assert.Equal(t, models.ProxyTypeReverseProxy, p.Type)
		}
	})

	t.Run("Success_FilterByMultipleTypes", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:  1,
			Limit: 10,
			Types: []string{models.ProxyTypeReverseProxy, models.ProxyTypeRedirect},
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		for _, p := range proxies {
			assert.Contains(t, []string{models.ProxyTypeReverseProxy, models.ProxyTypeRedirect}, p.Type)
		}
	})

	t.Run("Success_ExcludeTypes", func(t *testing.T) {
		resultProxies, _, err := repo.List(ProxyListParams{
			Page:         1,
			Limit:        10,
			TypesExclude: []string{models.ProxyTypeStatic},
		})
		require.NoError(t, err)
		for _, p := range resultProxies {
			assert.NotEqual(t, models.ProxyTypeStatic, p.Type)
		}
	})

	t.Run("Success_FilterByStatusActive", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Status: "active",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		for _, p := range proxies {
			assert.True(t, p.IsActive)
		}
	})

	t.Run("Success_FilterByStatusInactive", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Status: "inactive",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		for _, p := range proxies {
			assert.False(t, p.IsActive)
		}
	})

	t.Run("Success_FilterByStatusNot", func(t *testing.T) {
		proxies, _, err := repo.List(ProxyListParams{
			Page:      1,
			Limit:     10,
			StatusNot: "inactive",
		})
		require.NoError(t, err)
		for _, p := range proxies {
			assert.True(t, p.IsActive)
		}
	})

	t.Run("Success_Search", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "Alpha",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		for _, p := range proxies {
			assert.Contains(t, p.Name, "Alpha")
		}
	})

	t.Run("Success_SearchByHostname", func(t *testing.T) {
		_, total, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "alpha-api",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
	})

	t.Run("Success_SortByNameASC", func(t *testing.T) {
		proxies, _, err := repo.List(ProxyListParams{
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
		proxies, _, err := repo.List(ProxyListParams{
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

	t.Run("Success_SortByHostname", func(t *testing.T) {
		proxies, _, err := repo.List(ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "hostname",
			Order: "asc",
		})
		require.NoError(t, err)
		if len(proxies) >= 2 {
			assert.LessOrEqual(t, proxies[0].Hostname, proxies[1].Hostname)
		}
	})

	t.Run("Success_InvalidSortUsesDefault", func(t *testing.T) {
		proxies, _, err := repo.List(ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "invalid_field",
			Order: "asc",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, proxies)
	})

	t.Run("Success_InvalidOrderUsesDefault", func(t *testing.T) {
		proxies, _, err := repo.List(ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "name",
			Order: "INVALID",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, proxies)
	})

	t.Run("Success_CombinedFilters", func(t *testing.T) {
		proxies, _, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Types:  []string{models.ProxyTypeReverseProxy},
			Status: "active",
			Search: "API",
			Sort:   "name",
			Order:  "asc",
		})
		require.NoError(t, err)
		for _, p := range proxies {
			assert.Equal(t, models.ProxyTypeReverseProxy, p.Type)
			assert.True(t, p.IsActive)
		}
	})

	t.Run("Success_EmptyResult", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "NonExistentSearchTerm12345",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, proxies)
	})

	t.Run("Success_LargePageNumber", func(t *testing.T) {
		proxies, total, err := repo.List(ProxyListParams{
			Page:  999,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.Empty(t, proxies) // Page too high
	})
}

func TestProxyRepository_List_SSLFilter(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	// Create proxies with different SSL settings
	sslProxy := &models.Proxy{
		Type:       models.ProxyTypeReverseProxy,
		Name:       "SSL Enabled",
		Hostname:   "ssl-enabled.example.com",
		SSLEnabled: true,
		CreatedBy:  user.ID,
	}
	_ = repo.Create(sslProxy)

	noSslProxy := &models.Proxy{
		Type:       models.ProxyTypeReverseProxy,
		Name:       "SSL Disabled",
		Hostname:   "ssl-disabled.example.com",
		SSLEnabled: false,
		CreatedBy:  user.ID,
	}
	_ = repo.Create(noSslProxy)

	t.Run("FilterBySSLEnabled_True", func(t *testing.T) {
		sslEnabled := true
		proxies, _, err := repo.List(ProxyListParams{
			Page:       1,
			Limit:      10,
			SSLEnabled: &sslEnabled,
		})
		require.NoError(t, err)
		for _, p := range proxies {
			assert.True(t, p.SSLEnabled)
		}
	})

	t.Run("FilterBySSLEnabled_False", func(t *testing.T) {
		sslEnabled := false
		proxies, _, err := repo.List(ProxyListParams{
			Page:       1,
			Limit:      10,
			SSLEnabled: &sslEnabled,
		})
		require.NoError(t, err)
		for _, p := range proxies {
			assert.False(t, p.SSLEnabled)
		}
	})

	t.Run("FilterBySSLEnabled_Nil", func(t *testing.T) {
		proxies, _, err := repo.List(ProxyListParams{
			Page:       1,
			Limit:      10,
			SSLEnabled: nil,
		})
		require.NoError(t, err)
		// Should return all proxies
		assert.GreaterOrEqual(t, len(proxies), 2)
	})
}

func TestProxyRepository_List_TargetFilter(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	// Create proxies with different upstreams
	proxy1 := &models.Proxy{
		Type:      models.ProxyTypeReverseProxy,
		Name:      "Backend One",
		Hostname:  "backend-one.example.com",
		CreatedBy: user.ID,
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "internal-service.local",
				"port":   8080,
				"scheme": "http",
			},
		},
	}
	_ = repo.Create(proxy1)

	proxy2 := &models.Proxy{
		Type:      models.ProxyTypeRedirect,
		Name:      "Redirect Target",
		Hostname:  "redirect-target.example.com",
		CreatedBy: user.ID,
		RedirectConfig: models.JSONField{
			"target": "https://external-service.com",
		},
	}
	_ = repo.Create(proxy2)

	t.Run("FilterByUpstreamTarget", func(t *testing.T) {
		_, total, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Target: "internal-service",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
	})

	t.Run("FilterByRedirectTarget", func(t *testing.T) {
		_, total, err := repo.List(ProxyListParams{
			Page:   1,
			Limit:  10,
			Target: "external-service",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
	})
}

func TestProxyRepository_List_PopulatesACLSummary(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)
	// Start from a clean ACL/proxy state so the per-proxy summary counts are
	// deterministic regardless of test ordering.
	CleanACLTables(t, tdb.DB)
	defer CleanACLTables(t, tdb.DB)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	// Proxy A is protected by both ACL groups; proxy B has no assignments.
	proxyA := CreateTestProxy(t, tdb.DB, user.ID, "Protected", "protected.example.com", models.ProxyTypeReverseProxy)
	proxyB := CreateTestProxy(t, tdb.DB, user.ID, "Unprotected", "unprotected.example.com", models.ProxyTypeReverseProxy)

	vpnGroup := CreateTestACLGroup(t, tdb.DB, user.ID, "vpn")
	staffGroup := CreateTestACLGroup(t, tdb.DB, user.ID, "staff")

	// Priorities drive the ordering of ACLGroupNames (priority ASC).
	CreateTestProxyACLAssignment(t, tdb.DB, proxyA.ID, vpnGroup.ID, "/*", 0)
	CreateTestProxyACLAssignment(t, tdb.DB, proxyA.ID, staffGroup.ID, "/*", 1)

	proxies, total, err := repo.List(ProxyListParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(2))
	require.Len(t, proxies, 2)

	byID := make(map[int]models.Proxy, len(proxies))
	for _, p := range proxies {
		byID[p.ID] = p
	}

	gotA, ok := byID[proxyA.ID]
	require.True(t, ok, "proxy A should be in the list")
	assert.Equal(t, 2, gotA.ACLGroupCount)
	assert.Equal(t, []string{"vpn", "staff"}, gotA.ACLGroupNames)

	gotB, ok := byID[proxyB.ID]
	require.True(t, ok, "proxy B should be in the list")
	assert.Equal(t, 0, gotB.ACLGroupCount)
	assert.Equal(t, []string{}, gotB.ACLGroupNames)
}
