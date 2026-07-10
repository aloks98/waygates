package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/proxygroup"
)

// NOTE: `ptr` is already defined in builder_test.go for this package.

func buildJSONFor(t *testing.T, e proxygroup.EffectiveProxy) []byte {
	t.Helper()
	b := NewBuilder(WithLogger(zap.NewNop()))
	b.SetHTTPProxies([]proxygroup.EffectiveProxy{e})
	out, err := b.BuildJSON()
	require.NoError(t, err)
	return out
}

// A grouped proxy that inherits its settings must produce byte-identical Caddy
// JSON to an ungrouped proxy carrying those same settings written down.
//
// This is the guard against the resolver and the builder disagreeing. It cannot
// catch a wrong *system default* — both sides would be wrong identically. That
// is what TestResolve_SystemDefaultsMatchCreateHandler is for.
func TestBuildJSON_GroupedInheritsEqualsUngroupedExplicit(t *testing.T) {
	group := &models.ProxyGroup{
		ID:                    3,
		SSLEnabled:            ptr(true),
		BlockExploits:         ptr(false),
		TLSInsecureSkipVerify: ptr(true),
		CustomHeaders: models.CustomHeaders{
			Request: map[string]string{"X-Env": "prod"},
		},
	}

	inheriting := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true, GroupID: ptr(3),
		Upstreams: []interface{}{map[string]interface{}{"host": "127.0.0.1", "port": float64(8080), "scheme": "http"}},
	}

	explicit := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true,
		SSLEnabled:            ptr(true),
		BlockExploits:         ptr(false),
		TLSInsecureSkipVerify: ptr(true),
		CustomHeaders: models.CustomHeaders{
			Request: map[string]string{"X-Env": "prod"},
		},
		Upstreams: []interface{}{map[string]interface{}{"host": "127.0.0.1", "port": float64(8080), "scheme": "http"}},
	}

	got := buildJSONFor(t, proxygroup.Resolve(inheriting, group, nil, nil))
	want := buildJSONFor(t, proxygroup.Resolve(explicit, nil, nil, nil))

	require.JSONEq(t, string(want), string(got))
	require.Equal(t, string(want), string(got), "byte-identical, not merely JSON-equal")
}

// A proxy overriding its group must produce the same JSON as one that never had
// a group.
func TestBuildJSON_ProxyOverrideBeatsGroup(t *testing.T) {
	group := &models.ProxyGroup{ID: 3, BlockExploits: ptr(true)}

	overriding := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true, GroupID: ptr(3),
		BlockExploits: ptr(false),
		Upstreams:     []interface{}{map[string]interface{}{"host": "127.0.0.1", "port": float64(8080), "scheme": "http"}},
	}
	standalone := overriding
	standalone.GroupID = nil

	got := buildJSONFor(t, proxygroup.Resolve(overriding, group, nil, nil))
	want := buildJSONFor(t, proxygroup.Resolve(standalone, nil, nil, nil))

	require.Equal(t, string(want), string(got))
}

// BlockExploits drives whether security routes are emitted at all, so assert the
// group can turn them on for a silent member.
func TestBuildJSON_GroupEnablesSecurityRoutes(t *testing.T) {
	base := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true,
		BlockExploits: ptr(false),
		Upstreams:     []interface{}{map[string]interface{}{"host": "127.0.0.1", "port": float64(8080), "scheme": "http"}},
	}
	off := buildJSONFor(t, proxygroup.Resolve(base, nil, nil, nil))

	member := base
	member.BlockExploits = nil
	member.GroupID = ptr(3)
	on := buildJSONFor(t, proxygroup.Resolve(member, &models.ProxyGroup{ID: 3, BlockExploits: ptr(true)}, nil, nil))

	// Assert the direction of the difference, not merely that the bytes differ:
	// a change anywhere would satisfy NotEqual while security routes stayed off.
	// SecurityRoutesForHost prepends extra host-matched routes, so the hostname
	// appears strictly more often once BlockExploits resolves true.
	offN := strings.Count(string(off), "svc.acme.in")
	onN := strings.Count(string(on), "svc.acme.in")
	require.Greater(t, onN, offN, "group must enable security routes for a silent member")
}
