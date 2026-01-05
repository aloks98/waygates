package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/aloks98/waygates/backend/internal/models"
)

func TestNewAuditLogRepository(t *testing.T) {
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewAuditLogRepository(nil)
	if repo == nil {
		t.Fatal("Expected repository to be created")
	}
	if repo.db != nil {
		t.Error("Expected db to be nil")
	}
}

func TestAuditLogListParams_Structure(t *testing.T) {
	now := time.Now()
	userID := 1
	params := AuditLogListParams{
		Page:         1,
		Limit:        20,
		Search:       "proxy",
		Action:       "proxy.create",
		ResourceType: "proxy",
		UserID:       &userID,
		Status:       "success",
		DateFrom:     &now,
		DateTo:       &now,
		Sort:         "created_at",
		Order:        "desc",
	}

	if params.Page != 1 {
		t.Errorf("Expected Page 1, got %d", params.Page)
	}
	if params.Limit != 20 {
		t.Errorf("Expected Limit 20, got %d", params.Limit)
	}
	if params.Search != "proxy" {
		t.Errorf("Expected Search 'proxy', got '%s'", params.Search)
	}
	if params.Action != "proxy.create" {
		t.Errorf("Expected Action 'proxy.create', got '%s'", params.Action)
	}
	if params.ResourceType != "proxy" {
		t.Errorf("Expected ResourceType 'proxy', got '%s'", params.ResourceType)
	}
	if params.UserID == nil || *params.UserID != 1 {
		t.Error("Expected UserID 1")
	}
	if params.Status != "success" {
		t.Errorf("Expected Status 'success', got '%s'", params.Status)
	}
	if params.DateFrom == nil {
		t.Error("Expected DateFrom to be set")
	}
	if params.DateTo == nil {
		t.Error("Expected DateTo to be set")
	}
	if params.Sort != "created_at" {
		t.Errorf("Expected Sort 'created_at', got '%s'", params.Sort)
	}
	if params.Order != "desc" {
		t.Errorf("Expected Order 'desc', got '%s'", params.Order)
	}
}

func TestAuditLogListParams_Defaults(t *testing.T) {
	// Test zero value defaults
	params := AuditLogListParams{}

	if params.Page != 0 {
		t.Errorf("Expected default Page 0, got %d", params.Page)
	}
	if params.Limit != 0 {
		t.Errorf("Expected default Limit 0, got %d", params.Limit)
	}
	if params.Search != "" {
		t.Errorf("Expected empty Search, got '%s'", params.Search)
	}
	if params.Action != "" {
		t.Errorf("Expected empty Action, got '%s'", params.Action)
	}
	if params.ResourceType != "" {
		t.Errorf("Expected empty ResourceType, got '%s'", params.ResourceType)
	}
	if params.UserID != nil {
		t.Error("Expected UserID to be nil")
	}
	if params.Status != "" {
		t.Errorf("Expected empty Status, got '%s'", params.Status)
	}
	if params.DateFrom != nil {
		t.Error("Expected DateFrom to be nil")
	}
	if params.DateTo != nil {
		t.Error("Expected DateTo to be nil")
	}
}

// TestAuditLogList_SQLInjectionPrevented tests that SQL injection attempts are blocked
func TestAuditLogList_SQLInjectionPrevented(t *testing.T) {
	testCases := []struct {
		name          string
		sortInput     string
		expectedField string
		orderInput    string
		expectedOrder string
	}{
		{
			name:          "Valid sort field - action",
			sortInput:     "action",
			expectedField: "action",
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "Valid sort field - created_at",
			sortInput:     "created_at",
			expectedField: "created_at",
			orderInput:    "desc",
			expectedOrder: "DESC",
		},
		{
			name:          "SQL injection in sort - DROP TABLE",
			sortInput:     "action; DROP TABLE audit_logs;--",
			expectedField: "created_at", // Should fallback to default
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in sort - UNION SELECT",
			sortInput:     "action UNION SELECT * FROM users--",
			expectedField: "created_at", // Should fallback to default
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in order",
			sortInput:     "action",
			expectedField: "action",
			orderInput:    "ASC; DELETE FROM audit_logs;--",
			expectedOrder: "DESC", // Should fallback to default
		},
		{
			name:          "Case insensitive sort field",
			sortInput:     "ACTION",
			expectedField: "action",
			orderInput:    "DESC",
			expectedOrder: "DESC",
		},
		{
			name:          "Invalid sort field - fallback to default",
			sortInput:     "password_hash",
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
				if validField, ok := auditLogAllowedSortFields[strings.ToLower(tc.sortInput)]; ok {
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

// TestAuditLogAllowedSortFields verifies the whitelist is correctly defined
func TestAuditLogAllowedSortFields(t *testing.T) {
	expectedFields := []string{"id", "action", "resource_type", "resource_name", "status", "created_at", "user_id"}

	for _, field := range expectedFields {
		if _, ok := auditLogAllowedSortFields[field]; !ok {
			t.Errorf("Expected field '%s' to be in auditLogAllowedSortFields", field)
		}
	}

	// Verify dangerous fields are NOT in the whitelist
	dangerousFields := []string{"ip_address", "user_agent", "details", "error_message"}
	for _, field := range dangerousFields {
		if _, ok := auditLogAllowedSortFields[field]; ok {
			t.Errorf("Field '%s' should NOT be in auditLogAllowedSortFields (potential data exposure)", field)
		}
	}
}

// TestAuditLogStats_Structure tests the stats struct
func TestAuditLogStats_Structure(t *testing.T) {
	stats := models.AuditLogStats{
		TotalLogs:      100,
		ByAction:       make(map[string]int64),
		ByStatus:       make(map[string]int64),
		ByResourceType: make(map[string]int64),
		RecentActivity: []models.AuditLog{},
	}

	if stats.TotalLogs != 100 {
		t.Errorf("Expected TotalLogs 100, got %d", stats.TotalLogs)
	}

	// Test ByAction map
	stats.ByAction["proxy.create"] = 25
	stats.ByAction["proxy.update"] = 30
	stats.ByAction["auth.login"] = 45

	if stats.ByAction["proxy.create"] != 25 {
		t.Errorf("Expected ByAction['proxy.create'] 25, got %d", stats.ByAction["proxy.create"])
	}

	// Test ByStatus map
	stats.ByStatus["success"] = 90
	stats.ByStatus["failure"] = 10

	if stats.ByStatus["success"] != 90 {
		t.Errorf("Expected ByStatus['success'] 90, got %d", stats.ByStatus["success"])
	}

	// Test ByResourceType map
	stats.ByResourceType["proxy"] = 55
	stats.ByResourceType["user"] = 45

	if stats.ByResourceType["proxy"] != 55 {
		t.Errorf("Expected ByResourceType['proxy'] 55, got %d", stats.ByResourceType["proxy"])
	}
}

// TestAuditLogModel_Pointers tests that pointer fields work correctly
func TestAuditLogModel_Pointers(t *testing.T) {
	// Test with nil pointers
	log := models.AuditLog{
		ID:        1,
		Action:    "proxy.create",
		Status:    "success",
		CreatedAt: time.Now(),
	}

	if log.UserID != nil {
		t.Error("Expected UserID to be nil")
	}
	if log.ResourceType != nil {
		t.Error("Expected ResourceType to be nil")
	}
	if log.ResourceID != nil {
		t.Error("Expected ResourceID to be nil")
	}
	if log.ResourceName != nil {
		t.Error("Expected ResourceName to be nil")
	}
	if log.IPAddress != nil {
		t.Error("Expected IPAddress to be nil")
	}
	if log.UserAgent != nil {
		t.Error("Expected UserAgent to be nil")
	}
	if log.ErrorMessage != nil {
		t.Error("Expected ErrorMessage to be nil")
	}

	// Test with values
	userID := 1
	resourceType := "proxy"
	resourceID := 5
	resourceName := "My Proxy"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	errorMessage := "Some error"

	logWithValues := models.AuditLog{
		ID:           2,
		UserID:       &userID,
		Action:       "proxy.update",
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		ResourceName: &resourceName,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Status:       "failure",
		ErrorMessage: &errorMessage,
		CreatedAt:    time.Now(),
	}

	if logWithValues.UserID == nil || *logWithValues.UserID != 1 {
		t.Error("Expected UserID to be 1")
	}
	if logWithValues.ResourceType == nil || *logWithValues.ResourceType != "proxy" {
		t.Error("Expected ResourceType to be 'proxy'")
	}
	if logWithValues.ResourceID == nil || *logWithValues.ResourceID != 5 {
		t.Error("Expected ResourceID to be 5")
	}
	if logWithValues.ResourceName == nil || *logWithValues.ResourceName != "My Proxy" {
		t.Error("Expected ResourceName to be 'My Proxy'")
	}
	if logWithValues.ErrorMessage == nil || *logWithValues.ErrorMessage != "Some error" {
		t.Error("Expected ErrorMessage to be 'Some error'")
	}
}

// TestAuditConfig tests the audit configuration struct
func TestAuditConfig(t *testing.T) {
	// Test default config
	config := models.DefaultAuditConfig()

	if !config.ProxyCreate {
		t.Error("Expected ProxyCreate to be true by default")
	}
	if !config.AuthLogin {
		t.Error("Expected AuthLogin to be true by default")
	}
	if !config.SettingsUpdate {
		t.Error("Expected SettingsUpdate to be true by default")
	}
	if !config.SyncStarted {
		t.Error("Expected SyncStarted to be true by default")
	}
	if !config.SystemStartup {
		t.Error("Expected SystemStartup to be true by default")
	}

	// Test custom config
	customConfig := models.AuditConfig{
		ProxyCreate:        true,
		ProxyUpdate:        true,
		ProxyDelete:        true,
		ProxyEnable:        true,
		ProxyDisable:       true,
		AuthLogin:          false,
		AuthLogout:         false,
		AuthRegister:       false,
		AuthPasswordChange: false,
		AuthLoginFailed:    false,
		SettingsUpdate:     true,
		SyncStarted:        false,
		SyncCompleted:      false,
		SyncFailed:         false,
		SystemStartup:      true,
		CaddyReload:        true,
	}

	if !customConfig.ProxyCreate {
		t.Error("Expected ProxyCreate to be true")
	}
	if customConfig.AuthLogin {
		t.Error("Expected AuthLogin to be false")
	}
	if !customConfig.SettingsUpdate {
		t.Error("Expected SettingsUpdate to be true")
	}
	if customConfig.SyncStarted {
		t.Error("Expected SyncStarted to be false")
	}
	if !customConfig.SystemStartup {
		t.Error("Expected SystemStartup to be true")
	}
}

// TestAuditEvent tests the audit event struct
func TestAuditEvent(t *testing.T) {
	userID := 1
	resourceID := 5

	event := models.AuditEvent{
		UserID:       &userID,
		Action:       "proxy.create",
		ResourceType: "proxy",
		ResourceID:   &resourceID,
		ResourceName: "Test Proxy",
		Details: map[string]interface{}{
			"hostname": "test.example.com",
			"type":     "reverse_proxy",
		},
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		Status:       "success",
		ErrorMessage: "",
	}

	if event.UserID == nil || *event.UserID != 1 {
		t.Error("Expected UserID to be 1")
	}
	if event.Action != "proxy.create" {
		t.Errorf("Expected Action 'proxy.create', got '%s'", event.Action)
	}
	if event.ResourceType != "proxy" {
		t.Errorf("Expected ResourceType 'proxy', got '%s'", event.ResourceType)
	}
	if event.ResourceID == nil || *event.ResourceID != 5 {
		t.Error("Expected ResourceID to be 5")
	}
	if event.ResourceName != "Test Proxy" {
		t.Errorf("Expected ResourceName 'Test Proxy', got '%s'", event.ResourceName)
	}
	if event.Details["hostname"] != "test.example.com" {
		t.Error("Expected Details['hostname'] to be 'test.example.com'")
	}
	if event.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IPAddress '192.168.1.1', got '%s'", event.IPAddress)
	}
	if event.UserAgent != "Mozilla/5.0" {
		t.Errorf("Expected UserAgent 'Mozilla/5.0', got '%s'", event.UserAgent)
	}
	if event.Status != "success" {
		t.Errorf("Expected Status 'success', got '%s'", event.Status)
	}
}

// TestAuditActionConstants tests that all action constants are defined
func TestAuditActionConstants(t *testing.T) {
	// Proxy actions
	if models.AuditActionProxyCreate != "proxy.create" {
		t.Errorf("Expected AuditActionProxyCreate 'proxy.create', got '%s'", models.AuditActionProxyCreate)
	}
	if models.AuditActionProxyUpdate != "proxy.update" {
		t.Errorf("Expected AuditActionProxyUpdate 'proxy.update', got '%s'", models.AuditActionProxyUpdate)
	}
	if models.AuditActionProxyDelete != "proxy.delete" {
		t.Errorf("Expected AuditActionProxyDelete 'proxy.delete', got '%s'", models.AuditActionProxyDelete)
	}
	if models.AuditActionProxyEnable != "proxy.enable" {
		t.Errorf("Expected AuditActionProxyEnable 'proxy.enable', got '%s'", models.AuditActionProxyEnable)
	}
	if models.AuditActionProxyDisable != "proxy.disable" {
		t.Errorf("Expected AuditActionProxyDisable 'proxy.disable', got '%s'", models.AuditActionProxyDisable)
	}

	// Auth actions
	if models.AuditActionAuthLogin != "auth.login" {
		t.Errorf("Expected AuditActionAuthLogin 'auth.login', got '%s'", models.AuditActionAuthLogin)
	}
	if models.AuditActionAuthLogout != "auth.logout" {
		t.Errorf("Expected AuditActionAuthLogout 'auth.logout', got '%s'", models.AuditActionAuthLogout)
	}
	if models.AuditActionAuthRegister != "auth.register" {
		t.Errorf("Expected AuditActionAuthRegister 'auth.register', got '%s'", models.AuditActionAuthRegister)
	}
	if models.AuditActionAuthPasswordChange != "auth.password_change" {
		t.Errorf("Expected AuditActionAuthPasswordChange 'auth.password_change', got '%s'", models.AuditActionAuthPasswordChange)
	}
	if models.AuditActionAuthLoginFailed != "auth.login_failed" {
		t.Errorf("Expected AuditActionAuthLoginFailed 'auth.login_failed', got '%s'", models.AuditActionAuthLoginFailed)
	}

	// Settings actions
	if models.AuditActionSettingsUpdate != "settings.update" {
		t.Errorf("Expected AuditActionSettingsUpdate 'settings.update', got '%s'", models.AuditActionSettingsUpdate)
	}

	// Sync actions
	if models.AuditActionSyncStarted != "sync.started" {
		t.Errorf("Expected AuditActionSyncStarted 'sync.started', got '%s'", models.AuditActionSyncStarted)
	}
	if models.AuditActionSyncCompleted != "sync.completed" {
		t.Errorf("Expected AuditActionSyncCompleted 'sync.completed', got '%s'", models.AuditActionSyncCompleted)
	}
	if models.AuditActionSyncFailed != "sync.failed" {
		t.Errorf("Expected AuditActionSyncFailed 'sync.failed', got '%s'", models.AuditActionSyncFailed)
	}

	// System actions
	if models.AuditActionSystemStartup != "system.startup" {
		t.Errorf("Expected AuditActionSystemStartup 'system.startup', got '%s'", models.AuditActionSystemStartup)
	}
	if models.AuditActionCaddyReload != "caddy.reload" {
		t.Errorf("Expected AuditActionCaddyReload 'caddy.reload', got '%s'", models.AuditActionCaddyReload)
	}
}

// TestAuditStatusConstants tests that all status constants are defined
func TestAuditStatusConstants(t *testing.T) {
	if models.AuditStatusSuccess != "success" {
		t.Errorf("Expected AuditStatusSuccess 'success', got '%s'", models.AuditStatusSuccess)
	}
	if models.AuditStatusFailure != "failure" {
		t.Errorf("Expected AuditStatusFailure 'failure', got '%s'", models.AuditStatusFailure)
	}
}

// TestAuditResourceTypeConstants tests that all resource type constants are defined
func TestAuditResourceTypeConstants(t *testing.T) {
	if models.AuditResourceTypeProxy != "proxy" {
		t.Errorf("Expected AuditResourceTypeProxy 'proxy', got '%s'", models.AuditResourceTypeProxy)
	}
	if models.AuditResourceTypeUser != "user" {
		t.Errorf("Expected AuditResourceTypeUser 'user', got '%s'", models.AuditResourceTypeUser)
	}
	if models.AuditResourceTypeSettings != "settings" {
		t.Errorf("Expected AuditResourceTypeSettings 'settings', got '%s'", models.AuditResourceTypeSettings)
	}
	if models.AuditResourceTypeSystem != "system" {
		t.Errorf("Expected AuditResourceTypeSystem 'system', got '%s'", models.AuditResourceTypeSystem)
	}
}

// TestAuditLogTableName tests the table name
func TestAuditLogTableName(t *testing.T) {
	log := models.AuditLog{}
	if log.TableName() != "audit_logs" {
		t.Errorf("Expected table name 'audit_logs', got '%s'", log.TableName())
	}
}

// TestAuditLogListResponse tests the list response struct
func TestAuditLogListResponse(t *testing.T) {
	response := models.AuditLogListResponse{
		Items: []models.AuditLog{
			{ID: 1, Action: "proxy.create", Status: "success"},
			{ID: 2, Action: "proxy.update", Status: "success"},
		},
		Total:      100,
		Page:       1,
		Limit:      20,
		TotalPages: 5,
	}

	if len(response.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(response.Items))
	}
	if response.Total != 100 {
		t.Errorf("Expected Total 100, got %d", response.Total)
	}
	if response.Page != 1 {
		t.Errorf("Expected Page 1, got %d", response.Page)
	}
	if response.Limit != 20 {
		t.Errorf("Expected Limit 20, got %d", response.Limit)
	}
	if response.TotalPages != 5 {
		t.Errorf("Expected TotalPages 5, got %d", response.TotalPages)
	}
}

// TestSettingAuditConfigKey tests the settings key constant
func TestSettingAuditConfigKey(t *testing.T) {
	if models.SettingAuditConfig != "audit_config" {
		t.Errorf("Expected SettingAuditConfig 'audit_config', got '%s'", models.SettingAuditConfig)
	}
}
