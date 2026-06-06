//go:build traffic

package integration

import (
	"net/http"
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
}
