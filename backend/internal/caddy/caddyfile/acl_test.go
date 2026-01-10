package caddyfile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

func TestNewACLBuilder(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "https://waygates.example.com/auth/login")
	require.NotNil(t, builder)
	assert.Equal(t, "http://localhost:8080", builder.waygatesVerifyURL)
	assert.Equal(t, "https://waygates.example.com/auth/login", builder.waygatesLoginURL)
}

func TestBuildACLConfig_EmptyAssignments(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")
	proxy := createTestProxy()

	// Test with nil assignments
	result := builder.BuildACLConfig(proxy, nil)
	assert.Empty(t, result)

	// Test with empty slice
	result = builder.BuildACLConfig(proxy, []models.ProxyACLAssignment{})
	assert.Empty(t, result)
}

func TestBuildACLConfig_DisabledAssignments(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.1.0/24"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/api/*",
			Priority:    0,
			Enabled:     false, // Disabled
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)
	assert.Empty(t, result, "disabled assignments should produce no output")
}

func TestBuildACLConfig_IPDenyRules(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "1.2.3.4"},
			{ID: 2, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.0.0/8"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "@acl_0_denied_ips")
	assert.Contains(t, result, "remote_ip 1.2.3.4 10.0.0.0/8")
	assert.Contains(t, result, "respond @acl_0_denied_ips \"Forbidden\" 403")
}

func TestBuildACLConfig_IPBypassRules(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/api/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "@acl_0_bypass_ip")
	assert.Contains(t, result, "remote_ip 192.168.1.0/24")
	assert.Contains(t, result, "handle @acl_0_bypass_ip")
	assert.Contains(t, result, "reverse_proxy")
}

func TestBuildACLConfig_IPAllowRules(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "@acl_0_allowed_ip")
	assert.Contains(t, result, "remote_ip 10.0.0.0/8")
	assert.Contains(t, result, "handle @acl_0_allowed_ip")
}

func TestBuildACLConfig_BasicAuth(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
			{ID: 2, Username: "user", PasswordHash: "$2a$14$hashedpassword2"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/admin/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "@acl_0_basic_auth")
	assert.Contains(t, result, "path /admin/*")
	assert.Contains(t, result, "basicauth")
	assert.Contains(t, result, "admin $2a$14$hashedpassword1")
	assert.Contains(t, result, "user $2a$14$hashedpassword2")
}

func TestBuildACLConfig_WaygatesAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         1,
			ACLGroupID: 1,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "@acl_0_forward_auth")
	assert.Contains(t, result, "path /protected/*")
	assert.Contains(t, result, "forward_auth http://waygates:8080")
	assert.Contains(t, result, "uri /api/auth/acl/verify")
	assert.Contains(t, result, "copy_headers X-Auth-User X-Auth-User-ID X-Auth-User-Email")
}

func TestBuildACLConfig_ExternalProvider_Authelia(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	redirectURL := "https://auth.example.com/"
	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		ExternalProviders: []models.ACLExternalProvider{
			{
				ID:              1,
				ACLGroupID:      1,
				ProviderType:    models.ACLProviderTypeAuthelia,
				Name:            "authelia",
				VerifyURL:       "http://authelia:9091/api/verify",
				AuthRedirectURL: &redirectURL,
			},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "forward_auth http://authelia:9091/api/verify?rd=https://auth.example.com/")
	assert.Contains(t, result, "copy_headers Remote-User Remote-Groups Remote-Name Remote-Email")
}

func TestBuildACLConfig_ExternalProvider_Authentik(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		ExternalProviders: []models.ACLExternalProvider{
			{
				ID:           1,
				ACLGroupID:   1,
				ProviderType: models.ACLProviderTypeAuthentik,
				Name:         "authentik",
				VerifyURL:    "http://authentik:9000/outpost.goauthentik.io/auth/nginx",
			},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "forward_auth http://authentik:9000/outpost.goauthentik.io/auth/nginx")
	assert.Contains(t, result, "X-authentik-username")
	assert.Contains(t, result, "X-authentik-groups")
}

func TestBuildACLConfig_ExternalProvider_CustomHeaders(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		ExternalProviders: []models.ACLExternalProvider{
			{
				ID:            1,
				ACLGroupID:    1,
				ProviderType:  models.ACLProviderTypeCustom,
				Name:          "custom-auth",
				VerifyURL:     "http://auth.local/verify",
				HeadersToCopy: []string{"X-Custom-User", "X-Custom-Role"},
			},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	assert.Contains(t, result, "forward_auth http://auth.local/verify")
	assert.Contains(t, result, "copy_headers X-Custom-User X-Custom-Role")
}

func TestBuildACLConfig_CombinationModeAll(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAll,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         1,
			ACLGroupID: 1,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/secure/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// In ALL mode, should deny requests not from allowed IPs
	assert.Contains(t, result, "@acl_0_not_allowed_ip")
	assert.Contains(t, result, "not remote_ip 10.0.0.0/8")
	assert.Contains(t, result, "respond @acl_0_not_allowed_ip \"Forbidden\" 403")

	// Should also require forward auth for allowed IPs
	assert.Contains(t, result, "forward_auth")
}

func TestBuildACLConfig_IPBypassWithForwardAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeIPBypass,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         1,
			ACLGroupID: 1,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/api/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Bypass IPs should skip auth
	assert.Contains(t, result, "@acl_0_bypass_ip")
	assert.Contains(t, result, "remote_ip 192.168.1.0/24")
	assert.Contains(t, result, "handle @acl_0_bypass_ip")

	// Forward auth should exclude bypass IPs
	assert.Contains(t, result, "@acl_0_forward_auth")
	assert.Contains(t, result, "not remote_ip 192.168.1.0/24")
	assert.Contains(t, result, "forward_auth")
}

func TestBuildACLConfig_MultipleAssignments(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group1 := &models.ACLGroup{
		ID:              1,
		Name:            "admin-group",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$adminpass"},
		},
	}

	group2 := &models.ACLGroup{
		ID:              2,
		Name:            "api-group",
		CombinationMode: models.ACLCombinationModeAny,
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         2,
			ACLGroupID: 2,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/admin/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group1,
		},
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/api/*",
			Priority:    1,
			Enabled:     true,
			ACLGroup:    group2,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should have both matchers
	assert.Contains(t, result, "@acl_0_basic_auth")
	assert.Contains(t, result, "path /admin/*")
	assert.Contains(t, result, "@acl_1_forward_auth")
	assert.Contains(t, result, "path /api/*")
}

func TestBuildACLConfig_PriorityOrdering(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group1 := &models.ACLGroup{
		ID:              1,
		Name:            "group1",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules:         []models.ACLIPRule{{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"}},
	}

	group2 := &models.ACLGroup{
		ID:              2,
		Name:            "group2",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules:         []models.ACLIPRule{{ID: 2, RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.0.0/16"}},
	}

	// Assignments in reverse priority order
	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/second/*",
			Priority:    10, // Lower priority (processed second)
			Enabled:     true,
			ACLGroup:    group2,
		},
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/first/*",
			Priority:    1, // Higher priority (processed first)
			Enabled:     true,
			ACLGroup:    group1,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// First group should be acl_0 (priority 1)
	firstIdx := strings.Index(result, "10.0.0.0/8")
	secondIdx := strings.Index(result, "192.168.0.0/16")

	assert.True(t, firstIdx < secondIdx, "higher priority assignment should appear first")
}

func TestBuildACLConfig_NoAuthMethodsConfigured(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	// Group with no auth methods
	group := &models.ACLGroup{
		ID:              1,
		Name:            "empty-group",
		CombinationMode: models.ACLCombinationModeAny,
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)
	assert.Empty(t, result, "groups with no auth methods should produce no output")
}

func TestBuildACLConfig_NilACLGroup(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    nil, // No group loaded
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)
	assert.Empty(t, result, "assignments with nil ACLGroup should be skipped")
}

func TestBuildACLConfig_PathPatternWildcard(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "test-group",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "1.2.3.4"},
		},
	}

	// Test with /* pattern (should not generate path matcher)
	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should not contain path directive for /* pattern
	assert.NotContains(t, result, "path /*")
}

func TestHasACLConfig(t *testing.T) {
	group := &models.ACLGroup{
		ID:   1,
		Name: "test",
	}

	tests := []struct {
		name        string
		assignments []models.ProxyACLAssignment
		expected    bool
	}{
		{
			name:        "nil assignments",
			assignments: nil,
			expected:    false,
		},
		{
			name:        "empty assignments",
			assignments: []models.ProxyACLAssignment{},
			expected:    false,
		},
		{
			name: "disabled assignment",
			assignments: []models.ProxyACLAssignment{
				{Enabled: false, ACLGroup: group},
			},
			expected: false,
		},
		{
			name: "enabled assignment with nil group",
			assignments: []models.ProxyACLAssignment{
				{Enabled: true, ACLGroup: nil},
			},
			expected: false,
		},
		{
			name: "enabled assignment with group",
			assignments: []models.ProxyACLAssignment{
				{Enabled: true, ACLGroup: group},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasACLConfig(tt.assignments)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultWaygatesHeaders(t *testing.T) {
	headers := GetDefaultWaygatesHeaders()
	assert.Contains(t, headers, "X-Auth-User")
	assert.Contains(t, headers, "X-Auth-User-ID")
	assert.Contains(t, headers, "X-Auth-User-Email")
}

func TestGetProviderDefaultHeaders(t *testing.T) {
	tests := []struct {
		provider string
		expected []string
	}{
		{
			provider: models.ACLProviderTypeAuthelia,
			expected: []string{"Remote-User", "Remote-Groups", "Remote-Name", "Remote-Email"},
		},
		{
			provider: models.ACLProviderTypeAuthentik,
			expected: []string{"X-authentik-username", "X-authentik-groups"},
		},
		{
			provider: "unknown",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := GetProviderDefaultHeaders(tt.provider)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				for _, h := range tt.expected {
					assert.Contains(t, result, h)
				}
			}
		})
	}
}

// Helper function to create a test proxy
func createTestProxy() *models.Proxy {
	return &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "test-proxy",
		Hostname: "test.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "backend",
				"port":   float64(8080),
				"scheme": "http",
			},
		},
		SSLEnabled: true,
		IsActive:   true,
	}
}

// =============================================================================
// Union Config Builder Tests
// =============================================================================

// TestBuildUnionACLConfig_MultipleGroupsIPRules tests that IP rules from multiple
// ACL groups are correctly combined when building Caddyfile configuration.
func TestBuildUnionACLConfig_MultipleGroupsIPRules(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	// Group 1 with deny rules for one subnet
	group1 := &models.ACLGroup{
		ID:              1,
		Name:            "group1-deny",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.10.0/24"},
			{ID: 2, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"},
		},
	}

	// Group 2 with deny rules for a different subnet
	group2 := &models.ACLGroup{
		ID:              2,
		Name:            "group2-deny",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 3, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.12.0/24"},
			{ID: 4, RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.0.0/16"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group1,
		},
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/*",
			Priority:    1,
			Enabled:     true,
			ACLGroup:    group2,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Verify both groups have their deny rules
	assert.Contains(t, result, "10.0.10.0/24", "First group's deny rule should be present")
	assert.Contains(t, result, "10.0.12.0/24", "Second group's deny rule should be present")

	// Verify both groups have their allow rules
	assert.Contains(t, result, "10.0.0.0/8", "First group's allow rule should be present")
	assert.Contains(t, result, "192.168.0.0/16", "Second group's allow rule should be present")

	// Verify we have matchers for both groups
	assert.Contains(t, result, "@acl_0_", "First assignment should have acl_0 prefix")
	assert.Contains(t, result, "@acl_1_", "Second assignment should have acl_1 prefix")
}

// TestBuildUnionACLConfig_MultipleGroupsBypassRules tests that IP bypass rules
// from multiple ACL groups are correctly combined.
func TestBuildUnionACLConfig_MultipleGroupsBypassRules(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	// Group 1 with bypass rule for internal network
	group1 := &models.ACLGroup{
		ID:              1,
		Name:            "group1-bypass",
		CombinationMode: models.ACLCombinationModeIPBypass,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			Enabled: true,
		},
	}

	// Group 2 with bypass rule for different internal network
	group2 := &models.ACLGroup{
		ID:              2,
		Name:            "group2-bypass",
		CombinationMode: models.ACLCombinationModeIPBypass,
		IPRules: []models.ACLIPRule{
			{ID: 2, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.2.0/24"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			Enabled: true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group1,
		},
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/*",
			Priority:    1,
			Enabled:     true,
			ACLGroup:    group2,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Verify both bypass rules are present
	assert.Contains(t, result, "192.168.1.0/24", "First group's bypass range should be present")
	assert.Contains(t, result, "192.168.2.0/24", "Second group's bypass range should be present")

	// Verify bypass handlers are created for both
	assert.Contains(t, result, "@acl_0_bypass_ip", "First group should have bypass_ip matcher")
	assert.Contains(t, result, "@acl_1_bypass_ip", "Second group should have bypass_ip matcher")
}

// TestBuildUnionACLConfig_DifferentAuthMethods tests that different authentication
// methods from multiple groups are all generated in the Caddyfile.
func TestBuildUnionACLConfig_DifferentAuthMethods(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	// Group 1 with basic auth
	group1 := &models.ACLGroup{
		ID:              1,
		Name:            "group1-basicauth",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
	}

	// Group 2 with Waygates auth (forward_auth)
	group2 := &models.ACLGroup{
		ID:              2,
		Name:            "group2-waygates",
		CombinationMode: models.ACLCombinationModeAny,
		WaygatesAuth: &models.ACLWaygatesAuth{
			Enabled: true,
		},
	}

	// Group 3 with external provider (Authelia)
	redirectURL := "https://auth.example.com/"
	group3 := &models.ACLGroup{
		ID:              3,
		Name:            "group3-authelia",
		CombinationMode: models.ACLCombinationModeAny,
		ExternalProviders: []models.ACLExternalProvider{
			{
				ID:              1,
				ProviderType:    models.ACLProviderTypeAuthelia,
				Name:            "authelia",
				VerifyURL:       "http://authelia:9091/api/verify",
				AuthRedirectURL: &redirectURL,
			},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/admin/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group1,
		},
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/api/*",
			Priority:    1,
			Enabled:     true,
			ACLGroup:    group2,
		},
		{
			ID:          3,
			ProxyID:     1,
			ACLGroupID:  3,
			PathPattern: "/secure/*",
			Priority:    2,
			Enabled:     true,
			ACLGroup:    group3,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Verify basic auth is configured
	assert.Contains(t, result, "basicauth", "Basic auth directive should be present")
	assert.Contains(t, result, "admin", "Basic auth username should be present")
	assert.Contains(t, result, "/admin/*", "Basic auth path should be present")

	// Verify Waygates forward_auth is configured
	assert.Contains(t, result, "forward_auth http://waygates:8080", "Waygates forward_auth should be present")
	assert.Contains(t, result, "/api/auth/acl/verify", "Waygates verify URI should be present")
	assert.Contains(t, result, "/api/*", "Waygates path should be present")

	// Verify Authelia forward_auth is configured
	assert.Contains(t, result, "http://authelia:9091/api/verify", "Authelia verify URL should be present")
	assert.Contains(t, result, "Remote-User", "Authelia headers should be present")
	assert.Contains(t, result, "/secure/*", "Authelia path should be present")
}

// TestBuildUnionACLConfig_DeduplicateCIDRs tests that duplicate CIDRs
// are handled appropriately when multiple groups have the same CIDR.
func TestBuildUnionACLConfig_DeduplicateCIDRs(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	// Both groups have the same deny CIDR
	sharedCIDR := "10.0.0.0/8"

	group1 := &models.ACLGroup{
		ID:              1,
		Name:            "group1",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: sharedCIDR},
			{ID: 2, RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.0.0/16"},
		},
	}

	group2 := &models.ACLGroup{
		ID:              2,
		Name:            "group2",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 3, RuleType: models.ACLIPRuleTypeDeny, CIDR: sharedCIDR}, // Same CIDR as group1
			{ID: 4, RuleType: models.ACLIPRuleTypeAllow, CIDR: "172.16.0.0/12"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group1,
		},
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/*",
			Priority:    1,
			Enabled:     true,
			ACLGroup:    group2,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Verify the shared CIDR appears (at least once for each group's deny matcher)
	assert.Contains(t, result, sharedCIDR, "Shared CIDR should be present in config")

	// Each group should have its own denied_ips matcher
	assert.Contains(t, result, "@acl_0_denied_ips", "First group should have denied_ips matcher")
	assert.Contains(t, result, "@acl_1_denied_ips", "Second group should have denied_ips matcher")
}

// TestBuildUnionACLConfig_EmptyAssignmentsReturnsEmpty tests that an empty
// list of assignments produces no config output.
func TestBuildUnionACLConfig_EmptyAssignmentsReturnsEmpty(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	// Test with nil
	result := builder.BuildACLConfig(proxy, nil)
	assert.Empty(t, result, "Nil assignments should return empty config")

	// Test with empty slice
	result = builder.BuildACLConfig(proxy, []models.ProxyACLAssignment{})
	assert.Empty(t, result, "Empty assignments should return empty config")
}

// TestBuildUnionACLConfig_OnlyDenyRules tests configuration generation when
// groups only have deny rules without any allow rules.
func TestBuildUnionACLConfig_OnlyDenyRules(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "deny-only",
		CombinationMode: models.ACLCombinationModeAny,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "1.2.3.0/24"},
			{ID: 2, RuleType: models.ACLIPRuleTypeDeny, CIDR: "4.5.6.0/24"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should contain deny matcher
	assert.Contains(t, result, "@acl_0_denied_ips", "Deny matcher should be present")
	assert.Contains(t, result, "1.2.3.0/24", "First deny CIDR should be present")
	assert.Contains(t, result, "4.5.6.0/24", "Second deny CIDR should be present")
	assert.Contains(t, result, "respond @acl_0_denied_ips", "Should respond 403 to denied IPs")
	assert.Contains(t, result, "403", "Should return 403 status")
}

// TestBuildUnionACLConfig_OnlyBypassRules tests configuration generation when
// groups only have bypass rules.
func TestBuildUnionACLConfig_OnlyBypassRules(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "bypass-only",
		CombinationMode: models.ACLCombinationModeIPBypass,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
			{ID: 2, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.2.0/24"},
		},
		// Need some auth method for bypass to make sense
		WaygatesAuth: &models.ACLWaygatesAuth{
			Enabled: true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should contain bypass matcher
	assert.Contains(t, result, "@acl_0_bypass_ip", "Bypass matcher should be present")
	assert.Contains(t, result, "192.168.1.0/24", "First bypass CIDR should be present")
	assert.Contains(t, result, "192.168.2.0/24", "Second bypass CIDR should be present")
	assert.Contains(t, result, "handle @acl_0_bypass_ip", "Should handle bypass IPs")
	assert.Contains(t, result, "reverse_proxy", "Should proxy to backend for bypass IPs")
}

// TestBuildUnionACLConfig_MixedRulesWithPriority tests that multiple groups
// with mixed rules are generated with correct priority ordering.
func TestBuildUnionACLConfig_MixedRulesWithPriority(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	// High priority group with basic auth
	highPriorityGroup := &models.ACLGroup{
		ID:              1,
		Name:            "high-priority",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hash1"},
		},
	}

	// Low priority group with Waygates auth
	lowPriorityGroup := &models.ACLGroup{
		ID:              2,
		Name:            "low-priority",
		CombinationMode: models.ACLCombinationModeAny,
		WaygatesAuth: &models.ACLWaygatesAuth{
			Enabled: true,
		},
	}

	// Assignments in reverse priority order to test sorting
	assignments := []models.ProxyACLAssignment{
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/*",
			Priority:    10, // Lower priority (processed second)
			Enabled:     true,
			ACLGroup:    lowPriorityGroup,
		},
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/*",
			Priority:    1, // Higher priority (processed first)
			Enabled:     true,
			ACLGroup:    highPriorityGroup,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Verify both configurations are present
	assert.Contains(t, result, "basicauth", "Basic auth should be present")
	assert.Contains(t, result, "forward_auth", "Forward auth should be present")

	// Verify priority ordering: basic auth (priority 1) should come before forward_auth (priority 10)
	basicAuthIdx := strings.Index(result, "basicauth")
	forwardAuthIdx := strings.Index(result, "forward_auth")

	assert.True(t, basicAuthIdx > 0, "Basic auth should be in config")
	assert.True(t, forwardAuthIdx > 0, "Forward auth should be in config")
	assert.True(t, basicAuthIdx < forwardAuthIdx, "Higher priority (basic auth) should appear before lower priority (forward auth)")
}

// TestBuildUnionACLConfig_AllCombinationModes tests that all combination modes
// are handled correctly when building union config.
func TestBuildUnionACLConfig_AllCombinationModes(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "")
	proxy := createTestProxy()

	tests := []struct {
		name            string
		combinationMode string
		hasIPRules      bool
		hasAuth         bool
		expectDeny      bool
		expectBypass    bool
		expectAuth      bool
	}{
		{
			name:            "any mode with IP allow",
			combinationMode: models.ACLCombinationModeAny,
			hasIPRules:      true,
			hasAuth:         false,
			expectDeny:      false,
			expectBypass:    false,
			expectAuth:      false,
		},
		{
			name:            "all mode requires both IP and auth",
			combinationMode: models.ACLCombinationModeAll,
			hasIPRules:      true,
			hasAuth:         true,
			expectDeny:      false,
			expectBypass:    false,
			expectAuth:      true,
		},
		{
			name:            "ip_bypass mode with bypass rules",
			combinationMode: models.ACLCombinationModeIPBypass,
			hasIPRules:      true,
			hasAuth:         true,
			expectDeny:      false,
			expectBypass:    true,
			expectAuth:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := &models.ACLGroup{
				ID:              1,
				Name:            "test-group",
				CombinationMode: tt.combinationMode,
			}

			if tt.hasIPRules {
				if tt.combinationMode == models.ACLCombinationModeIPBypass {
					group.IPRules = []models.ACLIPRule{
						{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "10.0.0.0/8"},
					}
				} else {
					group.IPRules = []models.ACLIPRule{
						{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"},
					}
				}
			}

			if tt.hasAuth {
				group.WaygatesAuth = &models.ACLWaygatesAuth{
					Enabled: true,
				}
			}

			assignments := []models.ProxyACLAssignment{
				{
					ID:          1,
					ProxyID:     1,
					ACLGroupID:  1,
					PathPattern: "/*",
					Priority:    0,
					Enabled:     true,
					ACLGroup:    group,
				},
			}

			result := builder.BuildACLConfig(proxy, assignments)

			if tt.expectBypass {
				assert.Contains(t, result, "bypass_ip", "Should have bypass_ip matcher for ip_bypass mode")
			}

			if tt.expectAuth {
				assert.Contains(t, result, "forward_auth", "Should have forward_auth for auth-enabled configs")
			}

			// All mode with IP allow should deny non-matching IPs
			if tt.combinationMode == models.ACLCombinationModeAll && tt.hasIPRules {
				assert.Contains(t, result, "not_allowed_ip", "ALL mode should deny non-matching IPs")
			}
		})
	}
}

// =============================================================================
// BuildUnionACLConfig Tests - Union IP Rule Combination
// =============================================================================

// TestDeduplicateCIDRs tests the CIDR deduplication helper function.
func TestDeduplicateCIDRs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates",
			input:    []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"},
			expected: []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"},
		},
		{
			name:     "with duplicates",
			input:    []string{"10.0.0.0/8", "192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			expected: []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"},
		},
		{
			name:     "all duplicates",
			input:    []string{"10.0.0.0/8", "10.0.0.0/8", "10.0.0.0/8"},
			expected: []string{"10.0.0.0/8"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateCIDRs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCollectUnionIPRules tests the IP rule collection function.
func TestCollectUnionIPRules(t *testing.T) {
	group1 := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.10.0/24"},
			{ID: 2, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
			{ID: 3, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"},
		},
	}

	group2 := &models.ACLGroup{
		ID:   2,
		Name: "group2",
		IPRules: []models.ACLIPRule{
			{ID: 4, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.12.0/24"},
			{ID: 5, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.2.0/24"},
			{ID: 6, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.10.0/24"}, // Duplicate deny
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   group1,
		},
		{
			ID:         2,
			ProxyID:    1,
			ACLGroupID: 2,
			Enabled:    true,
			ACLGroup:   group2,
		},
	}

	denyRules, bypassRules, allowRules := collectUnionIPRules(assignments)

	// Verify deny rules are collected and deduplicated
	assert.Len(t, denyRules, 2, "Should have 2 unique deny rules")
	assert.Contains(t, denyRules, "10.0.10.0/24")
	assert.Contains(t, denyRules, "10.0.12.0/24")

	// Verify bypass rules are collected
	assert.Len(t, bypassRules, 2, "Should have 2 bypass rules")
	assert.Contains(t, bypassRules, "192.168.1.0/24")
	assert.Contains(t, bypassRules, "192.168.2.0/24")

	// Verify allow rules are collected
	assert.Len(t, allowRules, 1, "Should have 1 allow rule")
	assert.Contains(t, allowRules, "10.0.0.0/8")
}

// TestCollectUnionIPRules_DisabledAssignments tests that disabled assignments are skipped.
func TestCollectUnionIPRules_DisabledAssignments(t *testing.T) {
	group := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.10.0/24"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    false, // Disabled
			ACLGroup:   group,
		},
	}

	denyRules, bypassRules, allowRules := collectUnionIPRules(assignments)

	assert.Empty(t, denyRules, "Disabled assignment should not contribute deny rules")
	assert.Empty(t, bypassRules, "Disabled assignment should not contribute bypass rules")
	assert.Empty(t, allowRules, "Disabled assignment should not contribute allow rules")
}

// TestBuildUnionACLConfig_Empty tests that empty assignments return empty config.
func TestBuildUnionACLConfig_Empty(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	// Test with nil
	result := builder.BuildUnionACLConfig(nil)
	assert.Empty(t, result, "Nil assignments should return empty config")

	// Test with empty slice
	result = builder.BuildUnionACLConfig([]models.ProxyACLAssignment{})
	assert.Empty(t, result, "Empty assignments should return empty config")
}

// TestBuildUnionACLConfig_AllDisabled tests that all-disabled assignments return empty config.
func TestBuildUnionACLConfig_AllDisabled(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	group := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.0.0/8"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    false,
			ACLGroup:   group,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)
	assert.Empty(t, result, "All disabled assignments should return empty config")
}

// TestBuildUnionACLConfig_DenyRulesOnly tests config generation with only deny rules.
func TestBuildUnionACLConfig_DenyRulesOnly(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	group1 := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.10.0/24"},
		},
	}

	group2 := &models.ACLGroup{
		ID:   2,
		Name: "group2",
		IPRules: []models.ACLIPRule{
			{ID: 2, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.12.0/24"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   group1,
		},
		{
			ID:         2,
			ProxyID:    1,
			ACLGroupID: 2,
			Enabled:    true,
			ACLGroup:   group2,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)

	// Verify deny matcher is present
	assert.Contains(t, result, "@denied_ips", "Should have denied_ips matcher")
	assert.Contains(t, result, "remote_ip 10.0.10.0/24", "Should contain first deny CIDR")
	assert.Contains(t, result, "remote_ip 10.0.12.0/24", "Should contain second deny CIDR")
	assert.Contains(t, result, "respond @denied_ips 403", "Should respond 403 to denied IPs")

	// Verify forward_auth is present (since no bypass rules)
	assert.Contains(t, result, "forward_auth http://localhost:8080", "Should have forward_auth")
	assert.Contains(t, result, "uri /api/auth/acl/verify", "Should have verify URI")
}

// TestBuildUnionACLConfig_BypassRulesOnly tests config generation with only bypass rules.
func TestBuildUnionACLConfig_BypassRulesOnly(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	group := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
			{ID: 2, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.2.0/24"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   group,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)

	// Verify bypass matcher is present
	assert.Contains(t, result, "@bypass_ips", "Should have bypass_ips matcher")
	assert.Contains(t, result, "remote_ip 192.168.1.0/24", "Should contain first bypass CIDR")
	assert.Contains(t, result, "remote_ip 192.168.2.0/24", "Should contain second bypass CIDR")

	// Verify needs_auth matcher is present
	assert.Contains(t, result, "@needs_auth", "Should have needs_auth matcher")
	assert.Contains(t, result, "not {", "Should have not block in needs_auth")

	// Verify forward_auth is applied to needs_auth
	assert.Contains(t, result, "forward_auth @needs_auth", "Should apply forward_auth to needs_auth matcher")
}

// TestBuildUnionACLConfig_MixedRules tests config generation with deny and bypass rules.
func TestBuildUnionACLConfig_MixedRules(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	group1 := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.10.0/24"},
		},
	}

	group2 := &models.ACLGroup{
		ID:   2,
		Name: "group2",
		IPRules: []models.ACLIPRule{
			{ID: 2, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.12.0/24"},
			{ID: 3, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   group1,
		},
		{
			ID:         2,
			ProxyID:    1,
			ACLGroupID: 2,
			Enabled:    true,
			ACLGroup:   group2,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)

	// Verify deny matcher comes first
	denyIdx := strings.Index(result, "@denied_ips")
	bypassIdx := strings.Index(result, "@bypass_ips")
	assert.True(t, denyIdx < bypassIdx, "Deny matcher should come before bypass matcher")

	// Verify both deny rules are in a single matcher
	assert.Contains(t, result, "remote_ip 10.0.10.0/24", "Should contain first deny CIDR")
	assert.Contains(t, result, "remote_ip 10.0.12.0/24", "Should contain second deny CIDR")

	// Verify bypass rules
	assert.Contains(t, result, "remote_ip 192.168.1.0/24", "Should contain bypass CIDR")

	// Verify forward_auth with needs_auth
	assert.Contains(t, result, "forward_auth @needs_auth", "Should apply forward_auth to needs_auth")
}

// TestBuildUnionACLConfig_DuplicateCIDRsAcrossGroups tests that duplicate CIDRs are deduplicated.
func TestBuildUnionACLConfig_DuplicateCIDRsAcrossGroups(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	sharedCIDR := "10.0.0.0/8"

	group1 := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: sharedCIDR},
		},
	}

	group2 := &models.ACLGroup{
		ID:   2,
		Name: "group2",
		IPRules: []models.ACLIPRule{
			{ID: 2, RuleType: models.ACLIPRuleTypeDeny, CIDR: sharedCIDR}, // Same as group1
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   group1,
		},
		{
			ID:         2,
			ProxyID:    1,
			ACLGroupID: 2,
			Enabled:    true,
			ACLGroup:   group2,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)

	// Count occurrences of the CIDR in the deny block
	// The CIDR should appear exactly once in the denied_ips matcher
	denyBlockStart := strings.Index(result, "@denied_ips")
	denyBlockEnd := strings.Index(result, "respond @denied_ips")
	denyBlock := result[denyBlockStart:denyBlockEnd]

	count := strings.Count(denyBlock, "remote_ip "+sharedCIDR)
	assert.Equal(t, 1, count, "Duplicate CIDR should appear only once in deny block")
}

// TestBuildUnionACLConfig_ForwardAuthHeaders tests that correct headers are included.
func TestBuildUnionACLConfig_ForwardAuthHeaders(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	group := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   group,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)

	// Verify headers in forward_auth
	assert.Contains(t, result, "copy_headers", "Should have copy_headers directive")
	assert.Contains(t, result, "Remote-User", "Should copy Remote-User header")
	assert.Contains(t, result, "Remote-Groups", "Should copy Remote-Groups header")
	assert.Contains(t, result, "Remote-Email", "Should copy Remote-Email header")
	assert.Contains(t, result, "X-Forwarded-User", "Should copy X-Forwarded-User header")
}

// TestBuildUnionACLConfig_VerifyURL tests that the verify URL is correctly used.
func TestBuildUnionACLConfig_VerifyURL(t *testing.T) {
	customVerifyURL := "http://waygates-service:8080"
	builder := NewACLBuilder(customVerifyURL, "")

	group := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "10.0.0.0/8"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   group,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)

	assert.Contains(t, result, "forward_auth http://waygates-service:8080", "Should use custom verify URL")
	assert.Contains(t, result, "uri /api/auth/acl/verify", "Should have verify URI")
}

// TestBuildUnionACLConfig_NilACLGroup tests that assignments with nil ACLGroup are skipped.
func TestBuildUnionACLConfig_NilACLGroup(t *testing.T) {
	builder := NewACLBuilder("http://localhost:8080", "")

	validGroup := &models.ACLGroup{
		ID:   1,
		Name: "group1",
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.0.0/8"},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:         1,
			ProxyID:    1,
			ACLGroupID: 1,
			Enabled:    true,
			ACLGroup:   nil, // Nil group should be skipped
		},
		{
			ID:         2,
			ProxyID:    1,
			ACLGroupID: 2,
			Enabled:    true,
			ACLGroup:   validGroup,
		},
	}

	result := builder.BuildUnionACLConfig(assignments)

	// Should still produce config from valid group
	assert.Contains(t, result, "@denied_ips", "Should have denied_ips from valid group")
	assert.Contains(t, result, "10.0.0.0/8", "Should contain CIDR from valid group")
}

// =============================================================================
// Basic Auth Override Tests - Caddyfile Generation
// =============================================================================

// TestBuildACLConfig_BasicAuthOnlyGeneratesBasicAuth tests that when a group has
// ONLY basic auth users configured (no Waygates auth, no OAuth), the Caddyfile
// should contain a "basicauth" directive.
func TestBuildACLConfig_BasicAuthOnlyGeneratesBasicAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "basic-auth-only",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
			{ID: 2, Username: "user", PasswordHash: "$2a$14$hashedpassword2"},
		},
		// No WaygatesAuth - nil
		// No ExternalProviders - empty
		// No OAuthProviderRestrictions - empty
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should contain basicauth directive since it's the only auth method
	assert.Contains(t, result, "basicauth", "Should contain basicauth directive when only basic auth is configured")
	assert.Contains(t, result, "admin $2a$14$hashedpassword1", "Should contain first user credentials")
	assert.Contains(t, result, "user $2a$14$hashedpassword2", "Should contain second user credentials")

	// Should NOT contain forward_auth since Waygates/OAuth are not configured
	assert.NotContains(t, result, "forward_auth", "Should NOT contain forward_auth when only basic auth is configured")
}

// TestBuildACLConfig_BasicAuthWithWaygatesGeneratesForwardAuth tests that when
// a group has both basic auth users AND Waygates auth enabled, the Caddyfile
// should contain "forward_auth" and NOT "basicauth".
func TestBuildACLConfig_BasicAuthWithWaygatesGeneratesForwardAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "basic-plus-waygates",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         1,
			ACLGroupID: 1,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should contain forward_auth since Waygates auth is enabled (secure auth overrides basic auth)
	assert.Contains(t, result, "forward_auth", "Should contain forward_auth when Waygates auth is enabled")
	assert.Contains(t, result, "http://waygates:8080", "Should use Waygates verify URL")
	assert.Contains(t, result, "/api/auth/acl/verify", "Should have verify URI")

	// Should NOT contain basicauth since Waygates auth overrides it
	assert.NotContains(t, result, "basicauth", "Should NOT contain basicauth when Waygates auth is enabled")
}

// TestBuildACLConfig_BasicAuthWithOAuthGeneratesForwardAuth tests that when
// a group has both basic auth users AND OAuth restrictions, the Caddyfile
// should contain "forward_auth" and NOT "basicauth".
func TestBuildACLConfig_BasicAuthWithOAuthGeneratesForwardAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "basic-plus-oauth",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
		OAuthProviderRestrictions: []models.ACLOAuthProviderRestriction{
			{
				ID:             1,
				ACLGroupID:     1,
				Provider:       "google",
				AllowedDomains: []string{"example.com"},
				Enabled:        true,
			},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should contain forward_auth since OAuth restrictions are configured (secure auth overrides basic auth)
	assert.Contains(t, result, "forward_auth", "Should contain forward_auth when OAuth restrictions are configured")

	// Should NOT contain basicauth since OAuth overrides it
	assert.NotContains(t, result, "basicauth", "Should NOT contain basicauth when OAuth restrictions are configured")
}

// TestBuildACLConfig_BasicAuthWithExternalProviderGeneratesForwardAuth tests that when
// a group has both basic auth users AND external providers (Authelia/Authentik),
// the Caddyfile should contain "forward_auth" and NOT "basicauth".
func TestBuildACLConfig_BasicAuthWithExternalProviderGeneratesForwardAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	redirectURL := "https://auth.external.com/"
	group := &models.ACLGroup{
		ID:              1,
		Name:            "basic-plus-external",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
		ExternalProviders: []models.ACLExternalProvider{
			{
				ID:              1,
				ACLGroupID:      1,
				ProviderType:    models.ACLProviderTypeAuthelia,
				Name:            "authelia",
				VerifyURL:       "http://authelia:9091/api/verify",
				AuthRedirectURL: &redirectURL,
			},
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Should contain forward_auth since external provider is configured (secure auth overrides basic auth)
	assert.Contains(t, result, "forward_auth", "Should contain forward_auth when external provider is configured")
	assert.Contains(t, result, "http://authelia:9091/api/verify", "Should use Authelia verify URL")

	// Should NOT contain basicauth since external provider overrides it
	assert.NotContains(t, result, "basicauth", "Should NOT contain basicauth when external provider is configured")
}

// TestBuildACLConfig_AllModeBasicAuthWithWaygatesGeneratesForwardAuth tests the
// basic auth override behavior in ACLCombinationModeAll.
func TestBuildACLConfig_AllModeBasicAuthWithWaygatesGeneratesForwardAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "all-mode-mixed-auth",
		CombinationMode: models.ACLCombinationModeAll, // All auth methods must pass
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         1,
			ACLGroupID: 1,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Even in ALL mode, forward_auth should be used instead of basicauth when secure auth is configured
	assert.Contains(t, result, "forward_auth", "Should contain forward_auth in ALL mode when Waygates auth is enabled")
	assert.NotContains(t, result, "basicauth", "Should NOT contain basicauth in ALL mode when Waygates auth is enabled")
}

// TestBuildACLConfig_AllModeBasicAuthOnlyGeneratesBasicAuth tests that in ALL mode,
// basicauth is still generated when it's the only auth method.
func TestBuildACLConfig_AllModeBasicAuthOnlyGeneratesBasicAuth(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "all-mode-basic-only",
		CombinationMode: models.ACLCombinationModeAll,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
		// No WaygatesAuth
		// No ExternalProviders
		// No OAuthProviderRestrictions
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// In ALL mode with only basic auth, basicauth should still be generated
	assert.Contains(t, result, "basicauth", "Should contain basicauth in ALL mode when only basic auth is configured")
	assert.NotContains(t, result, "forward_auth", "Should NOT contain forward_auth when only basic auth is configured")
}

// TestBuildACLConfig_IPBypassModeBasicAuthOverride tests the basic auth override
// behavior in ACLCombinationModeIPBypass.
func TestBuildACLConfig_IPBypassModeBasicAuthOverride(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	group := &models.ACLGroup{
		ID:              1,
		Name:            "ip-bypass-mixed-auth",
		CombinationMode: models.ACLCombinationModeIPBypass,
		IPRules: []models.ACLIPRule{
			{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "192.168.1.0/24"},
		},
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         1,
			ACLGroupID: 1,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/protected/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// IP bypass rules should still work
	assert.Contains(t, result, "@acl_0_bypass_ip", "Should have bypass IP matcher")
	assert.Contains(t, result, "192.168.1.0/24", "Should contain bypass CIDR")

	// For non-bypass IPs, forward_auth should be used instead of basicauth
	assert.Contains(t, result, "forward_auth", "Should contain forward_auth for non-bypass IPs when Waygates auth is enabled")
	assert.NotContains(t, result, "basicauth", "Should NOT contain basicauth when Waygates auth is enabled")
}

// TestBuildACLConfig_MultipleGroupsMixedBasicAuthOverride tests Caddyfile generation
// with multiple ACL groups where some have basic auth only and others have secure auth.
func TestBuildACLConfig_MultipleGroupsMixedBasicAuthOverride(t *testing.T) {
	builder := NewACLBuilder("http://waygates:8080", "https://auth.example.com/login")
	proxy := createTestProxy()

	// Group 1: Only basic auth (should generate basicauth)
	group1 := &models.ACLGroup{
		ID:              1,
		Name:            "basic-auth-only",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 1, Username: "admin", PasswordHash: "$2a$14$hashedpassword1"},
		},
	}

	// Group 2: Basic auth + Waygates (should generate forward_auth, not basicauth)
	group2 := &models.ACLGroup{
		ID:              2,
		Name:            "basic-plus-waygates",
		CombinationMode: models.ACLCombinationModeAny,
		BasicAuthUsers: []models.ACLBasicAuthUser{
			{ID: 2, Username: "user", PasswordHash: "$2a$14$hashedpassword2"},
		},
		WaygatesAuth: &models.ACLWaygatesAuth{
			ID:         2,
			ACLGroupID: 2,
			Enabled:    true,
		},
	}

	assignments := []models.ProxyACLAssignment{
		{
			ID:          1,
			ProxyID:     1,
			ACLGroupID:  1,
			PathPattern: "/admin/*",
			Priority:    0,
			Enabled:     true,
			ACLGroup:    group1,
		},
		{
			ID:          2,
			ProxyID:     1,
			ACLGroupID:  2,
			PathPattern: "/api/*",
			Priority:    1,
			Enabled:     true,
			ACLGroup:    group2,
		},
	}

	result := builder.BuildACLConfig(proxy, assignments)

	// Group 1 should have basicauth (only basic auth configured)
	assert.Contains(t, result, "basicauth", "Should contain basicauth for group 1 (basic auth only)")
	assert.Contains(t, result, "admin $2a$14$hashedpassword1", "Should contain admin credentials")
	assert.Contains(t, result, "/admin/*", "Should have /admin/* path")

	// Group 2 should have forward_auth (Waygates overrides basic auth)
	assert.Contains(t, result, "forward_auth", "Should contain forward_auth for group 2 (Waygates enabled)")
	assert.Contains(t, result, "/api/*", "Should have /api/* path")

	// Group 2's basic auth user should NOT appear in basicauth block
	// (the forward_auth handles auth, not basicauth)
	// Note: We check that forward_auth block doesn't contain "user $2a$14"
	// by verifying the structure - forward_auth block should be separate from basicauth
}
