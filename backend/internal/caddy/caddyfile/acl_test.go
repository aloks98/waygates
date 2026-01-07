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
