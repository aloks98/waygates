package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// =============================================================================
// JSONStringArray Tests
// =============================================================================

func TestJSONStringArray_Value_WithValues(t *testing.T) {
	t.Parallel()

	arr := JSONStringArray{"value1", "value2", "value3"}
	val, err := arr.Value()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Value should be a []byte
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatal("Expected []byte type")
	}

	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 values, got %d", len(result))
	}
	if result[0] != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result[0])
	}
}

func TestJSONStringArray_Value_Nil(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	val, err := arr.Value()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if val != nil {
		t.Errorf("Expected nil, got %v", val)
	}
}

func TestJSONStringArray_Value_Empty(t *testing.T) {
	t.Parallel()

	arr := JSONStringArray{}
	val, err := arr.Value()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Empty slice should marshal to "[]"
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatal("Expected []byte type")
	}

	if string(bytes) != "[]" {
		t.Errorf("Expected '[]', got '%s'", string(bytes))
	}
}

func TestJSONStringArray_Scan_Bytes(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	input := []byte(`["value1", "value2"]`)

	err := arr.Scan(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(arr) != 2 {
		t.Errorf("Expected 2 values, got %d", len(arr))
	}
	if arr[0] != "value1" {
		t.Errorf("Expected 'value1', got '%s'", arr[0])
	}
	if arr[1] != "value2" {
		t.Errorf("Expected 'value2', got '%s'", arr[1])
	}
}

func TestJSONStringArray_Scan_String(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	input := `["admin", "user"]`

	err := arr.Scan(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(arr) != 2 {
		t.Errorf("Expected 2 values, got %d", len(arr))
	}
	if arr[0] != "admin" {
		t.Errorf("Expected 'admin', got '%s'", arr[0])
	}
}

func TestJSONStringArray_Scan_Nil(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	err := arr.Scan(nil)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if arr != nil {
		t.Errorf("Expected nil, got %v", arr)
	}
}

func TestJSONStringArray_Scan_EmptyBytes(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	err := arr.Scan([]byte{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if arr != nil {
		t.Errorf("Expected nil, got %v", arr)
	}
}

func TestJSONStringArray_Scan_EmptyString(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	err := arr.Scan("")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if arr != nil {
		t.Errorf("Expected nil, got %v", arr)
	}
}

func TestJSONStringArray_Scan_InvalidJSON(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	input := []byte(`{invalid json}`)

	err := arr.Scan(input)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestJSONStringArray_Scan_UnexpectedType(t *testing.T) {
	t.Parallel()

	var arr JSONStringArray
	err := arr.Scan(123) // Pass an int instead of []byte or string

	if err != nil {
		t.Error("Expected no error for unexpected type (returns nil)")
	}

	if arr != nil {
		t.Errorf("Expected nil for unexpected type, got %v", arr)
	}
}

func TestJSONStringArray_DriverValuer(t *testing.T) {
	t.Parallel()

	// Ensure it implements driver.Valuer
	var _ driver.Valuer = JSONStringArray{}
}

// =============================================================================
// ACLGroup Validation Tests
// =============================================================================

func TestACLGroup_Validate_Success(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		group ACLGroup
	}{
		{
			name: "valid with any mode",
			group: ACLGroup{
				Name:            "Test Group",
				CombinationMode: ACLCombinationModeAny,
			},
		},
		{
			name: "valid with all mode",
			group: ACLGroup{
				Name:            "Admin Group",
				CombinationMode: ACLCombinationModeAll,
			},
		},
		{
			name: "valid with ip_bypass mode",
			group: ACLGroup{
				Name:            "IP Bypass Group",
				CombinationMode: ACLCombinationModeIPBypass,
			},
		},
		{
			name: "valid with description",
			group: ACLGroup{
				Name:            "Described Group",
				CombinationMode: ACLCombinationModeAny,
				Description:     stringPtr("A description"),
			},
		},
		{
			name: "valid with max length name",
			group: ACLGroup{
				Name:            string(make([]byte, 255)),
				CombinationMode: ACLCombinationModeAny,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.group.Validate()
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestACLGroup_Validate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		group       ACLGroup
		expectedErr error
	}{
		{
			name: "empty name",
			group: ACLGroup{
				Name:            "",
				CombinationMode: ACLCombinationModeAny,
			},
			expectedErr: ErrACLGroupNameRequired,
		},
		{
			name: "name too long",
			group: ACLGroup{
				Name:            string(make([]byte, 256)),
				CombinationMode: ACLCombinationModeAny,
			},
			expectedErr: ErrACLGroupNameTooLong,
		},
		{
			name: "invalid combination mode",
			group: ACLGroup{
				Name:            "Test Group",
				CombinationMode: "invalid",
			},
			expectedErr: ErrInvalidCombinationMode,
		},
		{
			name: "empty combination mode",
			group: ACLGroup{
				Name:            "Test Group",
				CombinationMode: "",
			},
			expectedErr: ErrInvalidCombinationMode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.group.Validate()
			if tc.expectedErr != nil {
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("Expected error %v, got: %v", tc.expectedErr, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// =============================================================================
// ACLIPRule Validation Tests
// =============================================================================

func TestACLIPRule_Validate_Success(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		rule ACLIPRule
	}{
		{
			name: "valid allow rule",
			rule: ACLIPRule{
				RuleType: ACLIPRuleTypeAllow,
				CIDR:     "192.168.1.0/24",
			},
		},
		{
			name: "valid deny rule",
			rule: ACLIPRule{
				RuleType: ACLIPRuleTypeDeny,
				CIDR:     "10.0.0.0/8",
			},
		},
		{
			name: "valid bypass rule",
			rule: ACLIPRule{
				RuleType: ACLIPRuleTypeBypass,
				CIDR:     "172.16.0.0/12",
			},
		},
		{
			name: "valid single IP",
			rule: ACLIPRule{
				RuleType: ACLIPRuleTypeAllow,
				CIDR:     "192.168.1.100",
			},
		},
		{
			name: "valid IPv6 CIDR",
			rule: ACLIPRule{
				RuleType: ACLIPRuleTypeAllow,
				CIDR:     "2001:db8::/32",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rule.Validate()
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestACLIPRule_Validate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		rule        ACLIPRule
		expectedErr error
	}{
		{
			name: "invalid rule type",
			rule: ACLIPRule{
				RuleType: "invalid",
				CIDR:     "192.168.1.0/24",
			},
			expectedErr: ErrInvalidIPRuleType,
		},
		{
			name: "empty rule type",
			rule: ACLIPRule{
				RuleType: "",
				CIDR:     "192.168.1.0/24",
			},
			expectedErr: ErrInvalidIPRuleType,
		},
		{
			name: "empty CIDR",
			rule: ACLIPRule{
				RuleType: ACLIPRuleTypeAllow,
				CIDR:     "",
			},
			expectedErr: ErrInvalidCIDR,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rule.Validate()
			if tc.expectedErr != nil {
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("Expected error %v, got: %v", tc.expectedErr, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// =============================================================================
// ACLExternalProvider Validation Tests
// =============================================================================

func TestACLExternalProvider_Validate_Success(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		provider ACLExternalProvider
	}{
		{
			name: "valid authelia provider",
			provider: ACLExternalProvider{
				ProviderType: ACLProviderTypeAuthelia,
				Name:         "Authelia",
				VerifyURL:    "https://auth.example.com/api/verify",
			},
		},
		{
			name: "valid authentik provider",
			provider: ACLExternalProvider{
				ProviderType: ACLProviderTypeAuthentik,
				Name:         "Authentik",
				VerifyURL:    "https://auth.example.com/verify",
			},
		},
		{
			name: "valid custom provider",
			provider: ACLExternalProvider{
				ProviderType: ACLProviderTypeCustom,
				Name:         "Custom Auth",
				VerifyURL:    "https://custom.example.com/auth",
			},
		},
		{
			name: "valid with optional fields",
			provider: ACLExternalProvider{
				ProviderType:    ACLProviderTypeAuthelia,
				Name:            "Full Provider",
				VerifyURL:       "https://auth.example.com/verify",
				AuthRedirectURL: stringPtr("https://auth.example.com/login"),
				HeadersToCopy:   JSONStringArray{"X-Custom-Header"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.provider.Validate()
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestACLExternalProvider_Validate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		provider    ACLExternalProvider
		expectedErr error
	}{
		{
			name: "invalid provider type",
			provider: ACLExternalProvider{
				ProviderType: "invalid",
				Name:         "Test",
				VerifyURL:    "https://example.com/verify",
			},
			expectedErr: ErrInvalidProviderType,
		},
		{
			name: "empty provider type",
			provider: ACLExternalProvider{
				ProviderType: "",
				Name:         "Test",
				VerifyURL:    "https://example.com/verify",
			},
			expectedErr: ErrInvalidProviderType,
		},
		{
			name: "empty name",
			provider: ACLExternalProvider{
				ProviderType: ACLProviderTypeAuthelia,
				Name:         "",
				VerifyURL:    "https://example.com/verify",
			},
			expectedErr: ErrProviderNameRequired,
		},
		{
			name: "empty verify URL",
			provider: ACLExternalProvider{
				ProviderType: ACLProviderTypeAuthelia,
				Name:         "Test",
				VerifyURL:    "",
			},
			expectedErr: ErrProviderVerifyURLEmpty,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.provider.Validate()
			if tc.expectedErr != nil {
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("Expected error %v, got: %v", tc.expectedErr, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// =============================================================================
// ACLBasicAuthUser Password Tests
// =============================================================================

func TestACLBasicAuthUser_SetPassword(t *testing.T) {
	t.Parallel()

	user := &ACLBasicAuthUser{}

	err := user.SetPassword("testpassword123", 10) // Using cost 10 for faster tests
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if user.PasswordHash == "" {
		t.Error("Expected PasswordHash to be set")
	}

	if user.PasswordHash == "testpassword123" {
		t.Error("PasswordHash should not be plain text")
	}

	// Hash should start with bcrypt identifier
	if len(user.PasswordHash) < 4 || user.PasswordHash[:4] != "$2a$" {
		t.Errorf("Expected bcrypt hash, got: %s", user.PasswordHash[:10])
	}
}

func TestACLBasicAuthUser_SetPassword_DifferentCosts(t *testing.T) {
	t.Parallel()

	testCases := []int{4, 10, 12}

	for _, cost := range testCases {
		t.Run("cost_"+string(rune('0'+cost)), func(t *testing.T) {
			user := &ACLBasicAuthUser{}
			err := user.SetPassword("password", cost)
			if err != nil {
				t.Fatalf("Unexpected error with cost %d: %v", cost, err)
			}
			if user.PasswordHash == "" {
				t.Error("Expected PasswordHash to be set")
			}
		})
	}
}

func TestACLBasicAuthUser_CheckPassword_Correct(t *testing.T) {
	t.Parallel()

	user := &ACLBasicAuthUser{}
	password := "correctpassword"

	if err := user.SetPassword(password, 10); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	if !user.CheckPassword(password) {
		t.Error("Expected CheckPassword to return true for correct password")
	}
}

func TestACLBasicAuthUser_CheckPassword_Incorrect(t *testing.T) {
	t.Parallel()

	user := &ACLBasicAuthUser{}
	password := "correctpassword"

	if err := user.SetPassword(password, 10); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	if user.CheckPassword("wrongpassword") {
		t.Error("Expected CheckPassword to return false for incorrect password")
	}
}

func TestACLBasicAuthUser_CheckPassword_EmptyPassword(t *testing.T) {
	t.Parallel()

	user := &ACLBasicAuthUser{}

	if err := user.SetPassword("actualpassword", 10); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	if user.CheckPassword("") {
		t.Error("Expected CheckPassword to return false for empty password")
	}
}

func TestACLBasicAuthUser_CheckPassword_CaseSensitive(t *testing.T) {
	t.Parallel()

	user := &ACLBasicAuthUser{}

	if err := user.SetPassword("Password123", 10); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	if user.CheckPassword("password123") {
		t.Error("Expected CheckPassword to be case-sensitive")
	}

	if user.CheckPassword("PASSWORD123") {
		t.Error("Expected CheckPassword to be case-sensitive")
	}
}

// =============================================================================
// ACLSession Tests
// =============================================================================

func TestACLSession_IsExpired_NotExpired(t *testing.T) {
	t.Parallel()

	session := &ACLSession{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if session.IsExpired() {
		t.Error("Expected session not to be expired")
	}
}

func TestACLSession_IsExpired_Expired(t *testing.T) {
	t.Parallel()

	session := &ACLSession{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	if !session.IsExpired() {
		t.Error("Expected session to be expired")
	}
}

func TestACLSession_IsExpired_JustExpired(t *testing.T) {
	t.Parallel()

	session := &ACLSession{
		ExpiresAt: time.Now().Add(-1 * time.Millisecond),
	}

	if !session.IsExpired() {
		t.Error("Expected session to be expired")
	}
}

func TestACLSession_IsExpired_AboutToExpire(t *testing.T) {
	t.Parallel()

	session := &ACLSession{
		ExpiresAt: time.Now().Add(1 * time.Millisecond),
	}

	if session.IsExpired() {
		t.Error("Expected session not to be expired yet")
	}
}

// =============================================================================
// Validation Error Tests
// =============================================================================

func TestValidationErrors_ErrorMessages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err      error
		expected string
	}{
		{ErrACLGroupNameRequired, "ACL group name is required"},
		{ErrACLGroupNameTooLong, "ACL group name must be at most 255 characters"},
		{ErrInvalidCombinationMode, "invalid combination mode"},
		{ErrInvalidIPRuleType, "invalid IP rule type"},
		{ErrInvalidCIDR, "invalid CIDR format"},
		{ErrBasicAuthUsernameEmpty, "basic auth username cannot be empty"},
		{ErrBasicAuthPasswordEmpty, "basic auth password cannot be empty"},
		{ErrInvalidProviderType, "invalid external provider type"},
		{ErrProviderNameRequired, "external provider name is required"},
		{ErrProviderVerifyURLEmpty, "external provider verify URL is required"},
		{ErrInvalidPathPattern, "invalid path pattern"},
		{ErrSessionTokenEmpty, "session token cannot be empty"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			var validationErr *ValidationError
			if !errors.As(tc.err, &validationErr) {
				t.Fatal("Expected *ValidationError type")
			}
			if validationErr.Error() != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, validationErr.Error())
			}
		})
	}
}

// =============================================================================
// Model Default Values Tests
// =============================================================================

func TestACLGroup_DefaultCombinationMode(t *testing.T) {
	t.Parallel()

	// Test that the default GORM value is 'any'
	group := ACLGroup{
		Name: "Test Group",
	}

	// When CombinationMode is empty, validation should fail
	// because the service layer should set the default
	err := group.Validate()
	if !errors.Is(err, ErrInvalidCombinationMode) {
		t.Errorf("Expected ErrInvalidCombinationMode for empty mode, got: %v", err)
	}
}

func TestACLBranding_DefaultValues(t *testing.T) {
	t.Parallel()

	// Test default values according to GORM tags
	branding := ACLBranding{}

	// GORM defaults should be set by the database, not the struct
	// But we can verify the zero values
	if branding.PrimaryColor != "" {
		t.Error("Expected empty PrimaryColor for zero value")
	}
	if branding.BackgroundColor != "" {
		t.Error("Expected empty BackgroundColor for zero value")
	}
	if branding.Title != "" {
		t.Error("Expected empty Title for zero value")
	}
}

func TestProxyACLAssignment_DefaultValues(t *testing.T) {
	t.Parallel()

	assignment := ProxyACLAssignment{}

	// Test zero values
	if assignment.PathPattern != "" {
		t.Error("Expected empty PathPattern for zero value")
	}
	if assignment.Priority != 0 {
		t.Error("Expected 0 Priority for zero value")
	}
	if assignment.Enabled {
		t.Error("Expected false Enabled for zero value")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func stringPtr(s string) *string {
	return &s
}
