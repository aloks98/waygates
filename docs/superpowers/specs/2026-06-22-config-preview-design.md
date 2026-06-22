# B1 — Config Preview — Design

**Date:** 2026-06-22
**Status:** Approved — ready for implementation plan
**Context:** First of the backend-pipeline backlog items (B1). Lets an admin inspect the **generated Caddy configuration** in the UI — per-proxy (on the read-only proxy overview) and globally (the full `caddy.json`). Branch `feat/config-preview` off master.

## Goal

Surface the Caddy config Waygates generates so an admin can see/verify what each proxy produces and what the whole config looks like, without shelling into the container. **Generated config only** (what Waygates builds from the DB) — not the live admin-API config.

## Context (current state)

From exploration of `backend/internal/caddy`:

- **Config is generated in-memory** by `config.Builder` (`backend/internal/caddy/config/builder.go`): `Build() (*CaddyConfig, error)`, `BuildJSON() ([]byte, error)`, and **`BuildSingleProxy(proxy *models.Proxy) (*CaddyConfig, error)`** (already public — builds one proxy's routes). The builder is a fluent API: `SetHTTPProxies / SetACLGroups / SetACLAssignments / SetNotFoundSettings / SetLayer4App` then build.
- **`SyncService.performFullSyncJSON()`** (`service/sync_service.go`) gathers all proxies + ACL groups + assignments + L4 + not-found settings from the DB, configures the builder, calls `BuildJSON()`, and writes `/etc/caddy/caddy.json` + reloads. The gather+build is pure in-memory; the write/reload is separate.
- **Per-proxy routes are keyed by host matcher** (`NewHostMatcher(proxy.Hostname)`), not merged — so per-proxy extraction is clean, and `BuildSingleProxy` already does it.
- **Live config** (`GET localhost:2019/config/`) is available via the reloader's HTTP client but is out of scope (see below).
- **Read-endpoint pattern:** handler→service→routes (mirror `caddy_logs_handler.go` / `audit_handler.go`); routes in `api/routes/routes.go` gated by `chimw.RequirePermission(authAdapter, "<key>", mwConfig)`; permissions in `backend/rbac.yaml` (e.g. `caddy_logs:read`). The caddy builder/reloader/file-manager + `syncService` are constructed in `routes.go`.
- **UI:** `@e412/rnui-react` exports a **`JsonViewer`** (`data: Record<string, any>`, `showLineNumbers`, `defaultExpanded`, `collapseOn`, `title`) — not yet used. Data fetching = `ky` `api` client + React-Query `use-*` hooks. The HTTP proxy overview (`routes/_dashboard/proxies/$proxyId/overview.tsx`) renders cards (`ProxyConfigCard`, `ProxyAccessCard`, then a `lg:grid-cols-2` grid of HTTPS + Details); the new card slots in after `ProxyAccessCard`. Dashboard routes are code-based (`lib/router.tsx`); sidebar nav in `components/layout/sidebar.tsx`.

## Scope

**In:**
- Backend: `GET /api/caddy-config` (full generated config) and `GET /api/proxies/$proxyId/config-preview` (one proxy's generated slice), gated by a new `caddy_config:read` permission.
- A reusable no-write config build (refactor the gather+build out of `performFullSyncJSON`) so the global preview equals what a sync would push.
- UI: a per-proxy "Generated Caddy config" card (`JsonViewer` + copy) on the HTTP proxy overview, and a standalone `/dashboard/caddy-config` page (full config) + sidebar entry.

**Out of scope:**
- **Live config** (Caddy admin-API `GET :2019/config/`) and drift comparison — generated only.
- **L4 per-proxy** config card — no `BuildSingleL4Proxy` exists; the L4 overview links to the global view instead (a future item can add per-L4 extraction).
- No DB change; read-only (no config editing from the UI).

## Backend design

### Reusable build (no write)
Refactor `SyncService.performFullSyncJSON()` so the **gather + configure-builder + build** logic lives in a reusable method, e.g. `(s *SyncService) GenerateConfigJSON(ctx) ([]byte, error)` (or returns `*config.CaddyConfig`). `performFullSyncJSON` then = `GenerateConfigJSON()` + write + reload. This keeps the preview byte-identical to what sync pushes and avoids duplicating the gather.

### Endpoints (new `ConfigPreviewService` + handler, mirroring `caddy_logs`)
- **`GET /api/caddy-config`** → the full generated config. Service calls `syncService.GenerateConfigJSON()` (or the shared build); returns the parsed config object in the standard `utils.Success` envelope (`data` = the `CaddyConfig`).
- **`GET /api/proxies/$proxyId/config-preview`** → one proxy's generated config. Service loads the proxy by id (404 if missing), loads its ACL assignments + the referenced ACL groups, sets them on a fresh `config.Builder`, calls `BuildSingleProxy(proxy)`, returns the resulting `CaddyConfig` as `data`. (Including the proxy's ACL data so the preview reflects ACL/TLS handlers, not just the bare route.)
- Both `RequirePermission(authAdapter, "caddy_config:read", mwConfig)`. The per-proxy route lives alongside the existing `/api/proxies/$proxyId/*` routes; the global one as `/api/caddy-config`.
- `rbac.yaml`: add a "Caddy Config" group:
  ```yaml
  - name: "Caddy Config"
    permissions:
      - key: "caddy_config:read"
        name: "View Caddy Config"
        description: "View the generated Caddy configuration"
  ```

### Response shape
Return the config as a JSON object under `data` (the `CaddyConfig` marshals to the same JSON Caddy consumes). The UI feeds `data` straight into `JsonViewer`.

## UI design

### Per-proxy card (HTTP overview)
`components/proxy/overview/proxy-config-preview-card.tsx` → `ProxyConfigPreviewCard({ proxyId })`: a `Card` titled **"Generated Caddy config"** containing rnui `JsonViewer` (`data`, `showLineNumbers`, collapsible) + a **Copy** button (copies the JSON). Data via `useProxyConfigPreview(proxyId)` (React Query). Loading → `Skeleton`; error → a short inline message. Mounted in `proxies/$proxyId/overview.tsx` after `<ProxyAccessCard>`.

### Global page
`routes/_dashboard/caddy-config.tsx` → `CaddyConfigPage` (`space-y-6` + `<h1>Caddy Config</h1>`): the full generated config in a `JsonViewer` + Copy + a Refresh (refetch). Data via `useCaddyConfig()`. Registered in `lib/router.tsx` (`/caddy-config`, lazy named export) + a sidebar nav entry (lucide icon, e.g. `FileCode`/`Braces`). The L4 overview gets a small "View generated config" link to this page (since L4 per-proxy is deferred).

### Data layer
`hooks/use-config-preview.ts` (or split): `useCaddyConfig()` → `api.get('caddy-config')`; `useProxyConfigPreview(id)` → `api.get('proxies/'+id+'/config-preview')`. A `types/caddy-config.ts` can type the response loosely as `Record<string, unknown>` (the JsonViewer doesn't need a precise Caddy type).

## Decisions (approved)

- **Generated config only** (not live admin-API config); no drift view.
- **Per-proxy (HTTP) + global** scope; **L4 per-proxy deferred** (L4 overview links to the global page).
- **Global home:** a standalone `/dashboard/caddy-config` page + sidebar entry (not a Settings section).
- **Reuse the sync gather** via a no-write `GenerateConfigJSON` so preview == pushed config.
- Permission `caddy_config:read`; read-only.

## Architecture & files

- **Backend:** `service/sync_service.go` (extract `GenerateConfigJSON`), new `service/config_preview_service.go`, `handlers/config_preview_handler.go`, `api/routes/routes.go`, `rbac.yaml`. Tests alongside (the per-proxy build + the global build + handler).
- **Frontend:** `components/proxy/overview/proxy-config-preview-card.tsx`, `routes/_dashboard/caddy-config.tsx`, `hooks/use-config-preview.ts`, `types/caddy-config.ts`, `lib/router.tsx`, `components/layout/sidebar.tsx`, and the per-proxy overview mount (+ the L4 overview link).

## Decomposition (for the plan)

Subagent-driven, each gate-able:
1. **Backend** — refactor `GenerateConfigJSON` (no-write), `ConfigPreviewService` (global + per-proxy via `BuildSingleProxy` with ACL), handler, `caddy_config:read` RBAC, both routes; Go unit tests (global build returns valid config; per-proxy build returns that proxy's routes; missing proxy → not-found).
2. **UI per-proxy card** — `JsonViewer` card + `useProxyConfigPreview` + copy, mounted on the HTTP overview.
3. **UI global page + nav** — `/dashboard/caddy-config` page (`JsonViewer` + copy + refresh) + `useCaddyConfig` + router + sidebar + the L4-overview link.

## Testing

- **Backend:** Go unit tests (testify) — `GenerateConfigJSON` produces a valid config (matches what the builder yields); the per-proxy service returns a `CaddyConfig` containing the proxy's host-matched route(s) and 404s on a missing id; handler permission gating. No Docker/caddy needed (pure build).
- **Frontend:** gate (`pnpm --dir ui build` + `check` + `test`, existing tests stay green); unit-test any pure helper (likely none — the JsonViewer + hooks are integration-level, verified by the gate).

## Risks & notes

- **Per-proxy ACL fidelity:** `BuildSingleProxy` must be given the proxy's ACL groups/assignments so the previewed routes include ACL handlers; the service must load and set them (verify against how `performFullSyncJSON` sets ACL data).
- **Refactor safety:** extracting `GenerateConfigJSON` must not change what `performFullSyncJSON` writes/reloads — the sync behavior is unchanged, only restructured; cover with the existing/extended builder+sync tests.
- **Secrets in config:** the generated Caddy config is infrastructure config (hostnames, upstreams, TLS, ACL structure). It does not contain DB/JWT secrets, but reviewers should confirm no sensitive material (e.g. basic-auth hashes) is exposed in the preview; if any is, redact it in the service before returning. Gating on `caddy_config:read` limits exposure to authorized admins.
- **Read-only:** no editing; the preview never writes or reloads.
