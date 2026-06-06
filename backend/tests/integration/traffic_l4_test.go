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
}
