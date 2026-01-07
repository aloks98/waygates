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
// Test Helpers for ACL
// =============================================================================

// CreateTestACLGroup creates a test ACL group and returns it
func CreateTestACLGroup(t *testing.T, db *gorm.DB, userID int, name string) *models.ACLGroup {
	t.Helper()
	group := &models.ACLGroup{
		Name:            name,
		CombinationMode: models.ACLCombinationModeAny,
		CreatedBy:       userID,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create test ACL group: %v", err)
	}
	return group
}

// CreateTestACLGroupWithDescription creates a test ACL group with description
func CreateTestACLGroupWithDescription(t *testing.T, db *gorm.DB, userID int, name, description string) *models.ACLGroup {
	t.Helper()
	desc := description
	group := &models.ACLGroup{
		Name:            name,
		Description:     &desc,
		CombinationMode: models.ACLCombinationModeAny,
		CreatedBy:       userID,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create test ACL group: %v", err)
	}
	return group
}

// CreateTestIPRule creates a test IP rule for a group
func CreateTestIPRule(t *testing.T, db *gorm.DB, groupID int, ruleType, cidr string, priority int) *models.ACLIPRule {
	t.Helper()
	rule := &models.ACLIPRule{
		ACLGroupID: groupID,
		RuleType:   ruleType,
		CIDR:       cidr,
		Priority:   priority,
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("Failed to create test IP rule: %v", err)
	}
	return rule
}

// CreateTestBasicAuthUser creates a test basic auth user for a group
func CreateTestBasicAuthUser(t *testing.T, db *gorm.DB, groupID int, username, passwordHash string) *models.ACLBasicAuthUser {
	t.Helper()
	user := &models.ACLBasicAuthUser{
		ACLGroupID:   groupID,
		Username:     username,
		PasswordHash: passwordHash,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test basic auth user: %v", err)
	}
	return user
}

// CreateTestExternalProvider creates a test external provider for a group
func CreateTestExternalProvider(t *testing.T, db *gorm.DB, groupID int, name, providerType, verifyURL string) *models.ACLExternalProvider {
	t.Helper()
	provider := &models.ACLExternalProvider{
		ACLGroupID:   groupID,
		Name:         name,
		ProviderType: providerType,
		VerifyURL:    verifyURL,
	}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("Failed to create test external provider: %v", err)
	}
	return provider
}

// CreateTestWaygatesAuth creates a test Waygates auth config for a group
func CreateTestWaygatesAuth(t *testing.T, db *gorm.DB, groupID int, enabled bool, sessionTTL int) *models.ACLWaygatesAuth {
	t.Helper()
	auth := &models.ACLWaygatesAuth{
		ACLGroupID: groupID,
		Enabled:    enabled,
		SessionTTL: sessionTTL,
	}
	if err := db.Create(auth).Error; err != nil {
		t.Fatalf("Failed to create test Waygates auth: %v", err)
	}
	return auth
}

// CreateTestSession creates a test ACL session
func CreateTestSession(t *testing.T, db *gorm.DB, userID int, token string, expiresAt time.Time) *models.ACLSession {
	t.Helper()
	session := &models.ACLSession{
		SessionToken: token,
		UserID:       &userID,
		ExpiresAt:    expiresAt,
	}
	if err := db.Create(session).Error; err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}
	return session
}

// CreateTestProxyACLAssignment creates a test proxy ACL assignment
func CreateTestProxyACLAssignment(t *testing.T, db *gorm.DB, proxyID, groupID int, pathPattern string, priority int) *models.ProxyACLAssignment {
	t.Helper()
	assignment := &models.ProxyACLAssignment{
		ProxyID:     proxyID,
		ACLGroupID:  groupID,
		PathPattern: pathPattern,
		Priority:    priority,
		Enabled:     true,
	}
	if err := db.Create(assignment).Error; err != nil {
		t.Fatalf("Failed to create test proxy ACL assignment: %v", err)
	}
	return assignment
}

// CleanACLTables cleans all ACL-related tables
func CleanACLTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Delete in correct order due to foreign keys
	db.Exec("DELETE FROM acl_sessions")
	db.Exec("DELETE FROM proxy_acl_assignments")
	db.Exec("DELETE FROM acl_waygates_auth")
	db.Exec("DELETE FROM acl_external_providers")
	db.Exec("DELETE FROM acl_basic_auth_users")
	db.Exec("DELETE FROM acl_ip_rules")
	db.Exec("DELETE FROM acl_groups")
	db.Exec("DELETE FROM acl_branding")
	db.Exec("DELETE FROM proxies")
	db.Exec("DELETE FROM users")
}

// =============================================================================
// ACL Group Tests
// =============================================================================

func TestACLRepository_CreateGroup(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	t.Run("success", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user = CreateTestUser(t, tdb.DB)

		group := &models.ACLGroup{
			Name:            "Test Group",
			CombinationMode: models.ACLCombinationModeAny,
			CreatedBy:       user.ID,
		}

		err := repo.CreateGroup(group)
		require.NoError(t, err)
		assert.NotZero(t, group.ID)
		assert.NotZero(t, group.CreatedAt)
	})

	t.Run("duplicate name fails", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user = CreateTestUser(t, tdb.DB)

		group1 := &models.ACLGroup{
			Name:            "Duplicate Name",
			CombinationMode: models.ACLCombinationModeAny,
			CreatedBy:       user.ID,
		}
		err := repo.CreateGroup(group1)
		require.NoError(t, err)

		group2 := &models.ACLGroup{
			Name:            "Duplicate Name",
			CombinationMode: models.ACLCombinationModeAll,
			CreatedBy:       user.ID,
		}
		err = repo.CreateGroup(group2)
		assert.Error(t, err)
	})
}

func TestACLRepository_GetGroupByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("success with preloads", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Test Group")
		CreateTestIPRule(t, tdb.DB, group.ID, models.ACLIPRuleTypeAllow, "192.168.1.0/24", 0)
		CreateTestBasicAuthUser(t, tdb.DB, group.ID, "testuser", "$2a$10$fakehash")
		CreateTestWaygatesAuth(t, tdb.DB, group.ID, true, 86400)

		result, err := repo.GetGroupByID(group.ID)
		require.NoError(t, err)
		assert.Equal(t, group.ID, result.ID)
		assert.Equal(t, "Test Group", result.Name)
		assert.NotNil(t, result.Creator)
		assert.Len(t, result.IPRules, 1)
		assert.Len(t, result.BasicAuthUsers, 1)
		assert.NotNil(t, result.WaygatesAuth)
	})

	t.Run("not found", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)

		result, err := repo.GetGroupByID(99999)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestACLRepository_GetGroupByName(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("success", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Unique Name")

		result, err := repo.GetGroupByName("Unique Name")
		require.NoError(t, err)
		assert.Equal(t, group.ID, result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)

		result, err := repo.GetGroupByName("Nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestACLRepository_ListGroups(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("pagination", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)

		// Create 15 groups
		for i := 1; i <= 15; i++ {
			CreateTestACLGroupWithDescription(t, tdb.DB, user.ID,
				"Group "+string(rune('A'+i-1)),
				"Description "+string(rune('A'+i-1)))
		}

		// Page 1
		groups, total, err := repo.ListGroups(ACLGroupListParams{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(15), total)
		assert.Len(t, groups, 10)

		// Page 2
		groups, total, err = repo.ListGroups(ACLGroupListParams{Page: 2, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(15), total)
		assert.Len(t, groups, 5)
	})

	t.Run("search", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)

		CreateTestACLGroupWithDescription(t, tdb.DB, user.ID, "Admin ACL", "Administrator access")
		CreateTestACLGroupWithDescription(t, tdb.DB, user.ID, "User ACL", "Regular user access")
		CreateTestACLGroupWithDescription(t, tdb.DB, user.ID, "Guest ACL", "Guest access")

		// Search in name
		groups, total, err := repo.ListGroups(ACLGroupListParams{Page: 1, Limit: 10, Search: "Admin"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Admin ACL", groups[0].Name)

		// Search in description
		groups, total, err = repo.ListGroups(ACLGroupListParams{Page: 1, Limit: 10, Search: "Regular"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "User ACL", groups[0].Name)
	})

	t.Run("sorting", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)

		CreateTestACLGroup(t, tdb.DB, user.ID, "Zebra")
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		CreateTestACLGroup(t, tdb.DB, user.ID, "Alpha")
		time.Sleep(10 * time.Millisecond)
		CreateTestACLGroup(t, tdb.DB, user.ID, "Beta")

		// Sort by name ascending
		groups, _, err := repo.ListGroups(ACLGroupListParams{Page: 1, Limit: 10, Sort: "name", Order: "asc"})
		require.NoError(t, err)
		assert.Equal(t, "Alpha", groups[0].Name)
		assert.Equal(t, "Beta", groups[1].Name)
		assert.Equal(t, "Zebra", groups[2].Name)

		// Sort by name descending
		groups, _, err = repo.ListGroups(ACLGroupListParams{Page: 1, Limit: 10, Sort: "name", Order: "desc"})
		require.NoError(t, err)
		assert.Equal(t, "Zebra", groups[0].Name)
	})
}

func TestACLRepository_UpdateGroup(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("success", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Original Name")

		group.Name = "Updated Name"
		group.CombinationMode = models.ACLCombinationModeAll

		err := repo.UpdateGroup(group)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetGroupByID(group.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, models.ACLCombinationModeAll, updated.CombinationMode)
	})
}

func TestACLRepository_DeleteGroup(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("cascade deletes related data", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Group to Delete")

		// Create related data
		CreateTestIPRule(t, tdb.DB, group.ID, models.ACLIPRuleTypeAllow, "10.0.0.0/8", 0)
		CreateTestBasicAuthUser(t, tdb.DB, group.ID, "user1", "$2a$10$hash")
		CreateTestExternalProvider(t, tdb.DB, group.ID, "Provider1", models.ACLProviderTypeAuthelia, "https://auth.example.com/verify")
		CreateTestWaygatesAuth(t, tdb.DB, group.ID, true, 3600)

		// Create proxy and assignment
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "test-proxy", "test.example.com", models.ProxyTypeReverseProxy)
		CreateTestProxyACLAssignment(t, tdb.DB, proxy.ID, group.ID, "/*", 0)

		// Delete group
		err := repo.DeleteGroup(group.ID)
		require.NoError(t, err)

		// Verify group is deleted
		_, err = repo.GetGroupByID(group.ID)
		assert.Error(t, err)

		// Verify related data is deleted
		var ipRuleCount int64
		tdb.DB.Model(&models.ACLIPRule{}).Where("acl_group_id = ?", group.ID).Count(&ipRuleCount)
		assert.Zero(t, ipRuleCount)

		var basicAuthCount int64
		tdb.DB.Model(&models.ACLBasicAuthUser{}).Where("acl_group_id = ?", group.ID).Count(&basicAuthCount)
		assert.Zero(t, basicAuthCount)

		var providerCount int64
		tdb.DB.Model(&models.ACLExternalProvider{}).Where("acl_group_id = ?", group.ID).Count(&providerCount)
		assert.Zero(t, providerCount)

		var waygatesAuthCount int64
		tdb.DB.Model(&models.ACLWaygatesAuth{}).Where("acl_group_id = ?", group.ID).Count(&waygatesAuthCount)
		assert.Zero(t, waygatesAuthCount)

		var assignmentCount int64
		tdb.DB.Model(&models.ProxyACLAssignment{}).Where("acl_group_id = ?", group.ID).Count(&assignmentCount)
		assert.Zero(t, assignmentCount)
	})
}

// =============================================================================
// IP Rule Tests
// =============================================================================

func TestACLRepository_IPRules(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("CRUD operations", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "IP Rules Test")

		// Create
		rule := &models.ACLIPRule{
			ACLGroupID: group.ID,
			RuleType:   models.ACLIPRuleTypeAllow,
			CIDR:       "192.168.1.0/24",
			Priority:   10,
		}
		err := repo.CreateIPRule(rule)
		require.NoError(t, err)
		assert.NotZero(t, rule.ID)

		// Read
		fetched, err := repo.GetIPRuleByID(rule.ID)
		require.NoError(t, err)
		assert.Equal(t, rule.CIDR, fetched.CIDR)

		// Update
		rule.CIDR = "10.0.0.0/8"
		rule.RuleType = models.ACLIPRuleTypeDeny
		err = repo.UpdateIPRule(rule)
		require.NoError(t, err)

		updated, err := repo.GetIPRuleByID(rule.ID)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.0/8", updated.CIDR)
		assert.Equal(t, models.ACLIPRuleTypeDeny, updated.RuleType)

		// Delete
		err = repo.DeleteIPRule(rule.ID)
		require.NoError(t, err)

		_, err = repo.GetIPRuleByID(rule.ID)
		assert.Error(t, err)
	})

	t.Run("list ordered by priority", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Priority Test")

		CreateTestIPRule(t, tdb.DB, group.ID, models.ACLIPRuleTypeAllow, "192.168.1.0/24", 100)
		CreateTestIPRule(t, tdb.DB, group.ID, models.ACLIPRuleTypeDeny, "10.0.0.0/8", 50)
		CreateTestIPRule(t, tdb.DB, group.ID, models.ACLIPRuleTypeBypass, "172.16.0.0/12", 10)

		rules, err := repo.ListIPRules(group.ID)
		require.NoError(t, err)
		assert.Len(t, rules, 3)
		// Should be ordered by priority ascending
		assert.Equal(t, 10, rules[0].Priority)
		assert.Equal(t, 50, rules[1].Priority)
		assert.Equal(t, 100, rules[2].Priority)
	})
}

// =============================================================================
// Basic Auth User Tests
// =============================================================================

func TestACLRepository_BasicAuthUsers(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("CRUD operations", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Basic Auth Test")

		// Create
		authUser := &models.ACLBasicAuthUser{
			ACLGroupID:   group.ID,
			Username:     "apiuser",
			PasswordHash: "$2a$14$hash",
		}
		err := repo.CreateBasicAuthUser(authUser)
		require.NoError(t, err)
		assert.NotZero(t, authUser.ID)

		// Read by ID
		fetched, err := repo.GetBasicAuthUserByID(authUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "apiuser", fetched.Username)

		// Read by group and username
		fetched2, err := repo.GetBasicAuthUser(group.ID, "apiuser")
		require.NoError(t, err)
		assert.Equal(t, authUser.ID, fetched2.ID)

		// Update
		authUser.PasswordHash = "$2a$14$newhash"
		err = repo.UpdateBasicAuthUser(authUser)
		require.NoError(t, err)

		updated, err := repo.GetBasicAuthUserByID(authUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "$2a$14$newhash", updated.PasswordHash)

		// Delete
		err = repo.DeleteBasicAuthUser(authUser.ID)
		require.NoError(t, err)

		_, err = repo.GetBasicAuthUserByID(authUser.ID)
		assert.Error(t, err)
	})

	t.Run("unique constraint on group+username", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Unique Test")

		CreateTestBasicAuthUser(t, tdb.DB, group.ID, "duplicate", "$hash1")

		// Try to create another with same username in same group
		duplicate := &models.ACLBasicAuthUser{
			ACLGroupID:   group.ID,
			Username:     "duplicate",
			PasswordHash: "$hash2",
		}
		err := repo.CreateBasicAuthUser(duplicate)
		assert.Error(t, err)
	})

	t.Run("list ordered by username", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "List Test")

		CreateTestBasicAuthUser(t, tdb.DB, group.ID, "charlie", "$hash")
		CreateTestBasicAuthUser(t, tdb.DB, group.ID, "alice", "$hash")
		CreateTestBasicAuthUser(t, tdb.DB, group.ID, "bob", "$hash")

		users, err := repo.ListBasicAuthUsers(group.ID)
		require.NoError(t, err)
		assert.Len(t, users, 3)
		assert.Equal(t, "alice", users[0].Username)
		assert.Equal(t, "bob", users[1].Username)
		assert.Equal(t, "charlie", users[2].Username)
	})
}

// =============================================================================
// External Provider Tests
// =============================================================================

func TestACLRepository_ExternalProviders(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("CRUD operations", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "External Provider Test")

		// Create
		provider := &models.ACLExternalProvider{
			ACLGroupID:   group.ID,
			ProviderType: models.ACLProviderTypeAuthelia,
			Name:         "My Authelia",
			VerifyURL:    "https://auth.example.com/api/verify",
		}
		err := repo.CreateExternalProvider(provider)
		require.NoError(t, err)
		assert.NotZero(t, provider.ID)

		// Read
		fetched, err := repo.GetExternalProviderByID(provider.ID)
		require.NoError(t, err)
		assert.Equal(t, "My Authelia", fetched.Name)

		// Update
		provider.Name = "Updated Authelia"
		redirectURL := "https://auth.example.com/login"
		provider.AuthRedirectURL = &redirectURL
		err = repo.UpdateExternalProvider(provider)
		require.NoError(t, err)

		updated, err := repo.GetExternalProviderByID(provider.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Authelia", updated.Name)
		assert.NotNil(t, updated.AuthRedirectURL)

		// Delete
		err = repo.DeleteExternalProvider(provider.ID)
		require.NoError(t, err)

		_, err = repo.GetExternalProviderByID(provider.ID)
		assert.Error(t, err)
	})

	t.Run("list ordered by name", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Provider List Test")

		CreateTestExternalProvider(t, tdb.DB, group.ID, "Zeta Provider", models.ACLProviderTypeCustom, "https://z.example.com")
		CreateTestExternalProvider(t, tdb.DB, group.ID, "Alpha Provider", models.ACLProviderTypeAuthelia, "https://a.example.com")
		CreateTestExternalProvider(t, tdb.DB, group.ID, "Beta Provider", models.ACLProviderTypeAuthentik, "https://b.example.com")

		providers, err := repo.ListExternalProviders(group.ID)
		require.NoError(t, err)
		assert.Len(t, providers, 3)
		assert.Equal(t, "Alpha Provider", providers[0].Name)
		assert.Equal(t, "Beta Provider", providers[1].Name)
		assert.Equal(t, "Zeta Provider", providers[2].Name)
	})
}

// =============================================================================
// Waygates Auth Tests
// =============================================================================

func TestACLRepository_WaygatesAuth(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("CRUD operations", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Waygates Auth Test")

		// Create
		auth := &models.ACLWaygatesAuth{
			ACLGroupID:   group.ID,
			Enabled:      true,
			AllowedUsers: []string{"admin", "user1"},
			SessionTTL:   3600,
		}
		err := repo.CreateWaygatesAuth(auth)
		require.NoError(t, err)
		assert.NotZero(t, auth.ID)

		// Read
		fetched, err := repo.GetWaygatesAuth(group.ID)
		require.NoError(t, err)
		assert.True(t, fetched.Enabled)
		assert.Contains(t, fetched.AllowedUsers, "admin")

		// Update
		auth.Enabled = false
		auth.Require2FA = true
		err = repo.UpdateWaygatesAuth(auth)
		require.NoError(t, err)

		updated, err := repo.GetWaygatesAuth(group.ID)
		require.NoError(t, err)
		assert.False(t, updated.Enabled)
		assert.True(t, updated.Require2FA)

		// Delete
		err = repo.DeleteWaygatesAuth(group.ID)
		require.NoError(t, err)

		_, err = repo.GetWaygatesAuth(group.ID)
		assert.Error(t, err)
	})

	t.Run("unique constraint on group", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Unique Waygates Test")

		CreateTestWaygatesAuth(t, tdb.DB, group.ID, true, 3600)

		// Try to create another for the same group
		duplicate := &models.ACLWaygatesAuth{
			ACLGroupID: group.ID,
			Enabled:    false,
			SessionTTL: 7200,
		}
		err := repo.CreateWaygatesAuth(duplicate)
		assert.Error(t, err)
	})
}

// =============================================================================
// Proxy ACL Assignment Tests
// =============================================================================

func TestACLRepository_ProxyACLAssignments(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("CRUD operations", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Assignment Test")
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "test-proxy", "test.example.com", models.ProxyTypeReverseProxy)

		// Create
		assignment := &models.ProxyACLAssignment{
			ProxyID:     proxy.ID,
			ACLGroupID:  group.ID,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
		}
		err := repo.CreateProxyACLAssignment(assignment)
		require.NoError(t, err)
		assert.NotZero(t, assignment.ID)

		// Read
		assignments, err := repo.GetProxyACLAssignments(proxy.ID)
		require.NoError(t, err)
		assert.Len(t, assignments, 1)
		assert.NotNil(t, assignments[0].ACLGroup)

		// Update
		assignment.PathPattern = "/api/*"
		assignment.Enabled = false
		err = repo.UpdateProxyACLAssignment(assignment)
		require.NoError(t, err)

		updated, err := repo.GetProxyACLAssignments(proxy.ID)
		require.NoError(t, err)
		assert.Equal(t, "/api/*", updated[0].PathPattern)
		assert.False(t, updated[0].Enabled)

		// Delete
		err = repo.DeleteProxyACLAssignment(assignment.ID)
		require.NoError(t, err)

		assignments, err = repo.GetProxyACLAssignments(proxy.ID)
		require.NoError(t, err)
		assert.Empty(t, assignments)
	})

	t.Run("delete by proxy and group", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Delete By Combo Test")
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "delete-test", "delete.example.com", models.ProxyTypeReverseProxy)

		CreateTestProxyACLAssignment(t, tdb.DB, proxy.ID, group.ID, "/*", 0)

		err := repo.DeleteProxyACLAssignmentByProxyAndGroup(proxy.ID, group.ID)
		require.NoError(t, err)

		assignments, err := repo.GetProxyACLAssignments(proxy.ID)
		require.NoError(t, err)
		assert.Empty(t, assignments)
	})

	t.Run("get by group", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group := CreateTestACLGroup(t, tdb.DB, user.ID, "Get By Group Test")
		proxy1 := CreateTestProxy(t, tdb.DB, user.ID, "proxy1", "proxy1.example.com", models.ProxyTypeReverseProxy)
		proxy2 := CreateTestProxy(t, tdb.DB, user.ID, "proxy2", "proxy2.example.com", models.ProxyTypeReverseProxy)

		CreateTestProxyACLAssignment(t, tdb.DB, proxy1.ID, group.ID, "/*", 0)
		CreateTestProxyACLAssignment(t, tdb.DB, proxy2.ID, group.ID, "/api/*", 10)

		assignments, err := repo.GetProxyACLAssignmentsByGroup(group.ID)
		require.NoError(t, err)
		assert.Len(t, assignments, 2)
		// Should preload proxy
		assert.NotNil(t, assignments[0].Proxy)
	})

	t.Run("ordered by priority", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group1 := CreateTestACLGroup(t, tdb.DB, user.ID, "Group 1")
		group2 := CreateTestACLGroup(t, tdb.DB, user.ID, "Group 2")
		group3 := CreateTestACLGroup(t, tdb.DB, user.ID, "Group 3")
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "priority-test", "priority.example.com", models.ProxyTypeReverseProxy)

		CreateTestProxyACLAssignment(t, tdb.DB, proxy.ID, group1.ID, "/high/*", 100)
		CreateTestProxyACLAssignment(t, tdb.DB, proxy.ID, group2.ID, "/low/*", 10)
		CreateTestProxyACLAssignment(t, tdb.DB, proxy.ID, group3.ID, "/medium/*", 50)

		assignments, err := repo.GetProxyACLAssignments(proxy.ID)
		require.NoError(t, err)
		assert.Len(t, assignments, 3)
		assert.Equal(t, 10, assignments[0].Priority)
		assert.Equal(t, 50, assignments[1].Priority)
		assert.Equal(t, 100, assignments[2].Priority)
	})

	t.Run("unique constraint on proxy+path", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		group1 := CreateTestACLGroup(t, tdb.DB, user.ID, "Group A")
		group2 := CreateTestACLGroup(t, tdb.DB, user.ID, "Group B")
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "unique-test", "unique.example.com", models.ProxyTypeReverseProxy)

		CreateTestProxyACLAssignment(t, tdb.DB, proxy.ID, group1.ID, "/*", 0)

		// Try to create another with same proxy+path (different group)
		duplicate := &models.ProxyACLAssignment{
			ProxyID:     proxy.ID,
			ACLGroupID:  group2.ID,
			PathPattern: "/*",
			Priority:    10,
			Enabled:     true,
		}
		err := repo.CreateProxyACLAssignment(duplicate)
		assert.Error(t, err)
	})
}

// =============================================================================
// Branding Tests
// =============================================================================

func TestACLRepository_Branding(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("get creates default", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)

		branding, err := repo.GetBranding()
		require.NoError(t, err)
		assert.NotNil(t, branding)
		assert.Equal(t, 1, branding.ID)
		assert.Equal(t, "#3b82f6", branding.PrimaryColor)
		assert.Equal(t, "#ffffff", branding.BackgroundColor)
		assert.Equal(t, "Login Required", branding.Title)
	})

	t.Run("update", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)

		// Get default
		branding, err := repo.GetBranding()
		require.NoError(t, err)

		// Update
		branding.PrimaryColor = "#ff0000"
		branding.Title = "Custom Login"
		subtitle := "Please authenticate"
		branding.Subtitle = &subtitle

		err = repo.UpdateBranding(branding)
		require.NoError(t, err)

		// Verify
		updated, err := repo.GetBranding()
		require.NoError(t, err)
		assert.Equal(t, "#ff0000", updated.PrimaryColor)
		assert.Equal(t, "Custom Login", updated.Title)
		assert.NotNil(t, updated.Subtitle)
		assert.Equal(t, "Please authenticate", *updated.Subtitle)
	})
}

// =============================================================================
// Session Tests
// =============================================================================

func TestACLRepository_Sessions(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)

	t.Run("CRUD operations", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)

		// Create
		session := &models.ACLSession{
			SessionToken: "test-token-123",
			UserID:       &user.ID,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}
		err := repo.CreateSession(session)
		require.NoError(t, err)
		assert.NotZero(t, session.ID)

		// Read
		fetched, err := repo.GetSessionByToken("test-token-123")
		require.NoError(t, err)
		assert.Equal(t, &user.ID, fetched.UserID)
		assert.NotNil(t, fetched.User)

		// Delete
		err = repo.DeleteSession("test-token-123")
		require.NoError(t, err)

		_, err = repo.GetSessionByToken("test-token-123")
		assert.Error(t, err)
	})

	t.Run("delete expired sessions", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)

		// Create expired session
		CreateTestSession(t, tdb.DB, user.ID, "expired-1", time.Now().Add(-1*time.Hour))
		CreateTestSession(t, tdb.DB, user.ID, "expired-2", time.Now().Add(-2*time.Hour))
		// Create valid session
		CreateTestSession(t, tdb.DB, user.ID, "valid-1", time.Now().Add(1*time.Hour))

		count, err := repo.DeleteExpiredSessions()
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Valid session should still exist
		_, err = repo.GetSessionByToken("valid-1")
		assert.NoError(t, err)
	})

	t.Run("delete user sessions", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user1 := CreateTestUser(t, tdb.DB)
		user2 := CreateTestUser(t, tdb.DB)

		CreateTestSession(t, tdb.DB, user1.ID, "user1-session-1", time.Now().Add(1*time.Hour))
		CreateTestSession(t, tdb.DB, user1.ID, "user1-session-2", time.Now().Add(1*time.Hour))
		CreateTestSession(t, tdb.DB, user2.ID, "user2-session-1", time.Now().Add(1*time.Hour))

		err := repo.DeleteUserSessions(user1.ID)
		require.NoError(t, err)

		// User1 sessions should be deleted
		_, err = repo.GetSessionByToken("user1-session-1")
		assert.Error(t, err)
		_, err = repo.GetSessionByToken("user1-session-2")
		assert.Error(t, err)

		// User2 session should still exist
		_, err = repo.GetSessionByToken("user2-session-1")
		assert.NoError(t, err)
	})

	t.Run("delete proxy sessions", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)
		proxy := CreateTestProxy(t, tdb.DB, user.ID, "session-proxy", "session.example.com", models.ProxyTypeReverseProxy)

		// Create session with proxy ID
		proxyID := proxy.ID
		session := &models.ACLSession{
			SessionToken: "proxy-session-1",
			UserID:       &user.ID,
			ProxyID:      &proxyID,
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		}
		err := tdb.DB.Create(session).Error
		require.NoError(t, err)

		// Create session without proxy ID
		CreateTestSession(t, tdb.DB, user.ID, "no-proxy-session", time.Now().Add(1*time.Hour))

		err = repo.DeleteProxySessions(proxy.ID)
		require.NoError(t, err)

		// Proxy session should be deleted
		_, err = repo.GetSessionByToken("proxy-session-1")
		assert.Error(t, err)

		// Non-proxy session should still exist
		_, err = repo.GetSessionByToken("no-proxy-session")
		assert.NoError(t, err)
	})

	t.Run("unique token constraint", func(t *testing.T) {
		CleanACLTables(t, tdb.DB)
		user := CreateTestUser(t, tdb.DB)

		CreateTestSession(t, tdb.DB, user.ID, "duplicate-token", time.Now().Add(1*time.Hour))

		// Try to create another with same token
		duplicate := &models.ACLSession{
			SessionToken: "duplicate-token",
			UserID:       &user.ID,
			ExpiresAt:    time.Now().Add(2 * time.Hour),
		}
		err := repo.CreateSession(duplicate)
		assert.Error(t, err)
	})
}

// =============================================================================
// Full Integration Test
// =============================================================================

func TestACLRepository_FullWorkflow(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewACLRepository(tdb.DB)
	CleanACLTables(t, tdb.DB)

	// Create test user and proxy
	user := CreateTestUser(t, tdb.DB)
	proxy := CreateTestProxy(t, tdb.DB, user.ID, "full-test-proxy", "full.example.com", models.ProxyTypeReverseProxy)

	// 1. Create ACL group
	group := &models.ACLGroup{
		Name:            "Full Workflow Group",
		CombinationMode: models.ACLCombinationModeIPBypass,
		CreatedBy:       user.ID,
	}
	err := repo.CreateGroup(group)
	require.NoError(t, err)

	// 2. Add IP rules
	ipRule := &models.ACLIPRule{
		ACLGroupID: group.ID,
		RuleType:   models.ACLIPRuleTypeBypass,
		CIDR:       "192.168.1.0/24",
		Priority:   0,
	}
	err = repo.CreateIPRule(ipRule)
	require.NoError(t, err)

	// 3. Add basic auth user
	basicAuth := &models.ACLBasicAuthUser{
		ACLGroupID:   group.ID,
		Username:     "apiuser",
		PasswordHash: "$2a$14$somehash",
	}
	err = repo.CreateBasicAuthUser(basicAuth)
	require.NoError(t, err)

	// 4. Add external provider
	extProvider := &models.ACLExternalProvider{
		ACLGroupID:   group.ID,
		ProviderType: models.ACLProviderTypeAuthelia,
		Name:         "Authelia",
		VerifyURL:    "https://auth.example.com/api/verify",
	}
	err = repo.CreateExternalProvider(extProvider)
	require.NoError(t, err)

	// 5. Configure Waygates auth
	waygatesAuth := &models.ACLWaygatesAuth{
		ACLGroupID:   group.ID,
		Enabled:      true,
		AllowedUsers: []string{"admin"},
		SessionTTL:   3600,
	}
	err = repo.CreateWaygatesAuth(waygatesAuth)
	require.NoError(t, err)

	// 6. Assign to proxy
	assignment := &models.ProxyACLAssignment{
		ProxyID:     proxy.ID,
		ACLGroupID:  group.ID,
		PathPattern: "/*",
		Priority:    0,
		Enabled:     true,
	}
	err = repo.CreateProxyACLAssignment(assignment)
	require.NoError(t, err)

	// 7. Verify complete group with all relations
	fullGroup, err := repo.GetGroupByID(group.ID)
	require.NoError(t, err)
	assert.Equal(t, "Full Workflow Group", fullGroup.Name)
	assert.Len(t, fullGroup.IPRules, 1)
	assert.Len(t, fullGroup.BasicAuthUsers, 1)
	assert.Len(t, fullGroup.ExternalProviders, 1)
	assert.NotNil(t, fullGroup.WaygatesAuth)
	assert.NotNil(t, fullGroup.Creator)

	// 8. Verify proxy assignments
	assignments, err := repo.GetProxyACLAssignments(proxy.ID)
	require.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, group.ID, assignments[0].ACLGroupID)

	// 9. Create a session
	session := &models.ACLSession{
		SessionToken: "workflow-session",
		UserID:       &user.ID,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	err = repo.CreateSession(session)
	require.NoError(t, err)

	// 10. Cleanup - delete group should cascade
	err = repo.DeleteGroup(group.ID)
	require.NoError(t, err)

	// Verify all related data is cleaned up
	_, err = repo.GetGroupByID(group.ID)
	assert.Error(t, err)

	assignments, err = repo.GetProxyACLAssignments(proxy.ID)
	require.NoError(t, err)
	assert.Empty(t, assignments)
}
