//go:build traffic

package integration

import (
	"net/http"
	"testing"
)

func TestTraffic_L4(t *testing.T) {
	env := SetupTrafficEnvironment(t)

	t.Run("any_tcp_echo", func(t *testing.T) {
		port := l4PortPool[0] // 9001
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-tcp", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "any",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "tcpecho", "port": 7000}},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		got, err := tcpEcho(t, addr, "ping-waygates")
		if err != nil {
			t.Fatalf("tcp echo: %v", err)
		}
		if got != "ping-waygates" {
			t.Fatalf("expected echo, got %q", got)
		}
	})

	t.Run("http_matcher", func(t *testing.T) {
		port := l4PortPool[1] // 9002
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-http", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "http",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "echo1", "port": 8080}},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		hn, err := httpEchoHostname(t, addr)
		if err != nil {
			t.Fatalf("http through L4: %v", err)
		}
		if hn != "echo1" {
			t.Fatalf("expected echo1 via L4 http matcher, got %q", hn)
		}
	})

	t.Run("round_robin_lb", func(t *testing.T) {
		port := l4PortPool[2] // 9003
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-lb", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "http",
				"load_balancing_policy": "round_robin",
				"upstreams": []map[string]any{
					{"host": "echo1", "port": 8080},
					{"host": "echo2", "port": 8080},
				},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		seen := map[string]bool{}
		for i := 0; i < 10; i++ {
			hn, err := httpEchoHostname(t, addr)
			if err != nil {
				t.Fatalf("request %d: %v", i, err)
			}
			seen[hn] = true
		}
		if !seen["echo1"] || !seen["echo2"] {
			t.Fatalf("expected both backends, saw %v", seen)
		}
	})

	t.Run("remote_ip_allow_deny", func(t *testing.T) {
		// Allow: 0.0.0.0/0 always includes the client → echoes.
		allowPort := l4PortPool[3] // 9004
		r1 := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-allow", "listen_port": allowPort, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "remote_ip",
				"allowed_ip_ranges":     []string{"0.0.0.0/0"},
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "tcpecho", "port": 7000}},
			}},
		})
		_ = r1.Body.Close()
		if r1.StatusCode != http.StatusCreated {
			t.Fatalf("create allow l4 proxy: %d", r1.StatusCode)
		}
		// Deny: 203.0.113.0/24 (TEST-NET-3) cannot contain the client → no route → closed.
		denyPort := l4PortPool[4] // 9005
		r2 := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-deny", "listen_port": denyPort, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "remote_ip",
				"allowed_ip_ranges":     []string{"203.0.113.0/24"},
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "tcpecho", "port": 7000}},
			}},
		})
		_ = r2.Body.Close()
		if r2.StatusCode != http.StatusCreated {
			t.Fatalf("create deny l4 proxy: %d", r2.StatusCode)
		}
		env.triggerSync(t)

		allowAddr := env.l4Addr(t, allowPort)
		env.waitL4(t, allowAddr)
		if got, err := tcpEcho(t, allowAddr, "allowed"); err != nil || got != "allowed" {
			t.Fatalf("allow case should echo, got %q err %v", got, err)
		}

		// Deny case: TCP connect to Caddy may succeed, but with no matching route the
		// connection is closed, so the echo read must fail / return nothing.
		denyAddr := env.l4Addr(t, denyPort)
		if got, err := tcpEcho(t, denyAddr, "blocked"); err == nil && got == "blocked" {
			t.Fatalf("deny case should NOT echo, but it did")
		}
	})

	t.Run("postgres_matcher", func(t *testing.T) {
		port := l4PortPool[5] // 9006
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-pg", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "postgres",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "pgtarget", "port": 5432}},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		if err := pgSelect1(t, addr); err != nil {
			t.Fatalf("SELECT 1 through L4 postgres proxy: %v", err)
		}
	})

	t.Run("tls_sni_passthrough", func(t *testing.T) {
		port := l4PortPool[6] // 9007
		// One L4 proxy, two tls routes. Each route matches a distinct SNI and
		// passes the TLS bytes straight through to a different echo's HTTPS port
		// (8443, mendhak's built-in self-signed cert). Routing is asserted via the
		// backend hostname in the echoed JSON. The tls matcher serializes as
		// {"tls":{"sni":[...]}} (an object), which is the shape caddy-l4 expects.
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-tls", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{
				{
					"matcher_type":          "tls",
					"sni_hostnames":         []string{"a.sni.test"},
					"tls_passthrough":       true,
					"tls_terminate":         false,
					"load_balancing_policy": "round_robin",
					"upstreams":             []map[string]any{{"host": "echo1", "port": 8443}},
				},
				{
					"matcher_type":          "tls",
					"sni_hostnames":         []string{"b.sni.test"},
					"tls_passthrough":       true,
					"tls_terminate":         false,
					"load_balancing_policy": "round_robin",
					"upstreams":             []map[string]any{{"host": "echo2", "port": 8443}},
				},
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		hnA, err := tlsEchoHostname(t, addr, "a.sni.test")
		if err != nil {
			t.Fatalf("tls dial a: %v", err)
		}
		if hnA != "echo1" {
			t.Fatalf("SNI a.sni.test should reach echo1, got %q", hnA)
		}
		hnB, err := tlsEchoHostname(t, addr, "b.sni.test")
		if err != nil {
			t.Fatalf("tls dial b: %v", err)
		}
		if hnB != "echo2" {
			t.Fatalf("SNI b.sni.test should reach echo2, got %q", hnB)
		}
	})
}
