# Proxy Groups — Design

**Date:** 2026-07-10
**Status:** Approved — ready for implementation plan
**Context:** Proxies today are a flat list. Settings that logically belong to a fleet — "every internal service requires SSO", "every service under `*.corp.acme.in`" — must be restated on each proxy. This spec adds `ProxyGroup`: a named, config-bearing parent that HTTP proxies inherit from.

## Goal

Let an operator define a group once (base domain, TLS defaults, security defaults, custom headers, ACL assignments) and have its member proxies inherit those values, while any individual proxy can still override what it needs.

## Naming

The codebase already has `ACLGroup` (table `acl_groups`), an **auth** grouping joined to proxies via `proxy_acl_assignments`. This feature is a different concept and takes a distinct name throughout: entity `ProxyGroup`, table `proxy_groups`, RBAC prefix `proxygroups`, UI label "Proxy Groups". No code, column, or endpoint may call this a "group" unqualified.

## Scope

**In:**
- `proxy_groups` entity with CRUD API, RBAC, and audit logging.
- A **nullable, single** group per HTTP proxy (`proxies.group_id`).
- Inheritable settings: `base_domain`, `ssl_enabled`, `ssl_forced`, `tls_insecure_skip_verify`, `block_exploits`, `custom_headers`, ACL assignments.
- Base-domain hostname composition: a member proxy is addressed by a DNS label; its effective hostname is `<label>.<group.base_domain>`.
- A single resolver that both the sync service and the API read path are structurally forced to use.
- Group column + filter on the proxy list; group management UI; tri-state inheritance controls in the proxy form.

**Out of scope:** L4/TCP/UDP proxy grouping (every inheritable field here is an L7 concept; L4 grouping earns its own spec if it earns one at all); nested/parent groups; many-to-many membership; group-scoped RBAC (users restricted to their groups); a group-level `is_active` kill switch; inheriting `load_balancing`.

## Locked decisions

- **Purpose:** shared config inheritance, not merely organizational labelling.
- **Cardinality:** a proxy belongs to exactly one group, or none. No precedence rules between groups are needed, and none exist.
- **Override direction:** the group provides *defaults*; the proxy overrides. This requires the inheritable proxy columns to be tri-state (`NULL` = inherit).
- **Collections merge:** headers merge per key (proxy wins); ACL assignments union by `acl_group_id` (proxy's row wins wholesale).
- **`load_balancing` is NOT inheritable.** Considered and rejected: a policy block is not key-mergeable (a proxy's `{policy: "ip_hash"}` merged into a group's `{policy: "round_robin", health_checks: {…}}` produces a config nobody authored), and whole-block override adds a field for no gain.
- **Hostname is materialized, settings are resolved lazily.** See "The asymmetry" below — this is deliberate.
- **Deleting a non-empty group is blocked** (409), enforced both in the service and by `ON DELETE RESTRICT`.
- **HTTP proxies only.**

## The asymmetry

Settings resolve lazily; the hostname materializes eagerly. These pull in opposite directions on purpose:

- **Settings must stay unresolved in the DB.** Writing a group's value down onto the proxy row would erase the difference between "inherited `true`" and "explicitly `true`", and the next group edit could not tell which members to update. `NULL` must survive in storage.
- **The hostname must be resolved in the DB.** `proxies.hostname` is `NOT NULL UNIQUE` and is the key that the Caddy builder, certificate provisioning, metrics, and log correlation all read. Postgres cannot enforce uniqueness across a join, so a computed hostname would push collision detection into application queries and route every existing `proxy.Hostname` reader through the resolver.

So `hostname` is a denormalized cache with exactly one writer (the proxy/group service), guarded by an invariant test. This also makes detaching a proxy from a group free: the full hostname is already on the row.

## Data model

### Migration `000014_create_proxy_groups`

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
  path_pattern   VARCHAR(255) NOT NULL DEFAULT '/*',
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

ALTER TABLE proxies
  ADD COLUMN group_id INT NULL REFERENCES proxy_groups(id) ON DELETE RESTRICT,
  ADD COLUMN hostname_label VARCHAR(63) NULL,
  ADD CONSTRAINT chk_proxies_label_requires_group
    CHECK (hostname_label IS NULL OR group_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_proxies_group_id ON proxies(group_id);

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              DROP NOT NULL,
  ALTER COLUMN ssl_forced               DROP NOT NULL,
  ALTER COLUMN block_exploits           DROP NOT NULL,
  ALTER COLUMN tls_insecure_skip_verify DROP NOT NULL;
```

`proxy_group_acl_assignments` intentionally mirrors `proxy_acl_assignments` column-for-column so the resolver merges like with like.

The down migration must backfill before restoring `NOT NULL`: `ssl_forced` `NULL → true`, the other three `NULL → false`. It drops `group_id` and `hostname_label` (and therefore silently discards label-addressing — acceptable, since a down-migration to a schema without groups has no way to represent them; `hostname` already holds the correct absolute value).

### Model changes — `backend/internal/models/proxy.go`

`SSLEnabled`, `SSLForced`, `BlockExploits`, `TLSInsecureSkipVerify` become `*bool`. **Remove the `default:true` GORM tag from `SSLForced`.** That tag is what causes GORM to omit an explicit `false` on INSERT; a pointer field is the correct fix and the tag must not survive alongside it.

New persisted fields: `GroupID *int`, `HostnameLabel *string`.
New computed fields (`gorm:"-"`, populated by a summary join in `ProxyRepository.List`, mirroring the existing `ACLGroupCount`/`ACLGroupNames`): `GroupName *string`.

New model `backend/internal/models/proxy_group.go` with an explicit `TableName()` and explicit `column:` tags on every field.

### The cross-table rule

A CHECK constraint cannot span tables, so the **service** owns this invariant:

> A proxy is label-addressed **iff** it has a group *and* that group has a non-null `base_domain`.
> Label-addressed: `hostname_label` is set, `hostname = hostname_label + "." + group.base_domain`.
> Otherwise: `hostname_label IS NULL` and `hostname` is an absolute hostname supplied by the caller.

### Membership edge cases

| Operation | Behavior |
|---|---|
| Ungrouped proxy → base-domain group | Caller must supply `hostname_label`. UI pre-fills by stripping the base domain from the current hostname when it already matches. |
| Ungrouped proxy → group with no `base_domain` | `hostname` is kept as-is; `hostname_label` stays `NULL`. |
| Proxy leaves a group | `group_id` and `hostname_label` are set to `NULL`. `hostname` is unchanged (already materialized). |
| Group's `base_domain` changes | One transaction: recompute every member's `hostname`. Any collision aborts the whole transaction → 409. |
| Group's `base_domain` set to `NULL` while members hold labels | 409. Those members would have no hostname. |
| Group deleted with members | 409 with the member count. |

`base_domain` is **not** unique across groups. Two groups may share one, and a label collision between their members is caught by the existing unique index on `proxies.hostname` like any other. Nothing further is needed.

## Resolution

One pure function, no DB access, in a new package `backend/internal/service/proxygroup`:

```go
// Resolve merges a group's defaults into a proxy. g may be nil.
func Resolve(
    p        models.Proxy,
    g        *models.ProxyGroup,
    proxyACL []models.ProxyACLAssignment,
    groupACL []models.ProxyGroupACLAssignment,
) EffectiveProxy

type EffectiveACLAssignment struct {
    ACLGroupID  int
    PathPattern string
    Priority    int
    Enabled     bool
}
```

`EffectiveProxy` mirrors `Proxy` but with plain `bool` fields and an `ACL []EffectiveACLAssignment` — every value decided, nothing left to interpret. The two ACL row types are distinct Go types over identical columns, which is why the resolver normalizes both into `EffectiveACLAssignment` rather than merging one into the other.

**Scalars** (`ssl_enabled`, `ssl_forced`, `tls_insecure_skip_verify`, `block_exploits`) resolve in three steps: proxy value if non-nil → group value if non-nil → system default. System defaults preserve today's behavior: `ssl_forced` is `true`, the other three are `false`.

**Headers** merge per key — `Request` and `Response` maps independently — with the proxy's value winning on collision.

**ACL assignments** union by `acl_group_id`. Where both group and proxy assign the same `acl_group_id`, the proxy's assignment row wins *wholesale* (its `path_pattern`, `priority`, and `enabled` together; never a field-wise merge, which would produce a row neither side authored). A proxy opts out of an inherited ACL by assigning the same `acl_group_id` with `enabled = false`.

**Not inherited:** `hostname`, `upstreams`, `type`, `name`, `description`, `is_active`, `load_balancing`, `redirect_config`, `static_config`.

### Enforcing single-source resolution

Two call sites need effective values: `sync_service.buildConfigBytes` (what Caddy serves) and the API read path (what the user sees). If each inlines its own nil-checks, they will eventually diverge, and a divergence between the UI and the served config is the worst failure this feature can produce — the proxy detail page would report an ACL that Caddy is not enforcing, with no error anywhere.

This is prevented structurally, not by convention. The Caddy builder accepts `EffectiveProxy`, not `models.Proxy`:

```go
func BuildHTTPProxy(p proxygroup.EffectiveProxy) (json.RawMessage, error)
```

`EffectiveProxy` exposes no constructor other than `Resolve`. Passing an unresolved proxy to the builder does not compile. The read path is forced through the same door.

### Loading the inputs

`Resolve` takes no repository, so its callers must supply the group and both ACL sets:

- **Single-proxy paths** (`GetProxyByID`, `GenerateProxyConfigJSON`, `BuildSingleProxy`): the proxy repository preloads `Group`, and the ACL repository is asked for the proxy's and the group's assignments.
- **`buildConfigBytes`**: it already loads all proxies flat and all `acl_groups` once. It additionally loads all `proxy_groups` and all `proxy_group_acl_assignments` once into maps keyed by group ID, then resolves each proxy against its map entry. No N+1.

## API

Standard `{success, data, error}` envelope. New handler `handlers/proxy_group.go`, service `service/proxy_group_service.go`, repository `repository/proxy_group_repository.go` + interface in `repository/interfaces.go`.

| Method · Path | Perm | Notes |
|---|---|---|
| `GET /api/proxy-groups` | `proxygroups:read` | List rows carry `member_count` |
| `GET /api/proxy-groups/{id}` | `proxygroups:read` | Includes ACL assignments |
| `POST /api/proxy-groups` | `proxygroups:create` | |
| `PUT /api/proxy-groups/{id}` | `proxygroups:update` | Handles the `base_domain` rewrite |
| `DELETE /api/proxy-groups/{id}` | `proxygroups:delete` | 409 + member count if non-empty |
| `GET /api/proxy-groups/{id}/acl` | `acl:read` | |
| `POST /api/proxy-groups/{id}/acl` | `acl:update` | |
| `PUT /api/proxy-groups/{id}/acl/{assignmentId}` | `acl:update` | |
| `DELETE /api/proxy-groups/{id}/acl/{aclGroupId}` | `acl:delete` | |

The nested ACL routes mirror the existing proxy ACL routes and reuse the `acl:*` permissions — the same capability applied to a different subject.

### Changes to existing proxy endpoints

- `POST`/`PUT /api/proxies` accept `group_id` and `hostname_label`. `hostname` is required unless the proxy is label-addressed.
- `GET /api/proxies/{id}` returns the raw row **and** an `effective` object with a `_source` map:

```json
{
  "block_exploits": null,
  "group_id": 3,
  "effective": {
    "block_exploits": true,
    "_source": { "block_exploits": "group" }
  }
}
```

The edit form needs the raw nullable values to distinguish "inherit" from an explicit choice; the overview page and config preview need the effective values; `_source` is what lets the UI say *where* a value came from. Without it, "Inherit (on)" and "On" are indistinguishable to the user — the same lie the resolver exists to prevent, relocated into the form. `_source` values: `"proxy"`, `"group"`, `"default"`.

- `GET /api/proxies` list rows gain `group_id` and `group_name`.
- Group filtering reuses the existing `parseFilterParam` operator syntax: `?group=eq:3`, `?group=in:1,2`, `?group=not:3`. Threaded through `ListProxiesRequest` → `ProxyListParams` → a `WHERE` clause.

### RBAC — `backend/rbac.yaml`

New permission group:

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

`admin` already holds `*`. `operator` gains `proxygroups:*`; `viewer` gains `proxygroups:read`.

## Sync

A group mutation can change the effective config of every member at once.

- **Any group mutation** — settings, ACL assignments, deletion — triggers a **full config rebuild**, not per-proxy sync. `buildConfigBytes` already reconstructs the whole Caddy config from scratch, so this is cheaper to reason about than incremental invalidation and matches what the 60-second loop does anyway.
- **A `base_domain` change** is one transaction: update the group, recompute each member's materialized `hostname`, let the existing unique index catch collisions. A conflict aborts the transaction and returns 409 naming the colliding hostname. The rebuild happens only after commit.
- Proxy-level mutations continue to call `syncService.SyncProxy(id)` as they do today.
- Group create/update/delete mirrors the existing create-then-rollback-on-sync-failure pattern in `ProxyService.CreateProxy`.

Because `hostname` stays materialized and unique-indexed, `HostnameExists` and `ExistingHostnames` need no changes.

### Operational consequence

Renaming a group's `base_domain` re-homes every member proxy. Caddy will provision fresh certificates for all new hostnames, and the old hostnames stop resolving immediately. On a large group this is a burst of ACME activity. This is correct behavior for the feature, but it feels destructive, so the UI confirms it with the affected member count, as the bulk-delete bar does.

### Audit

`proxy_group.create`, `proxy_group.update`, `proxy_group.delete`, and ACL assignment changes are audit-logged. A `base_domain` rewrite additionally logs the affected proxy IDs — otherwise a mass re-homing leaves no trace of what moved.

## Frontend

- `types/proxy-group.ts`; `hooks/use-proxy-groups.ts` (React Query, key `['proxy-groups']`).
- Routes under `routes/_dashboard/proxy-groups/`: `index.tsx`, `new.tsx`, `$groupId/edit.tsx`.
- `components/proxy-group/proxy-group-data-grid.tsx` using `DataGrid` from `@e412/rnui-react`, with a `member_count` column.
- Group forms bound with RHF `Form`/`FormField`; Zod schema in `lib/form-validation.ts`.
- `components/proxy/forms/group-selector.tsx`, mirroring `acl-selector.tsx`.
- `proxy-data-grid.tsx` gains a `group` column beside the `acl` column; `filterFields` gains a `group` entry threaded through `params` → `UseProxiesOptions`.

Two components carry the UX weight:

**Tri-state control.** Each inheritable boolean becomes `Inherit (on)` / `On` / `Off`. The Inherit label resolves live against the selected group's value, so the user never has to open the group elsewhere to learn what "inherit" means. With no group selected it collapses to a plain switch (inherit resolves to the system default; there is nothing to show).

**Hostname field.** When the selected group has a `base_domain`, the input narrows to a single label with the base domain rendered as a static suffix adornment. Moving a proxy into a base-domain group prompts for the label, pre-filled by stripping the suffix when the current hostname already ends in it. Moving out clears the group and restores a normal, editable hostname field holding the current value.

**Cache invalidation.** A group mutation invalidates `['proxies']` as well as `['proxy-groups']`; every member's effective config changed and a stale grid would show pre-edit values.

## Testing

`Resolve` holds the correctness, so it carries the heaviest coverage: a table-driven unit test over proxy `nil/true/false` × group `nil/true/false` for each scalar, plus header-merge cases and the ACL union including the `enabled = false` opt-out.

Four further tests each pin an invariant this design leans on:

- **Cache-drift.** For every label-addressed proxy, `hostname == hostname_label + "." + group.base_domain`. Asserted after every mutation path. This is the single invariant Approach 2 trades for; it fails loudly.
- **Equivalence.** A grouped proxy and an ungrouped proxy carrying the group's values written down produce byte-identical Caddy JSON.
- **`base_domain` rename** (integration, testcontainers): recomputes all members; a collision aborts the transaction and returns 409 with nothing written.
- **Delete with members** returns 409 — asserted at the service layer *and* against the raw DB, proving `ON DELETE RESTRICT` is present rather than only the Go-side check.

Plus a **schema test** asserting the `ProxyGroup` and `Proxy` model columns match the migration. This codebase has been bitten by GORM naming drift before; explicit `column:` tags plus this test are the guard.

## Error handling

| Condition | Response |
|---|---|
| Hostname collision (any path) | 409, existing `ErrHostnameConflict`, naming the colliding hostname |
| Delete group with members | 409, `ErrGroupHasMembers`, with member count |
| `base_domain` set to `NULL` while members hold labels | 409, naming affected members |
| `hostname_label` set without a group | 400 (also caught by `chk_proxies_label_requires_group`) |
| Label-addressed proxy given an absolute `hostname` | 400 |
| Group name collision | 409, `uq_proxy_groups_name` |

The `ON DELETE RESTRICT` violation surfaces as a Postgres foreign-key error; the repository maps it to `ErrGroupHasMembers` rather than leaking a driver error.

## Build sequence

1. Migration `000014` + models (`*bool` conversion, `ProxyGroup`) + schema test.
2. `proxygroup.Resolve` + `EffectiveProxy` + the resolver unit-test table. No callers yet.
3. Switch the Caddy builder to accept `EffectiveProxy`; thread `Resolve` through `sync_service`. Equivalence test.
4. `ProxyGroupRepository` + `ProxyGroupService` (including the `base_domain` transaction) + integration tests.
5. Handler, routes, `rbac.yaml`, audit logging.
6. Proxy endpoint changes: `group_id`/`hostname_label` accepted, `effective` + `_source` returned, group filter.
7. Frontend: group CRUD surface, then the group selector, tri-state control, and hostname field in the proxy form.

Steps 1–3 land without any user-visible change: `Resolve` with a `nil` group is the identity function on today's behavior, which is what the equivalence test asserts.
