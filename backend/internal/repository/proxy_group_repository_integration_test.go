package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

// NOTE: `ptr` is already defined in testhelpers_test.go (package repository) —
// do not redefine it here.

func createGroup(t *testing.T, repo *ProxyGroupRepository, userID int, name string, base *string) *models.ProxyGroup {
	t.Helper()
	g := &models.ProxyGroup{Name: name, BaseDomain: base, CreatedBy: userID}
	require.NoError(t, repo.Create(g))
	return g
}

func TestProxyGroupRepository_DeleteWithMembersIsBlockedByTheDatabase(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	groupRepo := NewProxyGroupRepository(tdb.DB)
	proxyRepo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	g := createGroup(t, groupRepo, user.ID, "internal", ptr("group.acme.in"))

	p := &models.Proxy{
		Type: models.ProxyTypeReverseProxy, Name: "svc", Hostname: "abc.group.acme.in",
		HostnameLabel: ptr("abc"), GroupID: &g.ID, IsActive: true, CreatedBy: user.ID,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}
	require.NoError(t, proxyRepo.Create(p))

	// Bypass the service entirely: ON DELETE RESTRICT must hold on its own.
	err := tdb.DB.Exec("DELETE FROM proxy_groups WHERE id = ?", g.ID).Error
	require.Error(t, err, "ON DELETE RESTRICT must block this at the database")
}

func TestProxyGroupRepository_UpdateBaseDomainRehomesMembers(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	groupRepo := NewProxyGroupRepository(tdb.DB)
	proxyRepo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	g := createGroup(t, groupRepo, user.ID, "internal", ptr("group.acme.in"))
	for _, label := range []string{"abc", "def"} {
		require.NoError(t, proxyRepo.Create(&models.Proxy{
			Type: models.ProxyTypeReverseProxy, Name: label,
			Hostname: label + ".group.acme.in", HostnameLabel: ptr(label),
			GroupID: &g.ID, IsActive: true, CreatedBy: user.ID,
			Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
		}))
	}

	update := &models.ProxyGroup{ID: g.ID, Name: g.Name, BaseDomain: ptr("g2.acme.in")}
	require.NoError(t, groupRepo.UpdateGroupTx(update, true))

	members, err := groupRepo.ListMembers(g.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	hosts := []string{members[0].Hostname, members[1].Hostname}
	assert.ElementsMatch(t, []string{"abc.g2.acme.in", "def.g2.acme.in"}, hosts)
}

// The cache-drift invariant: hostname == label + "." + base_domain, always.
func TestProxyGroupRepository_HostnameCacheNeverDrifts(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	groupRepo := NewProxyGroupRepository(tdb.DB)
	proxyRepo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	g := createGroup(t, groupRepo, user.ID, "internal", ptr("group.acme.in"))
	require.NoError(t, proxyRepo.Create(&models.Proxy{
		Type: models.ProxyTypeReverseProxy, Name: "svc", Hostname: "abc.group.acme.in",
		HostnameLabel: ptr("abc"), GroupID: &g.ID, IsActive: true, CreatedBy: user.ID,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}))
	require.NoError(t, groupRepo.UpdateGroupTx(&models.ProxyGroup{ID: g.ID, Name: g.Name, BaseDomain: ptr("g2.acme.in")}, true))

	assertNoHostnameDrift(t, tdb)
}

// assertNoHostnameDrift is the invariant guard. Call it at the end of any test
// that mutates groups or members.
func assertNoHostnameDrift(t *testing.T, tdb *TestDB) {
	t.Helper()
	var bad int64
	// IS DISTINCT FROM (not <>): when g.base_domain IS NULL, label || '.' ||
	// base_domain is NULL too, and `p.hostname <> NULL` evaluates to NULL, not
	// TRUE — so a drifted row whose group has a NULL base_domain would silently
	// not count as bad. IS DISTINCT FROM treats NULL like any other value, so a
	// label-addressed member of a NULL-base_domain group (application code
	// should prevent this state, but nothing at the DB level forbids it) is
	// correctly flagged as drift.
	err := tdb.DB.Raw(`
		SELECT COUNT(*) FROM proxies p
		JOIN proxy_groups g ON g.id = p.group_id
		WHERE p.hostname_label IS NOT NULL
		  AND p.hostname IS DISTINCT FROM p.hostname_label || '.' || g.base_domain
	`).Scan(&bad).Error
	require.NoError(t, err)
	require.Zero(t, bad, "materialized hostname drifted from label + base_domain")
}

// A rename that would collide must write nothing.
func TestProxyGroupRepository_UpdateBaseDomainCollisionRollsBack(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	groupRepo := NewProxyGroupRepository(tdb.DB)
	proxyRepo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	g := createGroup(t, groupRepo, user.ID, "internal", ptr("group.acme.in"))
	require.NoError(t, proxyRepo.Create(&models.Proxy{
		Type: models.ProxyTypeReverseProxy, Name: "svc", Hostname: "abc.group.acme.in",
		HostnameLabel: ptr("abc"), GroupID: &g.ID, IsActive: true, CreatedBy: user.ID,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}))
	// An ungrouped proxy already occupies the destination hostname.
	require.NoError(t, proxyRepo.Create(&models.Proxy{
		Type: models.ProxyTypeReverseProxy, Name: "squatter", Hostname: "abc.g2.acme.in",
		IsActive: true, CreatedBy: user.ID,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}))

	update := &models.ProxyGroup{ID: g.ID, Name: g.Name, BaseDomain: ptr("g2.acme.in")}
	err := groupRepo.UpdateGroupTx(update, true)
	require.Error(t, err, "colliding rename must fail")

	reloaded, err := groupRepo.GetByID(g.ID)
	require.NoError(t, err)
	assert.Equal(t, "group.acme.in", *reloaded.BaseDomain, "group must be unchanged")

	members, err := groupRepo.ListMembers(g.ID)
	require.NoError(t, err)
	assert.Equal(t, "abc.group.acme.in", members[0].Hostname, "member must be unchanged")
	assertNoHostnameDrift(t, tdb)
}

// The atomicity invariant: a collision on the group's *name* (unrelated to
// hostnames) must still roll back the base_domain change and the member
// re-home together, in the same transaction. Before this fix, UpdateGroup
// called UpdateBaseDomainTx (one committed transaction: base_domain + member
// hostnames) and then a separate Update(g) (a second committed transaction:
// name + other settings). A name collision failed only the second call, so
// the rename and re-home from the first call were already durably committed
// — the caller saw an error, but the database had already moved. With
// UpdateGroupTx, both writes happen inside one r.db.Transaction, so this
// name-only collision must leave every column exactly as it was.
func TestProxyGroupRepository_UpdateGroupTxNameCollisionRollsBackEverything(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	groupRepo := NewProxyGroupRepository(tdb.DB)
	proxyRepo := NewProxyRepository(tdb.DB)
	user := CreateTestUser(t, tdb.DB)

	g := createGroup(t, groupRepo, user.ID, "internal", ptr("group.acme.in"))
	require.NoError(t, proxyRepo.Create(&models.Proxy{
		Type: models.ProxyTypeReverseProxy, Name: "svc", Hostname: "abc.group.acme.in",
		HostnameLabel: ptr("abc"), GroupID: &g.ID, IsActive: true, CreatedBy: user.ID,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}))
	// A second group already holds the name we're about to rename into. No
	// hostname collision is involved at all — g2.acme.in is untouched by
	// anything else, so if this test fails, it can only be the name write.
	createGroup(t, groupRepo, user.ID, "taken", ptr("other.acme.in"))

	update := &models.ProxyGroup{ID: g.ID, Name: "taken", BaseDomain: ptr("g2.acme.in")}
	err := groupRepo.UpdateGroupTx(update, true)
	require.Error(t, err, "name collision must fail the whole update")

	reloaded, err := groupRepo.GetByID(g.ID)
	require.NoError(t, err)
	assert.Equal(t, "internal", reloaded.Name, "group name must be unchanged")
	require.NotNil(t, reloaded.BaseDomain)
	assert.Equal(t, "group.acme.in", *reloaded.BaseDomain, "group base_domain must be unchanged")

	members, err := groupRepo.ListMembers(g.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "abc.group.acme.in", members[0].Hostname, "member hostname must be unchanged")

	assertNoHostnameDrift(t, tdb)
}
