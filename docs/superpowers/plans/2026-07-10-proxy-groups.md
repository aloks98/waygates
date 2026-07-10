# Proxy Groups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an operator define a `ProxyGroup` once (base domain, TLS/security defaults, custom headers, ACL assignments) and have member HTTP proxies inherit those values, with any proxy free to override.

**Architecture:** A nullable `proxies.group_id` gives each HTTP proxy at most one group. The four inheritable booleans on `proxies` become nullable (`NULL` = inherit). A single pure function, `proxygroup.Resolve`, merges group defaults into a proxy and returns an `EffectiveProxy` with every value decided. The Caddy builder is retyped to accept only `EffectiveProxy`, so an unresolved proxy cannot reach config generation. The hostname is the one exception to lazy resolution: it is materialized into `proxies.hostname` (which stays `NOT NULL UNIQUE`) by the service, so every existing reader and the unique index keep working.

**Tech Stack:** Go 1.25, chi/v5, GORM, golang-migrate, zap, testify, testcontainers-go. React 19, TypeScript, Vite, TanStack Router/Query/Table, React Hook Form + Zod, Zustand, ky, Tailwind 4, `@e412/rnui-react`.

**Spec:** `docs/superpowers/specs/2026-07-10-proxy-groups-design.md`

**Branch:** `feat/proxy-groups` (already checked out; spec committed).

## Global Constraints

- **Naming.** `ACLGroup` already exists and is an *auth* grouping. This feature is never called a "group" unqualified. Entity `ProxyGroup`, table `proxy_groups`, RBAC prefix `proxygroups`, UI label "Proxy Groups".
- **System defaults are `ssl_enabled=true`, `ssl_forced=true`, `block_exploits=true`, `tls_insecure_skip_verify=false`.** Lifted verbatim from `handlers/proxy.go:235-248`. They are NOT all-false. Defaulting `ssl_enabled` to false serves new proxies over plaintext.
- **Never add a GORM `default:` tag to a `bool` or `*bool` field.** A `default:` tag makes GORM omit the column from INSERT when the field holds its zero value, silently flipping an explicit `false`. This codebase documents the hazard in `models/proxy.go:17-30`.
- **Every new model field carries an explicit `column:` tag.** GORM's naming strategy has drifted from migrations here before.
- **`Resolve` is the only place a default is applied.** Handlers pass `*bool` straight through.
- **Migrations** are `NNNNNN_snake_description.{up,down}.sql`, 6-digit sequential, next is `000014`. Use `CREATE TABLE IF NOT EXISTS`, named constraints (`uq_`, `fk_`, `chk_`), `CREATE INDEX IF NOT EXISTS idx_<table>_<col>`.
- **Data tables in the UI use `DataGrid` from `@e412/rnui-react`**, never the bare `Table`.
- **Forms bind with React Hook Form `Form`/`FormField`**, with Zod schemas in `ui/src/lib/form-validation.ts`.
- **Local gate before every commit:** `gofmt`, `go build ./...`, `go test ./... -short`. `golangci-lint`, `goimports`, `caddy`, and Docker are not installed locally; `golangci-lint` runs in CI only.
- **UI gate:** `make lint-ui`.

## File Structure

**Created (backend)**
| Path | Responsibility |
|---|---|
| `backend/migrations/000014_create_proxy_groups.up.sql` / `.down.sql` | Schema |
| `backend/internal/models/proxy_group.go` | `ProxyGroup`, `ProxyGroupACLAssignment` models |
| `backend/internal/models/proxy_group_test.go` | Schema/column-tag test |
| `backend/internal/proxygroup/resolve.go` | `EffectiveProxy`, `Resolve`, system defaults |
| `backend/internal/proxygroup/resolve_test.go` | Resolver table tests |
| `backend/internal/repository/proxy_group_repository.go` | Data access |
| `backend/internal/repository/proxy_group_repository_integration_test.go` | Testcontainer tests |
| `backend/internal/service/proxy_group_service.go` | Business logic, base-domain transaction |
| `backend/internal/service/proxy_group_service_test.go` | Unit tests with mocks |
| `backend/internal/api/handlers/proxy_group.go` | HTTP handlers |

**Modified (backend)**
| Path | Change |
|---|---|
| `backend/internal/models/proxy.go` | 4 bools → `*bool`; add `GroupID`, `HostnameLabel`, `GroupName` |
| `backend/internal/caddy/config/builder.go` | Retype to `EffectiveProxy` |
| `backend/internal/caddy/config/http_builder.go` | Retype to `EffectiveProxy` |
| `backend/internal/service/sync_service.go` | Load groups, resolve before `SetHTTPProxies` |
| `backend/internal/repository/proxy_repository.go` | `GroupID` filter, group-name summary join |
| `backend/internal/repository/interfaces.go` | Add `ProxyGroupRepositoryInterface` |
| `backend/internal/service/interfaces.go` | Add `ProxyGroupServiceInterface`, `GroupSyncer` |
| `backend/internal/api/handlers/proxy.go` | Stop defaulting; accept `group_id`/`hostname_label`; return `effective` |
| `backend/internal/api/routes/routes.go` | Wire group routes |
| `backend/rbac.yaml` | `proxygroups:*` permission group + role grants |

**Created (frontend)**: `ui/src/types/proxy-group.ts`, `ui/src/hooks/use-proxy-groups.ts`, `ui/src/routes/_dashboard/proxy-groups/{index,new}.tsx` + `$groupId/edit.tsx`, `ui/src/components/proxy-group/proxy-group-data-grid.tsx`, `ui/src/components/proxy-group/proxy-group-form.tsx`, `ui/src/components/proxy/forms/group-selector.tsx`, `ui/src/components/proxy/forms/shared/inheritable-switch.tsx`, `ui/src/components/proxy/forms/shared/hostname-field.tsx`.

**Modified (frontend)**: `ui/src/types/proxy.ts`, `ui/src/hooks/use-proxies.ts`, `ui/src/components/proxy/proxy-data-grid.tsx`, `ui/src/components/proxy/cells/` (new group cell), `ui/src/routes/_dashboard/proxies/index.tsx`, `ui/src/components/proxy/forms/reverse-proxy-form.tsx`, `ui/src/lib/form-validation.ts`.

---

## Task 1: Schema and models

Adds the tables and flips the four inheritable booleans to nullable. No behavior change yet — nothing reads the new columns.

**Files:**
- Create: `backend/migrations/000014_create_proxy_groups.up.sql`
- Create: `backend/migrations/000014_create_proxy_groups.down.sql`
- Create: `backend/internal/models/proxy_group.go`
- Create: `backend/internal/models/proxy_group_test.go`
- Modify: `backend/internal/models/proxy.go:11-53`

**Interfaces:**
- Produces: `models.ProxyGroup`, `models.ProxyGroupACLAssignment`; `models.Proxy.{SSLEnabled,SSLForced,BlockExploits,TLSInsecureSkipVerify} *bool`; `models.Proxy.{GroupID *int, HostnameLabel *string, GroupName *string}`.

- [ ] **Step 1: Write the up migration**

`backend/migrations/000014_create_proxy_groups.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS proxy_groups (
  id                       SERIAL PRIMARY KEY,
  name                     VARCHAR(255) NOT NULL,
  description              TEXT,
  base_domain              VARCHAR(255),
  ssl_enabled              BOOLEAN,
  ssl_forced               BOOLEAN,
  tls_insecure_skip_verify BOOLEAN,
  block_exploits           BOOLEAN,
  custom_headers           TEXT,
  created_by               INT NOT NULL,
  created_at               TIMESTAMP NOT NULL DEFAULT now(),
  updated_at               TIMESTAMP NOT NULL DEFAULT now(),
  CONSTRAINT uq_proxy_groups_name UNIQUE (name),
  CONSTRAINT fk_proxy_groups_created_by FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS proxy_group_acl_assignments (
  id             SERIAL PRIMARY KEY,
  proxy_group_id INT NOT NULL,
  acl_group_id   INT NOT NULL,
  path_pattern   VARCHAR(500) NOT NULL DEFAULT '/*',
  priority       INT NOT NULL DEFAULT 0,
  enabled        BOOLEAN NOT NULL DEFAULT true,
  created_at     TIMESTAMP NOT NULL DEFAULT now(),
  updated_at     TIMESTAMP NOT NULL DEFAULT now(),
  CONSTRAINT uq_pgaa_group_acl UNIQUE (proxy_group_id, acl_group_id),
  CONSTRAINT fk_pgaa_proxy_group FOREIGN KEY (proxy_group_id)
    REFERENCES proxy_groups(id) ON DELETE CASCADE,
  CONSTRAINT fk_pgaa_acl_group FOREIGN KEY (acl_group_id)
    REFERENCES acl_groups(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_pgaa_proxy_group_id ON proxy_group_acl_assignments(proxy_group_id);

ALTER TABLE proxies
  ADD COLUMN IF NOT EXISTS group_id INT NULL,
  ADD COLUMN IF NOT EXISTS hostname_label VARCHAR(63) NULL;

ALTER TABLE proxies
  ADD CONSTRAINT fk_proxies_group_id FOREIGN KEY (group_id)
    REFERENCES proxy_groups(id) ON DELETE RESTRICT;

ALTER TABLE proxies
  ADD CONSTRAINT chk_proxies_label_requires_group
    CHECK (hostname_label IS NULL OR group_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_proxies_group_id ON proxies(group_id);

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              DROP NOT NULL,
  ALTER COLUMN ssl_forced               DROP NOT NULL,
  ALTER COLUMN block_exploits           DROP NOT NULL,
  ALTER COLUMN tls_insecure_skip_verify DROP NOT NULL;

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              DROP DEFAULT,
  ALTER COLUMN ssl_forced               DROP DEFAULT,
  ALTER COLUMN block_exploits           DROP DEFAULT,
  ALTER COLUMN tls_insecure_skip_verify DROP DEFAULT;
```

The `DROP DEFAULT` block matters: with the columns nullable, a lingering DB-side `DEFAULT true` would fill in a value whenever GORM omits the column, which is exactly the "inherit" case. Inherit must reach the database as `NULL`.

- [ ] **Step 2: Write the down migration**

`backend/migrations/000014_create_proxy_groups.down.sql`:

```sql
UPDATE proxies SET ssl_enabled              = true  WHERE ssl_enabled IS NULL;
UPDATE proxies SET ssl_forced               = true  WHERE ssl_forced IS NULL;
UPDATE proxies SET block_exploits           = true  WHERE block_exploits IS NULL;
UPDATE proxies SET tls_insecure_skip_verify = false WHERE tls_insecure_skip_verify IS NULL;

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              SET DEFAULT true,
  ALTER COLUMN ssl_forced               SET DEFAULT true,
  ALTER COLUMN block_exploits           SET DEFAULT true,
  ALTER COLUMN tls_insecure_skip_verify SET DEFAULT false;

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              SET NOT NULL,
  ALTER COLUMN ssl_forced               SET NOT NULL,
  ALTER COLUMN block_exploits           SET NOT NULL,
  ALTER COLUMN tls_insecure_skip_verify SET NOT NULL;

ALTER TABLE proxies DROP CONSTRAINT IF EXISTS chk_proxies_label_requires_group;
ALTER TABLE proxies DROP CONSTRAINT IF EXISTS fk_proxies_group_id;
DROP INDEX IF EXISTS idx_proxies_group_id;
ALTER TABLE proxies DROP COLUMN IF EXISTS hostname_label;
ALTER TABLE proxies DROP COLUMN IF EXISTS group_id;

DROP TABLE IF EXISTS proxy_group_acl_assignments;
DROP TABLE IF EXISTS proxy_groups;
```

The backfill values are the system defaults from the Global Constraints, so a down-migrated database keeps serving what it served.

- [ ] **Step 3: Write the new models**

`backend/internal/models/proxy_group.go`:

```go
package models

import "time"

// ProxyGroup is a named parent that member HTTP proxies inherit configuration
// from. It is NOT an ACLGroup: ACLGroup is an auth grouping, this is a config
// grouping. Every settings field is a pointer; nil means "the group says
// nothing about this", not "false".
type ProxyGroup struct {
	ID          int     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string  `json:"name" gorm:"column:name;type:varchar(255);uniqueIndex;not null"`
	Description *string `json:"description,omitempty" gorm:"column:description;type:text"`
	BaseDomain  *string `json:"base_domain,omitempty" gorm:"column:base_domain;type:varchar(255)"`

	// Inheritable settings. No GORM `default:` tags — a default tag drops an
	// explicit false on INSERT (see models/proxy.go:17-30).
	SSLEnabled            *bool         `json:"ssl_enabled" gorm:"column:ssl_enabled"`
	SSLForced             *bool         `json:"ssl_forced" gorm:"column:ssl_forced"`
	TLSInsecureSkipVerify *bool         `json:"tls_insecure_skip_verify" gorm:"column:tls_insecure_skip_verify"`
	BlockExploits         *bool         `json:"block_exploits" gorm:"column:block_exploits"`
	CustomHeaders         CustomHeaders `json:"custom_headers,omitempty" gorm:"column:custom_headers;type:text"`

	CreatedBy int       `json:"-" gorm:"column:created_by;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// MemberCount is computed by the repository List query, not persisted.
	MemberCount int `json:"member_count" gorm:"-"`
}

func (ProxyGroup) TableName() string { return "proxy_groups" }

// ProxyGroupACLAssignment mirrors ProxyACLAssignment column-for-column so the
// resolver can merge the two sets without translating between shapes.
type ProxyGroupACLAssignment struct {
	ID           int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ProxyGroupID int       `json:"proxy_group_id" gorm:"column:proxy_group_id;not null;index"`
	ACLGroupID   int       `json:"acl_group_id" gorm:"column:acl_group_id;not null;index"`
	PathPattern  string    `json:"path_pattern" gorm:"column:path_pattern;type:varchar(500);not null"`
	Priority     int       `json:"priority" gorm:"column:priority;not null"`
	Enabled      bool      `json:"enabled" gorm:"column:enabled;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	ACLGroup *ACLGroup `json:"acl_group,omitempty" gorm:"foreignKey:ACLGroupID"`
}

func (ProxyGroupACLAssignment) TableName() string { return "proxy_group_acl_assignments" }
```

- [ ] **Step 4: Convert the Proxy model to tri-state**

In `backend/internal/models/proxy.go`, replace lines 17-45 (the comment block through `StaticConfig`) with:

```go
	// SSLEnabled / SSLForced / BlockExploits / TLSInsecureSkipVerify are
	// tri-state: nil means "inherit from the proxy's group, or the system
	// default if ungrouped". They carry no GORM `default:` tag — a default tag
	// makes GORM omit the column from INSERT on a zero value, which would both
	// drop an explicit false and destroy the nil/inherit signal.
	// proxygroup.Resolve is the only place a default is applied.
	SSLEnabled *bool `json:"ssl_enabled" gorm:"column:ssl_enabled"`
	SSLForced  *bool `json:"ssl_forced" gorm:"column:ssl_forced"`
	// IsActive omits the GORM `default` tag for the same reason. It is NOT
	// inheritable — a group-level kill switch would make "why is my proxy down"
	// a two-table question.
	IsActive  bool      `json:"is_active" gorm:"column:is_active;not null"`
	CreatedBy int       `json:"-" gorm:"column:created_by;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// GroupID is the proxy's ProxyGroup (config inheritance), never an ACLGroup.
	GroupID *int `json:"group_id,omitempty" gorm:"column:group_id"`
	// HostnameLabel is set iff the proxy has a group AND that group has a
	// base_domain. Hostname then holds the materialized <label>.<base_domain>.
	HostnameLabel *string     `json:"hostname_label,omitempty" gorm:"column:hostname_label;type:varchar(63)"`
	Group         *ProxyGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`

	// Type-specific fields (stored as JSON in database)
	Upstreams             interface{}   `json:"upstreams,omitempty" gorm:"column:upstreams;type:text;serializer:json"`
	LoadBalancing         JSONField     `json:"load_balancing,omitempty" gorm:"column:load_balancing;type:text"`
	BlockExploits         *bool         `json:"block_exploits" gorm:"column:block_exploits"`
	TLSInsecureSkipVerify *bool         `json:"tls_insecure_skip_verify" gorm:"column:tls_insecure_skip_verify"`
	CustomHeaders         CustomHeaders `json:"custom_headers,omitempty" gorm:"column:custom_headers;type:text"`
	RedirectConfig        JSONField     `json:"redirect,omitempty" gorm:"column:redirect_config;type:text"`
	StaticConfig          JSONField     `json:"static,omitempty" gorm:"column:static_config;type:text"`
```

Then add to the computed-fields block at the end of the struct (after `ACLGroupNames`):

```go
	// GroupName is computed by the repository List query, not persisted.
	GroupName *string `json:"group_name,omitempty" gorm:"-"`
```

- [ ] **Step 5: Relax hostname validation for label-addressed proxies**

`Proxy.Validate()` (`models/proxy.go:236`) currently hard-fails on an empty `Hostname`. A label-addressed proxy has its hostname materialized by the service *before* `Validate` runs, so this stays correct — but add the label rule. Replace the hostname block in `Validate`:

```go
	if p.HostnameLabel != nil {
		if p.GroupID == nil {
			return ErrLabelRequiresGroup
		}
		if err := validateHostname(*p.HostnameLabel); err != nil {
			return err
		}
		if strings.Contains(*p.HostnameLabel, ".") {
			return ErrLabelNotASingleLabel
		}
	}

	if p.Hostname == "" {
		return ErrProxyHostnameRequired
	}

	if err := validateHostname(p.Hostname); err != nil {
		return err
	}
```

And add to the error vars block:

```go
	ErrLabelRequiresGroup   = &ValidationError{Message: "hostname_label requires a group"}
	ErrLabelNotASingleLabel = &ValidationError{Message: "hostname_label must be a single DNS label (no dots)"}
```

- [ ] **Step 6: Write the schema test**

`backend/internal/models/proxy_group_test.go`:

```go
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
		HostnameLabel: &label,
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
```

If `gorm.io/gorm/utils/tests` is not already a dependency, drop that import and pass `schema.NamingStrategy{}` for both cache args using `&sync.Map{}` as the cache: `schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})`.

- [ ] **Step 7: Run the model tests, expect failure**

Run: `cd backend && go test ./internal/models/ -run 'Proxy' -v`
Expected: compile errors across the repo — `models.Proxy.SSLEnabled` is now `*bool` and ~30 call sites assign/read a `bool`.

- [ ] **Step 8: Fix the compile breakage mechanically**

`go build ./... 2>&1 | head -50` lists every site. Fix them with the smallest possible change, deferring real logic to later tasks:

- `handlers/proxy.go:237-247`: delete the defaulting. `proxy.SSLEnabled = req.SSLEnabled`, `proxy.BlockExploits = req.BlockExploits`, `proxy.SSLForced = nil`. Keep `proxy.IsActive = true`.
- `handlers/proxy.go:323-340` (import path): same — assign the pointers straight through, keep the `IsActive` default.
- `handlers/proxy.go:400-403` (update): `proxy.SSLEnabled = req.SSLEnabled` — an omitted field now means inherit, not "keep existing". Note this in the diff; Task 6 revisits the update semantics.
- `handlers/proxy.go:815-871` (audit diffing): compare via a `derefBool` helper. Add to that file:

```go
func derefBool(b *bool) bool { return b != nil && *b }
```

  and change `if old.SSLEnabled != updated.SSLEnabled` to `if derefBool(old.SSLEnabled) != derefBool(updated.SSLEnabled)`, likewise for `BlockExploits` and `TLSInsecureSkipVerify`. `IsActive` is unchanged.
- `repository/proxy_repository.go:96`: `query.Where("ssl_enabled = ?", *params.SSLEnabled)` still compiles (the param is already `*bool`). Leave it; Task 6 revisits whether the filter should match inherited-true.
- `caddy/config/builder.go:396,460,502` and `http_builder.go:238,242`: wrap with `derefBool` **temporarily**. Task 3 deletes these when the builder is retyped. Add the same one-line helper to `caddy/config/builder.go`.
- `service/proxy_service_test.go`, any fixture setting these fields: use `ptr(true)`. Add a test helper `func ptr[T any](v T) *T { return &v }` in `proxy_service_test.go`.

- [ ] **Step 9: Run build and full short test suite**

Run: `cd backend && gofmt -l . && go build ./... && go test ./... -short`
Expected: `gofmt` prints nothing; build succeeds; all tests pass.

The `-short` suite skips testcontainers, so the migration itself is not exercised here. That is intentional; Task 4 covers it.

- [ ] **Step 10: Commit**

```bash
git add backend/migrations/000014_create_proxy_groups.up.sql \
        backend/migrations/000014_create_proxy_groups.down.sql \
        backend/internal/models/proxy_group.go \
        backend/internal/models/proxy_group_test.go \
        backend/internal/models/proxy.go \
        backend/internal/api/handlers/proxy.go \
        backend/internal/caddy/config/builder.go \
        backend/internal/caddy/config/http_builder.go \
        backend/internal/service/proxy_service_test.go
git commit -m "feat(proxy-groups): schema + tri-state proxy settings

Adds proxy_groups and proxy_group_acl_assignments, plus proxies.group_id
and proxies.hostname_label. The four inheritable booleans become nullable
(NULL = inherit) and their Go fields become *bool.

Drops the DB-side DEFAULT on those columns: a lingering DEFAULT would fill
in a value whenever GORM omits the column, which is exactly the inherit case."
```

---

## Task 2: The resolver

The heart of the feature. A pure function with no DB access and no callers yet — added in isolation so its test table is the only thing gating it.

**Files:**
- Create: `backend/internal/proxygroup/resolve.go`
- Create: `backend/internal/proxygroup/resolve_test.go`

**Interfaces:**
- Consumes: `models.Proxy`, `models.ProxyGroup`, `models.ProxyACLAssignment`, `models.ProxyGroupACLAssignment` (Task 1).
- Produces: `proxygroup.EffectiveProxy`; `proxygroup.Resolve(p models.Proxy, g *models.ProxyGroup, proxyACL []models.ProxyACLAssignment, groupACL []models.ProxyGroupACLAssignment) EffectiveProxy`; `proxygroup.EffectiveHostname(label string, baseDomain string) string`; the exported default consts.

- [ ] **Step 1: Write the failing test**

`backend/internal/proxygroup/resolve_test.go`:

```go
package proxygroup

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aloks98/waygates/backend/internal/models"
)

func ptr[T any](v T) *T { return &v }

// TestResolve_SystemDefaultsMatchCreateHandler pins the defaults to what
// handlers/proxy.go:235-248 applies today. These are NOT all-false: an omitted
// ssl_enabled currently yields true, and defaulting it to false would serve new
// proxies over plaintext. The equivalence test cannot catch that, because it
// compares grouped against ungrouped and both would be wrong identically.
func TestResolve_SystemDefaultsMatchCreateHandler(t *testing.T) {
	e := Resolve(models.Proxy{}, nil, nil, nil)

	assert.True(t, e.SSLEnabled, "ssl_enabled default must be true")
	assert.True(t, e.SSLForced, "ssl_forced default must be true")
	assert.True(t, e.BlockExploits, "block_exploits default must be true")
	assert.False(t, e.TLSInsecureSkipVerify, "tls_insecure_skip_verify default must be false")
}

func TestResolve_ScalarPrecedence(t *testing.T) {
	cases := []struct {
		name  string
		proxy *bool
		group *bool
		want  bool
	}{
		{"proxy true over group false", ptr(true), ptr(false), true},
		{"proxy false over group true", ptr(false), ptr(true), false},
		{"inherit group true", nil, ptr(true), true},
		{"inherit group false", nil, ptr(false), false},
		{"no group, system default", nil, nil, false}, // tls_insecure default
		{"proxy true, no group", ptr(true), nil, true},
		{"proxy false, no group", ptr(false), nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var g *models.ProxyGroup
			if tc.group != nil {
				g = &models.ProxyGroup{TLSInsecureSkipVerify: tc.group}
			}
			e := Resolve(models.Proxy{TLSInsecureSkipVerify: tc.proxy}, g, nil, nil)
			assert.Equal(t, tc.want, e.TLSInsecureSkipVerify)
		})
	}
}

// A group with no opinion must not override a proxy that also has none — the
// system default wins, not false.
func TestResolve_SilentGroupFallsThroughToSystemDefault(t *testing.T) {
	e := Resolve(models.Proxy{}, &models.ProxyGroup{}, nil, nil)
	assert.True(t, e.SSLEnabled)
	assert.True(t, e.BlockExploits)
}

func TestResolve_HeadersMergeProxyWinsPerKey(t *testing.T) {
	g := &models.ProxyGroup{CustomHeaders: models.CustomHeaders{
		Request:  map[string]string{"X-Env": "prod", "X-Team": "core"},
		Response: map[string]string{"X-Cache": "miss"},
	}}
	p := models.Proxy{CustomHeaders: models.CustomHeaders{
		Request: map[string]string{"X-Team": "web"},
	}}

	e := Resolve(p, g, nil, nil)

	assert.Equal(t, map[string]string{"X-Env": "prod", "X-Team": "web"}, e.CustomHeaders.Request)
	assert.Equal(t, map[string]string{"X-Cache": "miss"}, e.CustomHeaders.Response)
}

// Resolve must not write through into the group's maps.
func TestResolve_HeaderMergeDoesNotMutateInputs(t *testing.T) {
	g := &models.ProxyGroup{CustomHeaders: models.CustomHeaders{
		Request: map[string]string{"X-Env": "prod"},
	}}
	p := models.Proxy{CustomHeaders: models.CustomHeaders{
		Request: map[string]string{"X-Team": "web"},
	}}

	_ = Resolve(p, g, nil, nil)

	assert.Equal(t, map[string]string{"X-Env": "prod"}, g.CustomHeaders.Request)
	assert.Equal(t, map[string]string{"X-Team": "web"}, p.CustomHeaders.Request)
}

func TestResolve_ACLUnion(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{
		{ID: 9, ProxyGroupID: 3, ACLGroupID: 1, PathPattern: "/*", Priority: 0, Enabled: true},
	}
	proxyACL := []models.ProxyACLAssignment{
		{ID: 5, ProxyID: 7, ACLGroupID: 2, PathPattern: "/admin", Priority: 1, Enabled: true},
	}

	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{ID: 3}, proxyACL, groupACL)

	assert.Len(t, e.ACL, 2)
	byGroup := map[int]models.ProxyACLAssignment{}
	for _, a := range e.ACL {
		byGroup[a.ACLGroupID] = a
	}
	// Inherited row is synthesized onto the proxy with ID 0 as its provenance mark.
	assert.Equal(t, 0, byGroup[1].ID)
	assert.Equal(t, 7, byGroup[1].ProxyID)
	assert.Equal(t, "/*", byGroup[1].PathPattern)
	// The proxy's own row is passed through untouched.
	assert.Equal(t, 5, byGroup[2].ID)
	assert.Equal(t, "/admin", byGroup[2].PathPattern)
}

func TestResolve_ACLProxyRowWinsWholesale(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{
		{ACLGroupID: 1, PathPattern: "/*", Priority: 0, Enabled: true},
	}
	proxyACL := []models.ProxyACLAssignment{
		{ID: 5, ProxyID: 7, ACLGroupID: 1, PathPattern: "/admin", Priority: 9, Enabled: true},
	}

	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{}, proxyACL, groupACL)

	assert.Len(t, e.ACL, 1)
	assert.Equal(t, "/admin", e.ACL[0].PathPattern, "proxy row must win wholesale, not field-wise")
	assert.Equal(t, 9, e.ACL[0].Priority)
}

// The documented opt-out: assign the same acl_group_id with enabled=false.
func TestResolve_ACLProxyCanOptOutOfInheritedGroup(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{
		{ACLGroupID: 1, PathPattern: "/*", Enabled: true},
	}
	proxyACL := []models.ProxyACLAssignment{
		{ID: 5, ProxyID: 7, ACLGroupID: 1, Enabled: false},
	}

	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{}, proxyACL, groupACL)

	assert.Len(t, e.ACL, 1)
	assert.False(t, e.ACL[0].Enabled, "opt-out row survives so the builder's Enabled filter drops it")
}

// A disabled group-level assignment must not be revived by inheritance.
func TestResolve_DisabledGroupACLStaysDisabled(t *testing.T) {
	groupACL := []models.ProxyGroupACLAssignment{{ACLGroupID: 1, Enabled: false}}
	e := Resolve(models.Proxy{ID: 7}, &models.ProxyGroup{}, nil, groupACL)
	assert.Len(t, e.ACL, 1)
	assert.False(t, e.ACL[0].Enabled)
}

func TestResolve_LoadBalancingIsNotInherited(t *testing.T) {
	g := &models.ProxyGroup{}
	p := models.Proxy{LoadBalancing: models.JSONField{"policy": "ip_hash"}}
	e := Resolve(p, g, nil, nil)
	assert.Equal(t, models.JSONField{"policy": "ip_hash"}, e.LoadBalancing)
}

func TestResolve_NilGroupIsIdentityOnScalars(t *testing.T) {
	p := models.Proxy{
		SSLEnabled:    ptr(false),
		BlockExploits: ptr(false),
		Hostname:      "a.example.com",
	}
	e := Resolve(p, nil, nil, nil)
	assert.False(t, e.SSLEnabled)
	assert.False(t, e.BlockExploits)
	assert.Equal(t, "a.example.com", e.Hostname)
}

func TestEffectiveHostname(t *testing.T) {
	assert.Equal(t, "abc.group.acme.in", EffectiveHostname("abc", "group.acme.in"))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/proxygroup/ -v`
Expected: FAIL — `no Go files in .../internal/proxygroup` (package does not exist).

- [ ] **Step 3: Write the implementation**

`backend/internal/proxygroup/resolve.go`:

```go
// Package proxygroup resolves a proxy's effective configuration by merging its
// ProxyGroup's defaults underneath its own explicit values.
//
// It lives at internal/proxygroup rather than internal/service/proxygroup
// because internal/caddy/config imports it, and the Caddy builder must not
// depend on the service layer.
package proxygroup

import (
	"github.com/aloks98/waygates/backend/internal/models"
)

// System defaults, applied when neither the proxy nor its group has an opinion.
// These are lifted verbatim from the defaults ProxyHandler.CreateProxy applied
// before this package existed (handlers/proxy.go:235-248). They are NOT
// all-false. Changing DefaultSSLEnabled or DefaultBlockExploits to false is a
// security regression: new proxies would be served over plaintext with exploit
// blocking off.
const (
	DefaultSSLEnabled            = true
	DefaultSSLForced             = true
	DefaultBlockExploits         = true
	DefaultTLSInsecureSkipVerify = false
)

// EffectiveProxy is a proxy with every inheritable value decided. It is the only
// type the Caddy builder accepts, so an unresolved models.Proxy cannot reach
// config generation.
type EffectiveProxy struct {
	ID       int
	Type     string
	Name     string
	Hostname string
	IsActive bool

	SSLEnabled            bool
	SSLForced             bool
	BlockExploits         bool
	TLSInsecureSkipVerify bool

	Upstreams      interface{}
	LoadBalancing  models.JSONField
	CustomHeaders  models.CustomHeaders
	RedirectConfig models.JSONField
	StaticConfig   models.JSONField

	// ACL is the merged assignment set, expressed in the same row type the
	// Caddy builder already consumes. Rows inherited from the group carry
	// ID == 0; the builder's existing Enabled filter implements opt-out.
	ACL []models.ProxyACLAssignment

	// GroupID is nil for an ungrouped proxy. Carried for display and logging.
	GroupID *int
}

// EffectiveHostname composes a label-addressed proxy's full hostname.
func EffectiveHostname(label, baseDomain string) string {
	return label + "." + baseDomain
}

// Resolve merges a group's defaults into a proxy. g may be nil.
//
// Scalars: proxy value if non-nil, else group value if non-nil, else the system
// default. Headers: per-key union, proxy wins. ACL: union by ACLGroupID, the
// proxy's row winning wholesale. LoadBalancing, RedirectConfig, StaticConfig,
// Hostname, Upstreams, Type, Name and IsActive are never inherited.
//
// Resolve does not mutate its arguments.
func Resolve(
	p models.Proxy,
	g *models.ProxyGroup,
	proxyACL []models.ProxyACLAssignment,
	groupACL []models.ProxyGroupACLAssignment,
) EffectiveProxy {
	var (
		gSSLEnabled, gSSLForced, gBlockExploits, gTLSInsecure *bool
		gHeaders                                              models.CustomHeaders
	)
	if g != nil {
		gSSLEnabled, gSSLForced = g.SSLEnabled, g.SSLForced
		gBlockExploits, gTLSInsecure = g.BlockExploits, g.TLSInsecureSkipVerify
		gHeaders = g.CustomHeaders
	}

	return EffectiveProxy{
		ID:       p.ID,
		Type:     p.Type,
		Name:     p.Name,
		Hostname: p.Hostname,
		IsActive: p.IsActive,
		GroupID:  p.GroupID,

		SSLEnabled:            resolveBool(p.SSLEnabled, gSSLEnabled, DefaultSSLEnabled),
		SSLForced:             resolveBool(p.SSLForced, gSSLForced, DefaultSSLForced),
		BlockExploits:         resolveBool(p.BlockExploits, gBlockExploits, DefaultBlockExploits),
		TLSInsecureSkipVerify: resolveBool(p.TLSInsecureSkipVerify, gTLSInsecure, DefaultTLSInsecureSkipVerify),

		Upstreams:      p.Upstreams,
		LoadBalancing:  p.LoadBalancing,
		CustomHeaders:  mergeHeaders(gHeaders, p.CustomHeaders),
		RedirectConfig: p.RedirectConfig,
		StaticConfig:   p.StaticConfig,

		ACL: mergeACL(p.ID, proxyACL, groupACL),
	}
}

// resolveBool walks proxy -> group -> system default.
func resolveBool(proxy, group *bool, systemDefault bool) bool {
	if proxy != nil {
		return *proxy
	}
	if group != nil {
		return *group
	}
	return systemDefault
}

// mergeHeaders unions the group's headers under the proxy's, per key and per
// direction. It allocates fresh maps so neither input is mutated.
func mergeHeaders(group, proxy models.CustomHeaders) models.CustomHeaders {
	return models.CustomHeaders{
		Request:  mergeStringMap(group.Request, proxy.Request),
		Response: mergeStringMap(group.Response, proxy.Response),
	}
}

func mergeStringMap(base, over map[string]string) map[string]string {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

// mergeACL unions the group's assignments under the proxy's, keyed by
// ACLGroupID. A proxy row wins wholesale — never field-wise, which would
// produce a row neither side authored. Inherited rows are synthesized onto the
// proxy with ID 0 marking their provenance.
//
// Disabled rows are kept, not dropped: the Caddy builder's SetACLAssignments
// already skips Enabled == false, and keeping the row is what lets a proxy opt
// out of an inherited ACL by re-assigning it with enabled = false.
func mergeACL(
	proxyID int,
	proxyACL []models.ProxyACLAssignment,
	groupACL []models.ProxyGroupACLAssignment,
) []models.ProxyACLAssignment {
	if len(proxyACL) == 0 && len(groupACL) == 0 {
		return nil
	}

	claimed := make(map[int]bool, len(proxyACL))
	for _, a := range proxyACL {
		claimed[a.ACLGroupID] = true
	}

	out := make([]models.ProxyACLAssignment, 0, len(proxyACL)+len(groupACL))
	for _, a := range groupACL {
		if claimed[a.ACLGroupID] {
			continue // the proxy overrides this one wholesale
		}
		out = append(out, models.ProxyACLAssignment{
			ID:          0, // inherited
			ProxyID:     proxyID,
			ACLGroupID:  a.ACLGroupID,
			PathPattern: a.PathPattern,
			Priority:    a.Priority,
			Enabled:     a.Enabled,
		})
	}
	out = append(out, proxyACL...)
	return out
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd backend && go test ./internal/proxygroup/ -v`
Expected: PASS, all 12 tests.

- [ ] **Step 5: Run the gate and commit**

```bash
cd backend && gofmt -l . && go build ./... && go test ./... -short
git add backend/internal/proxygroup/
git commit -m "feat(proxy-groups): pure resolver for effective proxy config

Resolve merges a group's defaults under a proxy's explicit values. Scalars
walk proxy -> group -> system default; headers merge per key; ACL unions by
acl_group_id with the proxy's row winning wholesale.

System defaults are pinned to what CreateProxy applied before this existed
(ssl_enabled/ssl_forced/block_exploits true), with a test that says why.

Inherited ACL rows are synthesized as models.ProxyACLAssignment with ID 0,
so the builder's existing Enabled filter is the opt-out mechanism."
```

---

## Task 3: Retype the Caddy builder, thread Resolve through sync

Makes it impossible to hand an unresolved proxy to config generation. Ships with no user-visible change: `Resolve` with a nil group is the identity function on today's behavior, which the equivalence test asserts.

**Files:**
- Modify: `backend/internal/caddy/config/builder.go:122,136,359,392,460,483,502`
- Modify: `backend/internal/caddy/config/http_builder.go:30,57,121,156,209`
- Modify: `backend/internal/service/sync_service.go:320-434`
- Create: `backend/internal/caddy/config/equivalence_test.go`

**Interfaces:**
- Consumes: `proxygroup.EffectiveProxy`, `proxygroup.Resolve` (Task 2).
- Produces: `Builder.SetHTTPProxies([]proxygroup.EffectiveProxy) *Builder`; `Builder.BuildSingleProxy(*proxygroup.EffectiveProxy) (*CaddyConfig, error)`; the `HTTPBuilder` methods retyped to `*proxygroup.EffectiveProxy`. `SetACLAssignments([]models.ProxyACLAssignment)` is unchanged.

- [ ] **Step 1: Write the failing equivalence test**

`backend/internal/caddy/config/equivalence_test.go`:

```go
package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/proxygroup"
)

// NOTE: if package `config`'s existing tests already define `ptr`, delete this.
func ptr[T any](v T) *T { return &v }

func buildJSONFor(t *testing.T, e proxygroup.EffectiveProxy) []byte {
	t.Helper()
	b := NewBuilder(zap.NewNop())
	b.SetHTTPProxies([]proxygroup.EffectiveProxy{e})
	out, err := b.BuildJSON()
	require.NoError(t, err)
	return out
}

// A grouped proxy that inherits its settings must produce byte-identical Caddy
// JSON to an ungrouped proxy carrying those same settings written down.
//
// This is the guard against the resolver and the builder disagreeing. It cannot
// catch a wrong *system default* — both sides would be wrong identically. That
// is what TestResolve_SystemDefaultsMatchCreateHandler is for.
func TestBuildJSON_GroupedInheritsEqualsUngroupedExplicit(t *testing.T) {
	group := &models.ProxyGroup{
		ID:                    3,
		SSLEnabled:            ptr(true),
		BlockExploits:         ptr(false),
		TLSInsecureSkipVerify: ptr(true),
		CustomHeaders: models.CustomHeaders{
			Request: map[string]string{"X-Env": "prod"},
		},
	}

	inheriting := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true, GroupID: ptr(3),
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:8080"}},
	}

	explicit := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true,
		SSLEnabled:            ptr(true),
		BlockExploits:         ptr(false),
		TLSInsecureSkipVerify: ptr(true),
		CustomHeaders: models.CustomHeaders{
			Request: map[string]string{"X-Env": "prod"},
		},
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:8080"}},
	}

	got := buildJSONFor(t, proxygroup.Resolve(inheriting, group, nil, nil))
	want := buildJSONFor(t, proxygroup.Resolve(explicit, nil, nil, nil))

	require.JSONEq(t, string(want), string(got))
	require.Equal(t, string(want), string(got), "byte-identical, not merely JSON-equal")
}

// A proxy overriding its group must produce the same JSON as one that never had
// a group.
func TestBuildJSON_ProxyOverrideBeatsGroup(t *testing.T) {
	group := &models.ProxyGroup{ID: 3, BlockExploits: ptr(true)}

	overriding := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true, GroupID: ptr(3),
		BlockExploits: ptr(false),
		Upstreams:     []interface{}{map[string]interface{}{"address": "http://127.0.0.1:8080"}},
	}
	standalone := overriding
	standalone.GroupID = nil

	got := buildJSONFor(t, proxygroup.Resolve(overriding, group, nil, nil))
	want := buildJSONFor(t, proxygroup.Resolve(standalone, nil, nil, nil))

	require.Equal(t, string(want), string(got))
}

// BlockExploits drives whether security routes are emitted at all, so assert the
// group can turn them on for a silent member.
func TestBuildJSON_GroupEnablesSecurityRoutes(t *testing.T) {
	base := models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "svc.acme.in", IsActive: true,
		BlockExploits: ptr(false),
		Upstreams:     []interface{}{map[string]interface{}{"address": "http://127.0.0.1:8080"}},
	}
	off := buildJSONFor(t, proxygroup.Resolve(base, nil, nil, nil))

	member := base
	member.BlockExploits = nil
	member.GroupID = ptr(3)
	on := buildJSONFor(t, proxygroup.Resolve(member, &models.ProxyGroup{ID: 3, BlockExploits: ptr(true)}, nil, nil))

	// Assert the direction of the difference, not merely that the bytes differ:
	// a change anywhere would satisfy NotEqual while security routes stayed off.
	// SecurityRoutesForHost prepends extra host-matched routes, so the hostname
	// appears strictly more often once BlockExploits resolves true.
	offN := strings.Count(string(off), "svc.acme.in")
	onN := strings.Count(string(on), "svc.acme.in")
	require.Greater(t, onN, offN, "group must enable security routes for a silent member")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/caddy/config/ -run TestBuildJSON_Grouped -v`
Expected: FAIL to compile — `cannot use []proxygroup.EffectiveProxy as []models.Proxy in argument to b.SetHTTPProxies`.

- [ ] **Step 3: Retype the builder**

In `backend/internal/caddy/config/builder.go`:

1. Add the import `"github.com/aloks98/waygates/backend/internal/proxygroup"`.
2. Change the `httpProxies` field type on `Builder` from `[]models.Proxy` to `[]proxygroup.EffectiveProxy`.
3. Retype these four signatures (bodies unchanged):

```go
func (b *Builder) SetHTTPProxies(proxies []proxygroup.EffectiveProxy) *Builder
func (b *Builder) buildProxyRoutes(proxy *proxygroup.EffectiveProxy) ([]*HTTPRoute, error)
func (b *Builder) BuildSingleProxy(proxy *proxygroup.EffectiveProxy) (*CaddyConfig, error)
```

`collectTLSDomains` takes no proxy argument — it ranges over `b.httpProxies` (`builder.go:455-465`), so retyping the field is enough. Its loop variable `proxy` becomes a `proxygroup.EffectiveProxy` and `proxy.SSLEnabled` at `builder.go:460` is now a plain `bool`. Likewise `builder.go:359` (`httpHostnames[b.httpProxies[i].Hostname] = true`) compiles untouched.

4. Delete the temporary `derefBool` helper added in Task 1 Step 8 and the `derefBool(...)` wrappers at `builder.go:396,460,502`. They read plain `bool` fields now:

```go
	if proxy.BlockExploits {                 // builder.go:396
	if proxy.SSLEnabled {                    // builder.go:460 and :502
```

5. `SetACLAssignments` is **unchanged** — it still takes `[]models.ProxyACLAssignment` and still filters `if a.Enabled`. That filter is the opt-out mechanism.

In `backend/internal/caddy/config/http_builder.go`, retype the five signatures and delete the `derefBool` wrappers at `:238,242`:

```go
func (b *HTTPBuilder) BuildReverseProxyRoutes(proxy *proxygroup.EffectiveProxy) ([]*HTTPRoute, error)
func (b *HTTPBuilder) BuildReverseProxyRoutesWithACL(proxy *proxygroup.EffectiveProxy, assignments []models.ProxyACLAssignment, aclGroups map[int64]*models.ACLGroup, aclBuilder *ACLBuilder) ([]*HTTPRoute, error)
func (b *HTTPBuilder) BuildRedirectRoutes(proxy *proxygroup.EffectiveProxy) ([]*HTTPRoute, error)
func (b *HTTPBuilder) BuildStaticRoutes(proxy *proxygroup.EffectiveProxy) ([]*HTTPRoute, error)
func (b *HTTPBuilder) buildReverseProxyHandler(proxy *proxygroup.EffectiveProxy, upstreams []*Upstream) *ReverseProxyHandler
```

The bodies compile untouched: `Hostname`, `CustomHeaders`, `LoadBalancing`, `TLSInsecureSkipVerify`, `Upstreams`, `Type` and `ID` all exist on `EffectiveProxy` under the same names, with `TLSInsecureSkipVerify` now a plain `bool`.

- [ ] **Step 4: Run the equivalence tests**

Run: `cd backend && go test ./internal/caddy/config/ -v`
Expected: PASS. If `TestBuildJSON_GroupEnablesSecurityRoutes` fails, `BlockExploits` is not reaching `buildProxyRoutes` — check step 3.4.

- [ ] **Step 5: Thread Resolve through sync_service.buildConfigBytes**

In `backend/internal/service/sync_service.go`, add the `proxygroup` import, then replace step 1's proxy load and step 6's `SetHTTPProxies` call.

After the existing step 1 (`proxies, _, err := s.proxyRepo.List(...)`), insert:

```go
	// 1b. Load every proxy group and its ACL assignments in two batch queries.
	// The per-proxy GetProxyACLAssignments call below is a pre-existing N+1; do
	// not add another one here.
	groupsByID := map[int]*models.ProxyGroup{}
	groupACLByGroupID := map[int][]models.ProxyGroupACLAssignment{}
	if s.proxyGroupRepo != nil {
		groups, listErr := s.proxyGroupRepo.ListAll()
		if listErr != nil {
			return nil, fmt.Errorf("failed to list proxy groups: %w", listErr)
		}
		for i := range groups {
			groupsByID[groups[i].ID] = &groups[i]
		}

		groupACLs, aclErr := s.proxyGroupRepo.ListAllACLAssignments()
		if aclErr != nil {
			return nil, fmt.Errorf("failed to list proxy group ACL assignments: %w", aclErr)
		}
		for _, a := range groupACLs {
			groupACLByGroupID[a.ProxyGroupID] = append(groupACLByGroupID[a.ProxyGroupID], a)
		}
	}
```

The group load returns an error rather than warning-and-continuing, unlike the ACL load above it. That is deliberate: a silently-empty group map would strip inherited ACLs from every member and serve them unauthenticated. Failing the sync leaves the last-good config in place.

Change the ACL-assignment accumulation loop to also collect per-proxy assignments into a map, then replace `s.jsonBuilder.SetHTTPProxies(proxies)` with:

```go
	// 6. Resolve every proxy against its group before the builder sees it.
	effective := make([]proxygroup.EffectiveProxy, 0, len(proxies))
	var resolvedACL []models.ProxyACLAssignment
	for i := range proxies {
		var g *models.ProxyGroup
		if proxies[i].GroupID != nil {
			g = groupsByID[*proxies[i].GroupID]
		}
		e := proxygroup.Resolve(
			proxies[i],
			g,
			aclByProxyID[proxies[i].ID],
			groupACLForProxy(g, groupACLByGroupID),
		)
		effective = append(effective, e)
		resolvedACL = append(resolvedACL, e.ACL...)
	}

	s.jsonBuilder.SetHTTPProxies(effective)
	s.jsonBuilder.SetACLGroups(aclGroups)
	s.jsonBuilder.SetACLAssignments(resolvedACL)
```

Note `SetACLAssignments` now receives the **resolved** assignments (`e.ACL`), not the raw per-proxy ones. That is what carries inherited ACLs into the config. Add the helper at the bottom of `sync_service.go`:

```go
// groupACLForProxy returns the ACL assignments of g, or nil when the proxy is
// ungrouped.
func groupACLForProxy(g *models.ProxyGroup, byGroupID map[int][]models.ProxyGroupACLAssignment) []models.ProxyGroupACLAssignment {
	if g == nil {
		return nil
	}
	return byGroupID[g.ID]
}
```

Replace the existing ACL accumulation loop body so it fills `aclByProxyID` (declare `aclByProxyID := map[int][]models.ProxyACLAssignment{}` before it) instead of appending into a flat `aclAssignments` slice, and delete the now-unused `aclAssignments` variable.

- [ ] **Step 6: Add the repo dependency to SyncService**

Add `proxyGroupRepo repository.ProxyGroupRepositoryInterface` to the `SyncService` struct and its config, defaulting to nil. Task 4 creates the interface; until then, declare it in `repository/interfaces.go` as:

```go
// ProxyGroupRepositoryInterface defines proxy group database operations.
type ProxyGroupRepositoryInterface interface {
	ListAll() ([]models.ProxyGroup, error)
	ListAllACLAssignments() ([]models.ProxyGroupACLAssignment, error)
}
```

Task 4 extends this interface; it does not replace these two methods.

Also retype `BuildSingleProxy`'s caller. `SyncService.GenerateProxyConfigJSON(proxyID)` must load the proxy, its group, both ACL sets, and pass `proxygroup.Resolve(...)` to `b.BuildSingleProxy(&e)`.

- [ ] **Step 7: Run build and full short suite**

Run: `cd backend && gofmt -l . && go build ./... && go test ./... -short`
Expected: everything passes. Any remaining `models.Proxy` reaching a builder is now a compile error, which is the point.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/caddy/config/ backend/internal/service/sync_service.go backend/internal/repository/interfaces.go
git commit -m "refactor(caddy): builder accepts EffectiveProxy only

Retypes every Caddy builder entry point from *models.Proxy to
*proxygroup.EffectiveProxy, so an unresolved proxy cannot reach config
generation — the divergence between what the UI shows and what Caddy
serves is now a compile error rather than a review catch.

sync_service resolves each proxy against its group (two batched queries,
no new N+1) and feeds the builder resolved ACL assignments, which is what
carries inherited ACLs into the config.

No behavior change: Resolve with a nil group is the identity function on
today's config, asserted byte-for-byte by the new equivalence test."
```

---

## Task 4: Repository and service

**Files:**
- Create: `backend/internal/repository/proxy_group_repository.go`
- Create: `backend/internal/repository/proxy_group_repository_integration_test.go`
- Create: `backend/internal/service/proxy_group_service.go`
- Create: `backend/internal/service/proxy_group_service_test.go`
- Modify: `backend/internal/repository/interfaces.go`
- Modify: `backend/internal/service/interfaces.go`

**Interfaces:**
- Consumes: `models.ProxyGroup`, `models.ProxyGroupACLAssignment` (Task 1); `proxygroup.EffectiveHostname` (Task 2).
- Produces:
  - `repository.ProxyGroupRepositoryInterface`: `ListAll()`, `ListAllACLAssignments()`, `List(ProxyGroupListParams) ([]models.ProxyGroup, int64, error)`, `GetByID(id int) (*models.ProxyGroup, error)`, `Create(*models.ProxyGroup) error`, `Update(*models.ProxyGroup) error`, `Delete(id int) error`, `MemberCount(id int) (int64, error)`, `ListMembers(id int) ([]models.Proxy, error)`, `UpdateBaseDomainTx(groupID int, newBase *string) error`, `ListACLAssignments(groupID int) ([]models.ProxyGroupACLAssignment, error)`, `CreateACLAssignment(*models.ProxyGroupACLAssignment) error`, `UpdateACLAssignment(*models.ProxyGroupACLAssignment) error`, `DeleteACLAssignment(groupID, aclGroupID int) error`
  - `service.ProxyGroupService` + `service.ErrGroupNotFound`, `ErrGroupNameConflict`, `ErrGroupHasMembers`, `ErrBaseDomainRequiredByMembers`
  - `service.GroupSyncer` interface: `RebuildAll() error`

- [ ] **Step 1: Write the failing integration test**

`backend/internal/repository/proxy_group_repository_integration_test.go`:

```go
package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

func ptr[T any](v T) *T { return &v }

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

	require.NoError(t, groupRepo.UpdateBaseDomainTx(g.ID, ptr("g2.acme.in")))

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
	require.NoError(t, groupRepo.UpdateBaseDomainTx(g.ID, ptr("g2.acme.in")))

	assertNoHostnameDrift(t, tdb)
}

// assertNoHostnameDrift is the invariant guard. Call it at the end of any test
// that mutates groups or members.
func assertNoHostnameDrift(t *testing.T, tdb *TestDB) {
	t.Helper()
	var bad int64
	err := tdb.DB.Raw(`
		SELECT COUNT(*) FROM proxies p
		JOIN proxy_groups g ON g.id = p.group_id
		WHERE p.hostname_label IS NOT NULL
		  AND p.hostname <> p.hostname_label || '.' || g.base_domain
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

	err := groupRepo.UpdateBaseDomainTx(g.ID, ptr("g2.acme.in"))
	require.Error(t, err, "colliding rename must fail")

	reloaded, err := groupRepo.GetByID(g.ID)
	require.NoError(t, err)
	assert.Equal(t, "group.acme.in", *reloaded.BaseDomain, "group must be unchanged")

	members, err := groupRepo.ListMembers(g.ID)
	require.NoError(t, err)
	assert.Equal(t, "abc.group.acme.in", members[0].Hostname, "member must be unchanged")
	assertNoHostnameDrift(t, tdb)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/repository/ -run TestProxyGroupRepository -v`
Expected: FAIL to compile — `undefined: NewProxyGroupRepository`.

- [ ] **Step 3: Write the repository**

`backend/internal/repository/proxy_group_repository.go`:

```go
package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/proxygroup"
)

// ProxyGroupRepository is the data access layer for ProxyGroup.
type ProxyGroupRepository struct {
	db *gorm.DB
}

func NewProxyGroupRepository(db *gorm.DB) *ProxyGroupRepository {
	return &ProxyGroupRepository{db: db}
}

// ProxyGroupListParams holds parameters for listing proxy groups.
type ProxyGroupListParams struct {
	Page   int
	Limit  int
	Search string
	Sort   string
	Order  string
}

func (r *ProxyGroupRepository) ListAll() ([]models.ProxyGroup, error) {
	var groups []models.ProxyGroup
	if err := r.db.Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("listing proxy groups: %w", err)
	}
	return groups, nil
}

func (r *ProxyGroupRepository) ListAllACLAssignments() ([]models.ProxyGroupACLAssignment, error) {
	var out []models.ProxyGroupACLAssignment
	if err := r.db.Find(&out).Error; err != nil {
		return nil, fmt.Errorf("listing proxy group ACL assignments: %w", err)
	}
	return out, nil
}

// List returns a paginated page of groups with MemberCount populated.
func (r *ProxyGroupRepository) List(params ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
	var groups []models.ProxyGroup
	var total int64

	query := r.db.Model(&models.ProxyGroup{})
	if params.Search != "" {
		query = query.Where("name LIKE ?", "%"+params.Search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := "created_at"
	if validField, ok := allowedGroupSortFields[params.Sort]; ok {
		sortField = validField
	}
	sortOrder := "DESC"
	if validOrder, ok := allowedSortOrders[params.Order]; ok {
		sortOrder = validOrder
	}

	offset := (params.Page - 1) * params.Limit
	if err := query.Order(sortField + " " + sortOrder).
		Offset(offset).Limit(params.Limit).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	if len(groups) > 0 {
		ids := make([]int, len(groups))
		for i := range groups {
			ids[i] = groups[i].ID
		}
		type countRow struct {
			GroupID int
			N       int
		}
		var rows []countRow
		if err := r.db.Table("proxies").
			Select("group_id, COUNT(*) AS n").
			Where("group_id IN ?", ids).
			Group("group_id").Scan(&rows).Error; err != nil {
			return nil, 0, fmt.Errorf("counting proxy group members: %w", err)
		}
		byID := make(map[int]int, len(rows))
		for _, row := range rows {
			byID[row.GroupID] = row.N
		}
		for i := range groups {
			groups[i].MemberCount = byID[groups[i].ID]
		}
	}

	return groups, total, nil
}

var allowedGroupSortFields = map[string]string{
	"id": "id", "name": "name", "base_domain": "base_domain",
	"created_at": "created_at", "updated_at": "updated_at",
}

func (r *ProxyGroupRepository) GetByID(id int) (*models.ProxyGroup, error) {
	var g models.ProxyGroup
	if err := r.db.First(&g, id).Error; err != nil {
		return nil, err
	}
	n, err := r.MemberCount(id)
	if err != nil {
		return nil, err
	}
	g.MemberCount = int(n)
	return &g, nil
}

func (r *ProxyGroupRepository) Create(g *models.ProxyGroup) error { return r.db.Create(g).Error }

// Update uses Select to write the nullable settings columns explicitly. A plain
// Save would skip nil pointers, making it impossible to clear an inherited
// value back to "the group says nothing".
func (r *ProxyGroupRepository) Update(g *models.ProxyGroup) error {
	return r.db.Model(&models.ProxyGroup{ID: g.ID}).
		Select("name", "description", "base_domain",
			"ssl_enabled", "ssl_forced", "tls_insecure_skip_verify",
			"block_exploits", "custom_headers", "updated_at").
		Updates(g).Error
}

func (r *ProxyGroupRepository) Delete(id int) error {
	return r.db.Delete(&models.ProxyGroup{}, id).Error
}

func (r *ProxyGroupRepository) MemberCount(id int) (int64, error) {
	var n int64
	err := r.db.Model(&models.Proxy{}).Where("group_id = ?", id).Count(&n).Error
	return n, err
}

func (r *ProxyGroupRepository) ListMembers(id int) ([]models.Proxy, error) {
	var out []models.Proxy
	err := r.db.Where("group_id = ?", id).Order("id ASC").Find(&out).Error
	return out, err
}

// UpdateBaseDomainTx sets the group's base_domain and recomputes every
// label-addressed member's materialized hostname, in one transaction. A
// collision trips the unique index on proxies.hostname and rolls the whole
// thing back, so a failed rename writes nothing.
func (r *ProxyGroupRepository) UpdateBaseDomainTx(groupID int, newBase *string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var members []models.Proxy
		if err := tx.Where("group_id = ? AND hostname_label IS NOT NULL", groupID).
			Find(&members).Error; err != nil {
			return fmt.Errorf("loading members: %w", err)
		}

		if newBase == nil && len(members) > 0 {
			return ErrBaseDomainRequiredByMembers
		}

		if err := tx.Model(&models.ProxyGroup{ID: groupID}).
			Select("base_domain", "updated_at").
			Updates(&models.ProxyGroup{BaseDomain: newBase}).Error; err != nil {
			return fmt.Errorf("updating base_domain: %w", err)
		}

		for i := range members {
			host := proxygroup.EffectiveHostname(*members[i].HostnameLabel, *newBase)
			if err := tx.Model(&models.Proxy{}).
				Where("id = ?", members[i].ID).
				Update("hostname", host).Error; err != nil {
				return fmt.Errorf("re-homing proxy %d to %q: %w", members[i].ID, host, err)
			}
		}
		return nil
	})
}

// ErrBaseDomainRequiredByMembers is returned when clearing base_domain would
// leave label-addressed members with no hostname.
var ErrBaseDomainRequiredByMembers = errors.New("group has label-addressed members; base_domain cannot be cleared")

func (r *ProxyGroupRepository) ListACLAssignments(groupID int) ([]models.ProxyGroupACLAssignment, error) {
	var out []models.ProxyGroupACLAssignment
	err := r.db.Preload("ACLGroup").Where("proxy_group_id = ?", groupID).
		Order("priority ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *ProxyGroupRepository) CreateACLAssignment(a *models.ProxyGroupACLAssignment) error {
	return r.db.Create(a).Error
}

func (r *ProxyGroupRepository) UpdateACLAssignment(a *models.ProxyGroupACLAssignment) error {
	return r.db.Model(&models.ProxyGroupACLAssignment{ID: a.ID}).
		Select("path_pattern", "priority", "enabled", "updated_at").Updates(a).Error
}

func (r *ProxyGroupRepository) DeleteACLAssignment(groupID, aclGroupID int) error {
	return r.db.Where("proxy_group_id = ? AND acl_group_id = ?", groupID, aclGroupID).
		Delete(&models.ProxyGroupACLAssignment{}).Error
}
```

Extend `repository/interfaces.go` — replace the two-method `ProxyGroupRepositoryInterface` stub from Task 3 with the full interface listed under **Interfaces** above, and add `_ ProxyGroupRepositoryInterface = (*ProxyGroupRepository)(nil)` to the assertion block.

- [ ] **Step 4: Run the integration tests**

Run: `cd backend && go test ./internal/repository/ -run TestProxyGroupRepository -v`
Expected: PASS (4 tests). Requires Docker for testcontainers; these are skipped under `-short`.

- [ ] **Step 5: Write the failing service test**

`backend/internal/service/proxy_group_service_test.go`:

```go
package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// NOTE: do NOT define `ptr` here. Task 1 Step 8 already added it to
// proxy_service_test.go, and both files are package `service` — a second
// definition will not compile.

type MockProxyGroupRepository struct {
	ListAllFunc             func() ([]models.ProxyGroup, error)
	GetByIDFunc             func(id int) (*models.ProxyGroup, error)
	CreateFunc              func(g *models.ProxyGroup) error
	UpdateFunc              func(g *models.ProxyGroup) error
	DeleteFunc              func(id int) error
	MemberCountFunc         func(id int) (int64, error)
	UpdateBaseDomainTxFunc  func(groupID int, newBase *string) error
}

func (m *MockProxyGroupRepository) ListAll() ([]models.ProxyGroup, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc()
	}
	return nil, nil
}
func (m *MockProxyGroupRepository) ListAllACLAssignments() ([]models.ProxyGroupACLAssignment, error) {
	return nil, nil
}
func (m *MockProxyGroupRepository) List(repository.ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
	return nil, 0, nil
}
func (m *MockProxyGroupRepository) GetByID(id int) (*models.ProxyGroup, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.ProxyGroup{ID: id}, nil
}
func (m *MockProxyGroupRepository) Create(g *models.ProxyGroup) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(g)
	}
	return nil
}
func (m *MockProxyGroupRepository) Update(g *models.ProxyGroup) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(g)
	}
	return nil
}
func (m *MockProxyGroupRepository) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}
func (m *MockProxyGroupRepository) MemberCount(id int) (int64, error) {
	if m.MemberCountFunc != nil {
		return m.MemberCountFunc(id)
	}
	return 0, nil
}
func (m *MockProxyGroupRepository) ListMembers(int) ([]models.Proxy, error) { return nil, nil }
func (m *MockProxyGroupRepository) UpdateBaseDomainTx(groupID int, newBase *string) error {
	if m.UpdateBaseDomainTxFunc != nil {
		return m.UpdateBaseDomainTxFunc(groupID, newBase)
	}
	return nil
}
func (m *MockProxyGroupRepository) ListACLAssignments(int) ([]models.ProxyGroupACLAssignment, error) {
	return nil, nil
}
func (m *MockProxyGroupRepository) CreateACLAssignment(*models.ProxyGroupACLAssignment) error { return nil }
func (m *MockProxyGroupRepository) UpdateACLAssignment(*models.ProxyGroupACLAssignment) error { return nil }
func (m *MockProxyGroupRepository) DeleteACLAssignment(int, int) error                        { return nil }

type MockGroupSyncer struct {
	RebuildAllFunc func() error
	Calls          int
}

func (m *MockGroupSyncer) RebuildAll() error {
	m.Calls++
	if m.RebuildAllFunc != nil {
		return m.RebuildAllFunc()
	}
	return nil
}

func newGroupService(repo repository.ProxyGroupRepositoryInterface, syncer GroupSyncer) *ProxyGroupService {
	return NewProxyGroupService(ProxyGroupServiceConfig{Repo: repo, Syncer: syncer})
}

func TestProxyGroupService_DeleteBlockedWhenGroupHasMembers(t *testing.T) {
	repo := &MockProxyGroupRepository{
		MemberCountFunc: func(int) (int64, error) { return 7, nil },
		DeleteFunc:      func(int) error { t.Fatal("Delete must not be called"); return nil },
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	err := svc.DeleteGroup(3)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupHasMembers)
	assert.Contains(t, err.Error(), "7")
}

func TestProxyGroupService_DeleteSucceedsWhenEmptyAndRebuilds(t *testing.T) {
	syncer := &MockGroupSyncer{}
	svc := newGroupService(&MockProxyGroupRepository{}, syncer)

	require.NoError(t, svc.DeleteGroup(3))
	assert.Equal(t, 1, syncer.Calls, "deleting a group must rebuild the whole config")
}

func TestProxyGroupService_UpdateRebuildsFullConfig(t *testing.T) {
	syncer := &MockGroupSyncer{}
	svc := newGroupService(&MockProxyGroupRepository{}, syncer)

	require.NoError(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, Name: "internal"}))
	assert.Equal(t, 1, syncer.Calls)
}

// A rename goes through the transactional path, not a plain Update.
func TestProxyGroupService_BaseDomainChangeUsesTransaction(t *testing.T) {
	var txCalled bool
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, Name: "internal", BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateBaseDomainTxFunc: func(int, *string) error { txCalled = true; return nil },
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	require.NoError(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, Name: "internal", BaseDomain: ptr("g2.acme.in")}))
	assert.True(t, txCalled, "base_domain change must go through UpdateBaseDomainTx")
}

// A failed rename must not rebuild the config.
func TestProxyGroupService_FailedRenameDoesNotRebuild(t *testing.T) {
	syncer := &MockGroupSyncer{}
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateBaseDomainTxFunc: func(int, *string) error { return errors.New("hostname conflict") },
	}
	svc := newGroupService(repo, syncer)

	require.Error(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, BaseDomain: ptr("g2.acme.in")}))
	assert.Zero(t, syncer.Calls, "a failed rename must leave the served config alone")
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `cd backend && go test ./internal/service/ -run TestProxyGroupService -v`
Expected: FAIL to compile — `undefined: NewProxyGroupService`.

- [ ] **Step 7: Write the service**

`backend/internal/service/proxy_group_service.go`:

```go
package service

import (
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// GroupSyncer rebuilds the entire Caddy config. A group mutation can change the
// effective config of every member at once, so there is no per-proxy sync path
// worth having: buildConfigBytes reconstructs the whole config anyway.
type GroupSyncer interface {
	RebuildAll() error
}

type ProxyGroupService struct {
	repo   repository.ProxyGroupRepositoryInterface
	syncer GroupSyncer
	logger *zap.Logger
}

type ProxyGroupServiceConfig struct {
	Repo   repository.ProxyGroupRepositoryInterface
	Syncer GroupSyncer
	Logger *zap.Logger
}

func NewProxyGroupService(cfg ProxyGroupServiceConfig) *ProxyGroupService {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &ProxyGroupService{
		repo:   cfg.Repo,
		syncer: cfg.Syncer,
		logger: cfg.Logger.Named("proxy-group-service"),
	}
}

var (
	ErrGroupNotFound     = errors.New("proxy group not found")
	ErrGroupNameConflict = errors.New("proxy group name already exists")
	ErrGroupHasMembers   = errors.New("proxy group has member proxies")
)

// ErrBaseDomainRequiredByMembers is re-exported from the repository so handlers
// map a single error to 409 without importing the repository package.
var ErrBaseDomainRequiredByMembers = repository.ErrBaseDomainRequiredByMembers

func (s *ProxyGroupService) ListGroups(params repository.ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
	return s.repo.List(params)
}

func (s *ProxyGroupService) GetGroupByID(id int) (*models.ProxyGroup, error) {
	g, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrGroupNotFound
	}
	return g, nil
}

func (s *ProxyGroupService) CreateGroup(g *models.ProxyGroup, userID int) error {
	g.CreatedBy = userID
	if err := s.repo.Create(g); err != nil {
		return fmt.Errorf("failed to create proxy group: %w", err)
	}
	if err := s.syncer.RebuildAll(); err != nil {
		if delErr := s.repo.Delete(g.ID); delErr != nil {
			return fmt.Errorf("failed to sync and rollback failed: %w", errors.Join(err, delErr))
		}
		return fmt.Errorf("failed to sync proxy group: %w", err)
	}
	return nil
}

// UpdateGroup writes the group's settings, routing a base_domain change through
// the transactional re-home path. The config is rebuilt only after a successful
// write — a failed rename must leave the served config alone.
func (s *ProxyGroupService) UpdateGroup(g *models.ProxyGroup) error {
	existing, err := s.repo.GetByID(g.ID)
	if err != nil {
		return ErrGroupNotFound
	}

	if baseDomainChanged(existing.BaseDomain, g.BaseDomain) {
		if err := s.repo.UpdateBaseDomainTx(g.ID, g.BaseDomain); err != nil {
			return err
		}
		s.logger.Info("proxy group base domain changed",
			zap.Int("group_id", g.ID),
			zap.Stringp("old", existing.BaseDomain),
			zap.Stringp("new", g.BaseDomain))
	}

	if err := s.repo.Update(g); err != nil {
		return fmt.Errorf("failed to update proxy group: %w", err)
	}
	return s.syncer.RebuildAll()
}

func (s *ProxyGroupService) DeleteGroup(id int) error {
	n, err := s.repo.MemberCount(id)
	if err != nil {
		return fmt.Errorf("failed to count members: %w", err)
	}
	if n > 0 {
		return fmt.Errorf("%w: %d member proxies; reassign or remove them first", ErrGroupHasMembers, n)
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete proxy group: %w", err)
	}
	return s.syncer.RebuildAll()
}

func baseDomainChanged(old, next *string) bool {
	switch {
	case old == nil && next == nil:
		return false
	case old == nil || next == nil:
		return true
	default:
		return *old != *next
	}
}
```

Add `RebuildAll()` to `SyncService` in `sync_service.go` (it is `performFullSyncJSON` under a name that says what callers want):

```go
// RebuildAll regenerates the entire Caddy config from current database state.
func (s *SyncService) RebuildAll() error { return s.performFullSyncJSON() }
```

And add to `service/interfaces.go`: `_ GroupSyncer = (*SyncService)(nil)`.

- [ ] **Step 8: Run the service tests**

Run: `cd backend && go test ./internal/service/ -run TestProxyGroupService -v`
Expected: PASS (5 tests).

- [ ] **Step 9: Gate and commit**

```bash
cd backend && gofmt -l . && go build ./... && go test ./... -short
git add backend/internal/repository/ backend/internal/service/
git commit -m "feat(proxy-groups): repository + service

Delete is blocked at both layers when a group has members: the service
returns ErrGroupHasMembers with a count, and ON DELETE RESTRICT holds even
if the service is bypassed (asserted against the raw DB).

A base_domain change re-homes every label-addressed member in one
transaction; a collision trips the unique index on proxies.hostname and
rolls the whole thing back, writing nothing. A failed rename does not
rebuild the config.

Repository Update uses Select() on the nullable settings columns — a plain
Save skips nil pointers, making it impossible to clear a value back to
'the group says nothing'."
```

---

## Task 5: Handler, routes, RBAC, audit

**Files:**
- Create: `backend/internal/api/handlers/proxy_group.go`
- Modify: `backend/internal/api/routes/routes.go:144-160,236-259`
- Modify: `backend/rbac.yaml:91-139`
- Modify: `backend/internal/service/interfaces.go` (audit methods)

**Interfaces:**
- Consumes: `service.ProxyGroupService` and its error vars (Task 4).
- Produces: `handlers.NewProxyGroupHandler(svc, auditService, logger) *ProxyGroupHandler` with methods `ListGroups`, `GetGroup`, `CreateGroup`, `UpdateGroup`, `DeleteGroup`, `GetGroupACL`, `AssignACLToGroup`, `UpdateGroupACLAssignment`, `RemoveACLFromGroup`.

- [ ] **Step 1: Add the RBAC permission group**

In `backend/rbac.yaml`, append after the `L4 Proxies` block (line 104):

```yaml
  - name: "Proxy Groups"
    permissions:
      - key: "proxygroups:read"
        name: "View Proxy Groups"
        description: "View proxy group configurations"
      - key: "proxygroups:create"
        name: "Create Proxy Groups"
        description: "Create new proxy groups"
      - key: "proxygroups:update"
        name: "Update Proxy Groups"
        description: "Modify existing proxy groups"
      - key: "proxygroups:delete"
        name: "Delete Proxy Groups"
        description: "Remove proxy groups"
```

Add `- "proxygroups:*"` to the `operator` role's permission list (after `- "l4proxies:*"`), and `- "proxygroups:read"` to the `viewer` role's list (after `- "l4proxies:read"`). `admin` already holds `*`.

- [ ] **Step 2: Write the handler**

`backend/internal/api/handlers/proxy_group.go`. Follow the shape of `handlers/proxy.go`: a struct holding `service`, `auditService`, `logger`; each method decodes, calls the service, maps errors, writes the `{success,data,error}` envelope via `utils`, and audit-logs after success.

The error mapping is the part that matters:

```go
func (h *ProxyGroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid group id")
		return
	}

	if err := h.service.DeleteGroup(id); err != nil {
		switch {
		case errors.Is(err, service.ErrGroupHasMembers):
			// 409 carries the member count in the message, so the UI can say
			// "7 member proxies" without a second round trip.
			utils.RespondError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrGroupNotFound):
			utils.RespondError(w, http.StatusNotFound, "proxy group not found")
		default:
			h.logger.Error("failed to delete proxy group", zap.Int("id", id), zap.Error(err))
			utils.RespondError(w, http.StatusInternalServerError, "failed to delete proxy group")
		}
		return
	}

	userID := getUserID(r)
	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogProxyGroupDelete(r.Context(), userID, id, getClientIP(r), r.UserAgent())
	}
	utils.RespondSuccess(w, http.StatusOK, nil)
}
```

`UpdateGroup` maps `service.ErrBaseDomainRequiredByMembers` → 409, and a `proxies.hostname` unique-violation from the rename transaction → 409 with the colliding hostname. Detect the latter with `errors.Is(err, gorm.ErrDuplicatedKey)` (GORM translates it when `TranslateError: true` is set on the config) or by matching the Postgres `23505` SQLSTATE; check how `handlers/proxy.go` already surfaces `service.ErrHostnameConflict` and mirror it.

`CreateGroup` maps a `uq_proxy_groups_name` violation → 409 via `service.ErrGroupNameConflict`.

Add these audit methods to `AuditServiceInterface` and its implementation, mirroring `LogProxyCreate`'s signature style:

```go
LogProxyGroupCreate(ctx context.Context, userID int, group *models.ProxyGroup, ip, userAgent string) error
LogProxyGroupUpdate(ctx context.Context, userID int, old, updated *models.ProxyGroup, ip, userAgent string) error
LogProxyGroupDelete(ctx context.Context, userID, groupID int, ip, userAgent string) error
// A base_domain rewrite re-homes members; without the IDs, a mass re-homing
// leaves no trace of what moved.
LogProxyGroupRehome(ctx context.Context, userID, groupID int, oldBase, newBase string, proxyIDs []int, ip, userAgent string) error
```

- [ ] **Step 3: Wire the routes**

In `backend/internal/api/routes/routes.go`, construct the service and handler alongside the others (near line 103 and 147):

```go
	proxyGroupRepo := repository.NewProxyGroupRepository(db)
	proxyGroupService := service.NewProxyGroupService(service.ProxyGroupServiceConfig{
		Repo:   proxyGroupRepo,
		Syncer: syncService,
		Logger: logger,
	})
	proxyGroupHandler := handlers.NewProxyGroupHandler(proxyGroupService, auditService, logger)
```

Pass `proxyGroupRepo` into the `SyncService` construction so `buildConfigBytes` can see groups (Task 3, Step 6).

Then, inside the protected route group, after the `/api/proxies` block:

```go
		// Proxy group routes (config inheritance parent — NOT ACL groups)
		r.Route("/api/proxy-groups", func(r chi.Router) {
			r.With(chimw.RequirePermission(authAdapter, "proxygroups:read", mwConfig)).Get("/", proxyGroupHandler.ListGroups)
			r.With(chimw.RequirePermission(authAdapter, "proxygroups:read", mwConfig)).Get("/{id}", proxyGroupHandler.GetGroup)
			r.With(chimw.RequirePermission(authAdapter, "proxygroups:create", mwConfig)).Post("/", proxyGroupHandler.CreateGroup)
			r.With(chimw.RequirePermission(authAdapter, "proxygroups:update", mwConfig)).Put("/{id}", proxyGroupHandler.UpdateGroup)
			r.With(chimw.RequirePermission(authAdapter, "proxygroups:delete", mwConfig)).Delete("/{id}", proxyGroupHandler.DeleteGroup)
		})

		// Proxy group ACL assignment routes — same acl:* permissions as the
		// per-proxy ACL routes; the same capability, a different subject.
		r.Route("/api/proxy-groups/{id}/acl", func(r chi.Router) {
			r.With(chimw.RequirePermission(authAdapter, "acl:read", mwConfig)).Get("/", proxyGroupHandler.GetGroupACL)
			r.With(chimw.RequirePermission(authAdapter, "acl:update", mwConfig)).Post("/", proxyGroupHandler.AssignACLToGroup)
			r.With(chimw.RequirePermission(authAdapter, "acl:update", mwConfig)).Put("/{assignmentId}", proxyGroupHandler.UpdateGroupACLAssignment)
			r.With(chimw.RequirePermission(authAdapter, "acl:delete", mwConfig)).Delete("/{aclGroupId}", proxyGroupHandler.RemoveACLFromGroup)
		})
```

- [ ] **Step 4: Verify routes and permissions by hand**

Run: `cd backend && go build ./... && go test ./... -short`

Then start the server (`make backend-run`) and check each permission gate:

```bash
TOKEN=<viewer token>
curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" localhost:8080/api/proxy-groups          # expect 200
curl -s -o /dev/null -w '%{http_code}\n' -X POST -H "Authorization: Bearer $TOKEN" localhost:8080/api/proxy-groups  # expect 403
```

Expected: `200` then `403`. A viewer holding `proxygroups:read` must not be able to create.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/ backend/rbac.yaml backend/internal/service/interfaces.go
git commit -m "feat(proxy-groups): handler, routes, RBAC, audit

Adds /api/proxy-groups CRUD behind proxygroups:* and nested ACL assignment
routes behind the existing acl:* permissions.

Deleting a non-empty group returns 409 carrying the member count, so the UI
can name it without a second round trip. A base_domain rewrite audit-logs
the affected proxy IDs — a mass re-homing must not be traceless."
```

---

## Task 6: Proxy endpoints — group membership, effective values, filtering

**Files:**
- Modify: `backend/internal/api/handlers/proxy.go:200-250,320-345,360-410`
- Modify: `backend/internal/repository/proxy_repository.go:40-52,54-149`
- Modify: `backend/internal/service/proxy_service.go`

**Interfaces:**
- Consumes: `proxygroup.Resolve`, `proxygroup.EffectiveHostname` (Task 2); `repository.ProxyGroupRepositoryInterface` (Task 4).
- Produces: `ProxyListParams.GroupID *int` and `.GroupIDNot *int`; `models.Proxy.GroupName` populated by `List`; response field `effective` with `_source`.

- [ ] **Step 1: Write the failing service test for hostname materialization**

Add to `backend/internal/service/proxy_service_test.go`:

```go
// A label-addressed proxy has its hostname materialized before it is written.
func TestProxyService_CreateMaterializesHostnameFromLabel(t *testing.T) {
	var written *models.Proxy
	repo := &MockProxyRepository{
		CreateFunc:         func(p *models.Proxy) error { written = p; return nil },
		HostnameExistsFunc: func(string, int) (bool, error) { return false, nil },
	}
	groupRepo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, BaseDomain: ptr("group.acme.in")}, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo: repo, GroupRepo: groupRepo, SyncService: &MockProxySyncer{},
	})

	p := &models.Proxy{
		Type: models.ProxyTypeReverseProxy, Name: "svc",
		GroupID: ptr(3), HostnameLabel: ptr("abc"), IsActive: true,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}
	require.NoError(t, svc.CreateProxy(p, 1))

	assert.Equal(t, "abc.group.acme.in", written.Hostname)
}

// A group with no base_domain leaves the absolute hostname alone and forbids a label.
func TestProxyService_CreateRejectsLabelWhenGroupHasNoBaseDomain(t *testing.T) {
	groupRepo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, BaseDomain: nil}, nil
		},
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo: &MockProxyRepository{}, GroupRepo: groupRepo, SyncService: &MockProxySyncer{},
	})

	p := &models.Proxy{
		Type: models.ProxyTypeReverseProxy, Name: "svc",
		GroupID: ptr(3), HostnameLabel: ptr("abc"), IsActive: true,
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}
	assert.ErrorIs(t, svc.CreateProxy(p, 1), ErrLabelRequiresBaseDomain)
}

// Detaching keeps the materialized hostname — that is what Approach 2 bought.
func TestProxyService_UpdateDetachKeepsHostname(t *testing.T) {
	existing := &models.Proxy{
		ID: 1, Type: models.ProxyTypeReverseProxy, Name: "svc",
		Hostname: "abc.group.acme.in", GroupID: ptr(3), HostnameLabel: ptr("abc"),
		Upstreams: []interface{}{map[string]interface{}{"address": "http://127.0.0.1:1"}},
	}
	var written *models.Proxy
	repo := &MockProxyRepository{
		GetByIDFunc:        func(int) (*models.Proxy, error) { return existing, nil },
		UpdateFunc:         func(p *models.Proxy) error { written = p; return nil },
		HostnameExistsFunc: func(string, int) (bool, error) { return false, nil },
	}
	svc := NewProxyService(ProxyServiceConfig{
		Repo: repo, GroupRepo: &MockProxyGroupRepository{}, SyncService: &MockProxySyncer{},
	})

	update := *existing
	update.GroupID = nil
	update.HostnameLabel = nil
	require.NoError(t, svc.UpdateProxy(1, &update))

	assert.Equal(t, "abc.group.acme.in", written.Hostname, "detach must not change the hostname")
	assert.Nil(t, written.GroupID)
	assert.Nil(t, written.HostnameLabel)
}
```

Move `MockProxyGroupRepository` from `proxy_group_service_test.go` into a shared `mocks_test.go` in the same package so both files use it (Go allows one definition per package).

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/service/ -run TestProxyService_Create -v`
Expected: FAIL to compile — `unknown field GroupRepo in ProxyServiceConfig`.

- [ ] **Step 3: Implement hostname materialization in ProxyService**

Add `groupRepo repository.ProxyGroupRepositoryInterface` to `ProxyService` and `ProxyServiceConfig`. Add:

```go
var ErrLabelRequiresBaseDomain = errors.New("hostname_label requires the group to have a base_domain")

// materializeHostname enforces the cross-table rule that no CHECK constraint can
// express: a proxy is label-addressed iff it has a group AND that group has a
// base_domain. It writes proxies.hostname, the denormalized cache that keeps the
// unique index and every existing proxy.Hostname reader working.
func (s *ProxyService) materializeHostname(p *models.Proxy) error {
	if p.GroupID == nil {
		if p.HostnameLabel != nil {
			return models.ErrLabelRequiresGroup
		}
		return nil
	}

	g, err := s.groupRepo.GetByID(*p.GroupID)
	if err != nil {
		return ErrGroupNotFound
	}

	if g.BaseDomain == nil {
		if p.HostnameLabel != nil {
			return ErrLabelRequiresBaseDomain
		}
		return nil // grouped, but addressed absolutely
	}

	if p.HostnameLabel == nil {
		return ErrLabelRequiredByBaseDomain
	}
	p.Hostname = proxygroup.EffectiveHostname(*p.HostnameLabel, *g.BaseDomain)
	return nil
}

var ErrLabelRequiredByBaseDomain = errors.New("group has a base_domain; hostname_label is required")
```

Call `s.materializeHostname(proxy)` at the top of `CreateProxy` (before `proxy.Validate()`, so `Validate` sees the real hostname) and at the same point in `UpdateProxy`. The rollback-on-sync-failure structure of `CreateProxy` is otherwise unchanged.

- [ ] **Step 4: Add the group filter and group-name summary to the repository**

In `repository/proxy_repository.go`, add to `ProxyListParams`:

```go
	GroupID    *int // Filter by group (nil = no filter)
	GroupIDNot *int // Exclude a group
	Ungrouped  bool // Filter to proxies with no group
```

In `List`, after the SSL filter:

```go
	if params.Ungrouped {
		query = query.Where("group_id IS NULL")
	} else if params.GroupID != nil {
		query = query.Where("group_id = ?", *params.GroupID)
	}
	if params.GroupIDNot != nil {
		query = query.Where("group_id IS DISTINCT FROM ?", *params.GroupIDNot)
	}
```

`IS DISTINCT FROM` rather than `<>`: a plain `group_id <> 3` silently drops every ungrouped proxy, because `NULL <> 3` is NULL, not true. "Not in group 3" must include the ungrouped.

Then, next to the existing ACL summary join, add the group-name summary:

```go
	if len(proxies) > 0 {
		type groupNameRow struct {
			ProxyID int
			Name    string
		}
		var rows []groupNameRow
		if err := r.db.
			Table("proxies").
			Select("proxies.id AS proxy_id, proxy_groups.name").
			Joins("JOIN proxy_groups ON proxy_groups.id = proxies.group_id").
			Where("proxies.id IN ?", ids).
			Scan(&rows).Error; err != nil {
			return nil, 0, fmt.Errorf("loading proxy group names: %w", err)
		}
		byProxy := make(map[int]string, len(rows))
		for _, row := range rows {
			byProxy[row.ProxyID] = row.Name
		}
		for i := range proxies {
			if name, ok := byProxy[proxies[i].ID]; ok {
				n := name
				proxies[i].GroupName = &n
			}
		}
	}
```

Reuse the `ids` slice already built for the ACL summary; do not rebuild it.

Add `"group_id"` to `allowedSortFields`.

The pre-existing `ssl_enabled` filter (`proxy_repository.go:96`) now matches only proxies with an **explicit** value, since inheriting proxies store NULL. Change it so `?ssl_enabled=true` means "effectively true", which is what a user filtering the grid expects:

```go
	if params.SSLEnabled != nil {
		// NULL means inherit. Resolve against the group, then the system default
		// (ssl_enabled defaults to true), entirely in SQL.
		query = query.Joins("LEFT JOIN proxy_groups ON proxy_groups.id = proxies.group_id").
			Where("COALESCE(proxies.ssl_enabled, proxy_groups.ssl_enabled, true) = ?", *params.SSLEnabled)
	}
```

The `true` in that `COALESCE` is `proxygroup.DefaultSSLEnabled`. If that constant ever changes, this query must change with it — add a comment saying so.

- [ ] **Step 5: Return `effective` and `_source` from the read path**

In `handlers/proxy.go`, `GetProxy` loads the proxy, its group and both ACL sets, calls `proxygroup.Resolve`, and marshals a response struct:

```go
type effectiveSource struct {
	SSLEnabled            string `json:"ssl_enabled"`
	SSLForced             string `json:"ssl_forced"`
	BlockExploits         string `json:"block_exploits"`
	TLSInsecureSkipVerify string `json:"tls_insecure_skip_verify"`
}

type effectiveView struct {
	SSLEnabled            bool            `json:"ssl_enabled"`
	SSLForced             bool            `json:"ssl_forced"`
	BlockExploits         bool            `json:"block_exploits"`
	TLSInsecureSkipVerify bool            `json:"tls_insecure_skip_verify"`
	CustomHeaders         models.CustomHeaders `json:"custom_headers"`
	Source                effectiveSource `json:"_source"`
}

type proxyDetailResponse struct {
	*models.Proxy
	Effective effectiveView `json:"effective"`
}

// sourceOf reports where a resolved value came from, so the UI can distinguish
// "Inherit (on)" from an explicit "On". Without it the form shows the same
// toggle for both, which is the divergence the resolver exists to prevent,
// relocated into the UI.
func sourceOf(proxyVal, groupVal *bool) string {
	switch {
	case proxyVal != nil:
		return "proxy"
	case groupVal != nil:
		return "group"
	default:
		return "default"
	}
}
```

`CreateProxy` / `UpdateProxy` request structs gain `GroupID *int json:"group_id"` and `HostnameLabel *string json:"hostname_label"`. `Hostname` becomes optional in validation when `HostnameLabel` is set. `SSLForced` gains a wire field (`*bool`), since a group can now set it and a proxy must be able to override it.

**`UpdateProxy` semantics change and this is deliberate.** Today (`handlers/proxy.go:400-403`) an omitted `ssl_enabled` means "keep the existing value". Under inheritance it must mean "inherit". The UI always sends all four fields explicitly (`null` for inherit), so the wire contract is unambiguous. Document it in `docs/API.md`.

- [ ] **Step 6: Add the group filter to the list handler**

Extend the existing `parseFilterParam` usage: `?group=eq:3`, `?group=in:1,2`, `?group=not:3`, `?group=eq:none` → `Ungrouped`. Map to `ProxyListParams.GroupID` / `.GroupIDNot` / `.Ungrouped`.

- [ ] **Step 7: Run the suite**

Run: `cd backend && gofmt -l . && go build ./... && go test ./... -short && go test ./internal/repository/ -run TestProxyGroupRepository`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/api/handlers/proxy.go backend/internal/repository/proxy_repository.go backend/internal/service/proxy_service.go backend/internal/service/proxy_service_test.go docs/API.md
git commit -m "feat(proxy-groups): proxy endpoints accept groups, return effective values

Proxies accept group_id + hostname_label; the service materializes
proxies.hostname from label + group.base_domain, enforcing the cross-table
rule no CHECK can express. Detaching keeps the hostname.

GET /api/proxies/{id} returns both the raw nullable row and an 'effective'
object with a _source map, so the edit form can distinguish 'Inherit (on)'
from an explicit 'On'.

The ssl_enabled list filter now COALESCEs through the group to the system
default, so filtering matches what is actually served rather than only
proxies with an explicit value.

BREAKING: an omitted boolean on PUT /api/proxies/{id} now means 'inherit',
not 'keep existing'."
```

---

## Task 7: Frontend — group CRUD surface

**Files:**
- Create: `ui/src/types/proxy-group.ts`, `ui/src/hooks/use-proxy-groups.ts`
- Create: `ui/src/routes/_dashboard/proxy-groups/index.tsx`, `new.tsx`, `$groupId/edit.tsx`
- Create: `ui/src/components/proxy-group/proxy-group-data-grid.tsx`, `proxy-group-form.tsx`
- Modify: `ui/src/lib/form-validation.ts`

**Interfaces:**
- Consumes: `GET/POST/PUT/DELETE /api/proxy-groups` (Task 5).
- Produces: `ProxyGroup` type; `useProxyGroups()`, `useProxyGroup(id)` hooks exporting `{ data, isLoading, create, update, remove }`.

- [ ] **Step 1: Define the type**

`ui/src/types/proxy-group.ts`:

```ts
export interface ProxyGroup {
  id: number;
  name: string;
  description?: string;
  base_domain?: string;
  /** null means "the group says nothing"; members fall through to the system default. */
  ssl_enabled: boolean | null;
  ssl_forced: boolean | null;
  tls_insecure_skip_verify: boolean | null;
  block_exploits: boolean | null;
  custom_headers?: { request?: Record<string, string>; response?: Record<string, string> };
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface ProxyGroupListResponse {
  items: ProxyGroup[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export type CreateProxyGroupRequest = Omit<
  ProxyGroup,
  'id' | 'member_count' | 'created_at' | 'updated_at'
>;
export type UpdateProxyGroupRequest = CreateProxyGroupRequest;
```

- [ ] **Step 2: Write the hook**

`ui/src/hooks/use-proxy-groups.ts`, mirroring `use-proxies.ts`:

```ts
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type {
  CreateProxyGroupRequest,
  ProxyGroup,
  ProxyGroupListResponse,
  UpdateProxyGroupRequest,
} from '@/types/proxy-group';

const QUERY_KEY = ['proxy-groups'] as const;
const PROXIES_KEY = ['proxies'] as const;

export function useProxyGroups() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: QUERY_KEY,
    queryFn: () => api.get('proxy-groups').json<ProxyGroupListResponse>(),
  });

  // A group mutation changes every member's effective config, so the proxies
  // cache is stale too. Without this the grid shows pre-edit values.
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    void queryClient.invalidateQueries({ queryKey: PROXIES_KEY });
  };

  const create = useMutation({
    mutationFn: (body: CreateProxyGroupRequest) =>
      api.post('proxy-groups', { json: body }).json<ProxyGroup>(),
    onSuccess: invalidate,
  });

  const update = useMutation({
    mutationFn: ({ id, ...body }: UpdateProxyGroupRequest & { id: number }) =>
      api.put(`proxy-groups/${id}`, { json: body }).json<ProxyGroup>(),
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: (id: number) => api.delete(`proxy-groups/${id}`).json<void>(),
    onSuccess: invalidate,
  });

  return { ...query, create, update, remove };
}

export function useProxyGroup(id: number) {
  return useQuery({
    queryKey: [...QUERY_KEY, id],
    queryFn: () => api.get(`proxy-groups/${id}`).json<ProxyGroup>(),
    enabled: Number.isFinite(id),
  });
}
```

- [ ] **Step 3: Add the Zod schema**

Append to `ui/src/lib/form-validation.ts`:

```ts
export const proxyGroupSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  description: z.string().optional(),
  base_domain: z
    .string()
    .regex(/^([a-z0-9-]+\.)+[a-z]{2,}$/i, 'Must be a valid domain, e.g. group.acme.in')
    .optional()
    .or(z.literal('')),
  // null = inherit / "the group says nothing". Not the same as false.
  ssl_enabled: z.boolean().nullable(),
  ssl_forced: z.boolean().nullable(),
  tls_insecure_skip_verify: z.boolean().nullable(),
  block_exploits: z.boolean().nullable(),
});

export type ProxyGroupFormData = z.infer<typeof proxyGroupSchema>;
```

- [ ] **Step 4: Build the grid and form**

`proxy-group-data-grid.tsx` uses `DataGrid` from `@e412/rnui-react` with columns `name`, `base_domain`, `member_count`, `actions`, each with a `meta.skeleton`. `proxy-group-form.tsx` binds with RHF `Form`/`FormField` against `proxyGroupSchema`.

The delete action must handle the 409: when `remove` rejects with a 409, surface the server's message (which carries the member count) rather than a generic failure toast.

The base-domain field, when editing a group that already has one and has members, opens a confirm dialog before submit:

> Changing the base domain re-homes all **{member_count}** member proxies. Their old hostnames stop resolving immediately and Caddy will request new certificates for every new hostname. Continue?

- [ ] **Step 5: Add the group ACL editor**

Task 5 built `GET/POST/PUT/DELETE /api/proxy-groups/{id}/acl`. Without this step that API has no caller, and "the group names one or more ACL groups" — the highest-value inheritable field per the spec — would be unreachable from the UI.

Add a hook `useProxyGroupAcl(groupId)` in `ui/src/hooks/use-proxy-groups.ts`:

```ts
export function useProxyGroupAcl(groupId: number) {
  const queryClient = useQueryClient();
  const key = ['proxy-groups', groupId, 'acl'] as const;

  const query = useQuery({
    queryKey: key,
    queryFn: () => api.get(`proxy-groups/${groupId}/acl`).json<ProxyGroupAclAssignment[]>(),
    enabled: Number.isFinite(groupId),
  });

  // Changing a group's ACL changes what Caddy enforces for every member.
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: key });
    void queryClient.invalidateQueries({ queryKey: ['proxies'] });
  };

  const assign = useMutation({
    mutationFn: (body: { acl_group_id: number; path_pattern: string; priority: number; enabled: boolean }) =>
      api.post(`proxy-groups/${groupId}/acl`, { json: body }).json<ProxyGroupAclAssignment>(),
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: (aclGroupId: number) =>
      api.delete(`proxy-groups/${groupId}/acl/${aclGroupId}`).json<void>(),
    onSuccess: invalidate,
  });

  return { ...query, assign, remove };
}
```

Add the row type to `ui/src/types/proxy-group.ts`:

```ts
export interface ProxyGroupAclAssignment {
  id: number;
  proxy_group_id: number;
  acl_group_id: number;
  path_pattern: string;
  priority: number;
  enabled: boolean;
  acl_group?: { id: number; name: string };
}
```

Render it in `$groupId/edit.tsx` by reusing `components/proxy/forms/acl-selector.tsx` — it already presents exactly this shape for a proxy. Only the mutation target differs, so lift its API calls into props rather than copying the component.

The edit page must state, above the ACL section, that these assignments apply to every member proxy that has not overridden them. An inherited SSO requirement the operator cannot see while editing a member is the failure mode this whole design is built to avoid.

- [ ] **Step 6: Add the routes**

`routes/_dashboard/proxy-groups/index.tsx` (list), `new.tsx`, `$groupId/edit.tsx`. Match the existing route file conventions in `routes/_dashboard/proxies/`. Add a nav entry labelled **Proxy Groups**, gated on the `proxygroups:read` permission via `usePermissions`.

- [ ] **Step 7: Lint and verify in the browser**

Run: `make lint-ui && pnpm --dir ui build`
Expected: clean.

Then `pnpm --dir ui dev`, log in, and confirm: create a group with a base domain, see it in the grid with `member_count` 0, attach an ACL group to it, edit it, and confirm deleting an empty group succeeds.

- [ ] **Step 8: Commit**

```bash
git add ui/src/types/proxy-group.ts ui/src/hooks/use-proxy-groups.ts ui/src/components/proxy-group/ ui/src/routes/_dashboard/proxy-groups/ ui/src/lib/form-validation.ts
git commit -m "feat(ui): proxy group CRUD surface

Group mutations invalidate the proxies cache as well as proxy-groups —
every member's effective config changed, so a stale grid lies.

Editing base_domain on a group with members confirms first, naming the
count: the rename re-homes them and triggers fresh ACME issuance."
```

---

## Task 8: Frontend — inheritance in the proxy form

The part the user actually feels. Two components carry the weight: a tri-state control that shows what "inherit" resolves to, and a hostname field that composes against the group's base domain.

**Files:**
- Create: `ui/src/components/proxy/forms/group-selector.tsx`
- Create: `ui/src/components/proxy/forms/shared/inheritable-switch.tsx`
- Create: `ui/src/components/proxy/forms/shared/hostname-field.tsx`
- Create: `ui/src/components/proxy/cells/proxy-group-cell.tsx`
- Modify: `ui/src/types/proxy.ts`, `ui/src/hooks/use-proxies.ts`
- Modify: `ui/src/components/proxy/proxy-data-grid.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx`
- Modify: `ui/src/components/proxy/forms/reverse-proxy-form.tsx`

**Interfaces:**
- Consumes: `useProxyGroups` (Task 7); the `effective` + `_source` response shape (Task 6).
- Produces: `<InheritableSwitch name value onChange groupValue systemDefault />`, `<HostnameField />`, `<GroupSelector />`.

- [ ] **Step 1: Update the proxy types**

In `ui/src/types/proxy.ts`, the four booleans become `boolean | null` on `ProxyConfig`, and add:

```ts
  group_id?: number | null;
  group_name?: string | null;
  hostname_label?: string | null;
  effective?: {
    ssl_enabled: boolean;
    ssl_forced: boolean;
    block_exploits: boolean;
    tls_insecure_skip_verify: boolean;
    custom_headers?: CustomHeaders;
    /** Where each resolved value came from: 'proxy' | 'group' | 'default'. */
    _source: Record<string, 'proxy' | 'group' | 'default'>;
  };
```

- [ ] **Step 2: Build the tri-state control**

`ui/src/components/proxy/forms/shared/inheritable-switch.tsx`. Three states: `Inherit` / `On` / `Off`, backed by `boolean | null`. The Inherit option's label resolves live:

```tsx
interface InheritableSwitchProps {
  value: boolean | null;
  onChange: (value: boolean | null) => void;
  /** The selected group's value for this field, or null if the group is silent. */
  groupValue: boolean | null;
  /** Applied when neither the proxy nor the group has an opinion. */
  systemDefault: boolean;
  /** When there is no group, collapse to a plain switch. */
  hasGroup: boolean;
  label: string;
}

export function InheritableSwitch({
  value, onChange, groupValue, systemDefault, hasGroup, label,
}: InheritableSwitchProps) {
  // Without a group, "inherit" resolves to the system default and there is
  // nothing to show the user — so don't offer a third state they can't reason
  // about. null still round-trips to the server, which is what keeps the proxy
  // inheriting if it is later added to a group.
  if (!hasGroup) {
    return (
      <Switch
        checked={value ?? systemDefault}
        onCheckedChange={(next) => onChange(next)}
        aria-label={label}
      />
    );
  }

  const inherited = groupValue ?? systemDefault;
  const inheritLabel = `Inherit (${inherited ? 'on' : 'off'})`;

  return (
    <Select
      value={value === null ? 'inherit' : value ? 'on' : 'off'}
      onValueChange={(next) =>
        onChange(next === 'inherit' ? null : next === 'on')
      }
    >
      <SelectTrigger aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="inherit">{inheritLabel}</SelectItem>
        <SelectItem value="on">On</SelectItem>
        <SelectItem value="off">Off</SelectItem>
      </SelectContent>
    </Select>
  );
}
```

Import `Switch`, `Select*` from `@e412/rnui-react`, matching how sibling form components import them.

- [ ] **Step 3: Build the hostname field**

`ui/src/components/proxy/forms/shared/hostname-field.tsx`. When the selected group has a `base_domain`, render a single-label input with the base domain as a static suffix; otherwise a normal hostname input.

```tsx
// Moving into a base-domain group: pre-fill the label by stripping the suffix
// when the current hostname already sits under it, so the common case is a
// no-op rather than a retype.
export function deriveLabel(hostname: string, baseDomain: string): string {
  const suffix = `.${baseDomain}`;
  return hostname.endsWith(suffix) ? hostname.slice(0, -suffix.length) : '';
}
```

Add a unit test for `deriveLabel` covering: exact suffix match, no match, and a hostname equal to the base domain itself (which yields `''`, an invalid label — the field must then show a validation error rather than submit an empty label).

- [ ] **Step 4: Wire the form**

In `reverse-proxy-form.tsx` (and the redirect/static forms), add `<GroupSelector />` and replace the four boolean switches with `<InheritableSwitch />`, feeding `groupValue` from the selected group fetched via `useProxyGroups`. Changing the group must re-render the inherit labels — key the switches on `group_id`.

Submitting sends `null` for inherited fields. Do not coerce `null` to `false`.

- [ ] **Step 5: Add the grid column and filter**

`proxy-group-cell.tsx` renders `group_name` as a badge, or an em dash when ungrouped. Add the column to `proxy-data-grid.tsx` beside the `acl` column. Add a `group` entry to `filterFields` in `routes/_dashboard/proxies/index.tsx` and thread it into the `params` `useMemo` and `UseProxiesOptions`.

- [ ] **Step 6: Lint, build, and verify end to end**

Run: `make lint-ui && pnpm --dir ui build`

Then, with the backend running, drive the real flow:

1. Create a group `internal` with base domain `group.acme.in` and `block_exploits = On`.
2. Create a proxy in that group with label `abc`. Confirm the form shows `abc` + a static `.group.acme.in`, and that Block Exploits shows `Inherit (on)`.
3. Save. Confirm the grid shows hostname `abc.group.acme.in` and group `internal`.
4. `GET /api/caddy-config` (the config preview) and confirm the security routes for `abc.group.acme.in` are present — the inherited `block_exploits` reached Caddy.
5. Set the proxy's Block Exploits to `Off`, save, and confirm the security routes disappear. That is the override path, verified against the served config rather than the form.
6. Try to delete the group. Confirm the UI reports 409 naming 1 member proxy.

Expected: each step behaves as described. Step 4 and 5 are the ones that matter — they check the UI and Caddy agree.

- [ ] **Step 7: Commit**

```bash
git add ui/src/
git commit -m "feat(ui): proxy group inheritance in the proxy form

InheritableSwitch is tri-state (Inherit/On/Off) and resolves the Inherit
label live against the selected group, so 'Inherit (on)' and 'On' are never
confused. With no group it collapses to a plain switch.

HostnameField composes <label>.<base_domain> and pre-fills the label by
stripping the suffix when the existing hostname already sits under it."
```

---

## Task 9: Documentation

**Files:**
- Modify: `docs/API.md`, `README.md`

- [ ] **Step 1: Document the endpoints and the breaking change**

Add the `/api/proxy-groups` table from the spec to `docs/API.md`, plus the `group_id` / `hostname_label` / `effective` fields on the proxy endpoints.

Call out the breaking change explicitly under a **Breaking changes** heading:

> **`PUT /api/proxies/{id}`** — an omitted `ssl_enabled`, `ssl_forced`, `block_exploits`, or `tls_insecure_skip_verify` now means *inherit from the group* (or the system default), not *keep the existing value*. Send an explicit `true`/`false` to set a value, or `null` to inherit. Clients that relied on partial updates preserving these fields must now send them.

Document that renaming a group's `base_domain` re-homes every member and triggers fresh certificate issuance.

- [ ] **Step 2: Commit**

```bash
git add docs/API.md README.md
git commit -m "docs: proxy groups API + breaking change on PUT /api/proxies/{id}"
```

---

## Verification checklist

Before opening the PR:

- [ ] `cd backend && gofmt -l .` prints nothing
- [ ] `cd backend && go build ./...` succeeds
- [ ] `cd backend && go test ./... -short` passes
- [ ] `cd backend && go test ./internal/repository/ -run TestProxyGroup` passes (needs Docker)
- [ ] `make lint-ui` passes
- [ ] `pnpm --dir ui build` succeeds
- [ ] The Task 8 Step 6 end-to-end walk was actually performed, and steps 4–5 confirmed the config preview changed
- [ ] `golangci-lint` is not installed locally — CI is the gate for `gosec`/`gocritic`/`misspell`
