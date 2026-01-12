package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

// TestNewACLRepository tests ACL repository creation
func TestNewACLRepository(t *testing.T) {
	t.Parallel()

	repo := NewACLRepository(nil)
	require.NotNil(t, repo, "Expected repository to be created")
	require.Nil(t, repo.db, "Expected db to be nil")
}

// TestACLGroupListParams_Structure tests the ACLGroupListParams struct
func TestACLGroupListParams_Structure(t *testing.T) {
	t.Parallel()

	params := ACLGroupListParams{
		Page:   1,
		Limit:  20,
		Search: "admin",
		Sort:   "name",
		Order:  "asc",
	}

	if params.Page != 1 {
		t.Errorf("Expected Page 1, got %d", params.Page)
	}
	if params.Limit != 20 {
		t.Errorf("Expected Limit 20, got %d", params.Limit)
	}
	if params.Search != "admin" {
		t.Errorf("Expected Search 'admin', got '%s'", params.Search)
	}
	if params.Sort != "name" {
		t.Errorf("Expected Sort 'name', got '%s'", params.Sort)
	}
	if params.Order != "asc" {
		t.Errorf("Expected Order 'asc', got '%s'", params.Order)
	}
}

// TestACLGroupListParams_Defaults tests zero value defaults
func TestACLGroupListParams_Defaults(t *testing.T) {
	t.Parallel()

	params := ACLGroupListParams{}

	if params.Page != 0 {
		t.Errorf("Expected default Page 0, got %d", params.Page)
	}
	if params.Limit != 0 {
		t.Errorf("Expected default Limit 0, got %d", params.Limit)
	}
	if params.Search != "" {
		t.Errorf("Expected empty Search, got '%s'", params.Search)
	}
	if params.Sort != "" {
		t.Errorf("Expected empty Sort, got '%s'", params.Sort)
	}
	if params.Order != "" {
		t.Errorf("Expected empty Order, got '%s'", params.Order)
	}
}

// TestACLGroupList_SQLInjectionPrevented tests that SQL injection attempts are blocked
func TestACLGroupList_SQLInjectionPrevented(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		sortInput     string
		expectedField string
		orderInput    string
		expectedOrder string
	}{
		{
			name:          "Valid sort field - name",
			sortInput:     "name",
			expectedField: "name",
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
			name:          "Valid sort field - updated_at",
			sortInput:     "updated_at",
			expectedField: "updated_at",
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in sort - DROP TABLE",
			sortInput:     "name; DROP TABLE acl_groups;--",
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
			name:          "SQL injection in order",
			sortInput:     "name",
			expectedField: "name",
			orderInput:    "ASC; DELETE FROM acl_groups;--",
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
				if validField, ok := aclGroupAllowedSortFields[strings.ToLower(tc.sortInput)]; ok {
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

// TestACLGroupAllowedSortFields verifies the whitelist is correctly defined
func TestACLGroupAllowedSortFields(t *testing.T) {
	t.Parallel()

	expectedFields := []string{"id", "name", "created_at", "updated_at"}

	for _, field := range expectedFields {
		if _, ok := aclGroupAllowedSortFields[field]; !ok {
			t.Errorf("Expected field '%s' to be in aclGroupAllowedSortFields", field)
		}
	}

	// Verify dangerous fields are NOT in the whitelist
	dangerousFields := []string{"password", "password_hash", "secret", "token", "created_by"}
	for _, field := range dangerousFields {
		if _, ok := aclGroupAllowedSortFields[field]; ok {
			t.Errorf("Dangerous field '%s' should NOT be in aclGroupAllowedSortFields", field)
		}
	}
}

// TestACLGroup_TableName tests the table name
func TestACLGroup_TableName(t *testing.T) {
	t.Parallel()

	group := models.ACLGroup{}
	if group.TableName() != "acl_groups" {
		t.Errorf("Expected table name 'acl_groups', got '%s'", group.TableName())
	}
}

// TestACLIPRule_TableName tests the table name
func TestACLIPRule_TableName(t *testing.T) {
	t.Parallel()

	rule := models.ACLIPRule{}
	if rule.TableName() != "acl_ip_rules" {
		t.Errorf("Expected table name 'acl_ip_rules', got '%s'", rule.TableName())
	}
}

// TestACLBasicAuthUser_TableName tests the table name
func TestACLBasicAuthUser_TableName(t *testing.T) {
	t.Parallel()

	user := models.ACLBasicAuthUser{}
	if user.TableName() != "acl_basic_auth_users" {
		t.Errorf("Expected table name 'acl_basic_auth_users', got '%s'", user.TableName())
	}
}

// TestACLExternalProvider_TableName tests the table name
func TestACLExternalProvider_TableName(t *testing.T) {
	t.Parallel()

	provider := models.ACLExternalProvider{}
	if provider.TableName() != "acl_external_providers" {
		t.Errorf("Expected table name 'acl_external_providers', got '%s'", provider.TableName())
	}
}

// TestACLWaygatesAuth_TableName tests the table name
func TestACLWaygatesAuth_TableName(t *testing.T) {
	t.Parallel()

	auth := models.ACLWaygatesAuth{}
	if auth.TableName() != "acl_waygates_auth" {
		t.Errorf("Expected table name 'acl_waygates_auth', got '%s'", auth.TableName())
	}
}

// TestProxyACLAssignment_TableName tests the table name
func TestProxyACLAssignment_TableName(t *testing.T) {
	t.Parallel()

	assignment := models.ProxyACLAssignment{}
	if assignment.TableName() != "proxy_acl_assignments" {
		t.Errorf("Expected table name 'proxy_acl_assignments', got '%s'", assignment.TableName())
	}
}

// TestACLBranding_TableName tests the table name
func TestACLBranding_TableName(t *testing.T) {
	t.Parallel()

	branding := models.ACLBranding{}
	if branding.TableName() != "acl_branding" {
		t.Errorf("Expected table name 'acl_branding', got '%s'", branding.TableName())
	}
}

// TestACLSession_TableName tests the table name
func TestACLSession_TableName(t *testing.T) {
	t.Parallel()

	session := models.ACLSession{}
	if session.TableName() != "acl_sessions" {
		t.Errorf("Expected table name 'acl_sessions', got '%s'", session.TableName())
	}
}

// TestACLGroupListResponse_Structure tests the ACLGroupListResponse struct
func TestACLGroupListResponse_Structure(t *testing.T) {
	t.Parallel()

	response := models.ACLGroupListResponse{
		Items: []models.ACLGroup{
			{ID: 1, Name: "Admin Group"},
			{ID: 2, Name: "User Group"},
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

// TestACLSessionListResponse_Structure tests the ACLSessionListResponse struct
func TestACLSessionListResponse_Structure(t *testing.T) {
	t.Parallel()

	userID1 := 1
	userID2 := 2
	response := models.ACLSessionListResponse{
		Items: []models.ACLSession{
			{ID: 1, UserID: &userID1},
			{ID: 2, UserID: &userID2},
		},
		Total:      50,
		Page:       2,
		Limit:      10,
		TotalPages: 5,
	}

	if len(response.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(response.Items))
	}
	if response.Total != 50 {
		t.Errorf("Expected Total 50, got %d", response.Total)
	}
	if response.Page != 2 {
		t.Errorf("Expected Page 2, got %d", response.Page)
	}
	if response.Limit != 10 {
		t.Errorf("Expected Limit 10, got %d", response.Limit)
	}
	if response.TotalPages != 5 {
		t.Errorf("Expected TotalPages 5, got %d", response.TotalPages)
	}
}

// TestACLGroupModel_Structure tests the ACLGroup model structure
func TestACLGroupModel_Structure(t *testing.T) {
	t.Parallel()

	description := "Test group description"
	now := time.Now()

	group := models.ACLGroup{
		ID:              1,
		Name:            "Test Group",
		Description:     &description,
		CombinationMode: models.ACLCombinationModeAny,
		CreatedBy:       1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if group.ID != 1 {
		t.Errorf("Expected ID 1, got %d", group.ID)
	}
	if group.Name != "Test Group" {
		t.Errorf("Expected Name 'Test Group', got '%s'", group.Name)
	}
	if group.Description == nil || *group.Description != "Test group description" {
		t.Error("Expected Description to be set")
	}
	if group.CombinationMode != models.ACLCombinationModeAny {
		t.Errorf("Expected CombinationMode 'any', got '%s'", group.CombinationMode)
	}
	if group.CreatedBy != 1 {
		t.Errorf("Expected CreatedBy 1, got %d", group.CreatedBy)
	}
}

// TestACLIPRuleModel_Structure tests the ACLIPRule model structure
func TestACLIPRuleModel_Structure(t *testing.T) {
	t.Parallel()

	description := "Allow internal network"
	now := time.Now()

	rule := models.ACLIPRule{
		ID:          1,
		ACLGroupID:  1,
		RuleType:    models.ACLIPRuleTypeAllow,
		CIDR:        "192.168.1.0/24",
		Description: &description,
		Priority:    10,
		CreatedAt:   now,
	}

	if rule.ID != 1 {
		t.Errorf("Expected ID 1, got %d", rule.ID)
	}
	if rule.ACLGroupID != 1 {
		t.Errorf("Expected ACLGroupID 1, got %d", rule.ACLGroupID)
	}
	if rule.RuleType != models.ACLIPRuleTypeAllow {
		t.Errorf("Expected RuleType 'allow', got '%s'", rule.RuleType)
	}
	if rule.CIDR != "192.168.1.0/24" {
		t.Errorf("Expected CIDR '192.168.1.0/24', got '%s'", rule.CIDR)
	}
	if rule.Description == nil || *rule.Description != "Allow internal network" {
		t.Error("Expected Description to be set")
	}
	if rule.Priority != 10 {
		t.Errorf("Expected Priority 10, got %d", rule.Priority)
	}
}

// TestACLBasicAuthUserModel_Structure tests the ACLBasicAuthUser model structure
func TestACLBasicAuthUserModel_Structure(t *testing.T) {
	t.Parallel()

	now := time.Now()

	user := models.ACLBasicAuthUser{
		ID:           1,
		ACLGroupID:   1,
		Username:     "admin",
		PasswordHash: "$2a$14$hashvalue",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if user.ID != 1 {
		t.Errorf("Expected ID 1, got %d", user.ID)
	}
	if user.ACLGroupID != 1 {
		t.Errorf("Expected ACLGroupID 1, got %d", user.ACLGroupID)
	}
	if user.Username != "admin" {
		t.Errorf("Expected Username 'admin', got '%s'", user.Username)
	}
	if user.PasswordHash != "$2a$14$hashvalue" {
		t.Errorf("Expected PasswordHash to be set")
	}
}

// TestACLExternalProviderModel_Structure tests the ACLExternalProvider model structure
func TestACLExternalProviderModel_Structure(t *testing.T) {
	t.Parallel()

	authRedirect := "https://auth.example.com/login"
	headers := models.JSONStringArray{"Authorization", "X-User-ID"}
	now := time.Now()

	provider := models.ACLExternalProvider{
		ID:              1,
		ACLGroupID:      1,
		ProviderType:    models.ACLProviderTypeAuthelia,
		Name:            "Company Auth",
		VerifyURL:       "https://auth.example.com/verify",
		AuthRedirectURL: &authRedirect,
		HeadersToCopy:   headers,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if provider.ID != 1 {
		t.Errorf("Expected ID 1, got %d", provider.ID)
	}
	if provider.ACLGroupID != 1 {
		t.Errorf("Expected ACLGroupID 1, got %d", provider.ACLGroupID)
	}
	if provider.ProviderType != models.ACLProviderTypeAuthelia {
		t.Errorf("Expected ProviderType 'authelia', got '%s'", provider.ProviderType)
	}
	if provider.Name != "Company Auth" {
		t.Errorf("Expected Name 'Company Auth', got '%s'", provider.Name)
	}
	if provider.VerifyURL != "https://auth.example.com/verify" {
		t.Errorf("Expected VerifyURL to be set")
	}
	if provider.AuthRedirectURL == nil || *provider.AuthRedirectURL != "https://auth.example.com/login" {
		t.Error("Expected AuthRedirectURL to be set")
	}
	if len(provider.HeadersToCopy) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(provider.HeadersToCopy))
	}
}

// TestACLWaygatesAuthModel_Structure tests the ACLWaygatesAuth model structure
func TestACLWaygatesAuthModel_Structure(t *testing.T) {
	t.Parallel()

	auth := models.ACLWaygatesAuth{
		ID:                   1,
		ACLGroupID:           1,
		Enabled:              true,
		AllowedUsers:         models.JSONStringArray{"admin", "user1"},
		AllowedRoles:         models.JSONStringArray{"admin", "editor"},
		AllowedEmailPatterns: models.JSONStringArray{"*@company.com"},
		Require2FA:           true,
		SessionTTL:           86400,
	}

	if auth.ID != 1 {
		t.Errorf("Expected ID 1, got %d", auth.ID)
	}
	if auth.ACLGroupID != 1 {
		t.Errorf("Expected ACLGroupID 1, got %d", auth.ACLGroupID)
	}
	if !auth.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if len(auth.AllowedUsers) != 2 {
		t.Errorf("Expected 2 allowed users, got %d", len(auth.AllowedUsers))
	}
	if len(auth.AllowedRoles) != 2 {
		t.Errorf("Expected 2 allowed roles, got %d", len(auth.AllowedRoles))
	}
	if len(auth.AllowedEmailPatterns) != 1 {
		t.Errorf("Expected 1 email pattern, got %d", len(auth.AllowedEmailPatterns))
	}
	if !auth.Require2FA {
		t.Error("Expected Require2FA to be true")
	}
	if auth.SessionTTL != 86400 {
		t.Errorf("Expected SessionTTL 86400, got %d", auth.SessionTTL)
	}
}

// TestProxyACLAssignmentModel_Structure tests the ProxyACLAssignment model structure
func TestProxyACLAssignmentModel_Structure(t *testing.T) {
	t.Parallel()

	now := time.Now()

	assignment := models.ProxyACLAssignment{
		ID:          1,
		ProxyID:     1,
		ACLGroupID:  2,
		PathPattern: "/api/*",
		Priority:    10,
		Enabled:     true,
		CreatedAt:   now,
	}

	if assignment.ID != 1 {
		t.Errorf("Expected ID 1, got %d", assignment.ID)
	}
	if assignment.ProxyID != 1 {
		t.Errorf("Expected ProxyID 1, got %d", assignment.ProxyID)
	}
	if assignment.ACLGroupID != 2 {
		t.Errorf("Expected ACLGroupID 2, got %d", assignment.ACLGroupID)
	}
	if assignment.PathPattern != "/api/*" {
		t.Errorf("Expected PathPattern '/api/*', got '%s'", assignment.PathPattern)
	}
	if assignment.Priority != 10 {
		t.Errorf("Expected Priority 10, got %d", assignment.Priority)
	}
	if !assignment.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

// TestACLBrandingModel_Structure tests the ACLBranding model structure
func TestACLBrandingModel_Structure(t *testing.T) {
	t.Parallel()

	logoURL := "https://example.com/logo.png"
	subtitle := "Welcome to our service"
	footerText := "Copyright 2024"
	customCSS := ".login-form { background: #fff; }"
	now := time.Now()

	branding := models.ACLBranding{
		ID:              1,
		LogoURL:         &logoURL,
		PrimaryColor:    "#3b82f6",
		BackgroundColor: "#ffffff",
		Title:           "Login Required",
		Subtitle:        &subtitle,
		FooterText:      &footerText,
		CustomCSS:       &customCSS,
		UpdatedAt:       now,
	}

	if branding.ID != 1 {
		t.Errorf("Expected ID 1, got %d", branding.ID)
	}
	if branding.LogoURL == nil || *branding.LogoURL != "https://example.com/logo.png" {
		t.Error("Expected LogoURL to be set")
	}
	if branding.PrimaryColor != "#3b82f6" {
		t.Errorf("Expected PrimaryColor '#3b82f6', got '%s'", branding.PrimaryColor)
	}
	if branding.BackgroundColor != "#ffffff" {
		t.Errorf("Expected BackgroundColor '#ffffff', got '%s'", branding.BackgroundColor)
	}
	if branding.Title != "Login Required" {
		t.Errorf("Expected Title 'Login Required', got '%s'", branding.Title)
	}
	if branding.Subtitle == nil || *branding.Subtitle != "Welcome to our service" {
		t.Error("Expected Subtitle to be set")
	}
	if branding.FooterText == nil || *branding.FooterText != "Copyright 2024" {
		t.Error("Expected FooterText to be set")
	}
	if branding.CustomCSS == nil || *branding.CustomCSS != ".login-form { background: #fff; }" {
		t.Error("Expected CustomCSS to be set")
	}
}

// TestACLSessionModel_Structure tests the ACLSession model structure
func TestACLSessionModel_Structure(t *testing.T) {
	t.Parallel()

	proxyID := 1
	ipAddress := "192.168.1.100"
	userAgent := "Mozilla/5.0"
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	userID := 1

	session := models.ACLSession{
		ID:           1,
		SessionToken: "token123abc",
		UserID:       &userID,
		ProxyID:      &proxyID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
	}

	if session.ID != 1 {
		t.Errorf("Expected ID 1, got %d", session.ID)
	}
	if session.SessionToken != "token123abc" {
		t.Errorf("Expected SessionToken 'token123abc', got '%s'", session.SessionToken)
	}
	if session.UserID == nil || *session.UserID != 1 {
		t.Error("Expected UserID to be 1")
	}
	if session.ProxyID == nil || *session.ProxyID != 1 {
		t.Error("Expected ProxyID to be 1")
	}
	if session.IPAddress == nil || *session.IPAddress != "192.168.1.100" {
		t.Error("Expected IPAddress to be set")
	}
	if session.UserAgent == nil || *session.UserAgent != "Mozilla/5.0" {
		t.Error("Expected UserAgent to be set")
	}
}

// TestACLCombinationModeConstants tests combination mode constants
func TestACLCombinationModeConstants(t *testing.T) {
	t.Parallel()

	if models.ACLCombinationModeAny != "any" {
		t.Errorf("Expected ACLCombinationModeAny 'any', got '%s'", models.ACLCombinationModeAny)
	}
	if models.ACLCombinationModeAll != "all" {
		t.Errorf("Expected ACLCombinationModeAll 'all', got '%s'", models.ACLCombinationModeAll)
	}
	if models.ACLCombinationModeIPBypass != "ip_bypass" {
		t.Errorf("Expected ACLCombinationModeIPBypass 'ip_bypass', got '%s'", models.ACLCombinationModeIPBypass)
	}
}

// TestACLIPRuleTypeConstants tests IP rule type constants
func TestACLIPRuleTypeConstants(t *testing.T) {
	t.Parallel()

	if models.ACLIPRuleTypeAllow != "allow" {
		t.Errorf("Expected ACLIPRuleTypeAllow 'allow', got '%s'", models.ACLIPRuleTypeAllow)
	}
	if models.ACLIPRuleTypeDeny != "deny" {
		t.Errorf("Expected ACLIPRuleTypeDeny 'deny', got '%s'", models.ACLIPRuleTypeDeny)
	}
	if models.ACLIPRuleTypeBypass != "bypass" {
		t.Errorf("Expected ACLIPRuleTypeBypass 'bypass', got '%s'", models.ACLIPRuleTypeBypass)
	}
}

// TestACLProviderTypeConstants tests provider type constants
func TestACLProviderTypeConstants(t *testing.T) {
	t.Parallel()

	if models.ACLProviderTypeAuthelia != "authelia" {
		t.Errorf("Expected ACLProviderTypeAuthelia 'authelia', got '%s'", models.ACLProviderTypeAuthelia)
	}
	if models.ACLProviderTypeAuthentik != "authentik" {
		t.Errorf("Expected ACLProviderTypeAuthentik 'authentik', got '%s'", models.ACLProviderTypeAuthentik)
	}
	if models.ACLProviderTypeCustom != "custom" {
		t.Errorf("Expected ACLProviderTypeCustom 'custom', got '%s'", models.ACLProviderTypeCustom)
	}
}

// TestACLGroupModel_WithRelations tests the ACLGroup model with related data
func TestACLGroupModel_WithRelations(t *testing.T) {
	t.Parallel()

	description := "Test group"
	now := time.Now()

	group := models.ACLGroup{
		ID:              1,
		Name:            "Test Group",
		Description:     &description,
		CombinationMode: models.ACLCombinationModeAny,
		CreatedBy:       1,
		CreatedAt:       now,
		UpdatedAt:       now,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.1.0/24"},
		},
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin"},
		},
		ExternalProviders: []models.ACLExternalProvider{
			{ID: 1, Name: "Auth Provider", ProviderType: models.ACLProviderTypeAuthelia},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:      1,
			Enabled: true,
		},
	}

	if len(group.IPRules) != 1 {
		t.Errorf("Expected 1 IP rule, got %d", len(group.IPRules))
	}
	if len(group.BasicAuthUsers) != 1 {
		t.Errorf("Expected 1 basic auth user, got %d", len(group.BasicAuthUsers))
	}
	if len(group.ExternalProviders) != 1 {
		t.Errorf("Expected 1 external provider, got %d", len(group.ExternalProviders))
	}
	if group.WaygatesAuth == nil {
		t.Error("Expected WaygatesAuth to be set")
	}
}
