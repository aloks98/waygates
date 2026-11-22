package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

func TestExternalVerificationEndpoint(t *testing.T) {
	for _, tc := range []struct{ url, dial, uri, sni, host string }{
		{"http://outpost:9000/outpost.goauthentik.io/auth/caddy?a=b%2Fc&x=1", "outpost:9000", "/outpost.goauthentik.io/auth/caddy?a=b%2Fc&x=1", "", ""},
		{"https://id.nexus.e412.in/outpost.goauthentik.io/auth/caddy", "id.nexus.e412.in:443", "/outpost.goauthentik.io/auth/caddy?", "id.nexus.e412.in", "id.nexus.e412.in"},
		{"https://auth.example:9443/a%2Fb?tenant=nexus", "auth.example:9443", "/a%2Fb?tenant=nexus", "auth.example", "auth.example:9443"},
		{"http://[::1]:9000", "[::1]:9000", "/?", "", ""},
	} {
		t.Run(tc.url, func(t *testing.T) {
			h := NewACLBuilder(nil).buildExternalProviderHandler(models.ACLExternalProvider{ProviderType: "custom", VerifyURL: tc.url})
			assert.Equal(t, tc.dial, h["upstreams"].([]*Upstream)[0].Dial)
			assert.Equal(t, &RewriteHeaders{Method: "GET", URI: tc.uri}, h["rewrite"])
			transport := h["transport"].(*HTTPTransport)
			if tc.sni == "" {
				assert.Nil(t, transport.TLS)
			} else {
				require.NotNil(t, transport.TLS)
				assert.Equal(t, tc.sni, transport.TLS.ServerName)
				assert.False(t, transport.TLS.InsecureSkipVerify)
				assert.Equal(t, []string{tc.host}, h["headers"].(*HeadersConfig).Request.Set["Host"])
			}
		})
	}
}

func TestExternalAuthRedirectDeprecated(t *testing.T) {
	provider := models.ACLExternalProvider{ProviderType: "authentik", VerifyURL: "http://outpost/verify"}
	original := NewACLBuilder(nil).buildExternalProviderHandler(provider)
	redirect := "https://misleading.example/login"
	provider.AuthRedirectURL = &redirect
	after := NewACLBuilder(nil).buildExternalProviderHandler(provider)
	assert.Equal(t, original, after)
	responses := after["handle_response"].([]*HandleResponse)
	require.Len(t, responses, 1)
	assert.Equal(t, []int{2}, responses[0].Match.StatusCode)
}

func TestAuthentikOutpostRoute(t *testing.T) {
	outpost := "https://outpost.example:9443"
	group := &models.ACLGroup{ExternalProviders: []models.ACLExternalProvider{{Name: "auth", ProviderType: "authentik", VerifyURL: "http://verify.example/check", OutpostURL: &outpost}}}
	route, err := authentikOutpostRoute("app.example", []*models.ACLGroup{group})
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, MatchPath{"/outpost.goauthentik.io", "/outpost.goauthentik.io/*"}, route.Match[0]["path"])
	handler := route.Handle[0]
	assert.NotContains(t, handler, "rewrite")
	assert.NotContains(t, handler, "handle_response")
	assert.Equal(t, "outpost.example:9443", handler["upstreams"].([]*Upstream)[0].Dial)
	assert.Equal(t, "outpost.example", handler["transport"].(*HTTPTransport).TLS.ServerName)
	other := &models.ACLGroup{ExternalProviders: []models.ACLExternalProvider{{Name: "other", ProviderType: "authentik", VerifyURL: "https://other.example/verify"}}}
	_, err = authentikOutpostRoute("app.example", []*models.ACLGroup{group, other})
	require.Error(t, err)
}

func TestACLInvalidConfigurationFailsClosed(t *testing.T) {
	for _, raw := range []string{"", "ftp://auth.example/verify", "https://user:pass@auth.example/verify"} {
		group := models.ACLGroup{ExternalProviders: []models.ACLExternalProvider{{ProviderType: "custom", Name: "broken", VerifyURL: raw}}}
		routes, err := NewACLBuilder(nil).BuildACLRoutes("app.example", "/private/*", &group, NewReverseProxyHandler(&Upstream{Dial: "app:8080"}))
		require.NoError(t, err)
		data, err := json.Marshal(routes)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"status_code":503`)
		assert.NotContains(t, string(data), `"handler":"reverse_proxy"`)
	}
	proxy := createReverseProxy(1, "app", "app.example", []interface{}{createTestUpstream("app", 8080, "http")}, true, false)
	routes, err := NewHTTPBuilder(nil).BuildReverseProxyRoutesWithACL(&proxy, []models.ProxyACLAssignment{{ACLGroupID: 99, PathPattern: "/*", Enabled: true}}, nil, NewACLBuilder(nil))
	require.NoError(t, err)
	require.Len(t, routes, 3)
	assert.Equal(t, 503, routes[1].Handle[0]["status_code"])
	assert.True(t, routes[1].Terminal)
}
