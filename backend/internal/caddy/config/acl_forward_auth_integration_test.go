package config

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Run with CADDY_BINARY pointing to the Caddy version bundled by Dockerfile.
// Each process uses loopback, a private data directory, and no admin API/ACME.
func requireCaddy(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("real Caddy integration test")
	}
	binary := os.Getenv("CADDY_BINARY")
	if binary == "" {
		t.Skip("set CADDY_BINARY to run real Caddy forward-auth tests")
	}
	return binary
}

func startACLCaddy(t *testing.T, routes []*HTTPRoute, extraEnv ...string) string {
	t.Helper()
	binary := requireCaddy(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	dir := t.TempDir()
	config := map[string]any{
		"admin": map[string]any{"disabled": true},
		"apps": map[string]any{"http": map[string]any{"servers": map[string]any{"test": map[string]any{
			"listen": []string{address}, "automatic_https": map[string]any{"disable": true}, "routes": routes,
		}}}},
	}
	data, err := json.Marshal(config)
	require.NoError(t, err)
	configPath := filepath.Join(dir, "caddy.json")
	require.NoError(t, os.WriteFile(configPath, data, 0600))
	env := append(os.Environ(), "XDG_DATA_HOME="+dir, "XDG_CONFIG_HOME="+dir)
	env = append(env, extraEnv...)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	validate := exec.CommandContext(ctx, binary, "validate", "--config", configPath)
	validate.Env = env
	output, err := validate.CombinedOutput()
	require.NoError(t, err, "Caddy JSON validation: %s\n%s", output, data)
	logPath := filepath.Join(dir, "caddy.log")
	log, err := os.Create(logPath)
	require.NoError(t, err)
	cmd := exec.Command(binary, "run", "--config", configPath)
	cmd.Env, cmd.Stdout, cmd.Stderr = env, log, log
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = log.Close()
		if t.Failed() {
			b, _ := os.ReadFile(logPath)
			t.Logf("Caddy: %s", b)
		}
	})
	require.Eventually(t, func() bool {
		connection, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = connection.Close()
		return true
	}, 5*time.Second, 20*time.Millisecond)
	return "http://" + address
}

func aclTestRoutes(t *testing.T, appURL string, builder *ACLBuilder, groups ...models.ACLGroup) []*HTTPRoute {
	t.Helper()
	parsed, err := url.Parse(appURL)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)
	proxy := createReverseProxy(1, "test", "app.example", []interface{}{createTestUpstream(parsed.Hostname(), port, "http")}, true, false)
	groupMap := make(map[int64]*models.ACLGroup)
	assignments := make([]models.ProxyACLAssignment, 0, len(groups))
	for i := range groups {
		groupMap[int64(groups[i].ID)] = &groups[i]
		path := "/*"
		if i == 0 && len(groups) > 1 {
			path = "/api/v3/movie"
		}
		assignments = append(assignments, models.ProxyACLAssignment{ACLGroupID: groups[i].ID, PathPattern: path, Priority: i, Enabled: true})
	}
	routes, err := NewHTTPBuilder(nil).BuildReverseProxyRoutesWithACL(&proxy, assignments, groupMap, builder)
	require.NoError(t, err)
	return routes
}

type authTestRequest struct {
	Method  string
	URI     string
	Host    string
	Body    []byte
	Headers http.Header
}

func captureAuthRequest(r *http.Request) authTestRequest {
	body, _ := io.ReadAll(r.Body)
	return authTestRequest{r.Method, r.RequestURI, r.Host, body, r.Header.Clone()}
}

func aclTestRequest(t *testing.T, base, method, path, body string, headers http.Header) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, base+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Host = "app.example"
	req.Header = headers.Clone()
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	res, err := client.Do(req)
	require.NoError(t, err)
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	return res, string(data)
}

func TestCaddyExternalForwardAuth(t *testing.T) {
	requireCaddy(t)
	authRequests := make(chan authTestRequest, 100)
	var appHits atomic.Int32
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authRequests <- captureAuthRequest(r)
		w.Header().Add("Set-Cookie", "session=verified; Path=/; HttpOnly")
		w.Header().Add("Set-Cookie", "state=callback; Path=/; Secure")
		if r.URL.Path != "/outpost.goauthentik.io/auth/caddy" {
			w.Header().Set("Location", "/callback-complete")
			w.WriteHeader(302)
			_, _ = io.WriteString(w, "outpost response")
			return
		}
		status, _ := strconv.Atoi(r.Header.Get("X-Test-Status"))
		if status == 0 {
			status = 200
		}
		w.Header().Set("X-Authentik-Username", "verified-user")
		w.Header().Set("X-Identity-Custom", "verified-custom")
		w.Header().Set("Remote-User", "verified-remote")
		if status >= 300 {
			w.Header().Set("Location", "https://id.example/flow/login?next=app")
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, "auth response")
	}))
	defer auth.Close()
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appHits.Add(1)
		record := captureAuthRequest(r)
		if strings.HasPrefix(r.URL.Path, "/api/v3/") && r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(401)
		}
		_ = json.NewEncoder(w).Encode(record)
	}))
	defer app.Close()
	provider := models.ACLExternalProvider{ProviderType: "authentik", Name: "test", VerifyURL: auth.URL + "/outpost.goauthentik.io/auth/caddy?tenant=nexus&scope=a%2Fb", HeadersToCopy: models.JSONStringArray{"x-authentik-username", "X-Authentik-Email", "X-Identity-Custom", "X-Identity-Missing"}}
	group := models.ACLGroup{ID: 2, Name: "protected", CombinationMode: "any", ExternalProviders: []models.ACLExternalProvider{provider}}
	exception := models.ACLGroup{ID: 1, Name: "application-api-key", CombinationMode: "any", IPRules: []models.ACLIPRule{{RuleType: "allow", CIDR: "0.0.0.0/0"}, {RuleType: "allow", CIDR: "::/0"}}}
	base := startACLCaddy(t, aclTestRoutes(t, app.URL, NewACLBuilder(nil), exception, group))
	spoof := http.Header{"X-Authentik-Username": {"attacker", "another"}, "X-Authentik-Email": {"attacker@example"}, "X-Authentik-Jwt": {"forged"}, "X-Identity-Custom": {"forged"}, "X-Identity-Missing": {"forged"}, "X-Auth-User": {"forged"}, "Remote-User": {"forged"}, "X-Forwarded-Method": {"DELETE"}, "X-Forwarded-Uri": {"/forged"}, "X-Forwarded-Host": {"evil.example"}}
	t.Run("GET subrequest preserves upload and verified identity", func(t *testing.T) {
		body := strings.Repeat("\x00upload-data\xff", 100000)
		headers := spoof.Clone()
		headers.Set("Content-Type", "application/octet-stream")
		res, data := aclTestRequest(t, base, "POST", "/upload?x=1&next=%2Fa", body, headers)
		require.Equal(t, 200, res.StatusCode)
		var got authTestRequest
		require.NoError(t, json.Unmarshal([]byte(data), &got))
		assert.Equal(t, sha256.Sum256([]byte(body)), sha256.Sum256(got.Body))
		assert.Equal(t, "POST", got.Method)
		assert.Equal(t, "/upload?x=1&next=%2Fa", got.URI)
		assert.Equal(t, "verified-user", got.Headers.Get("X-Authentik-Username"))
		assert.Equal(t, "verified-custom", got.Headers.Get("X-Identity-Custom"))
		for _, key := range []string{"X-Authentik-Email", "X-Identity-Missing", "X-Authentik-Jwt", "X-Auth-User", "Remote-User"} {
			assert.NotContains(t, got.Headers, key)
		}
		assert.Empty(t, res.Header.Get("X-Authentik-Username"))
		assert.Len(t, res.Header.Values("Set-Cookie"), 2)
		check := <-authRequests
		assert.Equal(t, "GET", check.Method)
		assert.Empty(t, check.Body)
		assert.Equal(t, "/outpost.goauthentik.io/auth/caddy?tenant=nexus&scope=a%2Fb", check.URI)
		assert.Equal(t, "POST", check.Headers.Get("X-Forwarded-Method"))
		assert.Equal(t, got.URI, check.Headers.Get("X-Forwarded-Uri"))
		assert.Equal(t, "app.example", check.Headers.Get("X-Forwarded-Host"))
		assert.Empty(t, check.Headers.Get("X-Authentik-Username"))
	})
	for _, status := range []int{201, 204, 207, 226, 302, 401, 403, 429, 500, 503} {
		t.Run(fmt.Sprintf("auth status %d", status), func(t *testing.T) {
			before := appHits.Load()
			headers := spoof.Clone()
			headers.Set("X-Test-Status", strconv.Itoa(status))
			res, data := aclTestRequest(t, base, "GET", "/private", "", headers)
			<-authRequests
			if status < 300 {
				assert.Equal(t, 200, res.StatusCode)
				assert.Equal(t, before+1, appHits.Load())
			} else {
				assert.Equal(t, status, res.StatusCode)
				assert.Equal(t, "auth response", data)
				assert.Equal(t, "https://id.example/flow/login?next=app", res.Header.Get("Location"))
				assert.Len(t, res.Header.Values("Set-Cookie"), 2)
				assert.Equal(t, before, appHits.Load())
			}
		})
	}
	for _, path := range []string{"/config.json", "/media/private.mp4", "/assets/private.js", "/.well-known/private", "/favicon.ico", "/static/styles.css", "/api/v30/movie", "/api/v3/system/status", "/api/v3", "/outpost.goauthentik.io.evil/callback"} {
		t.Run("protected "+path, func(t *testing.T) {
			before := appHits.Load()
			res, _ := aclTestRequest(t, base, "GET", path, "", http.Header{"X-Test-Status": {"403"}})
			assert.Equal(t, 403, res.StatusCode)
			assert.Equal(t, before, appHits.Load())
			<-authRequests
		})
	}
	for _, path := range []string{"/outpost.goauthentik.io/callback?code=test", "/outpost.goauthentik.io/sign_out", "/outpost.goauthentik.io/start?rd=%2F", "/outpost.goauthentik.io"} {
		t.Run("outpost "+path, func(t *testing.T) {
			before := appHits.Load()
			res, data := aclTestRequest(t, base, "POST", path, "callback-body", spoof)
			assert.Equal(t, 302, res.StatusCode)
			assert.Equal(t, "outpost response", data)
			assert.Len(t, res.Header.Values("Set-Cookie"), 2)
			assert.Equal(t, before, appHits.Load())
			got := <-authRequests
			assert.Equal(t, path, got.URI)
			assert.Equal(t, "POST", got.Method)
			assert.Equal(t, []byte("callback-body"), got.Body)
		})
	}
	t.Run("explicit API exception still requires application API key", func(t *testing.T) {
		headers := spoof.Clone()
		for _, key := range []string{"", "test-key"} {
			headers.Set("X-Api-Key", key)
			res, data := aclTestRequest(t, base, "GET", "/api/v3/movie", "", headers)
			expected := 401
			if key != "" {
				expected = 200
			}
			assert.Equal(t, expected, res.StatusCode)
			var got authTestRequest
			require.NoError(t, json.Unmarshal([]byte(data), &got))
			for _, name := range []string{"X-Authentik-Username", "X-Identity-Custom", "X-Identity-Missing", "Remote-User"} {
				assert.Empty(t, got.Headers.Get(name))
			}
			assert.Empty(t, authRequests)
		}
	})
	t.Run("auth outage fails closed", func(t *testing.T) {
		auth.Close()
		before := appHits.Load()
		res, _ := aclTestRequest(t, base, "POST", "/private", "upload", spoof)
		assert.Equal(t, 502, res.StatusCode)
		assert.Equal(t, before, appHits.Load())
	})
}

func TestCaddyForwardAuthModes(t *testing.T) {
	requireCaddy(t)
	var authHits, appHits atomic.Int32
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHits.Add(1)
		status, _ := strconv.Atoi(r.Header.Get("X-Test-Status"))
		if status == 0 {
			status = 200
		}
		w.Header().Set("X-Auth-User", "waygates-user")
		w.Header().Set("Remote-User", "authelia-user")
		w.WriteHeader(status)
	}))
	defer auth.Close()
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appHits.Add(1)
		_ = json.NewEncoder(w).Encode(captureAuthRequest(r))
	}))
	defer app.Close()
	for _, tc := range []struct {
		name, mode, rule, cidr string
		authStatus, want       int
		verify                 bool
	}{
		{"any allow", "any", "allow", "127.0.0.0/8", 403, 200, false},
		{"any unmatched", "any", "allow", "192.0.2.0/24", 403, 403, true},
		{"all allow still authenticates", "all", "allow", "127.0.0.0/8", 403, 403, true},
		{"all bypass still authenticates", "all", "bypass", "127.0.0.0/8", 200, 200, true},
		{"all unmatched", "all", "allow", "192.0.2.0/24", 200, 403, false},
		{"IP bypass", "ip_bypass", "bypass", "127.0.0.0/8", 503, 200, false},
		{"IP bypass unmatched", "ip_bypass", "bypass", "192.0.2.0/24", 503, 503, true},
		{"deny wins", "any", "deny", "127.0.0.0/8", 200, 403, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			group := models.ACLGroup{ID: 1, Name: "mode", CombinationMode: tc.mode, IPRules: []models.ACLIPRule{{RuleType: tc.rule, CIDR: tc.cidr}}, ExternalProviders: []models.ACLExternalProvider{{ProviderType: "authelia", Name: "auth", VerifyURL: auth.URL + "/verify"}}}
			if tc.rule == "deny" {
				group.IPRules = append(group.IPRules, models.ACLIPRule{RuleType: "bypass", CIDR: "127.0.0.0/8"})
			}
			base := startACLCaddy(t, aclTestRoutes(t, app.URL, NewACLBuilder(nil), group))
			beforeAuth, beforeApp := authHits.Load(), appHits.Load()
			res, data := aclTestRequest(t, base, "GET", "/private.json", "", http.Header{"X-Test-Status": {strconv.Itoa(tc.authStatus)}, "Remote-User": {"forged"}})
			assert.Equal(t, tc.want, res.StatusCode)
			if tc.verify {
				assert.Equal(t, beforeAuth+1, authHits.Load())
			} else {
				assert.Equal(t, beforeAuth, authHits.Load())
			}
			if tc.want == 200 {
				var got authTestRequest
				require.NoError(t, json.Unmarshal([]byte(data), &got))
				if tc.verify {
					assert.Equal(t, "authelia-user", got.Headers.Get("Remote-User"))
				} else {
					assert.Empty(t, got.Headers.Get("Remote-User"))
				}
			} else {
				assert.Equal(t, beforeApp, appHits.Load())
			}
		})
	}
	t.Run("Waygates GET and login behavior", func(t *testing.T) {
		group := models.ACLGroup{ID: 1, Name: "internal", CombinationMode: "any", WaygatesAuth: &models.ACLWaygatesAuth{Enabled: true}}
		base := startACLCaddy(t, aclTestRoutes(t, app.URL, NewACLBuilder(nil).SetWaygatesURLs(auth.URL, "https://login.example/auth/acl-login"), group))
		for _, status := range []int{200, 401, 403, 500} {
			before := appHits.Load()
			res, data := aclTestRequest(t, base, "POST", "/upload.json", "body", http.Header{"X-Test-Status": {strconv.Itoa(status)}, "X-Auth-User": {"forged"}})
			if status == 200 {
				var got authTestRequest
				require.NoError(t, json.Unmarshal([]byte(data), &got))
				assert.Equal(t, []byte("body"), got.Body)
				assert.Equal(t, "waygates-user", got.Headers.Get("X-Auth-User"))
			} else {
				assert.Equal(t, before, appHits.Load())
			}
			if status == 401 {
				assert.Equal(t, 302, res.StatusCode)
				assert.Contains(t, res.Header.Get("Location"), "https://login.example/auth/acl-login?redirect=")
			} else {
				assert.Equal(t, status, res.StatusCode)
			}
		}
	})
	t.Run("basic auth protects static-looking paths", func(t *testing.T) {
		user := models.ACLBasicAuthUser{Username: "alice"}
		require.NoError(t, user.SetPassword("test-password", 4))
		group := models.ACLGroup{ID: 1, Name: "basic", CombinationMode: "any", BasicAuthUsers: []models.ACLBasicAuthUser{user}}
		base := startACLCaddy(t, aclTestRoutes(t, app.URL, NewACLBuilder(nil), group))
		res, _ := aclTestRequest(t, base, "GET", "/assets/private.json", "", http.Header{})
		assert.Equal(t, 401, res.StatusCode)
		req, _ := http.NewRequest("GET", base+"/assets/private.json", nil)
		req.SetBasicAuth("alice", "test-password")
		res, _ = aclTestRequest(t, base, "GET", "/assets/private.json", "", req.Header)
		assert.Equal(t, 200, res.StatusCode)
	})
}

func TestCaddyForwardAuthTLS(t *testing.T) {
	requireCaddy(t)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	ca := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600))
	requests := make(chan authTestRequest, 10)
	auth := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- captureAuthRequest(r)
		w.Header().Set("X-Identity", "tls-user")
		w.WriteHeader(200)
	}))
	sni := make(chan string, 10)
	auth.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) { sni <- info.ServerName; return nil, nil }}
	auth.StartTLS()
	defer auth.Close()
	var appHits atomic.Int32
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appHits.Add(1)
		_, _ = io.WriteString(w, r.Header.Get("X-Identity"))
	}))
	defer app.Close()
	localURL := strings.Replace(auth.URL, "127.0.0.1", "localhost", 1)
	for _, tc := range []struct {
		name, endpoint string
		trust          bool
		want           int
	}{
		{"verified hostname and SNI", localURL, true, 200},
		{"untrusted certificate", localURL, false, 502},
		{"wrong certificate hostname", auth.URL, true, 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			group := models.ACLGroup{ID: 1, Name: "tls", CombinationMode: "any", ExternalProviders: []models.ACLExternalProvider{{Name: "tls", ProviderType: "custom", VerifyURL: tc.endpoint + "/verify", HeadersToCopy: models.JSONStringArray{"X-Identity"}}}}
			var env []string
			if tc.trust {
				env = []string{"SSL_CERT_FILE=" + ca}
			}
			base := startACLCaddy(t, aclTestRoutes(t, app.URL, NewACLBuilder(nil), group), env...)
			before := appHits.Load()
			res, body := aclTestRequest(t, base, "POST", "/upload?must-not-leak=1", "body", http.Header{})
			assert.Equal(t, tc.want, res.StatusCode)
			if tc.want == 200 {
				assert.Equal(t, "tls-user", body)
				got := <-requests
				assert.Equal(t, "/verify", got.URI)
				assert.Equal(t, strings.TrimPrefix(localURL, "https://"), got.Host)
				assert.Equal(t, "localhost", <-sni)
			} else {
				assert.Equal(t, before, appHits.Load())
				assert.Empty(t, requests)
			}
		})
	}
}

func TestCaddyOutpostOriginAndPathScope(t *testing.T) {
	requireCaddy(t)
	var verificationHits, appHits atomic.Int32
	verify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { verificationHits.Add(1); w.WriteHeader(403) }))
	defer verify.Close()
	outpost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(captureAuthRequest(r)) }))
	defer outpost.Close()
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { appHits.Add(1); w.WriteHeader(200) }))
	defer app.Close()
	u, err := url.Parse(app.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	proxy := createReverseProxy(1, "path-scoped", "app.example", []interface{}{createTestUpstream(u.Hostname(), port, "http")}, true, false)
	group := &models.ACLGroup{ID: 1, Name: "NexusID", CombinationMode: "all", ExternalProviders: []models.ACLExternalProvider{{Name: "auth", ProviderType: "authentik", VerifyURL: verify.URL + "/check", OutpostURL: &outpost.URL}}}
	routes, err := NewHTTPBuilder(nil).BuildReverseProxyRoutesWithACL(&proxy, []models.ProxyACLAssignment{{ACLGroupID: 1, PathPattern: "/private/*", Enabled: true}}, map[int64]*models.ACLGroup{1: group}, NewACLBuilder(nil))
	require.NoError(t, err)
	base := startACLCaddy(t, routes)
	res, data := aclTestRequest(t, base, "POST", "/outpost.goauthentik.io/callback?code=isolated", "callback-data", http.Header{"Cookie": {"state=isolated"}})
	require.Equal(t, 200, res.StatusCode)
	var got authTestRequest
	require.NoError(t, json.Unmarshal([]byte(data), &got))
	assert.Equal(t, "/outpost.goauthentik.io/callback?code=isolated", got.URI)
	assert.Equal(t, "state=isolated", got.Headers.Get("Cookie"))
	assert.Equal(t, []byte("callback-data"), got.Body)
	assert.Zero(t, verificationHits.Load())
	assert.Zero(t, appHits.Load())
	res, _ = aclTestRequest(t, base, "GET", "/private/media/config.json", "", http.Header{})
	assert.Equal(t, 403, res.StatusCode)
	assert.Equal(t, int32(1), verificationHits.Load())
	assert.Zero(t, appHits.Load())
}
