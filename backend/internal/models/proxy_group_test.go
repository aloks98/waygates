package models

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

// columnNames parses a model with GORM's schema package and returns its DB
// column names. GORM's naming strategy has drifted from our migrations before
// (an underscore is not inserted before a digit), so every column is asserted.
func columnNames(t *testing.T, model interface{}) map[string]bool {
	t.Helper()
	s, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)
	out := map[string]bool{}
	for _, f := range s.Fields {
		if f.DBName != "" {
			out[f.DBName] = true
		}
	}
	return out
}

func TestProxyGroup_ColumnsMatchMigration(t *testing.T) {
	cols := columnNames(t, &ProxyGroup{})
	for _, want := range []string{
		"id", "name", "description", "base_domain",
		"ssl_enabled", "ssl_forced", "tls_insecure_skip_verify",
		"block_exploits", "custom_headers",
		"created_by", "created_at", "updated_at",
	} {
		assert.True(t, cols[want], "ProxyGroup missing column %q", want)
	}
	assert.False(t, cols["member_count"], "member_count must not be persisted")
}

func TestProxyGroupACLAssignment_ColumnsMatchMigration(t *testing.T) {
	cols := columnNames(t, &ProxyGroupACLAssignment{})
	for _, want := range []string{
		"id", "proxy_group_id", "acl_group_id",
		"path_pattern", "priority", "enabled", "created_at", "updated_at",
	} {
		assert.True(t, cols[want], "ProxyGroupACLAssignment missing column %q", want)
	}
}

func TestProxy_GroupColumnsMatchMigration(t *testing.T) {
	cols := columnNames(t, &Proxy{})
	assert.True(t, cols["group_id"])
	assert.True(t, cols["hostname_label"])
	assert.False(t, cols["group_name"], "group_name must not be persisted")
}

func TestProxy_ValidateRejectsLabelWithoutGroup(t *testing.T) {
	label := "abc"
	p := &Proxy{
		Type: ProxyTypeRedirect, Name: "n", Hostname: "abc.example.com",
		HostnameLabel:  &label,
		RedirectConfig: JSONField{"to": "https://x"},
	}
	assert.ErrorIs(t, p.Validate(), ErrLabelRequiresGroup)
}

func TestProxy_ValidateRejectsDottedLabel(t *testing.T) {
	label, gid := "a.b", 1
	p := &Proxy{
		Type: ProxyTypeRedirect, Name: "n", Hostname: "a.b.example.com",
		GroupID: &gid, HostnameLabel: &label,
		RedirectConfig: JSONField{"to": "https://x"},
	}
	assert.ErrorIs(t, p.Validate(), ErrLabelNotASingleLabel)
}
