# Proxy Details (Overview) Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only overview page for each proxy (HTTP + L4) at its detail URL and move the edit form to a child `/edit` route, so deep-links land on an overview and navigation follows list → detail → edit.

**Architecture:** Code-based router (`ui/src/lib/router.tsx`, flat children of `dashboardRoute`, lazy-loaded by named export). The existing detail routes are repointed to new overview components; the current edit pages move to new `…/edit` routes (content unchanged except param-`from` strings and back-nav). New read-only overview components render existing data from `useProxy`/`useProxyACL`/`useL4Proxy`. No backend change.

**Tech Stack:** React 19 + TS, `@e412/rnui-react` (Card/Badge/Button/DropdownMenu/AlertDialog/Skeleton), TanStack React Router, lucide-react.

## Global Constraints

- **Frontend-only.** No backend/endpoint/type/hook change. Reuse `useProxy`, `useProxyACL`, `useL4Proxy`, `useProxies`/`useL4Proxies` (delete), and the existing `?duplicate=<id>` create flow.
- **Overview is read-only.** No form fields. The only mutations are Duplicate (navigate to create wizard seeded) and Delete (existing confirm + delete hook).
- **Route move must keep param names** (`proxyId`, `l4ProxyId`) and update the `useParams({ from })` strings to the new route paths; every nav call site that pointed at the detail route as "edit" must be repointed.
- **Container/layout:** overview uses `space-y-6 max-w-5xl` + the existing header pattern (back button, type icon, name + status badges, subtitle). Cards use rnui `Card`/`CardHeader`/`CardTitle`/`CardContent`.
- **Reserved slots:** leave the card layout open for future "Generated Config" (B1) and "Health" (B2) cards — do NOT stub fake data.
- **Gate per task (repo root):** `pnpm --dir ui build` && `pnpm --dir ui check` && `pnpm --dir ui test` — all green; the existing **57 tests** must keep passing. No new unit tests unless a pure helper is extracted.
- **Commit trailer:** `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Stage only the files listed per task; never `git add -A`.

## Reference Files (read before implementing)

- Current HTTP edit page (to MOVE to `…/edit`): `ui/src/routes/_dashboard/proxies/$proxyId.tsx` (exports `ProxyDetailPage`).
- Current L4 edit page (to MOVE): `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId.tsx` (exports `L4ProxyDetailPage`).
- Router: `ui/src/lib/router.tsx` (`proxyDetailRoute` ~103-110, `l4ProxyDetailRoute` ~131-138, `lazyRouteComponent` + `redirect` patterns, `routeTree.addChildren`).
- HTTP list nav: `ui/src/routes/_dashboard/proxies/index.tsx` (`handleEdit` ~184 → detail; `handleDuplicate` ~191; `ProxyActionsCell` wired `onEdit`/`onDelete`/`onDuplicate` ~337-339; **no `onRowClick` today**).
- L4 list nav: `ui/src/routes/_dashboard/l4-proxies/index.tsx` (`handleRowClick` ~173 → detail; `handleDuplicate` ~196; row dropdown has Duplicate+Delete, **no Edit action**; `onRowClick={handleRowClick}` ~505).
- Actions cell: `ui/src/components/proxy/cells/proxy-actions-cell.tsx` (`onEdit`/`onDuplicate`/`onDelete` callbacks).
- Field source: `ui/src/types/proxy.ts` (ProxyConfig + per-type), `ui/src/types/l4-proxy.ts`.
- rnui detail layout reference: `ui/src/routes/_dashboard/acl/$groupId.tsx`; read-only label/value reference: `ui/src/components/activity/activity-detail-sheet.tsx` (`MetaRow`).
- Type icon/label helpers: `getProxyTypeIcon` (`@/components/proxy/cells`), `getProxyTypeLabel` (`@/components/proxy`).

## Shared component — `DetailRow`

A generic read-only label/value row used by all overview cards. Create once in Task 2.

`ui/src/components/ui/detail-row.tsx`:
```tsx
import type { ReactNode } from 'react';

interface DetailRowProps {
  label: string;
  children: ReactNode;
}

export function DetailRow({ label, children }: DetailRowProps) {
  return (
    <div className="grid grid-cols-[160px_1fr] items-start gap-2 py-1.5 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="min-w-0 break-words">{children}</span>
    </div>
  );
}
```

---

## Task 1: HTTP routing split + overview shell

Move the HTTP edit form to `…/$proxyId/edit`, add a minimal read-only `ProxyOverviewPage` at `…/$proxyId`, wire the router, and repoint the HTTP list nav. (Content cards land in Tasks 2–3.)

**Files:**
- Create: `ui/src/routes/_dashboard/proxies/$proxyId/edit.tsx` (moved edit page)
- Create: `ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx` (new overview shell)
- Delete: `ui/src/routes/_dashboard/proxies/$proxyId.tsx`
- Modify: `ui/src/lib/router.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx`

**Interfaces:**
- Produces: `ProxyEditPage` (named export, moved), `ProxyOverviewPage` (named export) — consumed by `router.tsx`.

- [ ] **Step 1: Move the edit page.** Copy the entire current `ui/src/routes/_dashboard/proxies/$proxyId.tsx` to `ui/src/routes/_dashboard/proxies/$proxyId/edit.tsx`, then make exactly these changes in the new file:
  - Rename the export `ProxyDetailPage` → `ProxyEditPage`.
  - Change the param hook: `useParams({ from: '/dashboard/proxies/$proxyId' })` → `useParams({ from: '/dashboard/proxies/$proxyId/edit' })`.
  - Repoint back/cancel to the overview: the back-button `onClick` and `handleCancel` both `navigate({ to: '/dashboard/proxies' })` → `navigate({ to: '/dashboard/proxies/$proxyId', params: { proxyId: String(proxyId) } })`. (The not-found "Back to Proxies" button and `handleDelete`'s post-delete nav stay → `/dashboard/proxies`.)
  Everything else (forms, ACL update logic, delete dialog) is unchanged.

- [ ] **Step 2: Delete** the old `ui/src/routes/_dashboard/proxies/$proxyId.tsx`.

- [ ] **Step 3: Create the overview shell** — `ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx`. Minimal for this task (header + Edit/back + skeleton/not-found); cards are added in Tasks 2–3 (leave a `{/* config/access/details cards — Tasks 2-3 */}` comment where they go):

```tsx
import { Badge, Button, Skeleton } from '@e412/rnui-react';
import { useNavigate, useParams } from '@tanstack/react-router';
import { ArrowLeft, Globe, Pencil } from 'lucide-react';

import { getProxyTypeLabel } from '@/components/proxy';
import { getProxyTypeIcon } from '@/components/proxy/cells';
import { useProxy } from '@/hooks/use-proxies';

export function ProxyOverviewPage() {
  const params = useParams({ from: '/dashboard/proxies/$proxyId' });
  const proxyId = parseInt(params.proxyId, 10);
  const navigate = useNavigate();
  const { proxy, isLoading } = useProxy(proxyId);

  if (isLoading) {
    return (
      <div className="space-y-6 max-w-5xl">
        <div className="flex items-center gap-4">
          <Skeleton className="size-8 rounded" />
          <Skeleton className="size-10 rounded-lg" />
          <div className="space-y-2">
            <Skeleton className="h-7 w-48" />
            <Skeleton className="h-4 w-32" />
          </div>
        </div>
        <Skeleton className="h-48 rounded-lg" />
        <Skeleton className="h-32 rounded-lg" />
      </div>
    );
  }

  if (!proxy) {
    return (
      <div className="flex flex-col items-center justify-center py-12 space-y-4">
        <Globe className="size-12 text-muted-foreground" />
        <h2 className="text-xl font-semibold">Proxy Not Found</h2>
        <p className="text-muted-foreground">
          The proxy you're looking for doesn't exist or has been deleted.
        </p>
        <Button onClick={() => navigate({ to: '/dashboard/proxies' })}>
          <ArrowLeft className="size-4" />
          Back to Proxies
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-5xl">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/dashboard/proxies' })}>
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded bg-primary/10">
              {getProxyTypeIcon(proxy.type)}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold">{proxy.name}</h1>
                <Badge variant={proxy.is_active ? 'default' : 'secondary'}>
                  {proxy.is_active ? 'Active' : 'Inactive'}
                </Badge>
                {proxy.ssl_enabled && <Badge variant="outline">HTTPS</Badge>}
              </div>
              <p className="text-sm text-muted-foreground">
                {getProxyTypeLabel(proxy.type)} &middot; {proxy.hostname}
              </p>
            </div>
          </div>
        </div>
        <Button
          onClick={() =>
            navigate({ to: '/dashboard/proxies/$proxyId/edit', params: { proxyId: String(proxy.id) } })
          }
        >
          <Pencil className="size-4" />
          Edit
        </Button>
      </div>

      {/* config/access/details cards — Tasks 2-3 */}
    </div>
  );
}
```

- [ ] **Step 4: Wire the router.** In `ui/src/lib/router.tsx`, change `proxyDetailRoute`'s component import to the overview, and add a new `proxyEditRoute`:

```tsx
const proxyDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/$proxyId',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/proxies/$proxyId/overview'),
    'ProxyOverviewPage',
  ),
});

const proxyEditRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/$proxyId/edit',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/proxies/$proxyId/edit'),
    'ProxyEditPage',
  ),
});
```
Add `proxyEditRoute` to the `dashboardRoute.addChildren([...])` array (next to `proxyDetailRoute`).

- [ ] **Step 5: Repoint the HTTP list nav** in `ui/src/routes/_dashboard/proxies/index.tsx`:
  - `handleEdit` → the edit route: `navigate({ to: '/dashboard/proxies/$proxyId/edit', params: { proxyId: String(proxy.id) } })`.
  - Add a row-click → overview handler and pass it to the DataGrid as `onRowClick` (matching the L4 list). Add:
    ```tsx
    const handleRowClick = useCallback(
      (proxy: ProxyConfig) => {
        navigate({ to: '/dashboard/proxies/$proxyId', params: { proxyId: String(proxy.id) } });
      },
      [navigate],
    );
    ```
    and add `onRowClick={handleRowClick}` to the `<DataGrid …>` props. (`handleEdit` stays wired to `ProxyActionsCell` via `onEdit`.)
  - `new.tsx` already navigates to `/dashboard/proxies/$proxyId` on success → now the overview; no change needed.

- [ ] **Step 6: Gate.** `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` → build clean, 0 errors, 57/57. Reason through: `/dashboard/proxies/$proxyId` shows the overview shell; `…/$proxyId/edit` shows the edit form; row-click → overview; Edit action → edit; edit back/cancel → overview.

- [ ] **Step 7: Commit**
```bash
git add ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx \
  ui/src/routes/_dashboard/proxies/$proxyId/edit.tsx \
  ui/src/lib/router.tsx ui/src/routes/_dashboard/proxies/index.tsx
git rm ui/src/routes/_dashboard/proxies/$proxyId.tsx
git commit -m "feat(ui): HTTP proxy overview route + move edit to /edit (details page)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: HTTP overview content cards

Add the config-by-type, HTTPS/TLS, and Details cards, plus the Duplicate/Delete kebab. Behavior read-only.

**Files:**
- Create: `ui/src/components/ui/detail-row.tsx` (the `DetailRow` from the shared section above)
- Create: `ui/src/components/proxy/overview/proxy-config-card.tsx`
- Create: `ui/src/components/proxy/overview/proxy-meta-cards.tsx` (HTTPS + Details cards)
- Modify: `ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx` (mount cards + actions kebab)

**Interfaces:**
- Consumes: `DetailRow`; `ProxyConfig` from `@/types/proxy`.
- Produces: `ProxyConfigCard`, `ProxyHttpsCard`, `ProxyDetailsCard` (named exports, props `{ proxy: ProxyConfig }`).

- [ ] **Step 1: Create `DetailRow`** at `ui/src/components/ui/detail-row.tsx` (code in the Shared component section above).

- [ ] **Step 2: Create `ProxyConfigCard`** — `ui/src/components/proxy/overview/proxy-config-card.tsx`. Renders a `Card` titled "Configuration" whose body switches on `proxy.type`:
  - **reverse_proxy:** a "Upstreams" `DetailRow` listing each `proxy.upstreams[]` as `${scheme}://${host}:${port}` (one per line); `DetailRow` "Load balancing" → `proxy.load_balancing.strategy`; `DetailRow` "Block exploits" → Yes/No (`proxy.block_exploits`); `DetailRow` "Skip TLS verify" → Yes/No (`proxy.tls_insecure_skip_verify`); if `proxy.custom_headers?.request`/`.response` have entries, a `DetailRow` "Request headers"/"Response headers" listing `key: value` lines.
  - **redirect:** `DetailRow` "Target" → `proxy.redirect.target`; "Status code" → `proxy.redirect.status_code`; "Preserve path" → Yes/No; "Preserve query" → Yes/No.
  - **static:** `DetailRow` "Root path" → `proxy.static.root_path`; "Index file" → `proxy.static.index_file`; "Directory browse" → Yes/No (`proxy.static.browse`); "Template rendering" → Yes/No (`proxy.static.template_rendering`); if `proxy.static.try_files?.length`, "Try files" → joined list.

  Reference implementation (reverse branch shown; implement redirect/static analogously with the fields above):
```tsx
import { Card, CardContent, CardHeader, CardTitle } from '@e412/rnui-react';

import { DetailRow } from '@/components/ui/detail-row';
import type { ProxyConfig } from '@/types/proxy';

function yesNo(v: boolean | undefined) {
  return v ? 'Yes' : 'No';
}

export function ProxyConfigCard({ proxy }: { proxy: ProxyConfig }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
      </CardHeader>
      <CardContent className="divide-y">
        {proxy.type === 'reverse_proxy' && (
          <>
            <DetailRow label="Upstreams">
              <div className="flex flex-col gap-0.5">
                {proxy.upstreams?.map((u, i) => (
                  <span key={`${u.host}:${u.port}:${i}`} className="font-mono text-xs">
                    {u.scheme}://{u.host}:{u.port}
                  </span>
                ))}
              </div>
            </DetailRow>
            <DetailRow label="Load balancing">{proxy.load_balancing?.strategy ?? '—'}</DetailRow>
            <DetailRow label="Block exploits">{yesNo(proxy.block_exploits)}</DetailRow>
            <DetailRow label="Skip TLS verify">{yesNo(proxy.tls_insecure_skip_verify)}</DetailRow>
          </>
        )}
        {proxy.type === 'redirect' && (
          <>
            <DetailRow label="Target">{proxy.redirect?.target}</DetailRow>
            <DetailRow label="Status code">{proxy.redirect?.status_code}</DetailRow>
            <DetailRow label="Preserve path">{yesNo(proxy.redirect?.preserve_path)}</DetailRow>
            <DetailRow label="Preserve query">{yesNo(proxy.redirect?.preserve_query)}</DetailRow>
          </>
        )}
        {proxy.type === 'static' && (
          <>
            <DetailRow label="Root path">{proxy.static?.root_path}</DetailRow>
            <DetailRow label="Index file">{proxy.static?.index_file}</DetailRow>
            <DetailRow label="Directory browse">{yesNo(proxy.static?.browse)}</DetailRow>
            <DetailRow label="Template rendering">{yesNo(proxy.static?.template_rendering)}</DetailRow>
          </>
        )}
      </CardContent>
    </Card>
  );
}
```
  (Verify exact field names against `ui/src/types/proxy.ts`; render custom_headers/try_files only when present.)

- [ ] **Step 3: Create `ProxyHttpsCard` + `ProxyDetailsCard`** in `ui/src/components/proxy/overview/proxy-meta-cards.tsx`:
  - `ProxyHttpsCard`: Card "HTTPS / TLS" → `DetailRow` "HTTPS enabled" (Yes/No `ssl_enabled`), "Force HTTPS" (Yes/No `ssl_forced`).
  - `ProxyDetailsCard`: Card "Details" → `DetailRow` "Description" (`proxy.description || '—'`), "ID" (`proxy.id`), "Created" (`new Date(proxy.created_at).toLocaleString()`), "Updated" (`new Date(proxy.updated_at).toLocaleString()`). Use the same `Card`/`DetailRow` pattern as Step 2.

- [ ] **Step 4: Mount cards + actions kebab** in `overview.tsx`. Replace the `{/* config/access/details cards — Tasks 2-3 */}` comment with `<ProxyConfigCard proxy={proxy} />`, then (Task 3 adds the Access card here), then `<ProxyHttpsCard proxy={proxy} />` and `<ProxyDetailsCard proxy={proxy} />`. Wrap in `<div className="grid gap-6 lg:grid-cols-2">` for the meta cards as desired (config card full-width above). In the header, beside the Edit button, add a kebab `DropdownMenu` with **Duplicate** (`navigate({ to: '/dashboard/proxies/new', search: { type: proxy.type, duplicate: proxy.id } })`) and **Delete** (opens an `AlertDialog` reusing the exact delete pattern from the edit page — `useProxies().remove` + post-delete `navigate({ to: '/dashboard/proxies' })`). Import `useProxies` + `useState` for the dialog. (Mirror the edit page's delete dialog markup verbatim.)

- [ ] **Step 5: Gate.** `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` → green. Verify each proxy type renders its config; Duplicate/Delete work.

- [ ] **Step 6: Commit**
```bash
git add ui/src/components/ui/detail-row.tsx ui/src/components/proxy/overview/ \
  ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx
git commit -m "feat(ui): HTTP proxy overview config/HTTPS/details cards + actions

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: HTTP overview Access ("what protects this") card

**Files:**
- Create: `ui/src/components/proxy/overview/proxy-access-card.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx` (mount it)

**Interfaces:**
- Consumes: `useProxyACL` (`@/hooks`), `Link` (`@tanstack/react-router`).
- Produces: `ProxyAccessCard` (named export, props `{ proxyId: number }`).

- [ ] **Step 1: Create `ProxyAccessCard`** — `ui/src/components/proxy/overview/proxy-access-card.tsx`. Fetches `const { assignments, isLoading } = useProxyACL(proxyId)`. Card titled "Access — what protects this":
  - loading → a `Skeleton`.
  - empty (`assignments.length === 0`) → a "Unprotected" line (muted, with `ShieldOff` icon) + a `Link to="/dashboard/access"` CTA "Configure access".
  - otherwise → one row per assignment: the group name as a `Link to="/dashboard/access/$groupId" params={{ groupId: String(a.acl_group_id) }}` (name from `a.acl_group?.name ?? \`Group #${a.acl_group_id}\``), plus its `path_pattern`, `priority`, and an enabled/disabled `Badge`. Use `DetailRow` or a small list; keep it read-only.

```tsx
import { Badge, Card, CardContent, CardHeader, CardTitle, Skeleton } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { ShieldOff } from 'lucide-react';

import { useProxyACL } from '@/hooks';

export function ProxyAccessCard({ proxyId }: { proxyId: number }) {
  const { assignments, isLoading } = useProxyACL(proxyId);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Access — what protects this</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-12 w-full" />
        ) : assignments.length === 0 ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <ShieldOff className="size-4" />
            Unprotected ·{' '}
            <Link to="/dashboard/access" className="text-primary hover:underline">
              Configure access
            </Link>
          </div>
        ) : (
          <ul className="space-y-2">
            {assignments.map((a) => (
              <li key={a.acl_group_id} className="flex items-center justify-between gap-3 text-sm">
                <Link
                  to="/dashboard/access/$groupId"
                  params={{ groupId: String(a.acl_group_id) }}
                  className="font-medium text-primary hover:underline"
                >
                  {a.acl_group?.name ?? `Group #${a.acl_group_id}`}
                </Link>
                <span className="text-muted-foreground font-mono text-xs">{a.path_pattern}</span>
                <Badge variant={a.enabled ? 'default' : 'secondary'}>
                  {a.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
```
  (Verify `assignments` item field names against `useProxyACL`'s `ProxyACLAssignment` — `acl_group_id`, `acl_group?.name`, `path_pattern`, `priority`, `enabled`.)

- [ ] **Step 2: Mount it** in `overview.tsx` between the config card and the meta cards: `<ProxyAccessCard proxyId={proxy.id} />`.

- [ ] **Step 3: Gate.** `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` → green. Verify a protected proxy lists its groups (links resolve to the group detail) and an unprotected one shows the empty state.

- [ ] **Step 4: Commit**
```bash
git add ui/src/components/proxy/overview/proxy-access-card.tsx \
  ui/src/routes/_dashboard/proxies/$proxyId/overview.tsx
git commit -m "feat(ui): HTTP proxy overview Access card (what protects this)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: L4 routing split + overview

Mirror Tasks 1–2 for L4 (simpler — no ACL).

**Files:**
- Create: `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId/edit.tsx` (moved edit page)
- Create: `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId/overview.tsx` (new overview)
- Create: `ui/src/components/l4-proxy/overview/l4-proxy-config-card.tsx`
- Delete: `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId.tsx`
- Modify: `ui/src/lib/router.tsx`
- Modify: `ui/src/routes/_dashboard/l4-proxies/index.tsx`

**Interfaces:**
- Produces: `L4ProxyEditPage`, `L4ProxyOverviewPage` (named exports); `L4ProxyConfigCard` (`{ proxy }`).

- [ ] **Step 1: Move the L4 edit page.** Copy `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId.tsx` → `…/$l4ProxyId/edit.tsx`; rename export `L4ProxyDetailPage` → `L4ProxyEditPage`; change `useParams({ from: '/dashboard/proxies/tcp-udp/$l4ProxyId' })` → `…/$l4ProxyId/edit`; repoint the back-button `onClick` and `handleCancel` from `/dashboard/proxies/tcp-udp` → `navigate({ to: '/dashboard/proxies/tcp-udp/$l4ProxyId', params: { l4ProxyId: String(l4ProxyId) } })` (not-found button + post-delete nav stay → `/dashboard/proxies/tcp-udp`).

- [ ] **Step 2: Delete** `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId.tsx`.

- [ ] **Step 3: Create `L4ProxyConfigCard`** — `ui/src/components/l4-proxy/overview/l4-proxy-config-card.tsx`. Read `ui/src/types/l4-proxy.ts` for exact fields; render a "Configuration" Card with `DetailRow`s for: listen port, protocol, the routes → upstreams summary (each route's match + its upstream host:port list), and TLS mode (terminate/passthrough) when present. Reuse `DetailRow` from `@/components/ui/detail-row`.

- [ ] **Step 4: Create the L4 overview** — `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId/overview.tsx`. Same shell shape as the HTTP overview (Task 1 Step 3) but using `useL4Proxy(l4ProxyId)`, the `Network` icon + protocol/active badges + `Port {proxy.listen_port}` subtitle (mirror the current L4 detail header), an **Edit** button → `…/$l4ProxyId/edit`, a kebab with **Duplicate** (`navigate({ to: '/dashboard/proxies/tcp-udp/new', search: { duplicate: proxy.id } })`) and **Delete** (reuse the L4 edit page's delete dialog pattern — `useL4Proxies().remove` + nav to `/dashboard/proxies/tcp-udp`), then `<L4ProxyConfigCard proxy={proxy} />` and an L4 Details card (description, id, timestamps — reuse the `Card`/`DetailRow` pattern; no separate component needed). `useParams({ from: '/dashboard/proxies/tcp-udp/$l4ProxyId' })`.

- [ ] **Step 5: Wire the router.** In `router.tsx`, repoint `l4ProxyDetailRoute`'s component import to `…/$l4ProxyId/overview` `'L4ProxyOverviewPage'`, and add `l4ProxyEditRoute` (path `/proxies/tcp-udp/$l4ProxyId/edit` → `…/$l4ProxyId/edit` `'L4ProxyEditPage'`); add it to `addChildren`.

- [ ] **Step 6: Repoint L4 list nav** in `ui/src/routes/_dashboard/l4-proxies/index.tsx`: `handleRowClick` already targets `/dashboard/proxies/tcp-udp/$l4ProxyId` → now the overview (no change). **Add an Edit action** to the L4 row dropdown that navigates to `…/$l4ProxyId/edit` (mirror how `handleDuplicate` is wired into the row menu — add an Edit `DropdownMenuItem` gated on the existing can-edit/can-update permission used elsewhere in this list, calling `navigate({ to: '/dashboard/proxies/tcp-udp/$l4ProxyId/edit', params: { l4ProxyId: String(row.original.id) } })`). `new.tsx` success already → `…/$l4ProxyId` (now overview); no change.

- [ ] **Step 7: Gate.** `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` → green. Verify L4 row-click → overview, Edit action → edit form, Duplicate/Delete work, edit back → overview.

- [ ] **Step 8: Commit**
```bash
git add ui/src/routes/_dashboard/l4-proxies/$l4ProxyId/overview.tsx \
  ui/src/routes/_dashboard/l4-proxies/$l4ProxyId/edit.tsx \
  ui/src/components/l4-proxy/overview/ \
  ui/src/lib/router.tsx ui/src/routes/_dashboard/l4-proxies/index.tsx
git rm ui/src/routes/_dashboard/l4-proxies/$l4ProxyId.tsx
git commit -m "feat(ui): L4 proxy overview route + move edit to /edit (details page)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:** overview at `$proxyId` + edit at `$proxyId/edit` for HTTP (Task 1) and L4 (Task 4); HTTP content cards config/HTTPS/details + actions (Task 2); HTTP Access "what protects this" (Task 3); nav repoints (Tasks 1 & 4: HTTP add onRowClick + Edit→/edit; L4 row-click already → overview + add Edit action); read-only, no backend, reserved B1/B2 slots (overview leaves card region open); container `space-y-6 max-w-5xl`. No spec requirement unaddressed.

**Placeholder scan:** route/move steps give exact `from`-string and nav edits; overview shell + cards + Access card + DetailRow have full code; the redirect/static config branches and the L4 config card are specified by exact field + a worked reverse/HTTP template (the implementer fills the analogous fields, verifying names against the `types/*` files named in each step) — no "TBD"/"handle X".

**Type/name consistency:** export names (`ProxyOverviewPage`/`ProxyEditPage`/`L4ProxyOverviewPage`/`L4ProxyEditPage`) match their `lazyRouteComponent` import strings; `DetailRow` created in Task 2 is consumed by Tasks 2–4; `ProxyConfigCard`/`ProxyHttpsCard`/`ProxyDetailsCard`/`ProxyAccessCard`/`L4ProxyConfigCard` props are `{ proxy }`/`{ proxyId }` as mounted; `useParams` `from` strings match the new route paths.
