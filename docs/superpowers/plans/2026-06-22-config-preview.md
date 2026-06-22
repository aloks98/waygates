# B1 — Config Preview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an admin view the **generated** Caddy config — per-proxy on the HTTP proxy overview, and the full config on a standalone page — via read-only endpoints gated by a new `caddy_config:read` permission.

**Architecture:** Add two generate-only (no write/reload) methods to `SyncService` reusing the existing builder: `GenerateConfigJSON()` (full, reuses the shared `s.jsonBuilder` under `s.mu`) and `GenerateProxyConfigJSON(id)` (per-proxy, fresh builder via `BuildSingleProxy` with the proxy's ACL set). A thin `ConfigPreviewHandler` exposes them; the UI renders the JSON with rnui `JsonViewer`.

**Tech Stack:** Go 1.25 (chi, zap, testify), React 19 + `@e412/rnui-react` (`JsonViewer`), TanStack Router + Query, ky.

## Global Constraints

- **Generated config only** (no live admin-API config). **Read-only** — never writes/reloads.
- New permission **`caddy_config:read`** (mirror the `caddy_logs:read` group). Admin-only by default (like `caddy_logs:read`, not in viewer/operator role templates) — keep that default.
- **Show config as-is** (admin-gated); redaction of sensitive fields (basic-auth hashes / provider secrets) is a documented follow-up, NOT in this scope.
- **Don't change sync behavior:** refactoring `performFullSyncJSON` must produce byte-identical writes/reloads — only restructured.
- **Concurrency:** the shared `s.jsonBuilder` is used by the periodic sync; the global generate must hold `s.mu` while it mutates+builds. The per-proxy generate uses a FRESH builder (no shared mutation, no lock).
- **Per-task gates:** backend → `gofmt -l <files>` + `go build ./...` + `go test ./... -short` (golangci-lint/Docker unavailable locally → CI runs lint; handle every returned error incl. `Close()`, no appendAssign, US spelling — see the prior caddy-logs lint fixes). Frontend → `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` (existing tests stay green).
- **Commit trailer:** `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Stage only each task's files; never `git add -A`.

## Reference facts (from exploration)

- `service/sync_service.go`: `performFullSyncJSON()` — gather (proxies via `proxyRepo.List`, `settingsRepo.GetNotFoundSettings`, ACL groups via `aclRepo.ListGroups`, per-proxy assignments via `aclRepo.GetProxyACLAssignments`, L4 via `l4Builder.BuildL4Config`) at ~309–398; setters `SetHTTPProxies/SetACLGroups/SetACLAssignments/SetNotFoundSettings` (+ `SetLayer4App/SetL4TLSHostnames`) ~400–405; `BuildJSON()` ~406–410; write/validate/reload ~412–474. `SyncService` has `proxyRepo`, `settingsRepo`, `aclRepo`, `l4ProxyRepo`, `jsonBuilder *config.Builder`, `l4Builder`, `mu sync.RWMutex`. `initJSONBuilder(cfg, logger)` constructs the shared builder (`config.NewBuilder(...)` + `SetSettings` + the ACL builder).
- `caddy/config/builder.go`: `BuildJSON() ([]byte,error)` = `Build()` + `MarshalIndent`. `BuildSingleProxy(proxy *models.Proxy) (*CaddyConfig, error)` builds one proxy's routes via `buildProxyRoutes`, which reads `b.aclAssigns[proxy.ID]` + `b.aclGroups` + `b.aclBuilder` → **ACL handlers only appear if `SetACLGroups`/`SetACLAssignments` were called AND the builder was made with the ACL builder option** (`SetACLAssignments` keeps only `Enabled` assignments). Returns `*CaddyConfig` (marshal yourself). `NewBuilder(opts...)`; the ACL-builder option + `WithLogger`/`SetSettings` are how the shared builder is configured (see `initJSONBuilder`).
- `repository`: `proxyRepo.GetByID(id int) (*models.Proxy, error)`; `aclRepo.GetProxyACLAssignments(proxyID int) ([]models.ProxyACLAssignment, error)` (preloads `ACLGroup` + all its relations — derive `SetACLGroups` input from `assignment.ACLGroup`, no extra fetch).
- `routes.go`: `syncService` var (constructed ~83–100); repos `proxyRepo`/`aclRepo`/`settingsRepo` in scope; handler+service injection precedent (`caddyLogsService`/`caddyLogsHandler` ~162–163); the `/api/proxies/{id}/acl` nested route block (~311–317) and the `/api/caddy-logs` block (~319–323) are the registration precedents. **Path param is `{id}`.**
- `utils`: `Success(w, data, msg)`, `BadRequest(w, msg, details)`, `NotFound(w, msg)`, `InternalError(w, msg)`.
- `rbac.yaml`: `caddy_logs:read` group (~73–77).
- UI: `JsonViewer` from `@e412/rnui-react` (`data: Record<string,any>`, `showLineNumbers`, `defaultExpanded`, `collapseOn`, `title`). HTTP overview (`proxies/$proxyId/overview.tsx`) renders `<ProxyAccessCard proxyId={proxy.id} />` (~167) — mount the new card right after. L4 overview Details card (`l4-proxies/$l4ProxyId/overview.tsx` ~165–178). Router leaf pattern + `caddyLogsRoute` (~208–212) + `addChildren` (~266–289); sidebar `navItems` (~74–80, lucide icons imported ~43–56). `api` ky client (`prefixUrl '/api'`, no leading slash); hook pattern `api.get(\`proxies/${id}\`).json<ApiResponse<T>>()`.

---

## Task 1: Backend — generate-only methods + endpoint + RBAC

**Files:**
- Modify: `backend/internal/service/sync_service.go` (extract no-write build; add two generate methods)
- Create: `backend/internal/api/handlers/config_preview_handler.go`
- Modify: `backend/internal/api/routes/routes.go`, `backend/rbac.yaml`
- Test: `backend/internal/service/sync_service_test.go` (or a new `config_preview` test) + handler-adjacent test

**Interfaces produced:**
- `(s *SyncService) GenerateConfigJSON() (json.RawMessage, error)` — full generated config, no write.
- `(s *SyncService) GenerateProxyConfigJSON(proxyID int) (json.RawMessage, error)` — one proxy's generated config; returns a not-found sentinel error when the proxy doesn't exist.
- `handlers.NewConfigPreviewHandler(sync *service.SyncService, logger *zap.Logger)` with `GetFull(w,r)` and `GetForProxy(w,r)`.

- [ ] **Step 1: Extract a no-write build from `performFullSyncJSON`.** Refactor so the gather + setters + `BuildJSON()` (current ~309–410) live in a new method, and `performFullSyncJSON` calls it then does the write/validate/reload (current ~412–474). Add:
```go
// GenerateConfigJSON gathers current state and returns the generated Caddy JSON
// config WITHOUT writing or reloading. It reuses the shared builder, so it locks
// s.mu to avoid racing the periodic sync.
func (s *SyncService) GenerateConfigJSON() (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buildConfigBytes()
}

// buildConfigBytes performs the gather + builder configuration + BuildJSON.
// CALLER MUST HOLD s.mu. (Extracted verbatim from performFullSyncJSON steps 1–6.)
func (s *SyncService) buildConfigBytes() (json.RawMessage, error) {
	// ... the existing gather (proxies, 404 settings, ACL groups + assignments,
	// L4 app) + SetHTTPProxies/SetACLGroups/SetACLAssignments/SetNotFoundSettings
	// (+ SetLayer4App/SetL4TLSHostnames) + BuildJSON() — moved here unchanged ...
}
```
Then change `performFullSyncJSON` to:
```go
func (s *SyncService) performFullSyncJSON() error {
	s.mu.Lock()
	configBytes, err := s.buildConfigBytes()
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("failed to build JSON config: %w", err)
	}
	// ... steps 7–14 unchanged (path, ConfigChanged, backup, write, validate, ReloadJSON) ...
}
```
(If `performFullSyncJSON` already holds `s.mu` anywhere, do not double-lock — verify and adjust; the goal is build-under-lock, write-after-unlock. `configBytes` is `[]byte`; return it as `json.RawMessage`.)

- [ ] **Step 2: Per-proxy generate (fresh builder).** Add a helper that constructs a builder configured identically to the shared one (factor the construction out of `initJSONBuilder` into e.g. `s.newConfiguredBuilder()` returning a fresh `*config.Builder` with the same `SetSettings` + ACL-builder option + logger), then:
```go
// ErrProxyNotFound is returned by GenerateProxyConfigJSON for a missing proxy.
var ErrProxyNotFound = errors.New("proxy not found")

// GenerateProxyConfigJSON returns the generated Caddy config for a single proxy
// (with its ACL handlers), without writing/reloading. Uses a fresh builder, so
// no shared state is mutated and no lock is needed.
func (s *SyncService) GenerateProxyConfigJSON(proxyID int) (json.RawMessage, error) {
	proxy, err := s.proxyRepo.GetByID(proxyID)
	if err != nil { return nil, err }
	if proxy == nil { return nil, ErrProxyNotFound }

	var assignments []models.ProxyACLAssignment
	var groups []models.ACLGroup
	if s.aclRepo != nil {
		assignments, err = s.aclRepo.GetProxyACLAssignments(proxyID)
		if err != nil { return nil, err }
		seen := map[int]bool{}
		for i := range assignments {
			g := assignments[i].ACLGroup // preloaded
			if g.ID != 0 && !seen[g.ID] { seen[g.ID] = true; groups = append(groups, g) }
		}
	}

	b := s.newConfiguredBuilder()
	b.SetACLGroups(groups)
	b.SetACLAssignments(assignments)
	cfg, err := b.BuildSingleProxy(proxy)
	if err != nil { return nil, err }
	return json.MarshalIndent(cfg, "", "  ")
}
```
(Adjust `proxy == nil` to however `GetByID` signals not-found — it may return an error instead; map that to `ErrProxyNotFound`. Confirm `assignment.ACLGroup` field name + that `GetByID`'s id type is `int`.)

- [ ] **Step 3: Handler** — `config_preview_handler.go` (mirror `caddy_logs_handler.go`):
```go
type ConfigPreviewHandler struct {
	sync   *service.SyncService
	logger *zap.Logger
}
func NewConfigPreviewHandler(sync *service.SyncService, logger *zap.Logger) *ConfigPreviewHandler {
	return &ConfigPreviewHandler{sync: sync, logger: logger}
}
func (h *ConfigPreviewHandler) GetFull(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.sync.GenerateConfigJSON()
	if err != nil {
		h.logger.Error("generate config failed", zap.Error(err))
		utils.InternalError(w, "failed to generate Caddy config")
		return
	}
	utils.Success(w, cfg, "Caddy config generated successfully")
}
func (h *ConfigPreviewHandler) GetForProxy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil { utils.BadRequest(w, "invalid proxy id", nil); return }
	cfg, err := h.sync.GenerateProxyConfigJSON(id)
	if errors.Is(err, service.ErrProxyNotFound) { utils.NotFound(w, "proxy not found"); return }
	if err != nil {
		h.logger.Error("generate proxy config failed", zap.Error(err))
		utils.InternalError(w, "failed to generate proxy config")
		return
	}
	utils.Success(w, cfg, "Proxy config generated successfully")
}
```
(`utils.Success` with a `json.RawMessage` emits the config object verbatim under `data`.)

- [ ] **Step 4: RBAC** — add to `rbac.yaml`:
```yaml
  - name: "Caddy Config"
    permissions:
      - key: "caddy_config:read"
        name: "View Caddy Config"
        description: "View the generated Caddy configuration"
```

- [ ] **Step 5: Wire routes** in `routes.go`: instantiate `configPreviewHandler := handlers.NewConfigPreviewHandler(syncService, logger)` (near the caddyLogs wiring), and register:
```go
r.Route("/api/caddy-config", func(r chi.Router) {
	r.With(chimw.RequirePermission(authAdapter, "caddy_config:read", mwConfig)).Get("/", configPreviewHandler.GetFull)
})
r.Route("/api/proxies/{id}/config-preview", func(r chi.Router) {
	r.With(chimw.RequirePermission(authAdapter, "caddy_config:read", mwConfig)).Get("/", configPreviewHandler.GetForProxy)
})
```

- [ ] **Step 6: Tests** — `GenerateConfigJSON()` returns valid JSON that unmarshals to a config with `apps.http` (mirror the existing builder/sync test setup with a couple of proxies); `GenerateProxyConfigJSON(id)` returns a config whose HTTP server routes are that proxy's (host matcher == proxy hostname), and returns `ErrProxyNotFound` for a missing id; if a proxy has an enabled ACL assignment, the per-proxy config includes the ACL handler (assert presence). Use the existing test fixtures/helpers (`createReverseProxy`, etc.). Confirm `performFullSyncJSON` still builds (no behavior change) — the existing sync tests must stay green.

- [ ] **Step 7: Gate** — `gofmt -l` (clean) + `go build ./...` + `go test ./... -short` (all pass; if `make backend-test` needs Docker, use `-short`).

- [ ] **Step 8: Commit** — `feat(api): generated Caddy config preview endpoints (caddy_config:read)`.

---

## Task 2: UI — per-proxy config preview card

**Files:**
- Create: `ui/src/hooks/use-config-preview.ts`, `ui/src/types/caddy-config.ts`, `ui/src/components/proxy/overview/proxy-config-preview-card.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx`

- [ ] **Step 1: Types + hooks.** `types/caddy-config.ts`: `export type CaddyConfig = Record<string, unknown>;`. `hooks/use-config-preview.ts`:
```ts
export function useProxyConfigPreview(proxyId: number) {
  return useQuery({
    queryKey: ['proxy-config-preview', proxyId],
    queryFn: async () => (await api.get(`proxies/${proxyId}/config-preview`).json<ApiResponse<CaddyConfig>>()).data,
    enabled: proxyId > 0,
  });
}
export function useCaddyConfig() {
  return useQuery({
    queryKey: ['caddy-config'],
    queryFn: async () => (await api.get('caddy-config').json<ApiResponse<CaddyConfig>>()).data,
  });
}
```
(Match the project's exact `useQuery`/`ApiResponse` import style from `use-proxies.ts`.)

- [ ] **Step 2: Card** — `proxy-config-preview-card.tsx` → `ProxyConfigPreviewCard({ proxyId }: { proxyId: number })`: a `Card` titled "Generated Caddy config" with a Copy button (copies `JSON.stringify(data, null, 2)` via `navigator.clipboard`); body = `useProxyConfigPreview(proxyId)` → loading `Skeleton`, error → short muted message, data → `<JsonViewer data={data} showLineNumbers defaultExpanded={1} collapseOn="click" />`. (If `data` may be undefined, guard before rendering JsonViewer.)

- [ ] **Step 3: Mount** in `proxies/$proxyId/overview.tsx` immediately after `<ProxyAccessCard proxyId={proxy.id} />`: `<ProxyConfigPreviewCard proxyId={proxy.id} />` (add the import).

- [ ] **Step 4: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` (green).

- [ ] **Step 5: Commit** — `feat(ui): per-proxy generated Caddy config card on the overview`.

---

## Task 3: UI — global config page + nav

**Files:**
- Create: `ui/src/routes/_dashboard/caddy-config.tsx`
- Modify: `ui/src/lib/router.tsx`, `ui/src/components/layout/sidebar.tsx`, `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId/overview.tsx`

- [ ] **Step 1: Page** — `caddy-config.tsx` → `CaddyConfigPage`: `space-y-6` + `<h1 className="text-2xl font-bold">Caddy Config</h1>`; uses `useCaddyConfig()`; a Copy button + a Refresh (`refetch`); body = loading `Skeleton` / error message / `<JsonViewer data={data} showLineNumbers defaultExpanded={1} collapseOn="click" />`.

- [ ] **Step 2: Router** — add in `lib/router.tsx`:
```tsx
const caddyConfigRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/caddy-config',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/caddy-config'), 'CaddyConfigPage'),
});
```
and add `caddyConfigRoute` to `dashboardRoute.addChildren([...])`.

- [ ] **Step 3: Sidebar** — add a `navItems` entry `{ label: 'Caddy Config', path: '/dashboard/caddy-config', icon: <Braces className="size-4" /> }` (import `Braces` — or another unused lucide icon, e.g. `FileCode2` — from `lucide-react`).

- [ ] **Step 4: L4 link** — in the L4 overview, add a small link to `/dashboard/caddy-config` (a `DetailRow` "Generated config" with a `Link` "View full config", or a header action), since L4 has no per-proxy card.

- [ ] **Step 5: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` (green).

- [ ] **Step 6: Commit** — `feat(ui): global generated Caddy config page + nav`.

---

## Self-Review

**Spec coverage:** generated-only ✓ (both methods build, never write); per-proxy (T1 `GenerateProxyConfigJSON` + T2 card) + global (T1 `GenerateConfigJSON` + T3 page) ✓; `caddy_config:read`, admin-only ✓; reuse sync gather (T1 Step 1 extract) ✓; L4 deferred → global link (T3 Step 4) ✓; JsonViewer + copy ✓; standalone page + sidebar ✓; no DB change ✓. Sensitive-data: shown as-is, admin-gated (redaction = noted follow-up, per Global Constraints).

**Placeholder scan:** Go methods/handler/routes/rbac and the TS hooks/card/page are concrete; the few "confirm/adjust" notes (GetByID not-found signaling, `assignment.ACLGroup` field name, the `newConfiguredBuilder` extraction from `initJSONBuilder`) are explicit verification points, not vague placeholders.

**Type/name consistency:** `GenerateConfigJSON`/`GenerateProxyConfigJSON`/`ErrProxyNotFound` used consistently across service↔handler; `caddy_config:read` in rbac + both routes; `CaddyConfigPage`/`ProxyConfigPreviewCard` exports match their imports; `useCaddyConfig`/`useProxyConfigPreview` match the hooks file; route param `{id}` matches `chi.URLParam(r, "id")`; both generate methods return `json.RawMessage` → `utils.Success` emits the config object under `data` → UI `JsonViewer data`.
