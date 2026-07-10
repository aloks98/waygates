package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// TestProxyExportImport_Integration_RoundTripPreservesGroupInheritance is the
// regression guard for the export/import group-awareness bug: exporting a
// proxy that INHERITS ssl_enabled from its ProxyGroup (raw nil on the model,
// which proxygroup.Resolve resolves to true since the group has
// ssl_enabled=true) must carry that nil through export as JSON null — never
// coerce it to false — and importing the export must land back as a NULL
// ssl_enabled column: still inheriting (and therefore still HTTPS), not an
// explicit false (plaintext).
//
// This test fails against the old derefBool-based ProxyExport: derefBool(nil)
// returns false, so the export would carry "ssl_enabled":false, and importing
// that would persist an explicit false — silently flipping the proxy from
// HTTPS to plaintext on a plain export -> import round-trip. (Confirmed by
// temporarily reverting ProxyExport/newProxyExport to the derefBool shim and
// re-running this test: it fails both assertions below with "ssl_enabled
// must serialize as JSON null" / "imported proxy must still have ssl_enabled
// IS NULL", since export.SSLEnabled and fetched.SSLEnabled become non-nil
// *bool(false) instead of nil.)
func TestProxyExportImport_Integration_RoundTripPreservesGroupInheritance(t *testing.T) {
	tdb := setupImportTestDB(t)
	defer tdb.cleanup(t)

	user := createImportTestUser(t, tdb.db)

	proxyRepo := repository.NewProxyRepository(tdb.db)
	groupRepo := repository.NewProxyGroupRepository(tdb.db)
	svc := NewProxyService(ProxyServiceConfig{
		Repo:        proxyRepo,
		GroupRepo:   groupRepo,
		SyncService: &MockProxySyncer{}, // no-op SyncProxy
	})

	// A group with base_domain set (label-addressed members) and
	// ssl_enabled=true, so a member that inherits resolves to HTTPS.
	baseDomain := "group.acme.in"
	group := &models.ProxyGroup{
		Name:       "Acme Group",
		BaseDomain: &baseDomain,
		SSLEnabled: ptr(true),
		CreatedBy:  user.ID,
	}
	require.NoError(t, groupRepo.Create(group))

	// A member proxy, label-addressed into the group, that leaves
	// ssl_enabled nil — i.e. it inherits the group's true.
	label := "abc"
	member := &models.Proxy{
		Type:          models.ProxyTypeReverseProxy,
		Name:          "Member Proxy",
		GroupID:       &group.ID,
		HostnameLabel: &label,
		// SSLEnabled intentionally left nil: inherits from the group.
		Upstreams: []interface{}{
			map[string]interface{}{"address": "http://localhost:8080"},
		},
	}
	require.NoError(t, svc.CreateProxy(member, user.ID))
	require.Equal(t, "abc.group.acme.in", member.Hostname, "materializeHostname should compose label.base_domain")
	require.Nil(t, member.SSLEnabled, "precondition: member must start inheriting (nil), not an explicit value")

	// --- Export ---
	exports, err := svc.ExportProxies([]int{member.ID}, ListProxiesRequest{})
	require.NoError(t, err)
	require.Len(t, exports, 1)
	export := exports[0]

	assert.Nil(t, export.SSLEnabled, "export must carry ssl_enabled as null, not false, for an inheriting proxy")
	require.NotNil(t, export.GroupID, "export must carry group_id")
	assert.Equal(t, group.ID, *export.GroupID)
	require.NotNil(t, export.HostnameLabel, "export must carry hostname_label")
	assert.Equal(t, "abc", *export.HostnameLabel)

	// Confirm the wire JSON literally carries "ssl_enabled":null, not false —
	// belt-and-suspenders against a stray zero-value default anywhere in the
	// marshaling path.
	raw, err := json.Marshal(export)
	require.NoError(t, err)
	var wire map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &wire))
	sslRaw, present := wire["ssl_enabled"]
	require.True(t, present, "ssl_enabled key must be present in the export JSON")
	assert.Nil(t, sslRaw, "ssl_enabled must serialize as JSON null")
	assert.Equal(t, float64(group.ID), wire["group_id"])
	assert.Equal(t, "abc", wire["hostname_label"])

	// --- Import the export as a NEW proxy. Give it a different
	// hostname_label so it doesn't collide with the original member's
	// "abc.group.acme.in" — mirrors how the API handler decodes an import
	// item straight into a models.Proxy (matching JSON tags, no coercion). ---
	var imported models.Proxy
	require.NoError(t, json.Unmarshal(raw, &imported))
	newLabel := "xyz"
	imported.HostnameLabel = &newLabel
	imported.Name = "Imported Member Proxy"
	require.Nil(t, imported.SSLEnabled, "decoding the export JSON must not coerce ssl_enabled to a concrete bool")

	report := svc.ImportProxies([]ImportInput{{Proxy: &imported}}, false, user.ID)
	require.Equal(t, 1, report.Summary.Created, "import must succeed: %+v", report.Items)

	fetched, err := proxyRepo.GetByHostname("xyz.group.acme.in")
	require.NoError(t, err)
	assert.Nil(t, fetched.SSLEnabled,
		"imported proxy must have ssl_enabled IS NULL in the database (still inherits, resolves to HTTPS), not an explicit false (plaintext)")
	require.NotNil(t, fetched.GroupID)
	assert.Equal(t, group.ID, *fetched.GroupID)
}
