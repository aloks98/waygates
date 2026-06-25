package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuditLog_TableName verifies the table name
func TestAuditLog_TableName(t *testing.T) {
	t.Parallel()
	a := AuditLog{}
	assert.Equal(t, "audit_logs", a.TableName())
}

// TestDefaultAuditConfig tests the DefaultAuditConfig function
func TestDefaultAuditConfig(t *testing.T) {
	t.Parallel()

	config := DefaultAuditConfig()

	require.NotNil(t, config)

	// Verify all proxy events are enabled
	assert.True(t, config.ProxyCreate, "ProxyCreate should be enabled by default")
	assert.True(t, config.ProxyUpdate, "ProxyUpdate should be enabled by default")
	assert.True(t, config.ProxyDelete, "ProxyDelete should be enabled by default")
	assert.True(t, config.ProxyEnable, "ProxyEnable should be enabled by default")
	assert.True(t, config.ProxyDisable, "ProxyDisable should be enabled by default")

	// Verify all auth events are enabled
	assert.True(t, config.AuthLogin, "AuthLogin should be enabled by default")
	assert.True(t, config.AuthLogout, "AuthLogout should be enabled by default")
	assert.True(t, config.AuthRegister, "AuthRegister should be enabled by default")
	assert.True(t, config.AuthPasswordChange, "AuthPasswordChange should be enabled by default")
	assert.True(t, config.AuthLoginFailed, "AuthLoginFailed should be enabled by default")

	// Verify settings events are enabled
	assert.True(t, config.SettingsUpdate, "SettingsUpdate should be enabled by default")

	// Verify sync events are enabled
	assert.True(t, config.SyncStarted, "SyncStarted should be enabled by default")
	assert.True(t, config.SyncCompleted, "SyncCompleted should be enabled by default")
	assert.True(t, config.SyncFailed, "SyncFailed should be enabled by default")

	// Verify system events are enabled
	assert.True(t, config.SystemStartup, "SystemStartup should be enabled by default")
	assert.True(t, config.CaddyReload, "CaddyReload should be enabled by default")
}

// TestDefaultAuditConfig_ReturnsNewInstance tests that each call returns a new instance
func TestDefaultAuditConfig_ReturnsNewInstance(t *testing.T) {
	t.Parallel()

	config1 := DefaultAuditConfig()
	config2 := DefaultAuditConfig()

	// Should be different pointers
	assert.NotSame(t, config1, config2)

	// Modifying one should not affect the other
	config1.ProxyCreate = false
	assert.True(t, config2.ProxyCreate)
}

// TestGetAuditEventGroups tests the GetAuditEventGroups function
func TestGetAuditEventGroups(t *testing.T) {
	t.Parallel()

	groups := GetAuditEventGroups()

	require.NotNil(t, groups)
	require.Len(t, groups, 6, "Should have 6 event groups")

	// Test expected group keys
	expectedGroups := []string{"proxy", "auth", "settings", "sync", "system", "acl"}
	for i, expectedKey := range expectedGroups {
		assert.Equal(t, expectedKey, groups[i].Key)
	}
}

// TestGetAuditEventGroups_ProxyGroup tests the proxy event group
func TestGetAuditEventGroups_ProxyGroup(t *testing.T) {
	t.Parallel()

	groups := GetAuditEventGroups()
	proxyGroup := groups[0]

	assert.Equal(t, "proxy", proxyGroup.Key)
	assert.Equal(t, "Proxy Events", proxyGroup.Label)
	assert.Equal(t, "Proxy configuration changes", proxyGroup.Description)

	require.Len(t, proxyGroup.Events, 5)

	expectedEvents := []struct {
		key   string
		label string
	}{
		{"proxy_create", "Create"},
		{"proxy_update", "Update"},
		{"proxy_delete", "Delete"},
		{"proxy_enable", "Enable"},
		{"proxy_disable", "Disable"},
	}

	for i, expected := range expectedEvents {
		assert.Equal(t, expected.key, proxyGroup.Events[i].Key)
		assert.Equal(t, expected.label, proxyGroup.Events[i].Label)
	}
}

// TestGetAuditEventGroups_AuthGroup tests the auth event group
func TestGetAuditEventGroups_AuthGroup(t *testing.T) {
	t.Parallel()

	groups := GetAuditEventGroups()
	authGroup := groups[1]

	assert.Equal(t, "auth", authGroup.Key)
	assert.Equal(t, "Authentication Events", authGroup.Label)
	assert.Equal(t, "User authentication activities", authGroup.Description)

	require.Len(t, authGroup.Events, 5)

	expectedEvents := []struct {
		key   string
		label string
	}{
		{"auth_login", "Login"},
		{"auth_logout", "Logout"},
		{"auth_register", "Register"},
		{"auth_password_change", "Password Change"},
		{"auth_login_failed", "Failed Login"},
	}

	for i, expected := range expectedEvents {
		assert.Equal(t, expected.key, authGroup.Events[i].Key)
		assert.Equal(t, expected.label, authGroup.Events[i].Label)
	}
}

// TestGetAuditEventGroups_SettingsGroup tests the settings event group
func TestGetAuditEventGroups_SettingsGroup(t *testing.T) {
	t.Parallel()

	groups := GetAuditEventGroups()
	settingsGroup := groups[2]

	assert.Equal(t, "settings", settingsGroup.Key)
	assert.Equal(t, "Settings Events", settingsGroup.Label)
	assert.Equal(t, "Configuration changes", settingsGroup.Description)

	require.Len(t, settingsGroup.Events, 1)
	assert.Equal(t, "settings_update", settingsGroup.Events[0].Key)
	assert.Equal(t, "Settings Update", settingsGroup.Events[0].Label)
}

// TestGetAuditEventGroups_SyncGroup tests the sync event group
func TestGetAuditEventGroups_SyncGroup(t *testing.T) {
	t.Parallel()

	groups := GetAuditEventGroups()
	syncGroup := groups[3]

	assert.Equal(t, "sync", syncGroup.Key)
	assert.Equal(t, "Sync Events", syncGroup.Label)
	assert.Equal(t, "Caddy synchronization operations", syncGroup.Description)

	require.Len(t, syncGroup.Events, 3)

	expectedEvents := []struct {
		key   string
		label string
	}{
		{"sync_started", "Started"},
		{"sync_completed", "Completed"},
		{"sync_failed", "Failed"},
	}

	for i, expected := range expectedEvents {
		assert.Equal(t, expected.key, syncGroup.Events[i].Key)
		assert.Equal(t, expected.label, syncGroup.Events[i].Label)
	}
}

// TestGetAuditEventGroups_SystemGroup tests the system event group
func TestGetAuditEventGroups_SystemGroup(t *testing.T) {
	t.Parallel()

	groups := GetAuditEventGroups()
	systemGroup := groups[4]

	assert.Equal(t, "system", systemGroup.Key)
	assert.Equal(t, "System Events", systemGroup.Label)
	assert.Equal(t, "System and Caddy operations", systemGroup.Description)

	require.Len(t, systemGroup.Events, 2)

	expectedEvents := []struct {
		key   string
		label string
	}{
		{"system_startup", "System Startup"},
		{"caddy_reload", "Caddy Reload"},
	}

	for i, expected := range expectedEvents {
		assert.Equal(t, expected.key, systemGroup.Events[i].Key)
		assert.Equal(t, expected.label, systemGroup.Events[i].Label)
	}
}

// TestGetAuditEventGroups_ReturnsSameData tests that each call returns consistent data
func TestGetAuditEventGroups_ReturnsSameData(t *testing.T) {
	t.Parallel()

	groups1 := GetAuditEventGroups()
	groups2 := GetAuditEventGroups()

	require.Len(t, groups1, len(groups2))

	for i := range groups1 {
		assert.Equal(t, groups1[i].Key, groups2[i].Key)
		assert.Equal(t, groups1[i].Label, groups2[i].Label)
		assert.Equal(t, groups1[i].Description, groups2[i].Description)
		assert.Len(t, groups1[i].Events, len(groups2[i].Events))
	}
}

// TestAuditActionConstants tests the audit action constant values
func TestAuditActionConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		constant string
		expected string
	}{
		// Proxy actions
		{"AuditActionProxyCreate", AuditActionProxyCreate, "proxy.create"},
		{"AuditActionProxyUpdate", AuditActionProxyUpdate, "proxy.update"},
		{"AuditActionProxyDelete", AuditActionProxyDelete, "proxy.delete"},
		{"AuditActionProxyEnable", AuditActionProxyEnable, "proxy.enable"},
		{"AuditActionProxyDisable", AuditActionProxyDisable, "proxy.disable"},

		// Auth actions
		{"AuditActionAuthLogin", AuditActionAuthLogin, "auth.login"},
		{"AuditActionAuthLogout", AuditActionAuthLogout, "auth.logout"},
		{"AuditActionAuthRegister", AuditActionAuthRegister, "auth.register"},
		{"AuditActionAuthPasswordChange", AuditActionAuthPasswordChange, "auth.password_change"},
		{"AuditActionAuthLoginFailed", AuditActionAuthLoginFailed, "auth.login_failed"},

		// Settings actions
		{"AuditActionSettingsUpdate", AuditActionSettingsUpdate, "settings.update"},

		// Sync actions
		{"AuditActionSyncStarted", AuditActionSyncStarted, "sync.started"},
		{"AuditActionSyncCompleted", AuditActionSyncCompleted, "sync.completed"},
		{"AuditActionSyncFailed", AuditActionSyncFailed, "sync.failed"},

		// System actions
		{"AuditActionSystemStartup", AuditActionSystemStartup, "system.startup"},
		{"AuditActionCaddyReload", AuditActionCaddyReload, "caddy.reload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

// TestAuditStatusConstants tests the audit status constant values
func TestAuditStatusConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "success", AuditStatusSuccess)
	assert.Equal(t, "failure", AuditStatusFailure)
}

// TestAuditResourceTypeConstants tests the resource type constant values
func TestAuditResourceTypeConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "proxy", AuditResourceTypeProxy)
	assert.Equal(t, "user", AuditResourceTypeUser)
	assert.Equal(t, "settings", AuditResourceTypeSettings)
	assert.Equal(t, "system", AuditResourceTypeSystem)
}

// TestSettingAuditConfigConstant tests the setting key constant
func TestSettingAuditConfigConstant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "audit_config", SettingAuditConfig)
}

// TestAuditLog_StructFields tests AuditLog struct field assignment
func TestAuditLog_StructFields(t *testing.T) {
	t.Parallel()

	userID := 1
	resourceType := "proxy"
	resourceID := 42
	resourceName := "My Proxy"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	errorMessage := "Some error"

	log := AuditLog{
		ID:           1,
		UserID:       &userID,
		Action:       AuditActionProxyCreate,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		ResourceName: &resourceName,
		Details:      JSONField{"key": "value"},
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Status:       AuditStatusSuccess,
		ErrorMessage: &errorMessage,
	}

	assert.Equal(t, 1, log.ID)
	assert.Equal(t, &userID, log.UserID)
	assert.Equal(t, AuditActionProxyCreate, log.Action)
	assert.Equal(t, &resourceType, log.ResourceType)
	assert.Equal(t, &resourceID, log.ResourceID)
	assert.Equal(t, &resourceName, log.ResourceName)
	assert.Equal(t, JSONField{"key": "value"}, log.Details)
	assert.Equal(t, &ipAddress, log.IPAddress)
	assert.Equal(t, &userAgent, log.UserAgent)
	assert.Equal(t, AuditStatusSuccess, log.Status)
	assert.Equal(t, &errorMessage, log.ErrorMessage)
}

// TestAuditLog_NilFields tests AuditLog with nil optional fields
func TestAuditLog_NilFields(t *testing.T) {
	t.Parallel()

	log := AuditLog{
		ID:     1,
		Action: AuditActionSystemStartup,
		Status: AuditStatusSuccess,
	}

	assert.Equal(t, 1, log.ID)
	assert.Nil(t, log.UserID)
	assert.Equal(t, AuditActionSystemStartup, log.Action)
	assert.Nil(t, log.ResourceType)
	assert.Nil(t, log.ResourceID)
	assert.Nil(t, log.ResourceName)
	assert.Nil(t, log.IPAddress)
	assert.Nil(t, log.UserAgent)
	assert.Equal(t, AuditStatusSuccess, log.Status)
	assert.Nil(t, log.ErrorMessage)
}

// TestAuditLogListResponse tests AuditLogListResponse struct
func TestAuditLogListResponse(t *testing.T) {
	t.Parallel()

	resp := AuditLogListResponse{
		Items:      []AuditLog{},
		Total:      100,
		Page:       2,
		Limit:      25,
		TotalPages: 4,
	}

	assert.Empty(t, resp.Items)
	assert.Equal(t, int64(100), resp.Total)
	assert.Equal(t, 2, resp.Page)
	assert.Equal(t, 25, resp.Limit)
	assert.Equal(t, 4, resp.TotalPages)
}

// TestAuditLogStats tests AuditLogStats struct
func TestAuditLogStats(t *testing.T) {
	t.Parallel()

	stats := AuditLogStats{
		TotalLogs: 1000,
		ByAction: map[string]int64{
			"proxy.create": 100,
			"proxy.update": 200,
		},
		ByStatus: map[string]int64{
			"success": 900,
			"failure": 100,
		},
		ByResourceType: map[string]int64{
			"proxy": 500,
			"user":  300,
		},
		RecentActivity: []AuditLog{},
	}

	assert.Equal(t, int64(1000), stats.TotalLogs)
	assert.Equal(t, int64(100), stats.ByAction["proxy.create"])
	assert.Equal(t, int64(200), stats.ByAction["proxy.update"])
	assert.Equal(t, int64(900), stats.ByStatus["success"])
	assert.Equal(t, int64(100), stats.ByStatus["failure"])
	assert.Equal(t, int64(500), stats.ByResourceType["proxy"])
	assert.Equal(t, int64(300), stats.ByResourceType["user"])
	assert.Empty(t, stats.RecentActivity)
}

// TestAuditConfig_StructFields tests AuditConfig struct field assignment
func TestAuditConfig_StructFields(t *testing.T) {
	t.Parallel()

	config := AuditConfig{
		ProxyCreate:        true,
		ProxyUpdate:        false,
		ProxyDelete:        true,
		ProxyEnable:        false,
		ProxyDisable:       true,
		AuthLogin:          true,
		AuthLogout:         false,
		AuthRegister:       true,
		AuthPasswordChange: false,
		AuthLoginFailed:    true,
		SettingsUpdate:     false,
		SyncStarted:        true,
		SyncCompleted:      false,
		SyncFailed:         true,
		SystemStartup:      false,
		CaddyReload:        true,
	}

	assert.True(t, config.ProxyCreate)
	assert.False(t, config.ProxyUpdate)
	assert.True(t, config.ProxyDelete)
	assert.False(t, config.ProxyEnable)
	assert.True(t, config.ProxyDisable)
	assert.True(t, config.AuthLogin)
	assert.False(t, config.AuthLogout)
	assert.True(t, config.AuthRegister)
	assert.False(t, config.AuthPasswordChange)
	assert.True(t, config.AuthLoginFailed)
	assert.False(t, config.SettingsUpdate)
	assert.True(t, config.SyncStarted)
	assert.False(t, config.SyncCompleted)
	assert.True(t, config.SyncFailed)
	assert.False(t, config.SystemStartup)
	assert.True(t, config.CaddyReload)
}

// TestAuditEvent_StructFields tests AuditEvent struct field assignment
func TestAuditEvent_StructFields(t *testing.T) {
	t.Parallel()

	userID := 1
	resourceID := 42

	event := AuditEvent{
		UserID:       &userID,
		Action:       AuditActionProxyCreate,
		ResourceType: AuditResourceTypeProxy,
		ResourceID:   &resourceID,
		ResourceName: "My Proxy",
		Details: map[string]interface{}{
			"hostname": "example.com",
		},
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		Status:       AuditStatusSuccess,
		ErrorMessage: "",
	}

	assert.Equal(t, &userID, event.UserID)
	assert.Equal(t, AuditActionProxyCreate, event.Action)
	assert.Equal(t, AuditResourceTypeProxy, event.ResourceType)
	assert.Equal(t, &resourceID, event.ResourceID)
	assert.Equal(t, "My Proxy", event.ResourceName)
	assert.Equal(t, "example.com", event.Details["hostname"])
	assert.Equal(t, "192.168.1.1", event.IPAddress)
	assert.Equal(t, "Mozilla/5.0", event.UserAgent)
	assert.Equal(t, AuditStatusSuccess, event.Status)
	assert.Empty(t, event.ErrorMessage)
}

// TestAuditEventDefinition_StructFields tests AuditEventDefinition struct
func TestAuditEventDefinition_StructFields(t *testing.T) {
	t.Parallel()

	def := AuditEventDefinition{
		Key:   "proxy_create",
		Label: "Create",
	}

	assert.Equal(t, "proxy_create", def.Key)
	assert.Equal(t, "Create", def.Label)
}

// TestAuditEventGroup_StructFields tests AuditEventGroup struct
func TestAuditEventGroup_StructFields(t *testing.T) {
	t.Parallel()

	group := AuditEventGroup{
		Key:         "proxy",
		Label:       "Proxy Events",
		Description: "Proxy configuration changes",
		Events: []AuditEventDefinition{
			{Key: "proxy_create", Label: "Create"},
		},
	}

	assert.Equal(t, "proxy", group.Key)
	assert.Equal(t, "Proxy Events", group.Label)
	assert.Equal(t, "Proxy configuration changes", group.Description)
	require.Len(t, group.Events, 1)
	assert.Equal(t, "proxy_create", group.Events[0].Key)
}

// TestGetAuditEventGroups_TotalEvents tests the total number of events
func TestGetAuditEventGroups_TotalEvents(t *testing.T) {
	t.Parallel()

	groups := GetAuditEventGroups()

	totalEvents := 0
	for _, group := range groups {
		totalEvents += len(group.Events)
	}

	// 5 proxy + 5 auth + 1 settings + 3 sync + 2 system + 18 acl = 34 events
	assert.Equal(t, 34, totalEvents)
}
