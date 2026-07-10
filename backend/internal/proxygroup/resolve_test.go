package proxygroup

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aloks98/waygates/backend/internal/models"
)

func ptr[T any](v T) *T { return &v }

// TestResolve_SystemDefaultsMatchCreateHandler pins the defaults to what
// handlers/proxy.go:235-248 applies today. These are NOT all-false: an omitted
// ssl_enabled currently yields true, and defaulting it to false would serve new
// proxies over plaintext. The equivalence test cannot catch that, because it
// compares grouped against ungrouped and both would be wrong identically.
func TestResolve_SystemDefaultsMatchCreateHandler(t *testing.T) {
	e := Resolve(models.Proxy{}, nil, nil, nil)

	assert.True(t, e.SSLEnabled, "ssl_enabled default must be true")
	assert.True(t, e.SSLForced, "ssl_forced default must be true")
	assert.True(t, e.BlockExploits, "block_exploits default must be true")
	assert.False(t, e.TLSInsecureSkipVerify, "tls_insecure_skip_verify default must be false")
}

func TestResolve_ScalarPrecedence(t *testing.T) {
	cases := []struct {
		name  string
		proxy *bool
		group *bool
		want  bool
	}{
		{"proxy true over group false", ptr(true), ptr(false), true},
		{"proxy false over group true", ptr(false), ptr(true), false},
		{"inherit group true", nil, ptr(true), true},
		{"inherit group false", nil, ptr(false), false},
		{"no group, system default", nil, nil, false}, // tls_insecure default
		{"proxy true, no group", ptr(true), nil, true},
		{"proxy false, no group", ptr(false), nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var g *models.ProxyGroup
			if tc.group != nil {
				g = &models.ProxyGroup{TLSInsecureSkipVerify: tc.group}
			}
			e := Resolve(models.Proxy{TLSInsecureSkipVerify: tc.proxy}, g, nil, nil)
			assert.Equal(t, tc.want, e.TLSInsecureSkipVerify)
		})
	}
}

// A group with no opinion must not override a proxy that also has none — the
// system default wins, not false.
func TestResolve_SilentGroupFallsThroughToSystemDefault(t *testing.T) {
	e := Resolve(models.Proxy{}, &models.ProxyGroup{}, nil, nil)
	assert.True(t, e.SSLEnabled)
	assert.True(t, e.BlockExploits)
}

func TestResolve_HeadersMergeProxyWinsPerKey(t *testing.T) {
	g := &models.ProxyGroup{CustomHeaders: models.CustomHeaders{
		Request:  map[string]string{"X-Env": "prod", "X-Team": "core"},
		Response: map[string]string{"X-Cache": "miss"},
	}}
	p := models.Proxy{CustomHeaders: models.CustomHeaders{
		Request: map[string]string{"X-Team": "web"},
	}}

	e := Resolve(p, g, nil, nil)

	assert.Equal(t, map[string]string{"X-Env": "prod", "X-Team": "web"}, e.CustomHeaders.Request)
	assert.Equal(t, map[string]string{"X-Cache": "miss"}, e.CustomHeaders.Response)
}

// Resolve must not write through into the group's maps.
func TestResolve_HeaderMergeDoesNotMutateInputs(t *testing.T) {
	g := &models.ProxyGroup{CustomHeaders: models.CustomHeaders{
		Request: map[string]string{"X-Env": "prod"},
	}}
	p := models.Proxy{CustomHeaders: models.CustomHeaders{
		Request: map[string]string{"X-Team": "web"},
	}}

	_ = Resolve(p, g, nil, nil)

	assert.Equal(t, map[string]string{"X-Env": "prod"}, g.CustomHeaders.Request)
	assert.Equal(t, map[string]string{"X-Team": "web"}, p.CustomHeaders.Request)
}

func TestResolve_ACLUnion(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{
		{ID: 9, ProxyGroupID: 3, ACLGroupID: 1, PathPattern: "/*", Priority: 0, Enabled: true},
	}
	proxyACL := []models.ProxyACLAssignment{
		{ID: 5, ProxyID: 7, ACLGroupID: 2, PathPattern: "/admin", Priority: 1, Enabled: true},
	}

	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{ID: 3}, proxyACL, groupACL)

	assert.Len(t, e.ACL, 2)
	byGroup := map[int]models.ProxyACLAssignment{}
	for _, a := range e.ACL {
		byGroup[a.ACLGroupID] = a
	}
	// Inherited row is synthesized onto the proxy with ID 0 as its provenance mark.
	assert.Equal(t, 0, byGroup[1].ID)
	assert.Equal(t, 7, byGroup[1].ProxyID)
	assert.Equal(t, "/*", byGroup[1].PathPattern)
	// The proxy's own row is passed through untouched.
	assert.Equal(t, 5, byGroup[2].ID)
	assert.Equal(t, "/admin", byGroup[2].PathPattern)
}

func TestResolve_ACLProxyRowWinsWholesale(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{
		{ACLGroupID: 1, PathPattern: "/*", Priority: 0, Enabled: true},
	}
	proxyACL := []models.ProxyACLAssignment{
		{ID: 5, ProxyID: 7, ACLGroupID: 1, PathPattern: "/admin", Priority: 9, Enabled: true},
	}

	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{}, proxyACL, groupACL)

	assert.Len(t, e.ACL, 1)
	assert.Equal(t, "/admin", e.ACL[0].PathPattern, "proxy row must win wholesale, not field-wise")
	assert.Equal(t, 9, e.ACL[0].Priority)
}

// The documented opt-out: assign the same acl_group_id with enabled=false.
func TestResolve_ACLProxyCanOptOutOfInheritedGroup(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{
		{ACLGroupID: 1, PathPattern: "/*", Enabled: true},
	}
	proxyACL := []models.ProxyACLAssignment{
		{ID: 5, ProxyID: 7, ACLGroupID: 1, Enabled: false},
	}

	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{}, proxyACL, groupACL)

	assert.Len(t, e.ACL, 1)
	assert.False(t, e.ACL[0].Enabled, "opt-out row survives so the builder's Enabled filter drops it")
}

// A disabled group-level assignment must not be revived by inheritance.
func TestResolve_DisabledGroupACLStaysDisabled(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{{ACLGroupID: 1, Enabled: false}}
	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{}, nil, groupACL)
	assert.Len(t, e.ACL, 1)
	assert.False(t, e.ACL[0].Enabled)
}

func TestResolve_LoadBalancingIsNotInherited(t *testing.T) {
	g := &models.ProxyGroup{}
	p := models.Proxy{LoadBalancing: models.JSONField{"policy": "ip_hash"}}
	e := Resolve(p, g, nil, nil)
	assert.Equal(t, models.JSONField{"policy": "ip_hash"}, e.LoadBalancing)
}

func TestResolve_NilGroupIsIdentityOnScalars(t *testing.T) {
	p := models.Proxy{
		SSLEnabled:    ptr(false),
		BlockExploits: ptr(false),
		Hostname:      "a.example.com",
	}
	e := Resolve(p, nil, nil, nil)
	assert.False(t, e.SSLEnabled)
	assert.False(t, e.BlockExploits)
	assert.Equal(t, "a.example.com", e.Hostname)
}

func TestEffectiveHostname(t *testing.T) {
	assert.Equal(t, "abc.group.acme.in", EffectiveHostname("abc", "group.acme.in"))
}
