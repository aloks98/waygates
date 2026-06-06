//go:build traffic

package integration

import (
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
}
