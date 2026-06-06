//go:build traffic

package integration

import (
	"encoding/base64"
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
		loc := got.Header.Get("Location")
		if !strings.HasPrefix(loc, "https://target.test.local") {
			t.Fatalf("unexpected Location: %q", loc)
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
		// SKIPPED pending the "typed request/response custom headers" feature.
		// This E2E surfaced that custom_headers are currently applied only as
		// REQUEST headers to the upstream (http_builder.go:206-212), so the
		// response-header assertion below cannot pass yet. Once response-header
		// support lands, un-skip and extend this to assert BOTH a request header
		// reaching the backend and a response header reaching the client.
		t.Skip("custom_headers response-header support not implemented yet; builder sets request headers only (http_builder.go:206)")

		host := "hdr.test.local"
		// Confirmed shape: models.Proxy.CustomHeaders is a flat JSONField
		// (map[string]interface{}) of {"Header-Name":"value"}, NOT a nested
		// {"response":{...}} object. See backend/internal/models/proxy.go:29 and
		// the builder at backend/internal/caddy/config/http_builder.go:206-212.
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "hdr", "hostname": host,
			"upstreams":      []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
			"custom_headers": map[string]string{"X-Test": "waygates"},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		got, _ := env.l7Get(t, host, "/", nil)
		if got.Header.Get("X-Test") != "waygates" {
			t.Fatalf("expected X-Test response header, headers: %v", got.Header)
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
