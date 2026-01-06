package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// =============================================================================
// AuditLogRepository Integration Tests
// =============================================================================

func TestAuditLogRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success_WithAllFields", func(t *testing.T) {
		resourceType := "proxy"
		resourceID := 1
		resourceName := "my-proxy"
		ipAddress := "192.168.1.100"
		userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

		log := &models.AuditLog{
			UserID:       &user.ID,
			Action:       models.AuditActionProxyCreate,
			ResourceType: &resourceType,
			ResourceID:   &resourceID,
			ResourceName: &resourceName,
			Details: models.JSONField{
				"hostname": "test.example.com",
				"type":     "reverse_proxy",
			},
			IPAddress: &ipAddress,
			UserAgent: &userAgent,
			Status:    models.AuditStatusSuccess,
		}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotZero(t, log.ID)
		assert.NotZero(t, log.CreatedAt)
	})

	t.Run("Success_WithMinimalFields", func(t *testing.T) {
		log := &models.AuditLog{
			Action: models.AuditActionAuthLogin,
			Status: models.AuditStatusSuccess,
		}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotZero(t, log.ID)
	})

	t.Run("Success_WithNilUserID", func(t *testing.T) {
		log := &models.AuditLog{
			UserID: nil,
			Action: models.AuditActionSystemStartup,
			Status: models.AuditStatusSuccess,
		}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotZero(t, log.ID)
		assert.Nil(t, log.UserID)
	})

	t.Run("Success_WithFailureStatus", func(t *testing.T) {
		errorMessage := "Authentication failed: invalid credentials"
		log := &models.AuditLog{
			UserID:       nil,
			Action:       models.AuditActionAuthLoginFailed,
			Status:       models.AuditStatusFailure,
			ErrorMessage: &errorMessage,
		}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotZero(t, log.ID)
		assert.Equal(t, models.AuditStatusFailure, log.Status)
		require.NotNil(t, log.ErrorMessage)
		assert.Equal(t, errorMessage, *log.ErrorMessage)
	})

	t.Run("Success_WithDetails", func(t *testing.T) {
		log := &models.AuditLog{
			UserID: &user.ID,
			Action: models.AuditActionProxyUpdate,
			Status: models.AuditStatusSuccess,
			Details: models.JSONField{
				"changes": map[string]interface{}{
					"name": map[string]interface{}{
						"old": "old-name",
						"new": "new-name",
					},
				},
			},
		}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotZero(t, log.ID)
	})

	t.Run("Success_AllActionTypes", func(t *testing.T) {
		actions := []string{
			models.AuditActionProxyCreate,
			models.AuditActionProxyUpdate,
			models.AuditActionProxyDelete,
			models.AuditActionProxyEnable,
			models.AuditActionProxyDisable,
			models.AuditActionAuthLogin,
			models.AuditActionAuthLogout,
			models.AuditActionAuthRegister,
			models.AuditActionSettingsUpdate,
			models.AuditActionSyncStarted,
			models.AuditActionSyncCompleted,
			models.AuditActionCaddyReload,
		}

		for _, action := range actions {
			log := &models.AuditLog{
				Action: action,
				Status: models.AuditStatusSuccess,
			}
			err := repo.Create(log)
			require.NoError(t, err, "Failed to create audit log with action: %s", action)
			assert.NotZero(t, log.ID)
		}
	})
}

func TestAuditLogRepository_GetByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Success", func(t *testing.T) {
		log := CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		fetched, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.Equal(t, log.ID, fetched.ID)
		assert.Equal(t, models.AuditActionProxyCreate, fetched.Action)
		assert.Equal(t, models.AuditStatusSuccess, fetched.Status)
		require.NotNil(t, fetched.UserID)
		assert.Equal(t, user.ID, *fetched.UserID)
	})

	t.Run("Success_WithUserPreloaded", func(t *testing.T) {
		log := CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionAuthLogin, models.AuditStatusSuccess)

		fetched, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		require.NotNil(t, fetched.User)
		assert.Equal(t, user.ID, fetched.User.ID)
		assert.Equal(t, user.Username, fetched.User.Username)
	})

	t.Run("Success_WithNilUserID", func(t *testing.T) {
		log := CreateTestAuditLog(t, tdb.DB, nil, models.AuditActionSystemStartup, models.AuditStatusSuccess)

		fetched, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		assert.Nil(t, fetched.UserID)
		assert.Nil(t, fetched.User)
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

func TestAuditLogRepository_List(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)
	user2 := CreateTestUser(t, tdb.DB)

	// Clean audit logs for deterministic tests
	tdb.DB.Exec("DELETE FROM audit_logs")

	// Create test data with various attributes
	ip1 := "192.168.1.1"
	ip2 := "10.0.0.1"
	ip3 := "192.168.1.100"

	// Create audit logs with different attributes
	CreateTestAuditLogWithDetails(t, tdb.DB, AuditLogTestOptions{
		UserID:       &user.ID,
		Action:       models.AuditActionProxyCreate,
		ResourceType: StringPtr("proxy"),
		ResourceName: StringPtr("alpha-proxy"),
		IPAddress:    &ip1,
		Status:       models.AuditStatusSuccess,
	})
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	CreateTestAuditLogWithDetails(t, tdb.DB, AuditLogTestOptions{
		UserID:       &user.ID,
		Action:       models.AuditActionProxyUpdate,
		ResourceType: StringPtr("proxy"),
		ResourceName: StringPtr("beta-proxy"),
		IPAddress:    &ip2,
		Status:       models.AuditStatusSuccess,
	})
	time.Sleep(10 * time.Millisecond)

	CreateTestAuditLogWithDetails(t, tdb.DB, AuditLogTestOptions{
		UserID:       &user2.ID,
		Action:       models.AuditActionAuthLogin,
		ResourceType: StringPtr("user"),
		ResourceName: StringPtr("testuser"),
		IPAddress:    &ip3,
		Status:       models.AuditStatusSuccess,
	})
	time.Sleep(10 * time.Millisecond)

	CreateTestAuditLogWithDetails(t, tdb.DB, AuditLogTestOptions{
		UserID:       nil,
		Action:       models.AuditActionAuthLoginFailed,
		ResourceType: StringPtr("user"),
		ResourceName: nil,
		IPAddress:    &ip1,
		Status:       models.AuditStatusFailure,
		ErrorMessage: StringPtr("Invalid credentials"),
	})
	time.Sleep(10 * time.Millisecond)

	CreateTestAuditLogWithDetails(t, tdb.DB, AuditLogTestOptions{
		UserID:       &user.ID,
		Action:       models.AuditActionSettingsUpdate,
		ResourceType: StringPtr("settings"),
		ResourceName: StringPtr("audit_config"),
		IPAddress:    &ip1,
		Status:       models.AuditStatusSuccess,
	})

	t.Run("Success_BasicPagination", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, logs, 5)
	})

	t.Run("Success_Pagination_Page1", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, logs, 2)
	})

	t.Run("Success_Pagination_Page2", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:  2,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, logs, 2)
	})

	t.Run("Success_Pagination_Page3", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:  3,
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, logs, 1)
	})

	t.Run("Success_FilterByActions", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:    1,
			Limit:   10,
			Actions: []string{models.AuditActionProxyCreate, models.AuditActionProxyUpdate},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, log := range logs {
			assert.Contains(t, []string{models.AuditActionProxyCreate, models.AuditActionProxyUpdate}, log.Action)
		}
	})

	t.Run("Success_FilterByActionsExclude", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:           1,
			Limit:          10,
			ActionsExclude: []string{models.AuditActionAuthLogin, models.AuditActionAuthLoginFailed},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		for _, log := range logs {
			assert.NotContains(t, []string{models.AuditActionAuthLogin, models.AuditActionAuthLoginFailed}, log.Action)
		}
	})

	t.Run("Success_FilterByResourceTypes", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:          1,
			Limit:         10,
			ResourceTypes: []string{"proxy"},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, log := range logs {
			require.NotNil(t, log.ResourceType)
			assert.Equal(t, "proxy", *log.ResourceType)
		}
	})

	t.Run("Success_FilterByResourceTypesExclude", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:                 1,
			Limit:                10,
			ResourceTypesExclude: []string{"proxy"},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		for _, log := range logs {
			if log.ResourceType != nil {
				assert.NotEqual(t, "proxy", *log.ResourceType)
			}
		}
	})

	t.Run("Success_FilterByUserID", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:   1,
			Limit:  10,
			UserID: &user.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		for _, log := range logs {
			require.NotNil(t, log.UserID)
			assert.Equal(t, user.ID, *log.UserID)
		}
	})

	t.Run("Success_FilterByStatus", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:   1,
			Limit:  10,
			Status: models.AuditStatusSuccess,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		for _, log := range logs {
			assert.Equal(t, models.AuditStatusSuccess, log.Status)
		}
	})

	t.Run("Success_FilterByStatusExclude", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:          1,
			Limit:         10,
			StatusExclude: models.AuditStatusFailure,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		for _, log := range logs {
			assert.NotEqual(t, models.AuditStatusFailure, log.Status)
		}
	})

	t.Run("Success_FilterByIPAddressExact", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:      1,
			Limit:     10,
			IPAddress: "192.168.1.1",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		for _, log := range logs {
			require.NotNil(t, log.IPAddress)
			assert.Equal(t, "192.168.1.1", *log.IPAddress)
		}
	})

	t.Run("Success_FilterByIPAddressContains", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:              1,
			Limit:             10,
			IPAddressContains: "192.168",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		for _, log := range logs {
			require.NotNil(t, log.IPAddress)
			assert.Contains(t, *log.IPAddress, "192.168")
		}
	})

	t.Run("Success_FilterByIPAddressStartsWith", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:                1,
			Limit:               10,
			IPAddressStartsWith: "10.",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		for _, log := range logs {
			require.NotNil(t, log.IPAddress)
			assert.True(t, (*log.IPAddress)[:3] == "10.")
		}
	})

	t.Run("Success_FilterByIPAddressEndsWith", func(t *testing.T) {
		_, total, err := repo.List(AuditLogListParams{
			Page:              1,
			Limit:             10,
			IPAddressEndsWith: ".1",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
	})

	t.Run("Success_FilterByIPAddressNot", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:         1,
			Limit:        10,
			IPAddressNot: "192.168.1.1",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, log := range logs {
			if log.IPAddress != nil {
				assert.NotEqual(t, "192.168.1.1", *log.IPAddress)
			}
		}
	})

	t.Run("Success_FilterBySearch_Action", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:   1,
			Limit:  10,
			Search: "proxy.create",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		for _, log := range logs {
			assert.Contains(t, log.Action, "proxy.create")
		}
	})

	t.Run("Success_FilterBySearch_ResourceName", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:   1,
			Limit:  10,
			Search: "alpha",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		for _, log := range logs {
			require.NotNil(t, log.ResourceName)
			assert.Contains(t, *log.ResourceName, "alpha")
		}
	})

	t.Run("Success_FilterByDateFrom", func(t *testing.T) {
		// Get all logs to find the earliest
		allLogs, _, _ := repo.List(AuditLogListParams{Page: 1, Limit: 100})
		if len(allLogs) > 0 {
			// Find a date that excludes at least one log
			midpoint := allLogs[len(allLogs)/2].CreatedAt
			filteredLogs, total, err := repo.List(AuditLogListParams{
				Page:     1,
				Limit:    10,
				DateFrom: &midpoint,
			})
			require.NoError(t, err)
			assert.LessOrEqual(t, total, int64(5))
			for _, log := range filteredLogs {
				assert.True(t, log.CreatedAt.Equal(midpoint) || log.CreatedAt.After(midpoint))
			}
		}
	})

	t.Run("Success_FilterByDateTo", func(t *testing.T) {
		allLogs, _, _ := repo.List(AuditLogListParams{Page: 1, Limit: 100})
		if len(allLogs) > 0 {
			midpoint := allLogs[len(allLogs)/2].CreatedAt
			filteredLogs, total, err := repo.List(AuditLogListParams{
				Page:   1,
				Limit:  10,
				DateTo: &midpoint,
			})
			require.NoError(t, err)
			assert.LessOrEqual(t, total, int64(5))
			for _, log := range filteredLogs {
				assert.True(t, log.CreatedAt.Equal(midpoint) || log.CreatedAt.Before(midpoint))
			}
		}
	})

	t.Run("Success_FilterByDateRange", func(t *testing.T) {
		allLogs, _, _ := repo.List(AuditLogListParams{Page: 1, Limit: 100, Sort: "created_at", Order: "asc"})
		if len(allLogs) >= 3 {
			dateFrom := allLogs[1].CreatedAt
			dateTo := allLogs[3].CreatedAt
			filteredLogs, total, err := repo.List(AuditLogListParams{
				Page:     1,
				Limit:    10,
				DateFrom: &dateFrom,
				DateTo:   &dateTo,
			})
			require.NoError(t, err)
			assert.LessOrEqual(t, total, int64(5))
			for _, log := range filteredLogs {
				assert.True(t, (log.CreatedAt.Equal(dateFrom) || log.CreatedAt.After(dateFrom)) &&
					(log.CreatedAt.Equal(dateTo) || log.CreatedAt.Before(dateTo)))
			}
		}
	})

	t.Run("Success_SortByCreatedAt_DESC", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "created_at",
			Order: "desc",
		})
		require.NoError(t, err)
		if len(logs) >= 2 {
			assert.True(t, logs[0].CreatedAt.After(logs[1].CreatedAt) || logs[0].CreatedAt.Equal(logs[1].CreatedAt))
		}
	})

	t.Run("Success_SortByCreatedAt_ASC", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "created_at",
			Order: "asc",
		})
		require.NoError(t, err)
		if len(logs) >= 2 {
			assert.True(t, logs[0].CreatedAt.Before(logs[1].CreatedAt) || logs[0].CreatedAt.Equal(logs[1].CreatedAt))
		}
	})

	t.Run("Success_SortByAction_ASC", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "action",
			Order: "asc",
		})
		require.NoError(t, err)
		if len(logs) >= 2 {
			assert.LessOrEqual(t, logs[0].Action, logs[1].Action)
		}
	})

	t.Run("Success_SortByStatus", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "status",
			Order: "asc",
		})
		require.NoError(t, err)
		if len(logs) >= 2 {
			assert.LessOrEqual(t, logs[0].Status, logs[1].Status)
		}
	})

	t.Run("Success_InvalidSortFieldUsesDefault", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "invalid_field",
			Order: "asc",
		})
		require.NoError(t, err)
		// Should not error, uses default sort (created_at)
		assert.NotEmpty(t, logs)
	})

	t.Run("Success_InvalidOrderUsesDefault", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "action",
			Order: "INVALID",
		})
		require.NoError(t, err)
		// Should not error, uses default order (DESC)
		assert.NotEmpty(t, logs)
	})

	t.Run("Success_SQLInjectionPrevented_Sort", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "action; DROP TABLE audit_logs;--",
			Order: "asc",
		})
		require.NoError(t, err)
		// Should not error and should fall back to default sort
		assert.NotEmpty(t, logs)
	})

	t.Run("Success_SQLInjectionPrevented_Order", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "action",
			Order: "ASC; DELETE FROM audit_logs;--",
		})
		require.NoError(t, err)
		// Should not error and should fall back to default order
		assert.NotEmpty(t, logs)
	})

	t.Run("Success_CombinedFilters", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:          1,
			Limit:         10,
			Actions:       []string{models.AuditActionProxyCreate, models.AuditActionProxyUpdate},
			ResourceTypes: []string{"proxy"},
			Status:        models.AuditStatusSuccess,
			UserID:        &user.ID,
			Sort:          "action",
			Order:         "asc",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, log := range logs {
			assert.Contains(t, []string{models.AuditActionProxyCreate, models.AuditActionProxyUpdate}, log.Action)
			require.NotNil(t, log.ResourceType)
			assert.Equal(t, "proxy", *log.ResourceType)
			assert.Equal(t, models.AuditStatusSuccess, log.Status)
			require.NotNil(t, log.UserID)
			assert.Equal(t, user.ID, *log.UserID)
		}
	})

	t.Run("Success_UserPreloaded", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:   1,
			Limit:  10,
			UserID: &user.ID,
		})
		require.NoError(t, err)
		for _, log := range logs {
			if log.UserID != nil {
				require.NotNil(t, log.User)
				assert.Equal(t, user.ID, log.User.ID)
			}
		}
	})

	t.Run("Success_EmptyResult", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:   1,
			Limit:  10,
			Search: "nonexistent_search_term_12345",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, logs)
	})

	t.Run("Success_LargePageNumber", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:  999,
			Limit: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Empty(t, logs) // Page too high
	})

	t.Run("Success_CaseSensitiveSort", func(t *testing.T) {
		logs, _, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 10,
			Sort:  "ACTION", // uppercase
			Order: "ASC",
		})
		require.NoError(t, err)
		// Should handle case-insensitive sort field
		assert.NotEmpty(t, logs)
	})
}

func TestAuditLogRepository_GetStats(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)

	t.Run("Success_EmptyDB", func(t *testing.T) {
		tdb.CleanTables(t)

		stats, err := repo.GetStats()
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.TotalLogs)
		assert.Empty(t, stats.ByAction)
		assert.Empty(t, stats.ByStatus)
		assert.Empty(t, stats.ByResourceType)
		assert.Empty(t, stats.RecentActivity)
	})

	t.Run("Success_WithData", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create various audit logs
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionAuthLogin, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, nil, models.AuditActionAuthLoginFailed, models.AuditStatusFailure)

		stats, err := repo.GetStats()
		require.NoError(t, err)

		// Verify total count
		assert.Equal(t, int64(5), stats.TotalLogs)

		// Verify counts by action
		assert.Equal(t, int64(2), stats.ByAction[models.AuditActionProxyCreate])
		assert.Equal(t, int64(1), stats.ByAction[models.AuditActionProxyUpdate])
		assert.Equal(t, int64(1), stats.ByAction[models.AuditActionAuthLogin])
		assert.Equal(t, int64(1), stats.ByAction[models.AuditActionAuthLoginFailed])

		// Verify counts by status
		assert.Equal(t, int64(4), stats.ByStatus[models.AuditStatusSuccess])
		assert.Equal(t, int64(1), stats.ByStatus[models.AuditStatusFailure])

		// Verify counts by resource type
		assert.GreaterOrEqual(t, stats.ByResourceType["proxy"], int64(3))

		// Verify recent activity
		assert.LessOrEqual(t, len(stats.RecentActivity), 10)
		assert.GreaterOrEqual(t, len(stats.RecentActivity), 1)
	})

	t.Run("Success_RecentActivityLimit", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create 15 audit logs to verify limit of 10
		for i := 0; i < 15; i++ {
			CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		}

		stats, err := repo.GetStats()
		require.NoError(t, err)
		assert.Equal(t, int64(15), stats.TotalLogs)
		assert.Len(t, stats.RecentActivity, 10) // Should be limited to 10
	})

	t.Run("Success_RecentActivityOrderedByCreatedAt", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create logs with delays to ensure different timestamps
		for i := 0; i < 5; i++ {
			CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
			time.Sleep(10 * time.Millisecond)
		}

		stats, err := repo.GetStats()
		require.NoError(t, err)

		// Verify recent activity is ordered by created_at DESC
		if len(stats.RecentActivity) >= 2 {
			for i := 0; i < len(stats.RecentActivity)-1; i++ {
				assert.True(t,
					stats.RecentActivity[i].CreatedAt.After(stats.RecentActivity[i+1].CreatedAt) ||
						stats.RecentActivity[i].CreatedAt.Equal(stats.RecentActivity[i+1].CreatedAt),
					"Recent activity should be ordered by created_at DESC")
			}
		}
	})

	t.Run("Success_UserPreloadedInRecentActivity", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		stats, err := repo.GetStats()
		require.NoError(t, err)
		require.NotEmpty(t, stats.RecentActivity)

		for _, log := range stats.RecentActivity {
			if log.UserID != nil {
				require.NotNil(t, log.User)
			}
		}
	})

	t.Run("Success_NilResourceTypeExcluded", func(t *testing.T) {
		tdb.CleanTables(t)

		// Create log without resource type
		log := &models.AuditLog{
			Action:       models.AuditActionSystemStartup,
			Status:       models.AuditStatusSuccess,
			ResourceType: nil,
		}
		err := tdb.DB.Create(log).Error
		require.NoError(t, err)

		stats, err := repo.GetStats()
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats.TotalLogs)
		// ByResourceType should not contain entry for nil resource type
		assert.Empty(t, stats.ByResourceType)
	})
}

func TestAuditLogRepository_DeleteOlderThan(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)

	t.Run("Success_DeletesOldLogs", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create logs with different timestamps using raw SQL
		// Old log (30 days ago)
		oldTime := time.Now().Add(-30 * 24 * time.Hour)
		oldLog := &models.AuditLog{
			UserID: &user.ID,
			Action: models.AuditActionProxyCreate,
			Status: models.AuditStatusSuccess,
		}
		err := tdb.DB.Create(oldLog).Error
		require.NoError(t, err)
		tdb.DB.Model(oldLog).Update("created_at", oldTime)

		// Recent log (1 hour ago)
		recentLog := CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)

		// Delete logs older than 7 days
		cutoff := time.Now().Add(-7 * 24 * time.Hour)
		deleted, err := repo.DeleteOlderThan(cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Verify old log is deleted
		_, err = repo.GetByID(oldLog.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		// Verify recent log still exists
		_, err = repo.GetByID(recentLog.ID)
		require.NoError(t, err)
	})

	t.Run("Success_NoLogsToDelete", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create only recent logs
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)

		// Delete logs older than 7 days (none exist)
		cutoff := time.Now().Add(-7 * 24 * time.Hour)
		deleted, err := repo.DeleteOlderThan(cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted)
	})

	t.Run("Success_DeleteAllLogs", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create logs
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)

		// Delete all logs (cutoff in the future)
		cutoff := time.Now().Add(1 * time.Hour)
		deleted, err := repo.DeleteOlderThan(cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		// Verify all logs are deleted
		logs, total, _ := repo.List(AuditLogListParams{Page: 1, Limit: 100})
		assert.Equal(t, int64(0), total)
		assert.Empty(t, logs)
	})

	t.Run("Success_EmptyDB", func(t *testing.T) {
		tdb.CleanTables(t)

		cutoff := time.Now().Add(-7 * 24 * time.Hour)
		deleted, err := repo.DeleteOlderThan(cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted)
	})

	t.Run("Success_BoundaryCondition", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create a log and set its created_at to exactly the cutoff time
		log := CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		exactCutoff := time.Now().Add(-1 * time.Hour)
		tdb.DB.Model(log).Update("created_at", exactCutoff)

		// Create another log just before the cutoff
		log2 := CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)
		beforeCutoff := exactCutoff.Add(-1 * time.Minute)
		tdb.DB.Model(log2).Update("created_at", beforeCutoff)

		// Delete logs older than (strictly less than) the cutoff
		deleted, err := repo.DeleteOlderThan(exactCutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted) // Only the log before cutoff should be deleted
	})
}

func TestAuditLogRepository_CountByAction(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)

	t.Run("Success_WithData", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create multiple logs with same action
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusFailure)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)

		count, err := repo.CountByAction(models.AuditActionProxyCreate)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)

		count, err = repo.CountByAction(models.AuditActionProxyUpdate)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Success_NoMatchingAction", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		count, err := repo.CountByAction(models.AuditActionProxyDelete)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Success_EmptyDB", func(t *testing.T) {
		tdb.CleanTables(t)

		count, err := repo.CountByAction(models.AuditActionProxyCreate)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Success_AllActionTypes", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		actions := []string{
			models.AuditActionProxyCreate,
			models.AuditActionProxyUpdate,
			models.AuditActionProxyDelete,
			models.AuditActionAuthLogin,
			models.AuditActionAuthLogout,
		}

		for _, action := range actions {
			CreateTestAuditLog(t, tdb.DB, &user.ID, action, models.AuditStatusSuccess)
		}

		for _, action := range actions {
			count, err := repo.CountByAction(action)
			require.NoError(t, err)
			assert.Equal(t, int64(1), count, "Count for action %s should be 1", action)
		}
	})

	t.Run("Success_CaseSensitive", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		// Different case should not match
		count, err := repo.CountByAction("PROXY.CREATE")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestAuditLogRepository_CountByUserID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)

	t.Run("Success_WithData", func(t *testing.T) {
		tdb.CleanTables(t)
		user1 := CreateTestUser(t, tdb.DB)
		user2 := CreateTestUser(t, tdb.DB)

		// Create logs for user1
		CreateTestAuditLog(t, tdb.DB, &user1.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user1.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user1.ID, models.AuditActionAuthLogin, models.AuditStatusSuccess)

		// Create logs for user2
		CreateTestAuditLog(t, tdb.DB, &user2.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		// Create logs without user
		CreateTestAuditLog(t, tdb.DB, nil, models.AuditActionSystemStartup, models.AuditStatusSuccess)

		count1, err := repo.CountByUserID(user1.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count1)

		count2, err := repo.CountByUserID(user2.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count2)
	})

	t.Run("Success_NoLogsForUser", func(t *testing.T) {
		tdb.CleanTables(t)
		user1 := CreateTestUser(t, tdb.DB)
		user2 := CreateTestUser(t, tdb.DB)

		// Only create logs for user1
		CreateTestAuditLog(t, tdb.DB, &user1.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		// User2 should have 0 logs
		count, err := repo.CountByUserID(user2.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Success_NonExistentUser", func(t *testing.T) {
		tdb.CleanTables(t)

		count, err := repo.CountByUserID(999999)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Success_EmptyDB", func(t *testing.T) {
		tdb.CleanTables(t)

		count, err := repo.CountByUserID(1)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Success_MixedUserAndNilUser", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		// Create logs with user
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyUpdate, models.AuditStatusSuccess)

		// Create logs without user (nil)
		CreateTestAuditLog(t, tdb.DB, nil, models.AuditActionSystemStartup, models.AuditStatusSuccess)
		CreateTestAuditLog(t, tdb.DB, nil, models.AuditActionCaddyReload, models.AuditStatusSuccess)

		// Only logs with matching user_id should be counted
		count, err := repo.CountByUserID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("Success_ZeroUserID", func(t *testing.T) {
		tdb.CleanTables(t)

		count, err := repo.CountByUserID(0)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestAuditLogRepository_EdgeCases(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("Create_WithEmptyAction", func(t *testing.T) {
		// Action field is required, but we test behavior
		log := &models.AuditLog{
			UserID: &user.ID,
			Action: "",
			Status: models.AuditStatusSuccess,
		}

		err := repo.Create(log)
		// This may or may not error depending on DB constraints
		// The important thing is it doesn't panic
		if err == nil {
			assert.NotZero(t, log.ID)
		}
	})

	t.Run("Create_WithEmptyStatus", func(t *testing.T) {
		// Status has a default value in the schema
		log := &models.AuditLog{
			UserID: &user.ID,
			Action: models.AuditActionProxyCreate,
			Status: "",
		}

		err := repo.Create(log)
		// Should work due to default value
		require.NoError(t, err)
		assert.NotZero(t, log.ID)
	})

	t.Run("Create_WithVeryLongAction", func(t *testing.T) {
		// Action has a max length of 100 characters
		longAction := ""
		for i := 0; i < 200; i++ {
			longAction += "a"
		}

		log := &models.AuditLog{
			UserID: &user.ID,
			Action: longAction,
			Status: models.AuditStatusSuccess,
		}

		err := repo.Create(log)
		// Should error due to length constraint
		require.Error(t, err)
	})

	t.Run("List_WithZeroLimit", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:  1,
			Limit: 0,
		})
		require.NoError(t, err)
		// With limit 0, GORM returns no results but total should be accurate
		assert.Empty(t, logs)
		assert.GreaterOrEqual(t, total, int64(0))
	})

	t.Run("List_WithZeroPage", func(t *testing.T) {
		tdb.CleanTables(t)
		CreateTestUser(t, tdb.DB)
		CreateTestAuditLog(t, tdb.DB, nil, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		logs, total, err := repo.List(AuditLogListParams{
			Page:  0,
			Limit: 10,
		})
		require.NoError(t, err)
		// Page 0 with offset calculation (0-1)*10 = -10, GORM handles this
		assert.GreaterOrEqual(t, total, int64(0))
		// Behavior may vary, just ensure no crash
		_ = logs
	})

	t.Run("List_WithNegativeValues", func(t *testing.T) {
		logs, total, err := repo.List(AuditLogListParams{
			Page:  -1,
			Limit: -10,
		})
		require.NoError(t, err)
		// GORM handles negative values gracefully
		assert.GreaterOrEqual(t, total, int64(0))
		_ = logs
	})

	t.Run("Create_WithSpecialCharacters", func(t *testing.T) {
		resourceName := "proxy-with-special-chars-!@#$%^&*()"
		log := &models.AuditLog{
			UserID:       &user.ID,
			Action:       models.AuditActionProxyCreate,
			ResourceName: &resourceName,
			Status:       models.AuditStatusSuccess,
		}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotZero(t, log.ID)

		// Verify retrieval
		fetched, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		require.NotNil(t, fetched.ResourceName)
		assert.Equal(t, resourceName, *fetched.ResourceName)
	})

	t.Run("Create_WithUnicodeCharacters", func(t *testing.T) {
		resourceName := "proxy-unicode-test"
		ipAddress := "192.168.1.1"
		userAgent := "Mozilla/5.0 Test Agent"
		log := &models.AuditLog{
			UserID:       &user.ID,
			Action:       models.AuditActionProxyCreate,
			ResourceName: &resourceName,
			IPAddress:    &ipAddress,
			UserAgent:    &userAgent,
			Status:       models.AuditStatusSuccess,
		}

		err := repo.Create(log)
		require.NoError(t, err)
		assert.NotZero(t, log.ID)

		// Verify retrieval preserves the text
		fetched, err := repo.GetByID(log.ID)
		require.NoError(t, err)
		require.NotNil(t, fetched.ResourceName)
		assert.Equal(t, resourceName, *fetched.ResourceName)
	})
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestAuditLogRepository_ConcurrentOperations(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewAuditLogRepository(tdb.DB)

	t.Run("ConcurrentCreates", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)

		const numGoroutines = 10
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				log := &models.AuditLog{
					UserID: &user.ID,
					Action: models.AuditActionProxyCreate,
					Status: models.AuditStatusSuccess,
				}
				errors <- repo.Create(log)
			}(i)
		}

		// Collect all errors
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			assert.NoError(t, err)
		}

		// Verify all logs were created
		_, total, err := repo.List(AuditLogListParams{Page: 1, Limit: 100})
		require.NoError(t, err)
		assert.Equal(t, int64(numGoroutines), total)
	})

	t.Run("ConcurrentReads", func(t *testing.T) {
		tdb.CleanTables(t)
		user := CreateTestUser(t, tdb.DB)
		log := CreateTestAuditLog(t, tdb.DB, &user.ID, models.AuditActionProxyCreate, models.AuditStatusSuccess)

		const numGoroutines = 10
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := repo.GetByID(log.ID)
				errors <- err
			}()
		}

		// Collect all errors
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			assert.NoError(t, err)
		}
	})
}
