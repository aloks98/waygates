# Proxy Import — Design Spec

**Date:** 2026-06-21
**Status:** Approved (brainstorm)
**Builds on:** M2b export (`ui/src/lib/proxy-export.ts`, `ProxyExport`), the proxy create path, and the M2b bulk patterns.

## Overview

Import HTTP proxies from a JSON file (the export format), as the inverse of M2b's export. A **preview → confirm** flow: the user supplies the JSON (file or paste), the backend performs a **dry-run** that validates and detects conflicts without writing, the user reviews the per-item breakdown, and on confirm the backend applies the import — creating the valid, non-conflicting proxies and skipping the rest. A per-item report is shown.

The authoritative validation and hostname-conflict logic lives on the **backend** (it already owns `proxy.Validate()` and the hostname-conflict check used by create), so the same code path produces the preview and the apply — the preview is exactly what apply will do.

## Locked decisions

- **Flow:** Preview → confirm → report.
- **Conflicts:** Skip conflicting items, import the rest (never overwrite an existing proxy). Best-effort, non-transactional (partial success is the intent).
- **Input:** File upload (`.json`) **or** paste into a textarea.
- **Validation/conflict authority:** Backend, via a dry-run mode on a single import endpoint.
- **No backend overwrite/update on import.** No all-or-nothing transaction. Import is bulk *create* only.

## Backend

### Endpoint

`POST /api/proxies/import` — RBAC permission `proxies:create`.

**Request body:**
```json
{
  "dry_run": true,
  "proxies": [ /* array of ProxyExport objects (the export format) */ ]
}
```
A `ProxyExport` item: `{ type, name, hostname, description?, ssl_enabled, upstreams?, load_balancing?, block_exploits?, tls_insecure_skip_verify?, custom_headers?, redirect?, static? }` — i.e. the create-request shape (top-level `upstreams`; nested `redirect`/`static`).

**Response** (`utils.Success` envelope, `data`):
```json
{
  "summary": {
    "total": 17,
    "importable": 12,   // dry-run: would import; apply: same pre-apply count
    "conflicts": 3,
    "invalid": 2,
    "created": 0,        // apply only (0 on dry-run)
    "failed": 0          // apply only (0 on dry-run)
  },
  "items": [
    { "index": 0, "name": "api",  "hostname": "api.example.com", "type": "reverse_proxy", "status": "valid" },
    { "index": 1, "name": "old",  "hostname": "old.example.com", "type": "redirect",      "status": "conflict", "reason": "hostname already exists" },
    { "index": 2, "name": "",     "hostname": "x.example.com",   "type": "static",        "status": "invalid",  "reason": "name is required" }
  ]
}
```

**Per-item `status`:**
- **dry-run:** `valid` | `conflict` | `invalid` (with `reason`).
- **apply:** `created` | `skipped_conflict` | `invalid` | `failed` (with `reason` for the non-created).

### Service behavior — `ImportProxies(items, dryRun, userID) → report`

For each item (preserving input order / `index`):
1. **Map** the item to a `models.Proxy` reusing the **same request→model mapping the create handler uses** (the import item is a create request). A map/deserialize failure → `invalid` (reason from the error).
2. **Validate** via `proxy.Validate()` (the model-level validation create already runs). Failure → `invalid` (reason = validation message).
3. **Conflict check** — hostname already exists, using the same check create uses (`ErrHostnameConflict`/repo hostname lookup). Also detect **intra-file duplicates**: track hostnames seen earlier in this batch; a repeat → `conflict` (reason "duplicate hostname in import"). Existing-in-DB → `conflict` (reason "hostname already exists").
4. **Classify / apply:**
   - `dryRun = true`: record `valid` / `conflict` / `invalid`. **No writes.**
   - `dryRun = false`: for a `valid` item, call the existing `CreateProxy` (which re-validates + re-checks conflict + writes the Caddy file). Success → `created`; a create error that is a hostname conflict → `skipped_conflict`; any other create error → `failed` (reason). `conflict`/`invalid` items are not created and are reported as `skipped_conflict`/`invalid`.
5. Build the `summary` counts from the per-item statuses.

Best-effort: one item's failure never aborts the rest. Non-transactional — the report tells the user exactly what happened. (The per-item `CreateProxy` already triggers its own Caddy file write/reload as today; no special bulk reload handling in scope.)

### Wiring & quality
- Handler `ImportProxies` in `handlers/proxy.go` (parse body, call service, return report; cap the array length to a sane max — e.g. reject > 1000 items with `400`).
- Register the route under the proxies group with `proxies:create`.
- Update `docs/` for the new endpoint.
- Errors wrapped with `%w`; structured zap logging (count, dryRun).

### Tests
- **Unit** (mock repo): mixed batch → correct classification; dry-run writes nothing (repo create never called); apply creates only the valid non-conflicting items; intra-file duplicate flagged; invalid item reason surfaced.
- **Integration** (testcontainers, real PostgreSQL): seed an existing proxy, import a batch overlapping it → dry-run reports the conflict; apply creates the new ones, leaves the existing untouched, and the DB count matches.

## Frontend (thin — validation/conflicts come from the backend)

### `lib/proxy-import.ts` (+ test)
- `parseImportJson(text: string): { ok: true; items: unknown[] } | { ok: false; error: string }` — pure: valid JSON, must be a non-empty array; returns a clear error otherwise. (Per-item validation is the backend's job.)
- TypeScript types for the report (`ImportItemResult`, `ImportSummary`, `ImportReport`) matching the endpoint.

### `hooks/use-proxy-import.ts`
- `previewImport(items): Promise<ImportReport>` → `POST /api/proxies/import` with `dry_run: true`.
- `applyImport(items): Promise<ImportReport>` → same endpoint, `dry_run: false`; invalidates `['proxies']` after.
- Loading flags `isPreviewing` / `isApplying`.

### `components/proxy/proxy-import-dialog.tsx`
rnui `Dialog` with three stages in one surface:
1. **Input:** a `.json` file picker **and** a "paste JSON" textarea. On file-select or paste → `parseImportJson`; on parse error, show it inline. On success → `previewImport`.
2. **Preview:** the `summary` line ("12 importable · 3 conflicts · 2 invalid") + a scrollable per-row list, each row tagged by status (Import / Skip — hostname exists / Invalid — reason). Confirm button "Import N proxies" (N = importable), disabled when N=0. A back/re-pick affordance.
3. **Report:** after apply — "Created K · Skipped S · Failed M" + per-row final statuses; Close.

### `routes/_dashboard/proxies/index.tsx`
An **Import** button beside Export in the header → opens the dialog. On dialog close after a successful apply, the list is already invalidated/refetched.

### Tests
Unit tests for `parseImportJson` (valid array, non-JSON, non-array, empty). The dialog is thin UI over the tested endpoint + parser; no heavy component tests.

## Data flow
```
file / paste
  → parseImportJson (client: valid JSON + non-empty array)
  → previewImport(items)  → POST /api/proxies/import {dry_run:true}  → report (valid/conflict/invalid)
  → user reviews preview
  → applyImport(items)    → POST /api/proxies/import {dry_run:false} → report (created/skipped/failed)
  → invalidate ['proxies'] → list refreshes
```

## Out of scope
- Overwriting / updating existing proxies on import (skip-only).
- All-or-nothing transactional import / rollback.
- Importing ACL assignments, TCP/UDP (L4) proxies, or other entities (HTTP proxies only — mirrors export).
- A separate bulk Caddy reload optimization (per-item create reload is acceptable at import sizes).

## Testing summary
Backend: service unit tests (classification, dry-run-no-writes, apply-creates, intra-file dupes) + one integration test (real PostgreSQL). Frontend: `parseImportJson` unit tests. Gate: backend `make format/lint/test`; UI `build + check + test`.
