# Proxy Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Import HTTP proxies from a JSON file (the export format) via a preview → confirm flow, backed by a single backend endpoint with a dry-run mode.

**Architecture:** A new `POST /api/proxies/import` endpoint runs the same per-item `Validate()` + hostname-conflict checks the create path uses. `dry_run: true` classifies without writing (the preview); `dry_run: false` creates the valid, non-conflicting items (skips the rest) — both produce a per-item report from one service method (`ImportProxies`). The frontend parses JSON client-side (valid-array only), calls the endpoint for preview then apply, and renders the report in a dialog.

**Tech Stack:** Go 1.25 (chi, GORM/PostgreSQL, zap, testify, testcontainers), React 19 + TanStack Query, rnui (`@e412/rnui-react`, Base UI — `render` not `asChild`), ky, Vitest.

## Global Constraints

- **Skip-conflicts, best-effort, non-transactional.** One item's failure never aborts the rest. No overwrite/update of existing proxies. Import is bulk *create* only.
- **Backend is the validation/conflict authority.** Dry-run = exactly what apply will do (same code path). The frontend only checks "valid JSON + non-empty array".
- **JSON field naming is snake_case**: `dry_run`, `proxies`, `acl_group_count`-style. Per-item report fields: `index`, `name`, `hostname`, `type`, `status`, `reason`; summary: `total`, `importable`, `conflicts`, `invalid`, `created`, `failed`.
- **Per-item `status`:** dry-run → `valid` | `conflict` | `invalid`; apply → `created` | `skipped_conflict` | `invalid` | `failed`.
- **RBAC:** the endpoint requires `proxies:create`.
- **Array cap:** reject `> 1000` items with `400`.
- **Go conventions:** wrap errors `fmt.Errorf("...: %w", err)`; structured zap logging; repository pattern; tests with testify, integration tests with testcontainers + a `testing.Short()` skip guard matching siblings.
- **No `tsc` gate** for UI. Gate UI tasks on `pnpm --dir ui build` + `pnpm --dir ui check` + `pnpm --dir ui test run`. Gate backend tasks on `make format-backend` + `make lint-backend` + `make backend-test` (or `cd backend && go test ./... -short` if Docker is unavailable — note which ran).
- **rnui composes via `render`, never `asChild`.** Run from repo root `/home/aloks98/projects/waygates`.

---

## File Structure

**Backend — create:**
- `backend/internal/service/proxy_import.go` — `ImportProxies` + the import types/statuses (keeps `proxy_service.go` focused).
- `backend/internal/service/proxy_import_test.go` — unit tests (mock repo/syncer) + a service integration test (real DB + no-op syncer).

**Backend — modify:**
- `backend/internal/api/handlers/proxy.go` — `ImportProxies` handler + `importProxyRequest` type.
- `backend/internal/api/routes/routes.go` — register the route under the proxies group.
- The proxy API doc under `docs/` — document the new endpoint.
- The handler's `ProxyService` interface + its mock (if the handler depends on an interface — see Task 2 Step 0).

**Frontend — create:**
- `ui/src/lib/proxy-import.ts` (+ `.test.ts`) — `parseImportJson` + report types.
- `ui/src/hooks/use-proxy-import.ts` — `previewImport` / `applyImport`.
- `ui/src/components/proxy/proxy-import-dialog.tsx` — the dialog (input → preview → report).

**Frontend — modify:**
- `ui/src/routes/_dashboard/proxies/index.tsx` — Import button → open dialog.

---

## Task 1: Backend — `ImportProxies` service + types + tests

**Files:**
- Create: `backend/internal/service/proxy_import.go`
- Test: `backend/internal/service/proxy_import_test.go`

**Interfaces:**
- Consumes: `ProxyService.repo` (`HostnameExists(hostname string, excludeID int) (bool, error)`), `ProxyService.CreateProxy(proxy *models.Proxy, userID int) error`, `ErrHostnameConflict`.
- Produces: types `ImportInput`, `ImportItemResult`, `ImportSummary`, `ImportReport`; status consts; method `(s *ProxyService) ImportProxies(inputs []ImportInput, dryRun bool, userID int) ImportReport`.

- [ ] **Step 1: Write the failing unit tests**

Create `backend/internal/service/proxy_import_test.go`. Construct the service with `MockProxyRepository` + `MockProxySyncer` (both already defined in `proxy_service_test.go`, same package). Use a minimal valid reverse-proxy model in helpers (mirror how `proxy_service_test.go` builds a valid `models.Proxy` — read it for the exact required fields so `Validate()` passes).

```go
func importableProxy(name, hostname string) *models.Proxy {
	// Build a models.Proxy that passes Validate() — mirror the valid fixture
	// used elsewhere in proxy_service_test.go (type reverse_proxy + >=1 upstream).
	p := validReverseProxyFixture() // helper that returns a *models.Proxy passing Validate()
	p.Name = name
	p.Hostname = hostname
	return p
}

func TestImportProxies_DryRun_ClassifiesAndWritesNothing(t *testing.T) {
	created := 0
	repo := &MockProxyRepository{
		HostnameExistsFunc: func(hostname string, _ int) (bool, error) {
			return hostname == "taken.example.com", nil
		},
		CreateFunc: func(_ *models.Proxy) error { created++; return nil },
	}
	syncer := &MockProxySyncer{}
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: syncer})

	inputs := []ImportInput{
		{Proxy: importableProxy("ok", "fresh.example.com")},   // valid
		{Proxy: importableProxy("dupe", "taken.example.com")}, // conflict (DB)
		{Proxy: importableProxy("dup2", "fresh.example.com")}, // conflict (intra-batch)
		{Proxy: &models.Proxy{Type: "reverse_proxy"}},          // invalid (Validate fails)
		{DecodeError: "invalid item format: bad port"},         // invalid (decode)
	}
	report := svc.ImportProxies(inputs, true, 7)

	assert.Equal(t, 0, created, "dry-run must not create")
	assert.Equal(t, 5, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Importable)
	assert.Equal(t, 2, report.Summary.Conflicts)
	assert.Equal(t, 2, report.Summary.Invalid)
	assert.Equal(t, "valid", report.Items[0].Status)
	assert.Equal(t, "conflict", report.Items[1].Status)
	assert.Equal(t, "conflict", report.Items[2].Status)
	assert.Equal(t, "invalid", report.Items[3].Status)
	assert.Equal(t, "invalid", report.Items[4].Status)
	assert.NotEmpty(t, report.Items[4].Reason)
}

func TestImportProxies_Apply_CreatesValidSkipsConflicts(t *testing.T) {
	createdHosts := []string{}
	existing := map[string]bool{"taken.example.com": true}
	repo := &MockProxyRepository{
		HostnameExistsFunc: func(hostname string, _ int) (bool, error) { return existing[hostname], nil },
		CreateFunc: func(p *models.Proxy) error { createdHosts = append(createdHosts, p.Hostname); existing[p.Hostname] = true; return nil },
	}
	syncer := &MockProxySyncer{} // SyncProxy no-op (nil func)
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: syncer})

	inputs := []ImportInput{
		{Proxy: importableProxy("a", "a.example.com")},
		{Proxy: importableProxy("b", "taken.example.com")}, // conflict → skipped
		{Proxy: importableProxy("c", "c.example.com")},
	}
	report := svc.ImportProxies(inputs, false, 7)

	assert.Equal(t, []string{"a.example.com", "c.example.com"}, createdHosts)
	assert.Equal(t, 2, report.Summary.Created)
	assert.Equal(t, 1, report.Summary.Conflicts)
	assert.Equal(t, 0, report.Summary.Failed)
	assert.Equal(t, "created", report.Items[0].Status)
	assert.Equal(t, "skipped_conflict", report.Items[1].Status)
	assert.Equal(t, "created", report.Items[2].Status)
}
```

If `proxy_service_test.go` has no reusable valid fixture, write `validReverseProxyFixture()` in this test file using the fields `Validate()` requires (read `models.Proxy.Validate()` to get them exactly).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend && go test ./internal/service/ -run TestImportProxies`
Expected: FAIL — `ImportInput`/`ImportProxies` undefined.

- [ ] **Step 3: Implement `proxy_import.go`**

```go
package service

import (
	"errors"
	"fmt"

	"github.com/<module>/internal/models" // match the module path used in sibling files
)

// Import per-item statuses.
const (
	ImportStatusValid           = "valid"
	ImportStatusConflict        = "conflict"
	ImportStatusInvalid         = "invalid"
	ImportStatusCreated         = "created"
	ImportStatusSkippedConflict = "skipped_conflict"
	ImportStatusFailed          = "failed"
)

// ImportInput is one decoded item. Proxy is nil when the raw item could not be
// decoded, in which case DecodeError explains why.
type ImportInput struct {
	Proxy       *models.Proxy
	DecodeError string
}

type ImportItemResult struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
}

type ImportSummary struct {
	Total      int `json:"total"`
	Importable int `json:"importable"`
	Conflicts  int `json:"conflicts"`
	Invalid    int `json:"invalid"`
	Created    int `json:"created"`
	Failed     int `json:"failed"`
}

type ImportReport struct {
	Summary ImportSummary      `json:"summary"`
	Items   []ImportItemResult `json:"items"`
}

// ImportProxies validates and (when dryRun is false) creates the given proxies,
// skipping conflicts and invalid items. Best-effort: one failure never aborts
// the rest. Dry-run performs the identical validation/conflict checks but writes
// nothing, so it is an exact preview of apply.
func (s *ProxyService) ImportProxies(inputs []ImportInput, dryRun bool, userID int) ImportReport {
	report := ImportReport{Items: make([]ImportItemResult, 0, len(inputs))}
	seen := make(map[string]bool) // hostnames already accepted (valid/created) in this batch

	conflictStatus := ImportStatusConflict
	if !dryRun {
		conflictStatus = ImportStatusSkippedConflict
	}

	for i, in := range inputs {
		res := ImportItemResult{Index: i}

		if in.Proxy == nil {
			res.Status = ImportStatusInvalid
			res.Reason = in.DecodeError
			report.Items = append(report.Items, res)
			continue
		}

		proxy := in.Proxy
		res.Name = proxy.Name
		res.Hostname = proxy.Hostname
		res.Type = string(proxy.Type)

		if err := proxy.Validate(); err != nil {
			res.Status = ImportStatusInvalid
			res.Reason = err.Error()
			report.Items = append(report.Items, res)
			continue
		}

		if seen[proxy.Hostname] {
			res.Status = conflictStatus
			res.Reason = "duplicate hostname in import"
			report.Items = append(report.Items, res)
			continue
		}
		exists, err := s.repo.HostnameExists(proxy.Hostname, 0)
		if err != nil {
			res.Status = ImportStatusFailed
			res.Reason = fmt.Sprintf("failed to check hostname: %v", err)
			report.Items = append(report.Items, res)
			continue
		}
		if exists {
			res.Status = conflictStatus
			res.Reason = "hostname already exists"
			report.Items = append(report.Items, res)
			continue
		}

		if dryRun {
			res.Status = ImportStatusValid
			seen[proxy.Hostname] = true
			report.Items = append(report.Items, res)
			continue
		}

		if err := s.CreateProxy(proxy, userID); err != nil {
			if errors.Is(err, ErrHostnameConflict) {
				res.Status = ImportStatusSkippedConflict
				res.Reason = "hostname already exists"
			} else {
				res.Status = ImportStatusFailed
				res.Reason = err.Error()
			}
			report.Items = append(report.Items, res)
			continue
		}
		res.Status = ImportStatusCreated
		seen[proxy.Hostname] = true
		report.Items = append(report.Items, res)
	}

	report.Summary = summarizeImport(report.Items, len(inputs))
	return report
}

func summarizeImport(items []ImportItemResult, total int) ImportSummary {
	s := ImportSummary{Total: total}
	for _, it := range items {
		switch it.Status {
		case ImportStatusValid:
			s.Importable++
		case ImportStatusCreated:
			s.Importable++
			s.Created++
		case ImportStatusFailed:
			s.Importable++
			s.Failed++
		case ImportStatusConflict, ImportStatusSkippedConflict:
			s.Conflicts++
		case ImportStatusInvalid:
			s.Invalid++
		}
	}
	return s
}
```

Fix the `models` import path to match sibling service files. (`Importable` counts items that passed validation+conflict — `valid` on dry-run, `created`+`failed` on apply.)

- [ ] **Step 4: Run the unit tests to verify they pass**

Run: `cd backend && go test ./internal/service/ -run TestImportProxies`
Expected: PASS.

- [ ] **Step 5: Add a service integration test (real DB + no-op syncer)**

In `proxy_import_test.go`, add an integration test mirroring the existing service/repository integration-test setup (testcontainers PostgreSQL; same `SetupTestDB`/skip-guard the M2b repo integration test used). Construct a `ProxyService` with the **real** `repository.NewProxyRepository(db)` and a `&MockProxySyncer{}` (no-op `SyncProxy`). Seed one existing proxy (hostname `existing.example.com`). Then:
- `ImportProxies(dryRun=true)` on `[existing.example.com, new1.example.com]` → reports 1 conflict + 1 importable, and the DB still has exactly 1 proxy.
- `ImportProxies(dryRun=false)` on the same → creates `new1.example.com` only; DB now has 2; the existing proxy is unchanged.
Name it `TestImportProxies_Integration`. Include the `testing.Short()` skip guard.

- [ ] **Step 6: Run the backend gate**

Run: `make format-backend && make lint-backend && make backend-test`
(If Docker is unavailable: `cd backend && go test ./... -short` and note the integration test skipped.)
Expected: clean; unit + integration tests pass (or integration skipped under `-short`).

- [ ] **Step 7: Commit**

```bash
git add backend/internal/service/proxy_import.go backend/internal/service/proxy_import_test.go
git commit -m "feat(api): add ImportProxies service (dry-run + apply) (import)"
```

---

## Task 2: Backend — import handler, route, docs

**Files:**
- Modify: `backend/internal/api/handlers/proxy.go`
- Modify: `backend/internal/api/routes/routes.go`
- Modify: the proxy API doc under `docs/`
- Modify (if applicable): the handler's `ProxyService` interface + its mock

**Interfaces:**
- Consumes: `service.ImportInput`, `service.ImportReport`, `(*ProxyService).ImportProxies`; the existing `createProxyRequest` type + its ssl/block defaulting; `utils.Success`/`utils.BadRequest`/`utils.Unauthorized`.
- Produces: `ImportProxies(w, r)` handler; route `POST /api/proxies/import` (`proxies:create`).

- [ ] **Step 0: Check the handler↔service seam**

Read `proxy.go` to see how `ProxyHandler` holds its service (concrete `*service.ProxyService` vs an interface). If it is an **interface** (the handler integration tests mock the service), add `ImportProxies(inputs []service.ImportInput, dryRun bool, userID int) service.ImportReport` to that interface and to the mock used in `proxy_handler_integration_test.go`. If it is the concrete type, no interface change is needed.

- [ ] **Step 1: Add the handler**

In `proxy.go`, add near `createProxyRequest`:

```go
type importProxyRequest struct {
	DryRun  bool              `json:"dry_run"`
	Proxies []json.RawMessage `json:"proxies"`
}

const maxImportProxies = 1000

// ImportProxies handles POST /api/proxies/import
func (h *ProxyHandler) ImportProxies(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.Unauthorized(w, "Invalid user ID")
		return
	}

	var req importProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}
	if len(req.Proxies) == 0 {
		utils.BadRequest(w, "No proxies to import", nil)
		return
	}
	if len(req.Proxies) > maxImportProxies {
		utils.BadRequest(w, fmt.Sprintf("Too many proxies to import (max %d)", maxImportProxies), nil)
		return
	}

	// Decode each item the same way create does, capturing per-item decode errors.
	inputs := make([]service.ImportInput, 0, len(req.Proxies))
	for _, raw := range req.Proxies {
		var cpr createProxyRequest
		if err := json.Unmarshal(raw, &cpr); err != nil {
			inputs = append(inputs, service.ImportInput{DecodeError: "invalid item format: " + err.Error()})
			continue
		}
		proxy := cpr.Proxy
		if cpr.SSLEnabled != nil {
			proxy.SSLEnabled = *cpr.SSLEnabled
		} else {
			proxy.SSLEnabled = true
		}
		if cpr.BlockExploits != nil {
			proxy.BlockExploits = *cpr.BlockExploits
		} else {
			proxy.BlockExploits = true
		}
		proxy.SSLForced = true
		proxy.IsActive = true
		inputs = append(inputs, service.ImportInput{Proxy: &proxy})
	}

	report := h.service.ImportProxies(inputs, req.DryRun, userID)

	if h.logger != nil {
		h.logger.Info("proxy import processed",
			zap.Bool("dry_run", req.DryRun),
			zap.Int("total", report.Summary.Total),
			zap.Int("created", report.Summary.Created),
			zap.Int("user_id", userID),
		)
	}

	utils.Success(w, report, "Import processed")
}
```

Match the existing imports already in `proxy.go` (`json`, `fmt`, `strconv`, `zap`, `chimw`, `service`, `utils`).

- [ ] **Step 2: Register the route**

In `routes.go`, inside the `r.Route("/api/proxies", ...)` block, after the create POST line, add:

```go
r.With(chimw.RequirePermission(authAdapter, "proxies:create", mwConfig)).Post("/import", proxyHandler.ImportProxies)
```

- [ ] **Step 3: Add a handler test**

Add a test (in the handler test file, mirroring its existing style — mock service if the seam is an interface; otherwise drive a real service with a mock repo). Assert: a body with `dry_run:true` + two items (one valid, one malformed) returns 200 with a report whose malformed item is `invalid`; an empty `proxies` array returns 400; `> 1000` items returns 400.

- [ ] **Step 4: Document the endpoint**

Find the proxy API documentation under `docs/` (where `POST /api/proxies` is documented) and add `POST /api/proxies/import` — the request shape (`dry_run`, `proxies`), the response (`summary` + `items` with the status enum for both modes), `proxies:create` permission, and the 1000-item cap.

- [ ] **Step 5: Run the backend gate**

Run: `make format-backend && make lint-backend && make backend-test`
Expected: clean; handler test passes.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/api/handlers/proxy.go backend/internal/api/routes/routes.go docs/
git commit -m "feat(api): POST /api/proxies/import endpoint (dry-run + apply) (import)"
```

---

## Task 3: Frontend — `parseImportJson` + report types

**Files:**
- Create: `ui/src/lib/proxy-import.ts`
- Test: `ui/src/lib/proxy-import.test.ts`

**Interfaces:**
- Produces: `parseImportJson(text)` → `{ ok: true; items: unknown[] } | { ok: false; error: string }`; types `ImportItemStatus`, `ImportItemResult`, `ImportSummary`, `ImportReport`.

- [ ] **Step 1: Write the failing test**

`ui/src/lib/proxy-import.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { parseImportJson } from './proxy-import';

describe('parseImportJson', () => {
  it('accepts a non-empty array', () => {
    const r = parseImportJson('[{"type":"redirect","name":"x","hostname":"x.test"}]');
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.items).toHaveLength(1);
  });
  it('rejects non-JSON', () => {
    const r = parseImportJson('not json');
    expect(r).toEqual({ ok: false, error: 'Not valid JSON.' });
  });
  it('rejects a non-array (object)', () => {
    const r = parseImportJson('{"type":"redirect"}');
    expect(r).toEqual({ ok: false, error: 'Expected a JSON array of proxies.' });
  });
  it('rejects an empty array', () => {
    const r = parseImportJson('[]');
    expect(r).toEqual({ ok: false, error: 'The file contains no proxies.' });
  });
});
```

- [ ] **Step 2: Run to verify it fails** — `pnpm --dir ui test run proxy-import` → FAIL (module missing).

- [ ] **Step 3: Implement** `ui/src/lib/proxy-import.ts`:

```ts
export type ImportItemStatus =
  | 'valid'
  | 'conflict'
  | 'invalid'
  | 'created'
  | 'skipped_conflict'
  | 'failed';

export interface ImportItemResult {
  index: number;
  name: string;
  hostname: string;
  type: string;
  status: ImportItemStatus;
  reason?: string;
}

export interface ImportSummary {
  total: number;
  importable: number;
  conflicts: number;
  invalid: number;
  created: number;
  failed: number;
}

export interface ImportReport {
  summary: ImportSummary;
  items: ImportItemResult[];
}

export function parseImportJson(
  text: string,
): { ok: true; items: unknown[] } | { ok: false; error: string } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { ok: false, error: 'Not valid JSON.' };
  }
  if (!Array.isArray(parsed)) {
    return { ok: false, error: 'Expected a JSON array of proxies.' };
  }
  if (parsed.length === 0) {
    return { ok: false, error: 'The file contains no proxies.' };
  }
  return { ok: true, items: parsed };
}
```

- [ ] **Step 4: Run to verify it passes** — `pnpm --dir ui test run proxy-import` → PASS.

- [ ] **Step 5: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run` (clean; +4 tests).

- [ ] **Step 6: Commit**

```bash
git add ui/src/lib/proxy-import.ts ui/src/lib/proxy-import.test.ts
git commit -m "feat(ui): proxy import JSON parser + report types (import)"
```

---

## Task 4: Frontend — import hook, dialog, and Import button

**Files:**
- Create: `ui/src/hooks/use-proxy-import.ts`
- Create: `ui/src/components/proxy/proxy-import-dialog.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx`

**Interfaces:**
- Consumes: `parseImportJson`, `ImportReport` (Task 3); the `api` client + the proxies query key (from `use-proxies.ts`).
- Produces: `useProxyImport()` → `{ previewImport(items), applyImport(items), isPreviewing, isApplying }`; `ProxyImportDialog({ open, onOpenChange })`.

- [ ] **Step 1: Implement the hook** `ui/src/hooks/use-proxy-import.ts`

Match the existing `use-proxies.ts` patterns for the `api` import, the response-envelope unwrapping (`ApiResponse<T>`), and the proxies query key used by `invalidateQueries`:

```tsx
import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';

import { api } from '@/lib/api';
import type { ImportReport } from '@/lib/proxy-import';
import type { ApiResponse } from '@/types/api'; // match the type used in use-proxies.ts

export function useProxyImport() {
  const queryClient = useQueryClient();
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [isApplying, setIsApplying] = useState(false);

  const call = (items: unknown[], dryRun: boolean) =>
    api
      .post('proxies/import', { json: { dry_run: dryRun, proxies: items } })
      .json<ApiResponse<ImportReport>>()
      .then((r) => r.data);

  const previewImport = async (items: unknown[]): Promise<ImportReport> => {
    setIsPreviewing(true);
    try {
      return await call(items, true);
    } finally {
      setIsPreviewing(false);
    }
  };

  const applyImport = async (items: unknown[]): Promise<ImportReport> => {
    setIsApplying(true);
    try {
      const report = await call(items, false);
      await queryClient.invalidateQueries({ queryKey: ['proxies'] }); // match use-proxies.ts key
      return report;
    } finally {
      setIsApplying(false);
    }
  };

  return { previewImport, applyImport, isPreviewing, isApplying };
}
```

Verify the exact `ApiResponse<T>` shape and `.data` field against `use-proxies.ts`; verify the proxies query key matches the one used there.

- [ ] **Step 2: Build the dialog** `ui/src/components/proxy/proxy-import-dialog.tsx`

An rnui `Dialog` (read an existing dialog usage in the repo for the exact rnui Dialog import/parts and the `render`-prop conventions). Local stage state `'input' | 'preview' | 'report'`:

- **input:** a hidden `<input type="file" accept="application/json,.json">` behind a Button, and a `Textarea` for pasting. A "Preview" action reads the file (`await file.text()`) or the textarea value → `parseImportJson`. On `!ok`, show the error inline. On `ok`, call `previewImport(items)`, stash both `items` and the returned `report`, go to **preview**.
- **preview:** render `report.summary` ("{importable} importable · {conflicts} conflicts · {invalid} invalid") + a scrollable list of `report.items` (name, hostname, a status chip: `valid`→Import, `conflict`→Skip (reason), `invalid`→Invalid (reason)). A primary "Import {summary.importable} proxies" button (disabled when `importable === 0` or `isApplying`) → `applyImport(items)` → stash the apply report → go to **report**. A "Back" button returns to input.
- **report:** render the apply summary ("Created {created} · Skipped {conflicts} · Failed {failed}") + the per-item final statuses; a "Done"/Close button that calls `onOpenChange(false)`.

Reset stage + state to `'input'` when the dialog opens (so reopening is clean). Use `sonner` `toast` for a one-line apply summary if desired. rnui composes via `render`, never `asChild`.

- [ ] **Step 3: Wire the Import button in `index.tsx`**

Add an "Import" button beside the existing "Export" button in the header. `const [importOpen, setImportOpen] = useState(false);` → button sets it true → render `<ProxyImportDialog open={importOpen} onOpenChange={setImportOpen} />`. Gate the button on the create permission (`canCreateProxies`) since import creates proxies. After a successful apply, the hook already invalidated `['proxies']`, so the list refreshes when the dialog closes.

- [ ] **Step 4: Gate**

Run: `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run`
Expected: clean; tests still pass.

- [ ] **Step 5: Manual verification**

`pnpm --dir ui dev` (8008): export a few proxies, then Import that file → preview shows all as importable; edit the file to duplicate an existing hostname and add a malformed item → preview shows a conflict + an invalid; confirm → report shows created/skipped/failed and the list refreshes; paste-path works too. Document results in the report.

- [ ] **Step 6: Commit**

```bash
git add ui/src/hooks/use-proxy-import.ts ui/src/components/proxy/proxy-import-dialog.tsx ui/src/routes/_dashboard/proxies/index.tsx
git commit -m "feat(ui): proxy import dialog (preview + apply) (import)"
```

---

## Self-Review

1. **Spec coverage:** endpoint with dry-run/apply ✓ (Task 1 service + Task 2 handler/route); same Validate()+conflict path ✓ (Task 1 uses `Validate()` + `HostnameExists` + `CreateProxy`); intra-file duplicate detection ✓ (Task 1 `seen` map); per-item status enums ✓ (Task 1 consts, Task 3 TS type); best-effort skip-conflicts ✓; RBAC `proxies:create` ✓ (Task 2 route); 1000 cap ✓ (Task 2); docs ✓ (Task 2 Step 4); frontend parse (valid-array only) ✓ (Task 3); preview→apply hook ✓ (Task 4 Step 1); 3-stage dialog ✓ (Task 4 Step 2); Import button ✓ (Task 4 Step 3); tests — backend unit+integration (Task 1), handler (Task 2), parser units (Task 3) ✓.
2. **Placeholder scan:** the logic-bearing code (service method, summarizer, handler decode, parser, hook) is shown in full; the dialog is structural over tested logic with the rnui parts deferred to an existing-usage read (UI chrome, not logic). Module-path/`ApiResponse`/query-key/Dialog-parts are flagged as "match the existing file" rather than guessed.
3. **Type/name consistency:** status strings identical between Go consts (`valid`/`conflict`/`invalid`/`created`/`skipped_conflict`/`failed`) and the TS `ImportItemStatus`; summary field names (`total`/`importable`/`conflicts`/`invalid`/`created`/`failed`) identical across the Go `ImportSummary` json tags and the TS `ImportSummary`; `ImportInput`/`ImportReport`/`ImportProxies` names match between Task 1 (definition) and Task 2 (consumer).

## Execution notes

- Backend (Tasks 1–2) first — the frontend calls the endpoint. Tasks 3–4 are frontend.
- The whole feature is HTTP-proxies-only and create-only (no overwrite, no transaction, no ACL/L4) per the spec's out-of-scope.
