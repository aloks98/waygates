# M0b — IA Restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the dashboard information architecture: unify HTTP + TCP/UDP proxies under one **Proxies** section with tabs, and rename **ACL→Access** and **Audit Logs→Activity** — updating routes, all internal links, and the sidebar nav.

**Architecture:** Routing is **code-based** in `ui/src/lib/router.tsx` (explicit `createRoute()` + `routeTree.addChildren`). We change route **path strings** (URLs) only — route component **files keep their current locations** (`routes/_dashboard/l4-proxies/*`, `acl/*`, `audit-logs.tsx`); only the `path:` and the runtime `to`/`from`/`navigate` references change. A new `ProxiesTabs` component renders a shared HTTP/TCP-UDP tab bar atop the two list pages. Redirect routes preserve old bookmarks.

**Tech Stack:** TanStack Router v1 (code-based), React 19, rnui (`Tabs`), lucide-react, pnpm.

## Global Constraints

- pnpm; UI in `ui/`. Run as `pnpm --dir ui <script>`. Branch `feat/rnui-redesign-program`.
- **URL/route paths change; API endpoints DO NOT.** Never touch `api.get('l4-proxies/...')`, `api.get('acl/...')`, `audit-logs` API strings, query keys, or the component import path `@/components/audit-logs` (that's a folder, not a route).
- New URL scheme: HTTP proxies `/dashboard/proxies`; TCP/UDP proxies `/dashboard/proxies/tcp-udp` (+ `/new`, `/$l4ProxyId`); Access `/dashboard/access` (+ `/$groupId`); Activity `/dashboard/activity`. HTTP create/edit stay `/dashboard/proxies/new` and `/dashboard/proxies/$proxyId`.
- Verification gates: `pnpm --dir ui build` (success) + `pnpm --dir ui check:fix && pnpm --dir ui check` (clean; pre-existing oxlint warnings OK) + `pnpm --dir ui test run` (pass). **No `tsc` gate** (pre-existing type debt; do not fix unrelated type errors). Interactive nav smoke is done in a final `verify` pass by the controller.
- Commits: Conventional Commits + trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

### Route-path change table (authoritative)
| Old URL | New URL | Route file (unchanged) |
|---|---|---|
| `/dashboard/l4-proxies` | `/dashboard/proxies/tcp-udp` | `_dashboard/l4-proxies/index` |
| `/dashboard/l4-proxies/new` | `/dashboard/proxies/tcp-udp/new` | `_dashboard/l4-proxies/new` |
| `/dashboard/l4-proxies/$l4ProxyId` | `/dashboard/proxies/tcp-udp/$l4ProxyId` | `_dashboard/l4-proxies/$l4ProxyId` |
| `/dashboard/acl` | `/dashboard/access` | `_dashboard/acl/index` |
| `/dashboard/acl/$groupId` | `/dashboard/access/$groupId` | `_dashboard/acl/$groupId` |
| `/dashboard/audit-logs` | `/dashboard/activity` | `_dashboard/audit-logs` |

---

## Task 1: Rename routes in the router (paths + redirects)

**Files:**
- Modify: `ui/src/lib/router.tsx`

**Interfaces:**
- Produces: route paths per the change table; redirect routes from each old path.

- [ ] **Step 1: Change the `path:` strings**

In `ui/src/lib/router.tsx`, edit these route definitions' `path` values (leave their `getParentRoute`, `component`, and variable names unchanged):
- `l4ProxiesRoute`: `path: '/l4-proxies'` → `path: '/proxies/tcp-udp'`
- `l4ProxyCreateRoute`: `path: '/l4-proxies/new'` → `path: '/proxies/tcp-udp/new'`
- `l4ProxyDetailRoute`: `path: '/l4-proxies/$l4ProxyId'` → `path: '/proxies/tcp-udp/$l4ProxyId'`
- `aclRoute`: `path: '/acl'` → `path: '/access'`
- `aclGroupDetailRoute`: `path: '/acl/$groupId'` → `path: '/access/$groupId'`
- `auditLogsRoute`: `path: '/audit-logs'` → `path: '/activity'`

- [ ] **Step 2: Add redirect routes for the old paths**

Add the `redirect` import to the existing import from `@tanstack/react-router` (it already imports `createRoute, createRouter, lazyRouteComponent`):

```tsx
import { createRoute, createRouter, lazyRouteComponent, redirect } from '@tanstack/react-router';
```

Then add these redirect routes (place after `aclGroupDetailRoute`, before `themePreviewRoute`):

```tsx
const l4RedirectRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/l4-proxies',
  beforeLoad: () => {
    throw redirect({ to: '/dashboard/proxies/tcp-udp' });
  },
});

const aclRedirectRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/acl',
  beforeLoad: () => {
    throw redirect({ to: '/dashboard/access' });
  },
});

const auditRedirectRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/audit-logs',
  beforeLoad: () => {
    throw redirect({ to: '/dashboard/activity' });
  },
});
```

- [ ] **Step 3: Register the redirect routes in the route tree**

In the `dashboardRoute.addChildren([...])` array at the bottom, add `l4RedirectRoute, aclRedirectRoute, auditRedirectRoute` alongside the existing dashboard children.

- [ ] **Step 4: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/lib/router.tsx
git commit -m "refactor(ui): rename routes — l4-proxies→proxies/tcp-udp, acl→access, audit-logs→activity (+redirects)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Update all internal navigation references to the new paths

**Files (exact edits below):**
- Modify: `ui/src/routes/_dashboard/l4-proxies/new.tsx`, `ui/src/routes/_dashboard/l4-proxies/$l4ProxyId.tsx`, `ui/src/routes/_dashboard/l4-proxies/index.tsx`
- Modify: `ui/src/routes/_dashboard/acl/index.tsx`, `ui/src/routes/_dashboard/acl/$groupId.tsx`
- Modify: `ui/src/components/proxy/proxy-data-grid.tsx`, `ui/src/components/proxy/forms/acl-selector.tsx`, `ui/src/components/acl/acl-group-form-modal.tsx`
- Modify: `ui/src/routes/_dashboard/index.tsx`

**Interfaces:**
- Consumes: route paths from Task 1.
- Produces: all `to`/`from`/`navigate`/`Link` references resolve to the new URLs.

> Mechanical rule: replace every routing string per the change table. Do NOT touch API strings (`api.get('l4-proxies/...')`, query keys `['l4-proxies', ...]`, the `@/components/audit-logs` import).

- [ ] **Step 1: l4-proxies → proxies/tcp-udp references**

Replace in these files (string → string):
- `'/dashboard/l4-proxies/$l4ProxyId'` → `'/dashboard/proxies/tcp-udp/$l4ProxyId'` (in `l4-proxies/new.tsx`, `l4-proxies/$l4ProxyId.tsx` (incl. the `useParams({ from: ... })`), `l4-proxies/index.tsx`)
- `'/dashboard/l4-proxies/new'` → `'/dashboard/proxies/tcp-udp/new'` (in `l4-proxies/index.tsx:250`, `proxy-data-grid.tsx:224`, `_dashboard/index.tsx:234` and `:459`)
- `'/dashboard/l4-proxies'` → `'/dashboard/proxies/tcp-udp'` (in `l4-proxies/new.tsx` (×3), `l4-proxies/$l4ProxyId.tsx` (×4), `_dashboard/index.tsx:278`)

- [ ] **Step 2: acl → access references**

- `'/dashboard/acl/$groupId'` → `'/dashboard/access/$groupId'` (in `acl/index.tsx:108`, `acl/$groupId.tsx` `useParams({ from: ... })`, `acl-group-form-modal.tsx:108`)
- `'/dashboard/acl'` → `'/dashboard/access'` (in `acl/$groupId.tsx` (×3 navigate), `acl-selector.tsx:138` Link `to`)
- `_dashboard/index.tsx:346`: `` `/dashboard/acl/${log.resource_id}` `` → `` `/dashboard/access/${log.resource_id}` ``

- [ ] **Step 3: audit-logs → activity references**

- `_dashboard/index.tsx:425`: `<Button ... render={<Link to="/dashboard/audit-logs" />}>` → `to="/dashboard/activity"`

- [ ] **Step 4: Verify none remain (except API/imports)**

Run: `grep -rn "dashboard/l4-proxies\|dashboard/acl\|dashboard/audit-logs" ui/src` → expect **no matches**. (If any remain, they're route refs — fix them. The `@/components/audit-logs` import has no `dashboard/` prefix so it won't match.)

- [ ] **Step 5: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/routes ui/src/components
git commit -m "refactor(ui): update internal links to new proxies/access/activity routes

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: ProxiesTabs component (HTTP / TCP-UDP)

**Files:**
- Create: `ui/src/components/layout/proxies-tabs.tsx`
- Create: `ui/src/components/layout/proxies-tabs.test.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/index.tsx` (render the tab bar)
- Modify: `ui/src/routes/_dashboard/l4-proxies/index.tsx` (render the tab bar)

**Interfaces:**
- Produces: `ProxiesTabs({ active }: { active: 'http' | 'tcp-udp' })` — a two-item segmented nav (links to `/dashboard/proxies` and `/dashboard/proxies/tcp-udp`) highlighting the active one.

- [ ] **Step 1: Write the failing test `proxies-tabs.test.tsx`**

```tsx
import { render, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';

vi.mock('@tanstack/react-router', () => ({
  Link: ({ to, children, ...props }: any) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));

import { ProxiesTabs } from './proxies-tabs';

test('renders both tabs with correct hrefs', () => {
  render(<ProxiesTabs active="http" />);
  expect(screen.getByRole('link', { name: /HTTP/i })).toHaveAttribute('href', '/dashboard/proxies');
  expect(screen.getByRole('link', { name: /TCP\/UDP/i })).toHaveAttribute(
    'href',
    '/dashboard/proxies/tcp-udp',
  );
});

test('marks the active tab with aria-current', () => {
  render(<ProxiesTabs active="tcp-udp" />);
  expect(screen.getByRole('link', { name: /TCP\/UDP/i })).toHaveAttribute('aria-current', 'page');
  expect(screen.getByRole('link', { name: /HTTP/i })).not.toHaveAttribute('aria-current');
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir ui test run src/components/layout/proxies-tabs.test.tsx`
Expected: FAIL — cannot resolve `./proxies-tabs`.

- [ ] **Step 3: Implement `proxies-tabs.tsx`**

```tsx
import { cn } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { Globe, Network } from 'lucide-react';

const tabs = [
  { key: 'http', label: 'HTTP', to: '/dashboard/proxies', icon: Globe },
  { key: 'tcp-udp', label: 'TCP/UDP', to: '/dashboard/proxies/tcp-udp', icon: Network },
] as const;

export function ProxiesTabs({ active }: { active: 'http' | 'tcp-udp' }) {
  return (
    <div className="bg-muted/60 inline-flex items-center gap-1 rounded-lg p-1">
      {tabs.map(({ key, label, to, icon: Icon }) => {
        const isActive = key === active;
        return (
          <Link
            key={key}
            to={to}
            aria-current={isActive ? 'page' : undefined}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              isActive
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            <Icon className="size-4" />
            {label}
          </Link>
        );
      })}
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm --dir ui test run src/components/layout/proxies-tabs.test.tsx`
Expected: 2 passed.

- [ ] **Step 5: Render the tab bar atop both list pages**

In `ui/src/routes/_dashboard/proxies/index.tsx`: import `ProxiesTabs` (`import { ProxiesTabs } from '@/components/layout/proxies-tabs';`) and render `<ProxiesTabs active="http" />` at the top of the page's returned JSX (above the existing header/list, inside the outermost container, with a `mb-4` wrapper if needed for spacing).

In `ui/src/routes/_dashboard/l4-proxies/index.tsx`: same, with `<ProxiesTabs active="tcp-udp" />`.

(Keep the existing page content; just prepend the tab bar.)

- [ ] **Step 6: Build + test + commit**

Run: `pnpm --dir ui build` → success. `pnpm --dir ui test run` → all pass.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/layout/proxies-tabs.tsx ui/src/components/layout/proxies-tabs.test.tsx ui/src/routes/_dashboard/proxies/index.tsx ui/src/routes/_dashboard/l4-proxies/index.tsx
git commit -m "feat(ui): unify proxies under HTTP/TCP-UDP tab bar

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Update the sidebar nav to the new IA (5 items)

**Files:**
- Modify: `ui/src/components/layout/sidebar.tsx` (the `navItems` array, ~lines 68-99)

**Interfaces:**
- Consumes: new route paths.
- Produces: sidebar with Dashboard, Proxies, Access, Activity, Settings (TCP/UDP folded into the Proxies section via the tab bar).

- [ ] **Step 1: Replace the `navItems` array**

```tsx
const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/dashboard', icon: <Home className="size-4" /> },
  { label: 'Proxies', path: '/dashboard/proxies', icon: <Globe className="size-4" /> },
  { label: 'Access', path: '/dashboard/access', icon: <Shield className="size-4" /> },
  { label: 'Activity', path: '/dashboard/activity', icon: <ClipboardList className="size-4" /> },
  { label: 'Settings', path: '/dashboard/settings', icon: <Settings className="size-4" /> },
];
```

Remove the now-unused `Network` icon import if it is no longer referenced in the file (check first — it may still be used elsewhere). The TCP/UDP nav item is intentionally dropped (reachable via the Proxies tab bar).

- [ ] **Step 2: Make "Proxies" stay active on its sub-routes**

The active check is `location.pathname === item.path`. So `/dashboard/proxies/tcp-udp` would NOT highlight "Proxies". Change the active logic for nav highlighting to match sub-paths: for the Proxies item, treat it active when the path starts with `/dashboard/proxies`. Update the render map's `isActive`:

```tsx
{navItems.map((item) => {
  const isActive =
    item.path === '/dashboard'
      ? location.pathname === '/dashboard' || location.pathname === '/dashboard/'
      : location.pathname === item.path || location.pathname.startsWith(`${item.path}/`);
  return (
    <SidebarMenuItem key={item.path}>
      <SidebarMenuButton render={<Link to={item.path} />} isActive={isActive}>
        {item.icon}
        <span>{item.label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
})}
```

(The Dashboard special-case prevents it from matching every `/dashboard/*` path.)

- [ ] **Step 3: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/layout/sidebar.tsx
git commit -m "feat(ui): sidebar nav to new IA (Proxies/Access/Activity)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Full verification

**Files:** none (verification).

- [ ] **Step 1: No stale route refs**

Run: `grep -rn "dashboard/l4-proxies\|dashboard/acl\|dashboard/audit-logs" ui/src` → empty.

- [ ] **Step 2: Gates**

Run: `pnpm --dir ui build` → success · `pnpm --dir ui test run` → all pass · `pnpm --dir ui check` → clean.

- [ ] **Step 3: Note for controller's `verify` pass**

Interactive smoke (not part of this task): sidebar shows 5 items; Proxies page shows HTTP/TCP-UDP tab bar and switches lists; "Proxies" stays highlighted on `/proxies/tcp-udp`; Access + Activity load; old URLs (`/dashboard/l4-proxies`, `/dashboard/acl`, `/dashboard/audit-logs`) redirect to the new ones; create/edit flows for HTTP + TCP/UDP navigate correctly.

## Done criteria
- New URLs live; old URLs redirect; `grep` for old route paths is empty.
- Sidebar = Dashboard · Proxies · Access · Activity · Settings; Proxies has the HTTP/TCP-UDP tab bar.
- `pnpm --dir ui build` + `test run` + `check` all green.
- **Next M0b plans:** Shell QoL (top bar: breadcrumbs, sync/apply status, theme toggle, ⌘K palette) and Form Foundation (RHF+Zod + auth forms).
