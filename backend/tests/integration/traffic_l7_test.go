//go:build traffic

package integration

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestTraffic_L7(t *testing.T) {
	env := SetupTrafficEnvironment(t)

	t.Run("reverse_proxy", func(t *testing.T) {
		host := "rp.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "rp", "hostname": host,
			"upstreams": []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		got, body := env.l7Get(t, host, "/", nil)
		if got.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", got.StatusCode)
		}
		if hn := echoHostnameFromBody(body); hn != "echo1" {
			t.Fatalf("expected backend echo1, got %q", hn)
		}
	})

	t.Run("trusted_proxy_client_ip", func(t *testing.T) {
		// The harness trusts the test's source network and reads the client IP
		// from X-Forwarded-For, so Caddy resolves the real client from the
		// forwarded header and Waygates passes it upstream as X-Real-IP. This
		// validates the trusted_proxies config is accepted by live Caddy and that
		// client-IP resolution works (the tunnel / Cloudflare / Pangolin case).
		host := "tp.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "tp", "hostname": host,
			"upstreams": []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		_, body := env.l7Get(t, host, "/", map[string]string{"X-Forwarded-For": "203.0.113.77"})
		var echoed struct {
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal(body, &echoed); err != nil {
			t.Fatalf("parse echo body: %v (body=%s)", err, string(body))
		}
		if echoed.Headers["x-real-ip"] != "203.0.113.77" {
			t.Fatalf("expected upstream to receive real client IP 203.0.113.77, got headers: %v", echoed.Headers)
		}
	})

	t.Run("round_robin_lb", func(t *testing.T) {
		host := "lb.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "lb", "hostname": host,
			"upstreams": []map[string]any{
				{"host": "echo1", "port": 8080, "scheme": "http"},
				{"host": "echo2", "port": 8080, "scheme": "http"},
			},
			"load_balancing": map[string]any{"strategy": "round_robin"},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		seen := map[string]bool{}
		for i := 0; i < 10; i++ {
			_, body := env.l7Get(t, host, "/", nil)
			seen[echoHostnameFromBody(body)] = true
		}
		if !seen["echo1"] || !seen["echo2"] {
			t.Fatalf("expected both backends, saw %v", seen)
		}
	})

	t.Run("redirect", func(t *testing.T) {
		host := "rd.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "redirect", "name": "rd", "hostname": host,
			"redirect": map[string]any{
				"target": "https://target.test.local", "status_code": 301,
				"preserve_path": true, "preserve_query": true,
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusMovedPermanently)

		got, _ := env.l7Get(t, host, "/somepath", nil)
		if got.StatusCode != http.StatusMovedPermanently {
			t.Fatalf("expected 301, got %d", got.StatusCode)
		}
		// preserve_path must keep the path in the Location (verifies the
		// {http.request.uri} placeholder actually expands).
		loc := got.Header.Get("Location")
		if loc != "https://target.test.local/somepath" {
			t.Fatalf("expected path preserved in redirect, got Location %q", loc)
		}
	})

	t.Run("redirect_preserve_query", func(t *testing.T) {
		// Exercises the preserve_query branch (preserve_path = false), which
		// appends "?{query}". A request with a query string must be redirected
		// to target?<query> with the '?' separator present.
		host := "rdq.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "redirect", "name": "rdq", "hostname": host,
			"redirect": map[string]any{
				"target": "https://target.test.local", "status_code": 302,
				"preserve_path": false, "preserve_query": true,
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusFound)

		got, _ := env.l7Get(t, host, "/ignored?foo=bar&baz=1", nil)
		if got.StatusCode != http.StatusFound {
			t.Fatalf("expected 302, got %d", got.StatusCode)
		}
		loc := got.Header.Get("Location")
		if loc != "https://target.test.local?foo=bar&baz=1" {
			t.Fatalf("expected preserved query with '?' separator, got Location %q", loc)
		}
	})

	t.Run("static_try_files_spa", func(t *testing.T) {
		// Exercises try_files: an existing asset must be served as-is, and an
		// unknown path must fall back to index.html. Without the file matcher,
		// every request (including the asset) is rewritten to index.html.
		host := "spa.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "static", "name": "spa", "hostname": host,
			"static": map[string]any{
				"root_path": "/var/www/test", "index_file": "index.html",
				"try_files": []any{"{path}", "/index.html"},
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		// An existing asset is served as-is, NOT rewritten to index.html.
		_, asset := env.l7Get(t, host, "/asset.txt", nil)
		if !strings.Contains(string(asset), "WAYGATES ASSET OK") {
			t.Fatalf("expected asset.txt content, got: %q", string(asset))
		}

		// An unknown path falls back to index.html (SPA behavior).
		fb, body := env.l7Get(t, host, "/deep/client/route", nil)
		if fb.StatusCode != http.StatusOK || !strings.Contains(string(body), "WAYGATES STATIC OK") {
			t.Fatalf("expected SPA fallback to index.html, got %d / %q", fb.StatusCode, string(body))
		}
	})

	t.Run("block_exploits", func(t *testing.T) {
		host := "sec.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "sec", "hostname": host,
			"upstreams":      []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
			"block_exploits": true,
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK) // benign path is reachable

		// Benign request reaches the backend.
		ok, body := env.l7Get(t, host, "/", nil)
		if ok.StatusCode != http.StatusOK || echoHostnameFromBody(body) != "echo1" {
			t.Fatalf("benign request should reach echo1, got %d / %q", ok.StatusCode, echoHostnameFromBody(body))
		}

		// Malicious request is blocked with 403.
		//
		// NOTE: We deliberately do NOT use the plan's example trigger
		// "/index.php?cmd=../../etc/passwd". Every block_exploits rule matches on
		// "path_regexp" (Caddy's URL.Path only), so the "../" in the query string
		// is never inspected, and "/index.php" matches no rule (the vuln_scanners
		// pattern flags "xmlrpc.php", not generic ".php") -> that request reaches
		// the backend. We instead hit the vuln_scanners rule with "/.env", a clean
		// path that survives Go URL parsing and Caddy path cleaning unchanged.
		// See backend/internal/caddy/config/security.go:118-137 (pattern includes
		// "\.env"; static_response status_code 403, security.go:131).
		bad, _ := env.l7Get(t, host, "/.env", nil)
		if bad.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for exploit request, got %d", bad.StatusCode)
		}
	})

	t.Run("custom_headers", func(t *testing.T) {
		host := "hdr.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "hdr", "hostname": host,
			"upstreams": []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
			"custom_headers": map[string]any{
				"request":  map[string]string{"X-Req-Test": "req-value"},
				"response": map[string]string{"X-Res-Test": "res-value"},
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		got, body := env.l7Get(t, host, "/", nil)
		// Response header reaches the client.
		if got.Header.Get("X-Res-Test") != "res-value" {
			t.Fatalf("expected X-Res-Test response header, headers: %v", got.Header)
		}
		// Request header reaches the backend (mendhak echo reflects request
		// headers under a lowercased "headers" object in its JSON body).
		var echoed struct {
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal(body, &echoed); err != nil {
			t.Fatalf("parse echo body: %v (body=%s)", err, string(body))
		}
		if echoed.Headers["x-req-test"] != "req-value" {
			t.Fatalf("expected backend to receive X-Req-Test request header, got headers: %v", echoed.Headers)
		}
	})

	t.Run("static", func(t *testing.T) {
		host := "st.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "static", "name": "st", "hostname": host,
			"static": map[string]any{"root_path": "/var/www/test", "index_file": "index.html"},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		_, body := env.l7Get(t, host, "/", nil)
		if !strings.Contains(string(body), "WAYGATES STATIC OK") {
			t.Fatalf("static content not served, got: %q", string(body))
		}
	})

	t.Run("acl_basic_auth", func(t *testing.T) {
		// Confirmed enforcement mechanism: NATIVE Caddy HTTP Basic Auth (not
		// forward-auth). A group whose only configured method is basic-auth users
		// (hasBasicAuth && !hasSecureAuth) is built with Caddy's "authentication"
		// handler + "http_basic" provider, so unauthenticated -> 401 and correct
		// credentials -> upstream.
		// See backend/internal/caddy/config/acl_builder.go:307-310 (buildBasicAuthHandler)
		// and backend/internal/caddy/config/http_handlers.go:228-251 (AuthenticationHandler/http_basic).
		// Field names confirmed: CreateGroupRequest{name,description},
		// AddBasicAuthUserRequest{username,password}, AssignACLRequest{acl_group_id,path_pattern,priority}.
		host := "acl.test.local"

		// 1. Create the reverse proxy to protect.
		pr := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "acl", "hostname": host,
			"upstreams": []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
		})
		if pr.StatusCode != http.StatusCreated {
			_ = pr.Body.Close()
			t.Fatalf("create proxy: %d", pr.StatusCode)
		}
		var pid struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		env.ReadJSONResponse(t, pr, &pid)
		_ = pr.Body.Close()
		if pid.Data.ID == 0 {
			t.Fatalf("proxy id not returned")
		}

		// 2. Create the ACL group.
		gr := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/acl/groups", map[string]any{
			"name": "Traffic ACL", "description": "basic-auth e2e",
		})
		if gr.StatusCode != http.StatusCreated {
			_ = gr.Body.Close()
			t.Fatalf("create acl group: %d", gr.StatusCode)
		}
		var gid struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		env.ReadJSONResponse(t, gr, &gid)
		_ = gr.Body.Close()
		if gid.Data.ID == 0 {
			t.Fatalf("acl group id not returned")
		}

		// 3. Add a basic-auth user to the group.
		ba := env.MakeAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/acl/groups/%d/basic-auth", gid.Data.ID), map[string]any{
			"username": "acluser", "password": "aclpass123",
		})
		baStatus := ba.StatusCode
		_ = ba.Body.Close()
		if baStatus != http.StatusOK && baStatus != http.StatusCreated {
			t.Fatalf("add basic-auth user: %d", baStatus)
		}

		// 4. Assign the group to the proxy for all paths.
		as := env.MakeAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/proxies/%d/acl", pid.Data.ID), map[string]any{
			"acl_group_id": gid.Data.ID, "path_pattern": "/*", "priority": 10,
		})
		asStatus := as.StatusCode
		_ = as.Body.Close()
		if asStatus != http.StatusOK && asStatus != http.StatusCreated {
			t.Fatalf("assign acl to proxy: %d", asStatus)
		}

		env.triggerSync(t)
		env.waitL7(t, host, http.StatusUnauthorized) // protected: unauthenticated -> 401

		// Unauthenticated -> 401 (blocked by Caddy basic auth).
		un, _ := env.l7Get(t, host, "/", nil)
		if un == nil {
			t.Fatalf("unauthenticated request: no response")
		}
		if un.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 unauthenticated, got %d", un.StatusCode)
		}

		// Authenticated with correct Basic credentials -> 200 reaching echo1.
		cred := base64.StdEncoding.EncodeToString([]byte("acluser:aclpass123"))
		au, body := env.l7Get(t, host, "/", map[string]string{"Authorization": "Basic " + cred})
		if au == nil {
			t.Fatalf("authenticated request: no response")
		}
		if au.StatusCode != http.StatusOK || echoHostnameFromBody(body) != "echo1" {
			t.Fatalf("expected authed 200 from echo1, got %d / %q", au.StatusCode, echoHostnameFromBody(body))
		}
	})
}
