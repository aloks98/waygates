# M1 — Dashboard Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the dashboard into a scannable control-plane overview — system status, three key stat cards, a fleet-composition donut, quick actions, a recent-activity timeline, and a strong first-run onboarding state — using only honest existing data (no traffic/time-series).

**Architecture:** Extract the dashboard's pure formatting helpers into a tested `lib/dashboard-format.ts`. Build focused components under `components/dashboard/` (status bar, stat cards, fleet composition, quick actions, activity timeline, empty state). Compose them in `_dashboard/index.tsx` with a first-run gate (`total === 0` or `!user_setup_complete` → onboarding card). Dogfood rnui `StatusIndicator`, `StatCard`, `PieChart` (donut), `Timeline`, `EmptyState`. Reserve nothing for B2/B3 — those land with their pipelines.

**Tech Stack:** rnui (`StatusIndicator`, `StatCard`, `PieChart`, `Timeline*`, `EmptyState`, `Card`, `Button`, `Tooltip`, `Skeleton`), TanStack Router/Query, `date-fns`, lucide, Vitest.

## Global Constraints

- pnpm; UI in `ui/`. Run `pnpm --dir ui <script>`. Branch `feat/rnui-redesign-program`.
- **Honest data only.** No traffic/time-series charts (deferred to B3). The only chart is the **fleet-composition donut** (≤5 categories: Reverse/Redirect/Static/TCP/UDP). Per UX rules it MUST have a legend with **exact counts** (not color-alone).
- Composition prop in rnui is **`render`**, not `asChild` (e.g. `<Button render={<Link to="…" />}>`).
- Status colors: healthy 🟢 (`StatusIndicator state="active"`), unhealthy 🔴 (`down`), syncing 🟡 (`fixing`), unknown (`idle`).
- Routes (post-M0b): HTTP `/dashboard/proxies` (+`/new`), TCP/UDP `/dashboard/proxies/tcp-udp` (+`/new`), Access `/dashboard/access`, Activity `/dashboard/activity`.
- Verification gates: `pnpm --dir ui build` + `pnpm --dir ui check:fix && pnpm --dir ui check` (clean; pre-existing oxlint warnings OK) + `pnpm --dir ui test run`. **No `tsc` gate.** Interactive visual smoke in a final `verify` pass.
- Commits: Conventional Commits + trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

### rnui APIs (confirmed)
- `StatCard({ title, value, description?, icon?, trend?, className? })`
- `StatusIndicator({ state: 'active'|'down'|'fixing'|'idle', label?, size?, className? })`
- `PieChart({ data: {name,value}[], donut?, showLegend?, showLabels?, height?, ... })`
- `Timeline` + `TimelineItem({step})` / `TimelineIndicator` / `TimelineSeparator` / `TimelineContent` / `TimelineTitle` / `TimelineDate`
- `EmptyState({ icon?, title, description?, action?, className? })`
- Data hooks: `useDashboardData()` → `{ proxyStats, l4ProxyStats, recentProxies, recentActivity, isLoading }`; `useSyncStatus()`, `useHealthStatus()`, `useAppStatus()`; `useACLGroups({ page, limit })` → `{ total, isLoading, ... }`.

---

## Task 1: Extract + test dashboard format helpers

**Files:**
- Create: `ui/src/lib/dashboard-format.ts`
- Create: `ui/src/lib/dashboard-format.test.ts`

**Interfaces:**
- Produces: `formatUptime(raw: string): string`; `getActionLabel(action: string): string`; `getActionColor(action: string): string`; `getActivityLink(log: AuditLog): string | null`; `buildCompositionData(proxyStats?, l4Stats?): { name: string; value: number }[]`.

- [ ] **Step 1: Write the failing test `dashboard-format.test.ts`**

```ts
import { expect, test } from 'vitest';

import { buildCompositionData, formatUptime, getActionColor, getActivityLink } from './dashboard-format';

test('buildCompositionData maps types/protocols and drops zeros', () => {
  expect(
    buildCompositionData(
      { total: 9, active: 7, inactive: 2, by_type: { reverse_proxy: 6, redirect: 0, static: 1 } },
      { total_proxies: 3, active_proxies: 3, tcp_proxies: 2, udp_proxies: 1, total_routes: 4, total_upstreams: 5 },
    ),
  ).toEqual([
    { name: 'Reverse', value: 6 },
    { name: 'Static', value: 1 },
    { name: 'TCP', value: 2 },
    { name: 'UDP', value: 1 },
  ]);
});

test('buildCompositionData with no data is empty', () => {
  expect(buildCompositionData(undefined, undefined)).toEqual([]);
});

test('formatUptime parses a Go duration', () => {
  expect(formatUptime('49h5m10.048s')).toMatch(/2 days/);
});

test('getActionColor flags destructive and success', () => {
  expect(getActionColor('proxy.delete')).toContain('destructive');
  expect(getActionColor('proxy.create')).toContain('green');
});

test('getActivityLink builds resource links and skips deletes', () => {
  expect(getActivityLink({ action: 'proxy.update', resource_type: 'proxy', resource_id: 5 } as any)).toBe(
    '/dashboard/proxies/5',
  );
  expect(getActivityLink({ action: 'acl_group.update', resource_type: 'acl', resource_id: 2 } as any)).toBe(
    '/dashboard/access/2',
  );
  expect(getActivityLink({ action: 'proxy.delete', resource_type: 'proxy', resource_id: 5 } as any)).toBeNull();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir ui test run src/lib/dashboard-format.test.ts`
Expected: FAIL — cannot resolve `./dashboard-format`.

- [ ] **Step 3: Implement `ui/src/lib/dashboard-format.ts`**

Lift `formatUptime`, `getActionLabel`, `getActionColor`, `getActivityLink` verbatim from the current `ui/src/routes/_dashboard/index.tsx` (lines 30-42, 47-78, 343-354) and add `buildCompositionData`:

```ts
import { formatDuration, intervalToDuration } from 'date-fns';

import type { AuditLog } from '@/types/audit';
import type { ProxyStats } from '@/types/api';
import type { L4ProxyStats } from '@/types/l4-proxy';

export function formatUptime(raw: string): string {
  const hours = raw.match(/(\d+)h/);
  const minutes = raw.match(/(\d+)m/);
  const seconds = raw.match(/([\d.]+)s/);
  const totalSeconds =
    (hours ? Number.parseInt(hours[1], 10) * 3600 : 0) +
    (minutes ? Number.parseInt(minutes[1], 10) * 60 : 0) +
    (seconds ? Math.floor(Number.parseFloat(seconds[1])) : 0);
  const duration = intervalToDuration({ start: 0, end: totalSeconds * 1000 });
  return formatDuration(duration, { format: ['days', 'hours', 'minutes'], delimiter: ' ' });
}

export function getActionLabel(action: string): string {
  const labels: Record<string, string> = {
    'proxy.create': 'created a proxy',
    'proxy.update': 'updated a proxy',
    'proxy.delete': 'deleted a proxy',
    'proxy.enable': 'enabled a proxy',
    'proxy.disable': 'disabled a proxy',
    'auth.login': 'signed in',
    'auth.logout': 'signed out',
    'auth.register': 'registered',
    'auth.password_change': 'changed password',
    'auth.login_failed': 'failed login attempt',
    'settings.update': 'updated settings',
    'sync.started': 'sync started',
    'sync.completed': 'sync completed',
    'sync.failed': 'sync failed',
    'system.startup': 'system started',
    'caddy.reload': 'proxy server reloaded',
    'acl_group.create': 'created ACL group',
    'acl_group.update': 'updated ACL group',
    'acl_group.delete': 'deleted ACL group',
  };
  return labels[action] ?? action.replace(/[._]/g, ' ');
}

export function getActionColor(action: string): string {
  if (action.includes('delete') || action.includes('failed')) return 'text-destructive';
  if (action.includes('create') || action.includes('enable') || action === 'sync.completed')
    return 'text-green-600 dark:text-green-500';
  if (action.includes('disable')) return 'text-amber-600 dark:text-amber-500';
  return 'text-muted-foreground';
}

export function getActivityLink(log: AuditLog): string | null {
  if (!log.resource_id || log.action.includes('delete')) return null;
  switch (log.resource_type) {
    case 'proxy':
      return `/dashboard/proxies/${log.resource_id}`;
    case 'acl':
      return `/dashboard/access/${log.resource_id}`;
    default:
      return null;
  }
}

export function buildCompositionData(
  proxyStats?: ProxyStats,
  l4Stats?: L4ProxyStats,
): { name: string; value: number }[] {
  const t = proxyStats?.by_type ?? {};
  return [
    { name: 'Reverse', value: t.reverse_proxy ?? 0 },
    { name: 'Redirect', value: t.redirect ?? 0 },
    { name: 'Static', value: t.static ?? 0 },
    { name: 'TCP', value: l4Stats?.tcp_proxies ?? 0 },
    { name: 'UDP', value: l4Stats?.udp_proxies ?? 0 },
  ].filter((d) => d.value > 0);
}
```
(If `ProxyStats`/`L4ProxyStats` aren't exported from those paths, check `@/types/api` and `@/types/l4-proxy` and import the correct names.)

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm --dir ui test run src/lib/dashboard-format.test.ts`
Expected: 5 passed.

- [ ] **Step 5: Commit**

```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/lib/dashboard-format.ts ui/src/lib/dashboard-format.test.ts
git commit -m "feat(ui): extract + test dashboard format helpers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: SystemStatusBar component

**Files:**
- Create: `ui/src/components/dashboard/system-status-bar.tsx`

**Interfaces:**
- Consumes: `formatUptime` (Task 1), `useSyncStatus`/`useHealthStatus`/`useAppStatus`.
- Produces: `SystemStatusBar()` — Caddy/DB/Sync `StatusIndicator`s + uptime/version + an "Apply now" action.

- [ ] **Step 1: Implement**

```tsx
import { Button, Skeleton, StatusIndicator, Tooltip, TooltipContent, TooltipTrigger } from '@e412/rnui-react';
import { formatDistanceToNow } from 'date-fns';
import { RefreshCw } from 'lucide-react';

import { formatUptime } from '@/lib/dashboard-format';
import { useAppStatus, useHealthStatus, useSyncStatus } from '@/hooks';

export function SystemStatusBar() {
  const { syncStatus, triggerSync, isSyncing, isLoading: sl } = useSyncStatus();
  const { health, isLoading: hl } = useHealthStatus();
  const { appStatus, isLoading: al } = useAppStatus();

  if (sl || hl || al) {
    return (
      <div className="flex flex-wrap items-center gap-4">
        <Skeleton className="h-5 w-28" />
        <Skeleton className="h-5 w-28" />
        <Skeleton className="h-5 w-32" />
      </div>
    );
  }

  const caddy = appStatus?.caddy_status === 'healthy' ? 'active' : 'down';
  const db = health?.components?.database === 'healthy' ? 'active' : 'down';
  const syncState = isSyncing ? 'fixing' : syncStatus?.last_sync_success === false ? 'down' : 'active';
  const lastSync = syncStatus?.last_sync_time
    ? formatDistanceToNow(new Date(syncStatus.last_sync_time), { addSuffix: true })
    : 'never';

  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
      <StatusIndicator state={caddy} size="sm" label="Caddy" />
      <StatusIndicator state={db} size="sm" label="Database" />
      <Tooltip>
        <TooltipTrigger render={<StatusIndicator state={syncState} size="sm" label={isSyncing ? 'Syncing…' : `Synced ${lastSync}`} />} />
        <TooltipContent>
          {isSyncing
            ? 'Applying configuration to Caddy…'
            : syncStatus?.last_sync_success === false
              ? `Last sync failed${syncStatus?.last_error ? `: ${syncStatus.last_error}` : ''}`
              : 'Configuration is applied'}
        </TooltipContent>
      </Tooltip>
      {health?.uptime && <span className="text-muted-foreground">Up {formatUptime(health.uptime)}</span>}
      {health?.version && <span className="text-muted-foreground">v{health.version}</span>}
      <Button variant="outline" size="sm" className="ml-auto gap-1.5" disabled={isSyncing} onClick={() => triggerSync()}>
        <RefreshCw className={`size-3.5 ${isSyncing ? 'animate-spin' : ''}`} />
        Apply now
      </Button>
    </div>
  );
}
```

- [ ] **Step 2: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/dashboard/system-status-bar.tsx
git commit -m "feat(ui): dashboard system status bar (StatusIndicator + apply now)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: DashboardStatCards (3 cards)

**Files:**
- Create: `ui/src/components/dashboard/dashboard-stat-cards.tsx`

**Interfaces:**
- Consumes: `useDashboardData` data (passed as props) + `useACLGroups` (for Access groups count).
- Produces: `DashboardStatCards({ proxyStats, l4ProxyStats, isLoading })` — 3 `StatCard`s: Total proxies, Active, Access groups.

- [ ] **Step 1: Implement**

```tsx
import { Skeleton, StatCard } from '@e412/rnui-react';
import { CheckCircle2, Server, Shield } from 'lucide-react';

import { useACLGroups } from '@/hooks';
import type { ProxyStats } from '@/types/api';
import type { L4ProxyStats } from '@/types/l4-proxy';

export function DashboardStatCards({
  proxyStats,
  l4ProxyStats,
  isLoading,
}: {
  proxyStats?: ProxyStats;
  l4ProxyStats?: L4ProxyStats;
  isLoading: boolean;
}) {
  const { total: aclTotal, isLoading: aclLoading } = useACLGroups({ page: 1, limit: 1 });

  if (isLoading || aclLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-3">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-24 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  const totalProxies = (proxyStats?.total ?? 0) + (l4ProxyStats?.total_proxies ?? 0);
  const activeProxies = (proxyStats?.active ?? 0) + (l4ProxyStats?.active_proxies ?? 0);

  return (
    <div className="grid gap-4 sm:grid-cols-3">
      <StatCard
        title="Total proxies"
        value={totalProxies}
        description={`${proxyStats?.total ?? 0} HTTP · ${l4ProxyStats?.total_proxies ?? 0} TCP/UDP`}
        icon={<Server className="size-4" />}
      />
      <StatCard
        title="Active"
        value={`${activeProxies} / ${totalProxies}`}
        description={`${totalProxies - activeProxies} inactive`}
        icon={<CheckCircle2 className="size-4" />}
      />
      <StatCard
        title="Access groups"
        value={aclTotal ?? 0}
        description="Auth & IP rules"
        icon={<Shield className="size-4" />}
      />
    </div>
  );
}
```
(Confirm `useACLGroups` returns `total`; if the field differs, adapt. If `StatCard` lacks a `description` prop, drop it.)

- [ ] **Step 2: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/dashboard/dashboard-stat-cards.tsx
git commit -m "feat(ui): dashboard stat cards (total/active/access groups)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: FleetComposition donut

**Files:**
- Create: `ui/src/components/dashboard/fleet-composition.tsx`

**Interfaces:**
- Consumes: `buildCompositionData` (Task 1).
- Produces: `FleetComposition({ proxyStats, l4ProxyStats, isLoading })` — a donut + a counts legend (not color-alone).

- [ ] **Step 1: Implement**

```tsx
import { Card, CardContent, CardHeader, CardTitle, PieChart, Skeleton } from '@e412/rnui-react';

import { buildCompositionData } from '@/lib/dashboard-format';
import type { ProxyStats } from '@/types/api';
import type { L4ProxyStats } from '@/types/l4-proxy';

const CHART_COLORS = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
];

export function FleetComposition({
  proxyStats,
  l4ProxyStats,
  isLoading,
}: {
  proxyStats?: ProxyStats;
  l4ProxyStats?: L4ProxyStats;
  isLoading: boolean;
}) {
  const data = buildCompositionData(proxyStats, l4ProxyStats);
  const total = data.reduce((sum, d) => sum + d.value, 0);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Fleet composition</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-48 w-full" />
        ) : total === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">No proxies to summarize.</p>
        ) : (
          <div className="flex flex-col items-center gap-4 sm:flex-row">
            <div className="h-44 w-44 shrink-0">
              <PieChart data={data} donut showLegend={false} height={176} />
            </div>
            <ul className="flex-1 space-y-1.5">
              {data.map((d, i) => (
                <li key={d.name} className="flex items-center justify-between text-sm">
                  <span className="flex items-center gap-2">
                    <span
                      className="inline-block size-2.5 rounded-sm"
                      style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }}
                    />
                    {d.name}
                  </span>
                  <span className="font-medium tabular-nums">{d.value}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```
(If `PieChart` doesn't honor `var(--chart-*)` for slice colors out of the box, it auto-themes — keep the legend dots matching by relying on its default palette order; if the palette differs, pass a `colors`/`option` prop to align, or accept rnui's default and drop the manual dot colors. Verify the rendered colors match the legend during the visual pass.)

- [ ] **Step 2: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/dashboard/fleet-composition.tsx
git commit -m "feat(ui): fleet composition donut + counts legend

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: DashboardQuickActions

**Files:**
- Create: `ui/src/components/dashboard/dashboard-quick-actions.tsx`

**Interfaces:**
- Produces: `DashboardQuickActions()` — New HTTP / New TCP-UDP / New Access group action buttons.

- [ ] **Step 1: Implement**

```tsx
import { Button, Card, CardContent, CardHeader, CardTitle } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { Globe, Network, Shield } from 'lucide-react';

export function DashboardQuickActions() {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Quick actions</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <Button variant="outline" className="justify-start" render={<Link to="/dashboard/proxies/new" />}>
          <Globe className="size-4" />
          New HTTP proxy
        </Button>
        <Button variant="outline" className="justify-start" render={<Link to="/dashboard/proxies/tcp-udp/new" />}>
          <Network className="size-4" />
          New TCP/UDP proxy
        </Button>
        <Button variant="outline" className="justify-start" render={<Link to="/dashboard/access" />}>
          <Shield className="size-4" />
          New access group
        </Button>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 2: Build + commit**

```bash
pnpm --dir ui build && pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/dashboard/dashboard-quick-actions.tsx
git commit -m "feat(ui): dashboard quick actions card

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: ActivityTimeline

**Files:**
- Create: `ui/src/components/dashboard/activity-timeline.tsx`

**Interfaces:**
- Consumes: `getActionLabel`, `getActionColor`, `getActivityLink` (Task 1).
- Produces: `ActivityTimeline({ activity, isLoading })` — recent audit events as an rnui `Timeline`, status-colored, linking to resources.

- [ ] **Step 1: Implement**

```tsx
import {
  Button,
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
  Skeleton,
  Timeline,
  TimelineContent,
  TimelineDate,
  TimelineIndicator,
  TimelineItem,
  TimelineSeparator,
  TimelineTitle,
} from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { formatDistanceToNow } from 'date-fns';
import { Activity, ArrowRight, CheckCircle2, XCircle } from 'lucide-react';

import { getActionColor, getActionLabel, getActivityLink } from '@/lib/dashboard-format';
import type { AuditLog } from '@/types/audit';

export function ActivityTimeline({ activity, isLoading }: { activity: AuditLog[]; isLoading: boolean }) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Recent activity</CardTitle>
        <CardAction>
          <Button variant="ghost" size="sm" render={<Link to="/dashboard/activity" />}>
            View all
            <ArrowRight className="ml-1 size-3.5" />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="pt-0">
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3, 4].map((i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : activity.length === 0 ? (
          <div className="flex flex-col items-center py-6 text-center">
            <Activity className="size-5 text-muted-foreground/40" />
            <p className="mt-2 text-sm text-muted-foreground">No recent activity</p>
            <p className="mt-1 max-w-[220px] text-xs text-muted-foreground/70">
              Creating proxies and changing settings will appear here.
            </p>
          </div>
        ) : (
          <Timeline orientation="vertical">
            {activity.map((log, i) => {
              const actor = log.user?.username ?? 'system';
              const link = getActivityLink(log);
              const failed = log.status === 'failure';
              const title = (
                <span className="text-sm leading-snug">
                  <span className="font-medium">{actor}</span>{' '}
                  <span className={getActionColor(log.action)}>{getActionLabel(log.action)}</span>
                  {log.resource_name && <span className="text-muted-foreground"> · {log.resource_name}</span>}
                </span>
              );
              return (
                <TimelineItem key={log.id} step={i + 1}>
                  <TimelineIndicator>
                    {failed ? (
                      <XCircle className="size-3.5 text-destructive" />
                    ) : (
                      <CheckCircle2 className="size-3.5 text-green-500" />
                    )}
                  </TimelineIndicator>
                  <TimelineSeparator />
                  <TimelineContent>
                    <TimelineTitle>
                      {link ? (
                        <Link to={link} className="hover:underline">
                          {title}
                        </Link>
                      ) : (
                        title
                      )}
                    </TimelineTitle>
                    <TimelineDate>{formatDistanceToNow(new Date(log.created_at), { addSuffix: true })}</TimelineDate>
                  </TimelineContent>
                </TimelineItem>
              );
            })}
          </Timeline>
        )}
      </CardContent>
    </Card>
  );
}
```
(If rnui's `Timeline` requires a `defaultValue`/specific `step` semantics that render oddly, adjust the `step` usage to match its API — verify in the visual pass. The `TimelineItem`/`TimelineIndicator`/etc. names are confirmed exported.)

- [ ] **Step 2: Build + commit**

```bash
pnpm --dir ui build && pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/dashboard/activity-timeline.tsx
git commit -m "feat(ui): recent-activity timeline

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Onboarding empty state + compose the dashboard

**Files:**
- Create: `ui/src/components/dashboard/dashboard-empty-state.tsx`
- Rewrite: `ui/src/routes/_dashboard/index.tsx`

**Interfaces:**
- Consumes: all Task 2-6 components + `useDashboardData`, `useAppStatus`, `useAuthStore`.
- Produces: `DashboardEmptyState()`; the composed `DashboardIndex()`.

- [ ] **Step 1: Implement `dashboard-empty-state.tsx`**

```tsx
import { Button, Card, CardContent, EmptyState } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { Globe, Network, Rocket } from 'lucide-react';

export function DashboardEmptyState() {
  return (
    <Card>
      <CardContent className="py-10">
        <EmptyState
          icon={<Rocket className="size-8" />}
          title="Welcome to Waygates"
          description="You don't have any proxies yet. Create your first one to start routing traffic — Waygates handles HTTPS automatically."
          action={
            <div className="flex flex-wrap justify-center gap-2">
              <Button render={<Link to="/dashboard/proxies/new" />}>
                <Globe className="size-4" />
                Create HTTP proxy
              </Button>
              <Button variant="outline" render={<Link to="/dashboard/proxies/tcp-udp/new" />}>
                <Network className="size-4" />
                Create TCP/UDP proxy
              </Button>
            </div>
          }
        />
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 2: Rewrite `ui/src/routes/_dashboard/index.tsx`**

```tsx
import { useAppStatus, useDashboardData } from '@/hooks';
import { useAuthStore } from '@/stores/auth';

import { ActivityTimeline } from '@/components/dashboard/activity-timeline';
import { DashboardEmptyState } from '@/components/dashboard/dashboard-empty-state';
import { DashboardQuickActions } from '@/components/dashboard/dashboard-quick-actions';
import { DashboardStatCards } from '@/components/dashboard/dashboard-stat-cards';
import { FleetComposition } from '@/components/dashboard/fleet-composition';
import { SystemStatusBar } from '@/components/dashboard/system-status-bar';

export function DashboardIndex() {
  const { user } = useAuthStore();
  const { proxyStats, l4ProxyStats, recentActivity, isLoading } = useDashboardData();
  const { appStatus } = useAppStatus();

  const totalProxies = (proxyStats?.total ?? 0) + (l4ProxyStats?.total_proxies ?? 0);
  const isFirstRun = !isLoading && (totalProxies === 0 || appStatus?.user_setup_complete === false);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Welcome back, {user?.name || 'User'}</h1>
        <div className="mt-3">
          <SystemStatusBar />
        </div>
      </div>

      {isFirstRun ? (
        <DashboardEmptyState />
      ) : (
        <>
          <DashboardStatCards proxyStats={proxyStats} l4ProxyStats={l4ProxyStats} isLoading={isLoading} />
          <div className="grid gap-6 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <FleetComposition proxyStats={proxyStats} l4ProxyStats={l4ProxyStats} isLoading={isLoading} />
            </div>
            <div>
              <DashboardQuickActions />
            </div>
          </div>
        </>
      )}

      <ActivityTimeline activity={recentActivity} isLoading={isLoading} />
    </div>
  );
}
```
Delete the old sub-components and helpers now living in `dashboard-format.ts` (the whole old file body is replaced by the above). Keep the `page-enter` animation wrapper behavior (the dashboard layout route already wraps children with `page-enter`), so no per-element animation is needed.

- [ ] **Step 3: Build + commit**

Run: `pnpm --dir ui build` → success. `pnpm --dir ui test run` → all pass.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/dashboard/dashboard-empty-state.tsx ui/src/routes/_dashboard/index.tsx
git commit -m "feat(ui): compose redesigned dashboard + first-run onboarding

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Verification

**Files:** none.

- [ ] **Step 1: Gates** — `pnpm --dir ui build` → success · `pnpm --dir ui test run` → all pass · `pnpm --dir ui check` → clean.
- [ ] **Step 2: No dead code** — `grep -rn "SystemStatusStrip\|ProxyOverview\|ActivityItem" ui/src` → only the new files (old sub-components removed).
- [ ] **Step 3: Note for controller's `verify` pass** — fresh install (0 proxies) shows the onboarding card; with proxies: status bar (healthy/syncing/failed states), 3 stat cards, donut whose slice colors match the counts legend, quick actions navigate, activity timeline renders status-colored items linking to resources; light + dark both look right; "Apply now" triggers a sync.

## Done criteria
- Dashboard redesigned: status bar + 3 stat cards + composition donut + quick actions + activity timeline; first-run onboarding when empty.
- Only the honest composition donut chart; no traffic/time-series; no B2/B3 placeholders.
- `pnpm --dir ui build` + `test run` + `check` green.
- **Next:** M2 — Proxies (HTTP) redesign, or backend pipeline B1 (config preview) to light up the proxy/dashboard "inspect generated config" affordance.
