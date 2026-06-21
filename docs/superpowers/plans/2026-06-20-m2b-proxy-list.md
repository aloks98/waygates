# M2b — HTTP Proxy List Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add list-side power features to the HTTP proxies page — a per-proxy **ACL summary column** (backed by an efficient list-query augmentation), **row selection + bulk enable/disable/delete**, **duplicate a proxy** (seed the create wizard), and **JSON export**.

**Architecture:** One small backend slice augments `GET /api/proxies` so each proxy carries `acl_group_count` + `acl_group_names` (populated with a single extra query over the page's proxy ids — no N+1, no schema change). The rest is frontend: TanStack row-selection wired into the existing `ProxyDataGrid`, a contextual bulk-action bar driving client-side loops over the existing single-item mutations, a duplicate action that seeds the M2a create wizard from an existing proxy, and a client-side JSON export. Import is explicitly deferred.

**Tech Stack:** Go 1.25 (chi, GORM/PostgreSQL, zap, testify, testcontainers), React 19 + TanStack Router/Query/Table, rnui (`@e412/rnui-react`, Base UI — `render` not `asChild`), ky, Vitest + RTL.

## Global Constraints

- **No backend bulk/export endpoints** — bulk enable/disable/delete and export are **client-side loops/transforms** over the existing single-item endpoints (`POST /api/proxies/{id}/enable|disable`, `DELETE /api/proxies/{id}`, `GET /api/proxies`). The only backend change in M2b is the ACL summary on the list response.
- **No DB migration** — the ACL summary fields are computed (`gorm:"-"`), not persisted.
- **JSON field naming is snake_case** (matches existing proxy responses): `acl_group_count`, `acl_group_names`.
- **Go conventions:** wrap errors with `fmt.Errorf("...: %w", err)`; structured zap logging; repository pattern (interface in `repository/interfaces.go`, impl in `repository/proxy_repository.go`); tests with testify, integration tests with testcontainers and a `testing.Short()` skip guard if the existing repo tests use one.
- **rnui composes via `render`, never `asChild`.** rnui `DataGrid` has **no built-in selection** — wire TanStack `enableRowSelection` + `getRowId` + a manual checkbox column using rnui `Checkbox`.
- **No `tsc` gate** for UI. Gate every UI task on `pnpm --dir ui build` + `pnpm --dir ui check` + `pnpm --dir ui test run`. Gate the backend task on `make format-backend` + `make lint-backend` + `make backend-test` (or `go test ./... -short` if testcontainers is unavailable in the dev env — note which was run in the report).
- Run from repo root `/home/aloks98/projects/waygates`.

---

## File Structure

**Backend — modify:**
- `backend/internal/models/proxy.go` — add `ACLGroupCount`/`ACLGroupNames` computed fields to `Proxy`.
- `backend/internal/repository/proxy_repository.go` — after `List` fetches the page, populate the ACL summary via one extra query; extract a pure mapping helper.
- `backend/internal/repository/proxy_repository_test.go` (create if absent, else extend) — test the helper (unit) + the query (integration).

**Frontend — modify:**
- `ui/src/types/proxy.ts` — add `acl_group_count?`, `acl_group_names?` to `ProxyConfig`.
- `ui/src/components/proxy/cells/proxy-acl-cell.tsx` — **create**: the ACL summary cell.
- `ui/src/components/proxy/cells/index.ts` — export the new cell.
- `ui/src/components/proxy/proxy-data-grid.tsx` — ACL column, selection (checkbox column, `getRowId`, `enableRowSelection`, controlled `rowSelection`), duplicate action plumbing.
- `ui/src/components/proxy/cells/proxy-actions-cell.tsx` — add a Duplicate action.
- `ui/src/components/proxy/proxy-bulk-bar.tsx` — **create**: the contextual bulk-action bar.
- `ui/src/lib/proxy-export.ts` (+ `.test.ts`) — **create**: pure export transform + bulk-result summarizer (tested).
- `ui/src/hooks/use-proxies.ts` — add `bulkSetActive(ids, enable)` and `bulkRemove(ids)` (single invalidation, summary result, no per-item toast spam).
- `ui/src/routes/_dashboard/proxies/index.tsx` — selection state, bulk bar, export button, duplicate navigation.
- `ui/src/routes/_dashboard/proxies/new.tsx` — read a `duplicate` param, fetch the source proxy, pass it as `initialData` seed.
- `ui/src/components/proxy/forms/{reverse-proxy,redirect,static}-form.tsx` — seed `defaultValues`/`reset` from `initialData` whenever present (not only edit mode).

---

## Task 1: Backend — per-proxy ACL summary on the list response

**Files:**
- Modify: `backend/internal/models/proxy.go`
- Modify: `backend/internal/repository/proxy_repository.go`
- Test: `backend/internal/repository/proxy_repository_test.go`

**Interfaces:**
- Produces: `Proxy.ACLGroupCount int` (json `acl_group_count`), `Proxy.ACLGroupNames []string` (json `acl_group_names`), both `gorm:"-"`. `List` returns proxies with these populated. Repo interface signature in `interfaces.go` is **unchanged** (fields ride on the existing `[]models.Proxy`).

- [ ] **Step 1: Add computed fields to the Proxy model**

In `backend/internal/models/proxy.go`, add to the `Proxy` struct (near the `Creator` relation), with the other JSON fields:

```go
// ACL summary — computed by the repository List query, not persisted.
ACLGroupCount int      `json:"acl_group_count" gorm:"-"`
ACLGroupNames []string `json:"acl_group_names" gorm:"-"`
```

- [ ] **Step 2: Write the failing test for the mapping helper**

In `backend/internal/repository/proxy_repository_test.go` add a unit test for a pure helper that turns flat (proxy_id, name) rows into a per-proxy summary and applies it to a proxy slice. (If the file doesn't exist, create it with the package + imports used by sibling repo tests.)

```go
func TestApplyACLSummaries(t *testing.T) {
	proxies := []models.Proxy{{ID: 1}, {ID: 2}, {ID: 3}}
	rows := []aclSummaryRow{
		{ProxyID: 1, Name: "vpn"},
		{ProxyID: 1, Name: "staff"},
		{ProxyID: 3, Name: "vpn"},
	}
	applyACLSummaries(proxies, rows)

	assert.Equal(t, 2, proxies[0].ACLGroupCount)
	assert.Equal(t, []string{"vpn", "staff"}, proxies[0].ACLGroupNames)
	assert.Equal(t, 0, proxies[1].ACLGroupCount)
	assert.Equal(t, []string{}, proxies[1].ACLGroupNames) // never nil
	assert.Equal(t, 1, proxies[2].ACLGroupCount)
	assert.Equal(t, []string{"vpn"}, proxies[2].ACLGroupNames)
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd backend && go test ./internal/repository/ -run TestApplyACLSummaries`
Expected: FAIL — `aclSummaryRow` / `applyACLSummaries` undefined.

- [ ] **Step 4: Implement the helper + wire it into `List`**

In `backend/internal/repository/proxy_repository.go`:

```go
// aclSummaryRow is one (proxy, acl group name) pair from the summary query.
type aclSummaryRow struct {
	ProxyID int
	Name    string
}

// applyACLSummaries populates ACLGroupCount/ACLGroupNames on each proxy from the
// flat rows. Every proxy gets a non-nil names slice (empty when unprotected).
func applyACLSummaries(proxies []models.Proxy, rows []aclSummaryRow) {
	byProxy := make(map[int][]string, len(proxies))
	for _, row := range rows {
		byProxy[row.ProxyID] = append(byProxy[row.ProxyID], row.Name)
	}
	for i := range proxies {
		names := byProxy[proxies[i].ID]
		if names == nil {
			names = []string{}
		}
		proxies[i].ACLGroupNames = names
		proxies[i].ACLGroupCount = len(names)
	}
}
```

At the end of `List`, after the existing `query.Find(&proxies)` succeeds and before returning, add:

```go
	if len(proxies) > 0 {
		ids := make([]int, len(proxies))
		for i := range proxies {
			ids[i] = proxies[i].ID
		}
		var rows []aclSummaryRow
		if err := r.db.
			Table("proxy_acl_assignments").
			Select("proxy_acl_assignments.proxy_id, acl_groups.name").
			Joins("JOIN acl_groups ON acl_groups.id = proxy_acl_assignments.acl_group_id").
			Where("proxy_acl_assignments.proxy_id IN ?", ids).
			Order("proxy_acl_assignments.priority ASC, proxy_acl_assignments.id ASC").
			Scan(&rows).Error; err != nil {
			return nil, 0, fmt.Errorf("loading proxy ACL summaries: %w", err)
		}
		applyACLSummaries(proxies, rows)
	}
```

(Keep the existing `total` count return value unchanged — the summary query does not affect it.)

- [ ] **Step 5: Run the helper test to verify it passes**

Run: `cd backend && go test ./internal/repository/ -run TestApplyACLSummaries`
Expected: PASS.

- [ ] **Step 6: Add an integration test for the query (testcontainers)**

Follow the existing repository/handler integration test pattern (testcontainers + real PostgreSQL; mirror the setup helper used by the nearest existing integration test, and include the same `testing.Short()` skip guard if siblings use one). The test: seed 2 proxies, 2 ACL groups, and assignments (proxy A → both groups, proxy B → none); call `repo.List` with a params value that returns both; assert proxy A has `ACLGroupCount == 2` and the expected `ACLGroupNames`, and proxy B has `0` / `[]string{}`. Name it `TestProxyRepository_List_PopulatesACLSummary`.

- [ ] **Step 7: Run backend gate**

Run: `make format-backend && make lint-backend && make backend-test`
(If testcontainers/Docker is unavailable here, run `cd backend && go test ./... -short` instead and note it in the report.)
Expected: clean; unit test passes; integration test passes (or is skipped under `-short`).

- [ ] **Step 8: Commit**

```bash
git add backend/internal/models/proxy.go backend/internal/repository/proxy_repository.go backend/internal/repository/proxy_repository_test.go
git commit -m "feat(api): include per-proxy ACL group summary in proxy list (M2b)"
```

---

## Task 2: Frontend — ProxyConfig type + ACL summary column

**Files:**
- Modify: `ui/src/types/proxy.ts`
- Create: `ui/src/components/proxy/cells/proxy-acl-cell.tsx`
- Modify: `ui/src/components/proxy/cells/index.ts`
- Modify: `ui/src/components/proxy/proxy-data-grid.tsx`

**Interfaces:**
- Consumes: `acl_group_count`/`acl_group_names` from Task 1.
- Produces: `ProxyAclCell({ count, names })` component; a new "Access" column in the grid.

- [ ] **Step 1: Extend the type**

In `ui/src/types/proxy.ts`, add to `ProxyConfig` (after `created_by`):

```ts
  // ACL summary (from the list endpoint)
  acl_group_count?: number;
  acl_group_names?: string[];
```

- [ ] **Step 2: Create the ACL cell**

`ui/src/components/proxy/cells/proxy-acl-cell.tsx` — a shield + count badge; tooltip lists the group names; an unprotected state when count is 0. Use rnui `Badge`, `Tooltip`/`TooltipTrigger`/`TooltipContent`, lucide `Shield` / `ShieldOff`. Pattern:

```tsx
import { Badge, Tooltip, TooltipContent, TooltipTrigger } from '@e412/rnui-react';
import { Shield, ShieldOff } from 'lucide-react';

export function ProxyAclCell({ count, names }: { count?: number; names?: string[] }) {
  const n = count ?? 0;
  if (n === 0) {
    return (
      <span className="inline-flex items-center gap-1.5 text-muted-foreground">
        <ShieldOff className="size-3.5" />
        <span className="text-sm">Unprotected</span>
      </span>
    );
  }
  const list = names ?? [];
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Badge variant="secondary" className="gap-1">
            <Shield className="size-3.5" />
            {n}
          </Badge>
        }
      />
      <TooltipContent>{list.length ? list.join(', ') : `${n} ACL group(s)`}</TooltipContent>
    </Tooltip>
  );
}
```

> Verify `Tooltip`/`TooltipTrigger`/`TooltipContent` import names and the `render`-prop trigger usage against an existing cell (`proxy-actions-cell.tsx` already uses rnui `Tooltip`). Match its usage exactly.

- [ ] **Step 3: Export it** from `ui/src/components/proxy/cells/index.ts` (follow the existing export style).

- [ ] **Step 4: Add the column** to `proxy-data-grid.tsx` columns array (place it before the `actions` column):

```tsx
{
  id: 'acl',
  accessorKey: 'acl_group_count',
  header: ({ column }) => <DataGridColumnHeader column={column} title="Access" />,
  cell: ({ row }) => (
    <ProxyAclCell
      count={row.original.acl_group_count}
      names={row.original.acl_group_names}
    />
  ),
  enableSorting: false,
  meta: { skeleton: <Skeleton className="h-5 w-20" /> },
},
```

(Import `ProxyAclCell` from `./cells`. Match the existing column objects' shape — `meta.skeleton`, size hints — to siblings.)

- [ ] **Step 5: Gate**

Run: `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run`
Expected: clean; 26 tests still pass.

- [ ] **Step 6: Commit**

```bash
git add ui/src/types/proxy.ts ui/src/components/proxy/cells/proxy-acl-cell.tsx ui/src/components/proxy/cells/index.ts ui/src/components/proxy/proxy-data-grid.tsx
git commit -m "feat(ui): add ACL summary column to proxy list (M2b)"
```

---

## Task 3: Frontend — row selection in the data grid

**Files:**
- Modify: `ui/src/components/proxy/proxy-data-grid.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx`

**Interfaces:**
- Produces: `ProxyDataGrid` gains `rowSelection: RowSelectionState` + `onRowSelectionChange: OnChangeFn<RowSelectionState>` props; `getRowId` keyed by `String(proxy.id)`; a leading checkbox column (header select-all-on-page + per-row). The page owns `rowSelection` state. Selected ids derive from `Object.keys(rowSelection)`.

- [ ] **Step 1: Wire selection into the grid**

In `proxy-data-grid.tsx`:
- Import `type RowSelectionState`, `type OnChangeFn` from `@tanstack/react-table`, and rnui `Checkbox`.
- Add props `rowSelection` + `onRowSelectionChange` to the component props.
- In `useReactTable`: add `enableRowSelection: true`, `getRowId: (row) => String(row.id)`, `state: { pagination, rowSelection }`, `onRowSelectionChange`.
- Add a leading column:

```tsx
{
  id: 'select',
  header: ({ table }) => (
    <Checkbox
      checked={
        table.getIsAllPageRowsSelected()
          ? true
          : table.getIsSomePageRowsSelected()
            ? 'indeterminate'
            : false
      }
      onCheckedChange={(v) => table.toggleAllPageRowsSelected(!!v)}
      aria-label="Select all"
    />
  ),
  cell: ({ row }) => (
    <Checkbox
      checked={row.getIsSelected()}
      onCheckedChange={(v) => row.toggleSelected(!!v)}
      aria-label="Select row"
    />
  ),
  enableSorting: false,
  meta: { skeleton: <Skeleton className="size-4" /> },
},
```

> Verify rnui `Checkbox`'s indeterminate API — if it doesn't accept `'indeterminate'` for `checked`, fall back to `checked={row.getIsSelected()}` per-row and a plain boolean header (all-page). Note any deviation in the report.

- [ ] **Step 2: Own selection state in the page**

In `index.tsx`: `const [rowSelection, setRowSelection] = useState<RowSelectionState>({});` and pass `rowSelection` + `onRowSelectionChange={setRowSelection}` to `ProxyDataGrid`. Reset selection (`setRowSelection({})`) whenever `pagination`, `debouncedSearch`, or filters change (selection is per-page and shouldn't leak across pages/filters).

- [ ] **Step 3: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run` (clean).

- [ ] **Step 4: Manual check** — selecting rows + select-all toggles checkboxes; changing page/filter clears selection. Document in report.

- [ ] **Step 5: Commit**

```bash
git add ui/src/components/proxy/proxy-data-grid.tsx ui/src/routes/_dashboard/proxies/index.tsx
git commit -m "feat(ui): row selection on the proxy data grid (M2b)"
```

---

## Task 4: Frontend — bulk action bar (enable / disable / delete)

**Files:**
- Create: `ui/src/lib/proxy-export.ts` (the bulk-result summarizer lives here too) + `ui/src/lib/proxy-export.test.ts`
- Modify: `ui/src/hooks/use-proxies.ts`
- Create: `ui/src/components/proxy/proxy-bulk-bar.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx`

**Interfaces:**
- Produces:
  - `summarizeBulkResults(results: PromiseSettledResult<unknown>[]): { succeeded: number; failed: number }` (in `proxy-export.ts`).
  - `useProxies()` gains `bulkSetActive(ids: number[], enable: boolean): Promise<{succeeded:number;failed:number}>`, `bulkRemove(ids: number[]): Promise<{succeeded:number;failed:number}>`, `isBulkRunning: boolean`.
  - `ProxyBulkBar({ count, onEnable, onDisable, onDelete, onClear, running })`.

- [ ] **Step 1: Write the failing test for the summarizer**

`ui/src/lib/proxy-export.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { summarizeBulkResults } from './proxy-export';

describe('summarizeBulkResults', () => {
  it('counts fulfilled vs rejected', () => {
    const results: PromiseSettledResult<unknown>[] = [
      { status: 'fulfilled', value: 1 },
      { status: 'rejected', reason: new Error('x') },
      { status: 'fulfilled', value: 2 },
    ];
    expect(summarizeBulkResults(results)).toEqual({ succeeded: 2, failed: 1 });
  });
  it('handles empty', () => {
    expect(summarizeBulkResults([])).toEqual({ succeeded: 0, failed: 0 });
  });
});
```

- [ ] **Step 2: Run to verify it fails** — `pnpm --dir ui test run proxy-export` → FAIL (module missing).

- [ ] **Step 3: Implement the summarizer** in `ui/src/lib/proxy-export.ts`:

```ts
export function summarizeBulkResults(results: PromiseSettledResult<unknown>[]): {
  succeeded: number;
  failed: number;
} {
  let succeeded = 0;
  let failed = 0;
  for (const r of results) {
    if (r.status === 'fulfilled') succeeded++;
    else failed++;
  }
  return { succeeded, failed };
}
```

- [ ] **Step 4: Add bulk methods to `use-proxies.ts`**

Add (using the same `api` client + `queryClient` already in the hook; do NOT reuse the per-item toasting mutations — avoid one-toast-per-item):

```ts
const [isBulkRunning, setIsBulkRunning] = useState(false);

const bulkSetActive = async (ids: number[], enable: boolean) => {
  setIsBulkRunning(true);
  try {
    const action = enable ? 'enable' : 'disable';
    const results = await Promise.allSettled(
      ids.map((id) => api.post(`proxies/${id}/${action}`)),
    );
    await queryClient.invalidateQueries({ queryKey: ['proxies'] });
    return summarizeBulkResults(results);
  } finally {
    setIsBulkRunning(false);
  }
};

const bulkRemove = async (ids: number[]) => {
  setIsBulkRunning(true);
  try {
    const results = await Promise.allSettled(ids.map((id) => api.delete(`proxies/${id}`)));
    await queryClient.invalidateQueries({ queryKey: ['proxies'] });
    return summarizeBulkResults(results);
  } finally {
    setIsBulkRunning(false);
  }
};
```

Return `bulkSetActive`, `bulkRemove`, `isBulkRunning` from the hook. (Match the file's actual `api` import, query-key shape, and `queryClient` access — verify against the existing mutations in the file; the endpoint paths mirror the per-item `toggle`/`remove`.)

- [ ] **Step 5: Create the bulk bar** `ui/src/components/proxy/proxy-bulk-bar.tsx` — renders only when `count > 0`: a bar above the grid showing `{count} selected`, buttons Enable / Disable / Delete (destructive) / Clear, all disabled while `running`. rnui `Button` + lucide icons (`Check`, `Ban`, `Trash2`, `X`).

- [ ] **Step 6: Wire it in `index.tsx`** — compute `selectedIds = Object.keys(rowSelection).map(Number)`. Render `<ProxyBulkBar count={selectedIds.length} running={isBulkRunning} ... />`. Handlers:
  - Enable/Disable: `const s = await bulkSetActive(selectedIds, enable); toast.success(\`Enabled ${s.succeeded}\${s.failed ? \`, ${s.failed} failed\` : ''}\`); setRowSelection({});` (mirror for disable).
  - Delete: open a confirm `AlertDialog` (reuse the existing one's pattern) showing the count; on confirm `const s = await bulkRemove(selectedIds); toast(...); setRowSelection({});`.
  Use the existing `sonner` `toast` already used in the app.

- [ ] **Step 7: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run` (summarizer tests pass; 28 total).

- [ ] **Step 8: Manual check** — select rows → bar appears; enable/disable/delete act on the selection, show a summary toast, clear selection, and the grid refreshes. Document in report.

- [ ] **Step 9: Commit**

```bash
git add ui/src/lib/proxy-export.ts ui/src/lib/proxy-export.test.ts ui/src/hooks/use-proxies.ts ui/src/components/proxy/proxy-bulk-bar.tsx ui/src/routes/_dashboard/proxies/index.tsx
git commit -m "feat(ui): bulk enable/disable/delete on the proxy list (M2b)"
```

---

## Task 5: Frontend — duplicate a proxy (seed the create wizard)

**Files:**
- Modify: `ui/src/components/proxy/forms/{reverse-proxy,redirect,static}-form.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/new.tsx`
- Modify: `ui/src/components/proxy/cells/proxy-actions-cell.tsx`
- Modify: `ui/src/components/proxy/proxy-data-grid.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx`

**Interfaces:**
- Produces: forms seed from `initialData` whenever it's present (not only `mode === 'edit'`); `new.tsx` accepts `?type=<t>&duplicate=<id>`, fetches the source proxy, and passes a seed (hostname cleared, name suffixed " (copy)") as `initialData`; the actions cell gains an `onDuplicate` action.

- [ ] **Step 1: Seed forms from `initialData` regardless of mode**

In each of the three forms, change the `defaultValues` condition and the reset effect from mode-gated to initialData-gated. Reverse example (apply the analogous change to redirect/static with their mappers):

```tsx
// defaultValues
defaultValues: initialData ? mapProxyToReverseDefaults(initialData) : REVERSE_PROXY_DEFAULTS,
```
```tsx
// reset effect (when initialData lands async)
useEffect(() => {
  if (initialData) form.reset(mapProxyToReverseDefaults(initialData));
}, [initialData, form]);
```

This makes create-with-`initialData` (duplicate) seed the wizard while create-without stays on defaults; edit is unchanged. (`mode` is no longer referenced by these two lines — confirm it's still used elsewhere, e.g., layout selection and the `onInvalid` guard, so no unused-var lint.)

- [ ] **Step 2: Handle `duplicate` in `new.tsx`**

- Read `duplicate` from search params alongside `type`: `const dupId = Number(search.duplicate) || 0;`.
- When `dupId > 0`, fetch via `const { proxy: source } = useProxy(dupId);` and force `selectedType` to `source.type` once loaded.
- Build the seed (clear hostname, suffix name, drop identity) and pass as `initialData` to the active form:

```tsx
const seed: ProxyConfig | null = source
  ? { ...source, hostname: '', name: `${source.name} (copy)` }
  : null;
```
Pass `initialData={seed}` to whichever form matches `selectedType`. (For non-duplicate creates, `seed` is `null` → unchanged behavior.) Show the existing loading skeleton while `dupId > 0 && !source`.

- [ ] **Step 3: Add a Duplicate action to the actions cell**

In `proxy-actions-cell.tsx`, add an `onDuplicate?: (proxy: ProxyConfig) => void` prop and a third icon button (lucide `Copy`, tooltip "Duplicate") shown when `canCreate`-equivalent (reuse `canEdit`/the create permission already threaded; if no create perm is passed, gate on `onDuplicate` being defined). Keep the existing edit/delete buttons.

- [ ] **Step 4: Thread it through the grid + page**

- `proxy-data-grid.tsx`: add `onDuplicate?: (proxy: ProxyConfig) => void` prop, pass to `ProxyActionsCell`.
- `index.tsx`: `const handleDuplicate = (p: ProxyConfig) => navigate({ to: '/dashboard/proxies/new', search: { type: p.type, duplicate: p.id } });` and pass `onDuplicate={handleDuplicate}` to the grid. (Match the router's search param typing; the route already reads `type` loosely via `useSearch({ strict: false })`.)

- [ ] **Step 5: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run` (clean).

- [ ] **Step 6: Manual check** — Duplicate on a multi-upstream reverse proxy opens the create wizard pre-filled (upstreams, headers, toggles), hostname blank, name "X (copy)"; submitting creates a new proxy; the original is untouched. Verify redirect + static duplicate too. Document in report.

- [ ] **Step 7: Commit**

```bash
git add ui/src/components/proxy/forms/reverse-proxy-form.tsx ui/src/components/proxy/forms/redirect-form.tsx ui/src/components/proxy/forms/static-form.tsx ui/src/routes/_dashboard/proxies/new.tsx ui/src/components/proxy/cells/proxy-actions-cell.tsx ui/src/components/proxy/proxy-data-grid.tsx ui/src/routes/_dashboard/proxies/index.tsx
git commit -m "feat(ui): duplicate a proxy via seeded create wizard (M2b)"
```

---

## Task 6: Frontend — JSON export

**Files:**
- Modify: `ui/src/lib/proxy-export.ts` (+ extend `proxy-export.test.ts`)
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx`

**Interfaces:**
- Produces: `toExportPayload(proxies: ProxyConfig[]): ProxyExport[]` (pure, strips runtime/identity fields, keeps type + config) and `downloadJson(filename: string, data: unknown): void` (browser download). An Export button in the page header.

- [ ] **Step 1: Write the failing test for `toExportPayload`**

Extend `ui/src/lib/proxy-export.test.ts`:

```ts
import { toExportPayload } from './proxy-export';
import type { ProxyConfig } from '@/types/proxy';

describe('toExportPayload', () => {
  it('keeps config, drops identity/runtime fields', () => {
    const p = {
      id: 7, type: 'reverse_proxy', name: 'api', hostname: 'api.example.com',
      ssl_enabled: true, ssl_forced: false, is_active: true,
      created_at: 't', updated_at: 't', created_by: 1,
      upstreams: [{ host: 'h', port: 80, scheme: 'http' }],
      acl_group_count: 2, acl_group_names: ['a', 'b'],
    } as ProxyConfig;
    const [out] = toExportPayload([p]);
    expect(out).toMatchObject({
      type: 'reverse_proxy', name: 'api', hostname: 'api.example.com',
      ssl_enabled: true, upstreams: [{ host: 'h', port: 80, scheme: 'http' }],
    });
    expect(out).not.toHaveProperty('id');
    expect(out).not.toHaveProperty('created_at');
    expect(out).not.toHaveProperty('acl_group_count');
    expect(out).not.toHaveProperty('is_active');
  });
});
```

- [ ] **Step 2: Run to verify it fails** — `pnpm --dir ui test run proxy-export` → FAIL.

- [ ] **Step 3: Implement** in `proxy-export.ts`:

```ts
import type { ProxyConfig } from '@/types/proxy';

export interface ProxyExport {
  type: ProxyConfig['type'];
  name: string;
  hostname: string;
  description?: string;
  ssl_enabled: boolean;
  upstreams?: ProxyConfig['upstreams'];
  load_balancing?: ProxyConfig['load_balancing'];
  block_exploits?: boolean;
  tls_insecure_skip_verify?: boolean;
  custom_headers?: ProxyConfig['custom_headers'];
  redirect?: ProxyConfig['redirect'];
  static?: ProxyConfig['static'];
}

export function toExportPayload(proxies: ProxyConfig[]): ProxyExport[] {
  return proxies.map((p) => ({
    type: p.type,
    name: p.name,
    hostname: p.hostname,
    description: p.description,
    ssl_enabled: p.ssl_enabled,
    upstreams: p.upstreams,
    load_balancing: p.load_balancing,
    block_exploits: p.block_exploits,
    tls_insecure_skip_verify: p.tls_insecure_skip_verify,
    custom_headers: p.custom_headers,
    redirect: p.redirect,
    static: p.static,
  }));
}

export function downloadJson(filename: string, data: unknown): void {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
```

- [ ] **Step 4: Run to verify it passes** — `pnpm --dir ui test run proxy-export` → PASS.

- [ ] **Step 5: Add the Export button** in `index.tsx` header (next to "Add Proxy"). Behavior: exports the **selected** proxies if any are selected, else **all** matching the current filters. Selected → map from the current `proxies` page by id. All → fetch via the existing list call with a large limit (`useProxies({ ...params, limit: 10000 })` or a one-off `api.get('proxies?limit=10000')`); keep it simple — reuse the hook/api already imported. Then `downloadJson(\`waygates-proxies-${count}.json\`, toExportPayload(list))` and a success toast. Disable the button while exporting.

- [ ] **Step 6: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run` (clean; export tests pass).

- [ ] **Step 7: Manual check** — Export with no selection downloads all; with a selection downloads just those; the JSON contains config without `id`/timestamps. Document in report.

- [ ] **Step 8: Commit**

```bash
git add ui/src/lib/proxy-export.ts ui/src/lib/proxy-export.test.ts ui/src/routes/_dashboard/proxies/index.tsx
git commit -m "feat(ui): JSON export of proxies (M2b)"
```

---

## Self-Review (run after writing, before execution)

1. **Spec coverage:** ACL summary column ✓ (Task 1 backend + Task 2 frontend); bulk enable/disable/delete ✓ (Task 3 selection + Task 4 bar/hook); duplicate ✓ (Task 5); export ✓ (Task 6); import explicitly deferred ✓ (Global + not a task).
2. **Placeholder scan:** tricky logic shown in full (backend query + helper, summarizer, export transform, selection column, duplicate seed). UI chrome (bar layout, button placement) is described against existing patterns the implementer reads.
3. **Type/name consistency:** `acl_group_count`/`acl_group_names` identical across model JSON tags (Task 1), `ProxyConfig` (Task 2), and the cell/column (Task 2). `summarizeBulkResults`/`toExportPayload`/`downloadJson` defined in `proxy-export.ts` (Tasks 4/6) and consumed by the hook/page. `bulkSetActive`/`bulkRemove`/`isBulkRunning` defined in Task 4, consumed by the bar wiring.

## Execution notes

- **Task 1 (backend) is the only backend slice** and is independent — do it first so the column has real data, but Tasks 2–6 are frontend and build on each other (selection → bulk → export-selected; forms-seed → duplicate).
- Deferred to a follow-up: **proxy import** (JSON → bulk create with hostname-conflict detection + per-item error report) — design its conflict/validation UX separately.
- The bulk ops are client-side loops (`Promise.allSettled`) — acceptable for page-sized selections; if a future need arises for very large batches, add real bulk endpoints.
