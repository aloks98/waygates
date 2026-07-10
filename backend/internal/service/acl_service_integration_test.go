package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// TestACLService_AssignToProxy_Integration_EnabledFalsePersists is the regression guard
// for the ProxyACLAssignment.Enabled `default:true` GORM hazard: a `default:`
// tag makes GORM omit a zero-value column from INSERT entirely, so an
// explicit Enabled: false would silently fall through to the DB's own
// DEFAULT true and come back true on reload — exactly the bug this task
// fixes (models/acl.go:226 dropped the `default:true` tag).
//
// This test goes through the real ACLService.AssignToProxy backed by a real
// Postgres container (not a mock), and reloads the row from the DB rather
// than trusting the in-memory struct the service just wrote — a mock-repo
// test would not catch a GORM tag regression since mocks never round-trip
// through GORM's INSERT column selection.
//
// It reuses setupImportTestDB/createImportTestUser from proxy_import_test.go
// rather than repository.SetupTestDB: the latter lives in a _test.go file in
// package repository and is therefore not importable from here.
func TestACLService_AssignToProxy_Integration_EnabledFalsePersists(t *testing.T) {
	tdb := setupImportTestDB(t)
	defer tdb.cleanup(t)

	user := createImportTestUser(t, tdb.db)

	proxyRepo := repository.NewProxyRepository(tdb.db)
	aclRepo := repository.NewACLRepository(tdb.db)

	proxy := validReverseProxyFixture()
	proxy.CreatedBy = user.ID
	proxy.IsActive = true
	require.NoError(t, proxyRepo.Create(proxy))

	group := &models.ACLGroup{
		Name:            "Integration Test ACL Group",
		CombinationMode: models.ACLCombinationModeAny,
		CreatedBy:       user.ID,
	}
	require.NoError(t, aclRepo.CreateGroup(group))

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	err := svc.AssignToProxy(proxy.ID, group.ID, "/*", 0, false)
	require.NoError(t, err)

	// Reload from the DB — not the struct the service just built — so this
	// actually proves persistence, not just that the service set the field
	// in memory.
	assignments, err := aclRepo.GetProxyACLAssignments(proxy.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.False(t, assignments[0].Enabled,
		"an explicit enabled=false must survive INSERT, not be resurrected by the DB's DEFAULT true")

	// Sanity check the positive case in the same run: an explicit true must
	// still persist as true, so this test isn't just measuring "always false".
	group2 := &models.ACLGroup{
		Name:            "Integration Test ACL Group Enabled",
		CombinationMode: models.ACLCombinationModeAny,
		CreatedBy:       user.ID,
	}
	require.NoError(t, aclRepo.CreateGroup(group2))

	require.NoError(t, svc.AssignToProxy(proxy.ID, group2.ID, "/api/*", 0, true))

	assignments, err = aclRepo.GetProxyACLAssignments(proxy.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 2)
	for _, a := range assignments {
		if a.ACLGroupID == group2.ID {
			assert.True(t, a.Enabled)
		}
	}
}
