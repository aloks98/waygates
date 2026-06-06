//go:build traffic

package integration

import (
	"net/http"
	"testing"
)

func TestTraffic_Smoke(t *testing.T) {
	env := SetupTrafficEnvironment(t)
	resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/health", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected health 200, got %d", resp.StatusCode)
	}
}
