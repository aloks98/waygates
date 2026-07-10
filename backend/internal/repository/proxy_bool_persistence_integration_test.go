package repository

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

// TestProxyRepository_BoolDefaultPersistence reproduces the reported bug where
// turning OFF "Allow Self-Signed Certificates" (tls_insecure_skip_verify) does
// not persist, and the related ssl_enabled=false case. These columns are
// declared with `gorm:"default:..."`, which interacts with GORM's zero-value
// handling on Create.
func TestProxyRepository_BoolDefaultPersistence(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)
	defer tdb.CleanTables(t)

	repo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	// Scenario 1: CREATE with bool fields set to their NON-default values.
	//   tls_insecure_skip_verify: true  (column default false)
	//   ssl_enabled:              false (column default true)
	//   block_exploits:           false (column default true)
	//   is_active:                false (column default true) — import round-trip
	t.Run("Create_NonDefaultBools", func(t *testing.T) {
		proxy := &models.Proxy{
			Type:     models.ProxyTypeReverseProxy,
			Name:     "create-bools",
			Hostname: "create-bools.example.com",
			Upstreams: []interface{}{
				map[string]interface{}{"host": "backend", "port": 8080, "scheme": "http"},
			},
			SSLEnabled:            ptr(false), // wants HTTPS OFF
			BlockExploits:         ptr(false), // wants exploit-blocking OFF
			TLSInsecureSkipVerify: ptr(true),  // wants self-signed ON
			IsActive:              false,      // imported as inactive
			CreatedBy:             user.ID,
		}
		require.NoError(t, repo.Create(proxy))

		fetched, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)

		require.NotNil(t, fetched.TLSInsecureSkipVerify)
		assert.True(t, *fetched.TLSInsecureSkipVerify, "tls_insecure_skip_verify=true should persist on create")
		require.NotNil(t, fetched.SSLEnabled)
		assert.False(t, *fetched.SSLEnabled, "ssl_enabled=false should persist on create")
		require.NotNil(t, fetched.BlockExploits)
		assert.False(t, *fetched.BlockExploits, "block_exploits=false should persist on create")
		assert.False(t, fetched.IsActive, "is_active=false should persist on create (import round-trip)")
	})

	// Scenario 2: the exact reported bug. Create a proxy with self-signed ON,
	// then UPDATE it to OFF and verify the false value persists.
	t.Run("Update_SelfSignedOnToOff", func(t *testing.T) {
		proxy := &models.Proxy{
			Type:     models.ProxyTypeReverseProxy,
			Name:     "update-selfsigned",
			Hostname: "update-selfsigned.example.com",
			Upstreams: []interface{}{
				map[string]interface{}{"host": "backend", "port": 8080, "scheme": "http"},
			},
			SSLEnabled:            ptr(true),
			TLSInsecureSkipVerify: ptr(true), // starts ON
			CreatedBy:             user.ID,
		}
		require.NoError(t, repo.Create(proxy))

		before, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		require.NotNil(t, before.TLSInsecureSkipVerify)
		require.True(t, *before.TLSInsecureSkipVerify, "precondition: should start ON")

		// Now turn it OFF and Save (mirrors ProxyService.UpdateProxy -> repo.Update).
		before.TLSInsecureSkipVerify = ptr(false)
		require.NoError(t, repo.Update(before))

		after, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		require.NotNil(t, after.TLSInsecureSkipVerify)
		assert.False(t, *after.TLSInsecureSkipVerify, "tls_insecure_skip_verify=false should persist after update")
	})

	// Scenario 2b: production-faithful update. The handler decodes a fresh
	// models.Proxy from the JSON body (fields not sent are zero), the service
	// copies a few preserved fields from the existing row, then Save runs.
	t.Run("Update_SelfSignedOff_FreshStructFromJSON", func(t *testing.T) {
		seed := &models.Proxy{
			Type:     models.ProxyTypeReverseProxy,
			Name:     "fresh-json",
			Hostname: "fresh-json.example.com",
			Upstreams: []interface{}{
				map[string]interface{}{"host": "backend", "port": 8080, "scheme": "http"},
			},
			SSLEnabled:            ptr(true),
			TLSInsecureSkipVerify: ptr(true), // starts ON
			CreatedBy:             user.ID,
		}
		require.NoError(t, repo.Create(seed))

		// Mimic the handler: decode the PUT body the UI sends when the toggle is off.
		body := `{
			"type":"reverse_proxy",
			"name":"fresh-json",
			"hostname":"fresh-json.example.com",
			"upstreams":[{"host":"backend","port":8080,"scheme":"http"}],
			"block_exploits":true,
			"tls_insecure_skip_verify":false
		}`
		var fresh models.Proxy
		require.NoError(t, json.Unmarshal([]byte(body), &fresh))

		existing, err := repo.GetByID(seed.ID)
		require.NoError(t, err)

		// Mimic ProxyService.UpdateProxy field preservation.
		fresh.ID = seed.ID
		fresh.IsActive = existing.IsActive
		fresh.SSLForced = existing.SSLForced
		fresh.CreatedBy = existing.CreatedBy
		fresh.CreatedAt = existing.CreatedAt
		fresh.SSLEnabled = existing.SSLEnabled // handler keeps existing when not sent

		require.NoError(t, repo.Update(&fresh))

		after, err := repo.GetByID(seed.ID)
		require.NoError(t, err)
		require.NotNil(t, after.TLSInsecureSkipVerify)
		assert.False(t, *after.TLSInsecureSkipVerify, "tls_insecure_skip_verify=false should persist via the fresh-struct Save path")
	})

	// Scenario 3: ssl_enabled true->false via update (the field the handler
	// already special-cases with a *bool; confirms the Save path persists it).
	t.Run("Update_SSLEnabledOnToOff", func(t *testing.T) {
		proxy := &models.Proxy{
			Type:     models.ProxyTypeReverseProxy,
			Name:     "update-ssl",
			Hostname: "update-ssl.example.com",
			Upstreams: []interface{}{
				map[string]interface{}{"host": "backend", "port": 8080, "scheme": "http"},
			},
			SSLEnabled: ptr(true),
			CreatedBy:  user.ID,
		}
		require.NoError(t, repo.Create(proxy))

		before, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		before.SSLEnabled = ptr(false)
		require.NoError(t, repo.Update(before))

		after, err := repo.GetByID(proxy.ID)
		require.NoError(t, err)
		require.NotNil(t, after.SSLEnabled)
		assert.False(t, *after.SSLEnabled, "ssl_enabled=false should persist after update")
	})
}
