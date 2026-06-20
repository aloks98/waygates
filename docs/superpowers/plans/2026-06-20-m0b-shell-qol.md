# M0b — Shell QoL Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the control-plane top bar — breadcrumbs, a sync/apply status indicator, a light/dark theme toggle, and a global ⌘K command palette (navigate + quick actions) — in the dashboard shell.

**Architecture:** A new `TopBar` replaces the minimal header inside `AppSidebar`'s `SidebarInset`. It composes the existing `SidebarTrigger` + `Breadcrumbs` (left) and `SyncStatus` + `ThemeToggle` + a ⌘K trigger (right). The command palette is a globally-mounted `CommandDialog` driven by a small zustand store (so the ⌘K key, the top-bar button, and palette actions all share open-state). A shared `useLogout` hook removes logout duplication between the sidebar and the palette.

**Tech Stack:** rnui (`Breadcrumb*`, `StatusIndicator`, `Command*`/`CommandDialog`, `Kbd`, `Button`, `Tooltip`), `next-themes`, zustand, TanStack Router, `useSyncStatus`, `date-fns`, lucide-react.

## Global Constraints

- pnpm; UI in `ui/`. Run `pnpm --dir ui <script>`. Branch `feat/rnui-redesign-program`.
- rnui composition uses the **`render`** prop, never `asChild` (Base UI). E.g. `BreadcrumbLink render={<Link to="…" />}`.
- Theme via `next-themes` (`useTheme`); the provider is already `attribute="class"`, `defaultTheme="dark"`, `enableSystem={false}` — so guard `useTheme()` reads with a mounted flag to avoid hydration flue.
- Verification gates: `pnpm --dir ui build` (success) + `pnpm --dir ui check:fix && pnpm --dir ui check` (clean; pre-existing oxlint warnings OK) + `pnpm --dir ui test run` (pass). **No `tsc` gate.** Interactive smoke in a final `verify` pass.
- Tests one-shot: `pnpm --dir ui test run <path>` (never bare `vitest`).
- Commits: Conventional Commits + trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

### Reference facts
- `useSyncStatus()` (from `@/hooks/use-dashboard`) returns `{ syncStatus?: { is_syncing, last_sync_time?, last_sync_success, last_error? }, isLoading, triggerSync, isSyncing }`.
- `StatusIndicator({ state, label, size, className })`, `state: 'active'|'down'|'fixing'|'idle'`.
- `CommandDialog({ open, onOpenChange, title, description, children })` (extends Dialog props).
- Auth store `useAuthStore` exposes `logout()`; the existing logout flow is `await api.post('auth/logout'); logout(); router.navigate({ to: '/login' })`.
- `AppSidebar` (`ui/src/components/layout/sidebar.tsx`) renders the header at `<SidebarInset><header className="flex h-14 items-center gap-4 border-b border-border px-4"><SidebarTrigger /></header>...`.

---

## Task 1: Shared `useLogout` hook

**Files:**
- Create: `ui/src/hooks/use-logout.ts`
- Modify: `ui/src/components/layout/sidebar.tsx` (use the hook in `AppSidebar`)

**Interfaces:**
- Produces: `useLogout(): () => Promise<void>` — posts `auth/logout`, clears the auth store, navigates to `/login`.

- [ ] **Step 1: Create `ui/src/hooks/use-logout.ts`**

```ts
import { useRouter } from '@tanstack/react-router';
import { useCallback } from 'react';

import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

export function useLogout(): () => Promise<void> {
  const router = useRouter();
  const logout = useAuthStore((s) => s.logout);
  return useCallback(async () => {
    try {
      await api.post('auth/logout');
    } catch {
      // Ignore errors; log out locally anyway.
    }
    logout();
    router.navigate({ to: '/login' });
  }, [logout, router]);
}
```

- [ ] **Step 2: Use it in `AppSidebar`**

In `ui/src/components/layout/sidebar.tsx`, in `AppSidebar`, replace the inline `handleLogout` + the `const { user, logout } = useAuthStore();` destructure so logout comes from the hook:
- Change `const { user, logout } = useAuthStore();` → `const { user } = useAuthStore();`
- Remove the `handleLogout` function body and the `router`/`logout` usage it had; add `const handleLogout = useLogout();` (keep the same name so the `onClick={handleLogout}` JSX is unchanged).
- Add the import: `import { useLogout } from '@/hooks/use-logout';`
- If `useRouter` is now unused in the file, remove it from the `@tanstack/react-router` import. (Check — it may still be used.)

- [ ] **Step 3: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/hooks/use-logout.ts ui/src/components/layout/sidebar.tsx
git commit -m "refactor(ui): extract useLogout hook

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: ThemeToggle + SyncStatus components

**Files:**
- Create: `ui/src/components/layout/theme-toggle.tsx`
- Create: `ui/src/components/layout/sync-status.tsx`

**Interfaces:**
- Produces: `ThemeToggle()` (icon button toggling light/dark) and `SyncStatus()` (status indicator + tooltip + click-to-apply).

- [ ] **Step 1: Create `ui/src/components/layout/theme-toggle.tsx`**

```tsx
import { Button } from '@e412/rnui-react';
import { Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';
import { useEffect, useState } from 'react';

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  const isDark = mounted ? resolvedTheme === 'dark' : true;

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
    >
      {isDark ? <Sun className="size-4" /> : <Moon className="size-4" />}
    </Button>
  );
}
```

- [ ] **Step 2: Create `ui/src/components/layout/sync-status.tsx`**

```tsx
import { Button, StatusIndicator, Tooltip, TooltipContent, TooltipTrigger } from '@e412/rnui-react';
import { formatDistanceToNow } from 'date-fns';

import { useSyncStatus } from '@/hooks/use-dashboard';

export function SyncStatus() {
  const { syncStatus, isSyncing, triggerSync } = useSyncStatus();

  const state = isSyncing
    ? 'fixing'
    : syncStatus?.last_error
      ? 'down'
      : syncStatus?.last_sync_success
        ? 'active'
        : 'idle';

  const label = isSyncing
    ? 'Syncing…'
    : syncStatus?.last_error
      ? 'Sync failed'
      : syncStatus?.last_sync_success
        ? 'Synced'
        : 'Not synced';

  const tooltip = isSyncing
    ? 'Applying configuration to Caddy…'
    : syncStatus?.last_error
      ? `Last sync failed: ${syncStatus.last_error}`
      : syncStatus?.last_sync_time
        ? `Last synced ${formatDistanceToNow(new Date(syncStatus.last_sync_time), { addSuffix: true })}. Click to re-apply.`
        : 'Not yet synced. Click to apply now.';

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className="gap-2"
            disabled={isSyncing}
            onClick={() => triggerSync()}
          />
        }
      >
        <StatusIndicator state={state} size="sm" label={label} />
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}
```

- [ ] **Step 3: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/layout/theme-toggle.tsx ui/src/components/layout/sync-status.tsx
git commit -m "feat(ui): theme toggle + sync/apply status indicator

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Breadcrumbs component (+ tests)

**Files:**
- Create: `ui/src/components/layout/breadcrumbs.tsx`
- Create: `ui/src/components/layout/breadcrumbs.test.tsx`

**Interfaces:**
- Produces: `Breadcrumbs()` — derives crumbs from `useLocation().pathname`; all but the last are links, the last is the current page.
- Exports a pure helper `buildCrumbs(pathname: string): { label: string; href: string }[]` for testing.

- [ ] **Step 1: Write the failing test `breadcrumbs.test.tsx`**

```tsx
import { expect, test } from 'vitest';

import { buildCrumbs } from './breadcrumbs';

test('dashboard root → single Dashboard crumb', () => {
  expect(buildCrumbs('/dashboard')).toEqual([{ label: 'Dashboard', href: '/dashboard' }]);
});

test('nested proxies tcp-udp path', () => {
  expect(buildCrumbs('/dashboard/proxies/tcp-udp')).toEqual([
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Proxies', href: '/dashboard/proxies' },
    { label: 'TCP/UDP', href: '/dashboard/proxies/tcp-udp' },
  ]);
});

test('numeric id segment becomes Details', () => {
  expect(buildCrumbs('/dashboard/access/42')).toEqual([
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Access', href: '/dashboard/access' },
    { label: 'Details', href: '/dashboard/access/42' },
  ]);
});

test('new segment becomes New', () => {
  expect(buildCrumbs('/dashboard/proxies/new')).toEqual([
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Proxies', href: '/dashboard/proxies' },
    { label: 'New', href: '/dashboard/proxies/new' },
  ]);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir ui test run src/components/layout/breadcrumbs.test.tsx`
Expected: FAIL — cannot resolve `./breadcrumbs`.

- [ ] **Step 3: Implement `breadcrumbs.tsx`**

```tsx
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@e412/rnui-react';
import { Link, useLocation } from '@tanstack/react-router';
import { Fragment } from 'react';

const LABELS: Record<string, string> = {
  dashboard: 'Dashboard',
  proxies: 'Proxies',
  'tcp-udp': 'TCP/UDP',
  new: 'New',
  access: 'Access',
  activity: 'Activity',
  settings: 'Settings',
};

function labelFor(segment: string): string {
  if (LABELS[segment]) return LABELS[segment];
  if (/^\d+$/.test(segment)) return 'Details';
  return segment.charAt(0).toUpperCase() + segment.slice(1);
}

export function buildCrumbs(pathname: string): { label: string; href: string }[] {
  const segments = pathname.split('/').filter(Boolean);
  const crumbs: { label: string; href: string }[] = [];
  let href = '';
  for (const segment of segments) {
    href += `/${segment}`;
    crumbs.push({ label: labelFor(segment), href });
  }
  return crumbs;
}

export function Breadcrumbs() {
  const { pathname } = useLocation();
  const crumbs = buildCrumbs(pathname);

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {crumbs.map((crumb, i) => {
          const isLast = i === crumbs.length - 1;
          return (
            <Fragment key={crumb.href}>
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink render={<Link to={crumb.href} />}>{crumb.label}</BreadcrumbLink>
                )}
              </BreadcrumbItem>
              {!isLast && <BreadcrumbSeparator />}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm --dir ui test run src/components/layout/breadcrumbs.test.tsx`
Expected: 4 passed.

- [ ] **Step 5: Build + commit**

Run: `pnpm --dir ui build` → success.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/layout/breadcrumbs.tsx ui/src/components/layout/breadcrumbs.test.tsx
git commit -m "feat(ui): breadcrumbs derived from route path

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Command palette (store + component)

**Files:**
- Create: `ui/src/stores/command-palette.ts`
- Create: `ui/src/stores/command-palette.test.ts`
- Create: `ui/src/components/layout/command-palette.tsx`

**Interfaces:**
- Consumes: `useLogout` (Task 1).
- Produces: `useCommandPalette()` zustand store (`{ open, setOpen, toggle }`) and `CommandPalette()` (global ⌘K dialog with navigate + quick actions).

- [ ] **Step 1: Write the failing store test `command-palette.test.ts`**

```ts
import { expect, test } from 'vitest';

import { useCommandPalette } from './command-palette';

test('toggle flips open and setOpen sets it', () => {
  useCommandPalette.getState().setOpen(false);
  expect(useCommandPalette.getState().open).toBe(false);
  useCommandPalette.getState().toggle();
  expect(useCommandPalette.getState().open).toBe(true);
  useCommandPalette.getState().setOpen(false);
  expect(useCommandPalette.getState().open).toBe(false);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir ui test run src/stores/command-palette.test.ts`
Expected: FAIL — cannot resolve `./command-palette`.

- [ ] **Step 3: Implement the store `ui/src/stores/command-palette.ts`**

```ts
import { create } from 'zustand';

interface CommandPaletteState {
  open: boolean;
  setOpen: (open: boolean) => void;
  toggle: () => void;
}

export const useCommandPalette = create<CommandPaletteState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
  toggle: () => set((s) => ({ open: !s.open })),
}));
```

- [ ] **Step 4: Run store test to verify it passes**

Run: `pnpm --dir ui test run src/stores/command-palette.test.ts`
Expected: 1 passed.

- [ ] **Step 5: Implement `ui/src/components/layout/command-palette.tsx`**

```tsx
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@e412/rnui-react';
import { useNavigate } from '@tanstack/react-router';
import {
  Activity,
  ClipboardList,
  Globe,
  Home,
  LogOut,
  MoonStar,
  Network,
  Plus,
  RefreshCw,
  Settings,
  Shield,
} from 'lucide-react';
import { useTheme } from 'next-themes';
import { useEffect } from 'react';

import { useSyncStatus } from '@/hooks/use-dashboard';
import { useLogout } from '@/hooks/use-logout';
import { useCommandPalette } from '@/stores/command-palette';

export function CommandPalette() {
  const { open, setOpen, toggle } = useCommandPalette();
  const navigate = useNavigate();
  const { resolvedTheme, setTheme } = useTheme();
  const { triggerSync } = useSyncStatus();
  const logout = useLogout();

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        toggle();
      }
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [toggle]);

  const run = (fn: () => void) => {
    setOpen(false);
    fn();
  };

  const go = (to: string) => run(() => navigate({ to }));

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Command palette"
      description="Search for a page or action"
    >
      <CommandInput placeholder="Type a command or search…" />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>
        <CommandGroup heading="Navigate">
          <CommandItem onSelect={() => go('/dashboard')}>
            <Home className="size-4" />
            Dashboard
          </CommandItem>
          <CommandItem onSelect={() => go('/dashboard/proxies')}>
            <Globe className="size-4" />
            Proxies (HTTP)
          </CommandItem>
          <CommandItem onSelect={() => go('/dashboard/proxies/tcp-udp')}>
            <Network className="size-4" />
            Proxies (TCP/UDP)
          </CommandItem>
          <CommandItem onSelect={() => go('/dashboard/access')}>
            <Shield className="size-4" />
            Access
          </CommandItem>
          <CommandItem onSelect={() => go('/dashboard/activity')}>
            <Activity className="size-4" />
            Activity
          </CommandItem>
          <CommandItem onSelect={() => go('/dashboard/settings')}>
            <Settings className="size-4" />
            Settings
          </CommandItem>
        </CommandGroup>
        <CommandGroup heading="Actions">
          <CommandItem onSelect={() => go('/dashboard/proxies/new')}>
            <Plus className="size-4" />
            New HTTP proxy
          </CommandItem>
          <CommandItem onSelect={() => go('/dashboard/proxies/tcp-udp/new')}>
            <Plus className="size-4" />
            New TCP/UDP proxy
          </CommandItem>
          <CommandItem onSelect={() => run(() => triggerSync())}>
            <RefreshCw className="size-4" />
            Apply configuration now
          </CommandItem>
          <CommandItem
            onSelect={() => run(() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark'))}
          >
            <MoonStar className="size-4" />
            Toggle theme
          </CommandItem>
          <CommandItem onSelect={() => run(() => void logout())}>
            <LogOut className="size-4" />
            Sign out
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
```

- [ ] **Step 6: Build + test + commit**

Run: `pnpm --dir ui build` → success. `pnpm --dir ui test run` → all pass.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/stores/command-palette.ts ui/src/stores/command-palette.test.ts ui/src/components/layout/command-palette.tsx
git commit -m "feat(ui): ⌘K command palette (navigate + quick actions)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: TopBar — compose + wire into AppSidebar

**Files:**
- Create: `ui/src/components/layout/top-bar.tsx`
- Modify: `ui/src/components/layout/sidebar.tsx` (`AppSidebar` header → `<TopBar />`; mount `<CommandPalette />`)

**Interfaces:**
- Consumes: `Breadcrumbs`, `SyncStatus`, `ThemeToggle`, `useCommandPalette`, `CommandPalette`, rnui `SidebarTrigger`, `Button`, `Kbd`.
- Produces: `TopBar()` — the dashboard header.

- [ ] **Step 1: Create `ui/src/components/layout/top-bar.tsx`**

```tsx
import { Button, Kbd, SidebarTrigger } from '@e412/rnui-react';
import { Search } from 'lucide-react';

import { Breadcrumbs } from '@/components/layout/breadcrumbs';
import { SyncStatus } from '@/components/layout/sync-status';
import { ThemeToggle } from '@/components/layout/theme-toggle';
import { useCommandPalette } from '@/stores/command-palette';

export function TopBar() {
  const setOpen = useCommandPalette((s) => s.setOpen);

  return (
    <header className="flex h-14 items-center gap-3 border-b border-border px-4">
      <SidebarTrigger />
      <Breadcrumbs />
      <div className="ml-auto flex items-center gap-1.5">
        <Button
          variant="outline"
          size="sm"
          className="text-muted-foreground gap-2"
          onClick={() => setOpen(true)}
        >
          <Search className="size-4" />
          <span className="hidden sm:inline">Search</span>
          <Kbd>⌘K</Kbd>
        </Button>
        <SyncStatus />
        <ThemeToggle />
      </div>
    </header>
  );
}
```

- [ ] **Step 2: Wire into `AppSidebar`**

In `ui/src/components/layout/sidebar.tsx`:
- Add imports: `import { TopBar } from '@/components/layout/top-bar';` and `import { CommandPalette } from '@/components/layout/command-palette';`
- Replace the `<header>...</header>` block inside `<SidebarInset>` with `<TopBar />`:

```tsx
      <SidebarInset>
        <TopBar />
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </SidebarInset>
```
- Remove the now-unused `SidebarTrigger` import from `@e412/rnui-react` in sidebar.tsx (it moved into TopBar). Verify it's not referenced elsewhere in the file first.
- Mount the palette once, alongside the existing dialogs near the end of the returned JSX:

```tsx
      <ProfileDialog open={profileOpen} onOpenChange={setProfileOpen} />
      <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
      <CommandPalette />
```

- [ ] **Step 3: Build + test + commit**

Run: `pnpm --dir ui build` → success. `pnpm --dir ui test run` → all pass.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/layout/top-bar.tsx ui/src/components/layout/sidebar.tsx
git commit -m "feat(ui): control-plane top bar (breadcrumbs, sync, theme, ⌘K)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Full verification

**Files:** none.

- [ ] **Step 1: Gates**

Run: `pnpm --dir ui build` → success · `pnpm --dir ui test run` → all pass · `pnpm --dir ui check` → clean.

- [ ] **Step 2: Note for the controller's `verify` pass**

Interactive smoke (not part of this task): top bar shows breadcrumbs reflecting the route; the sync chip shows Synced/Syncing/Sync-failed and re-applies on click; the theme toggle flips light/dark and persists; ⌘K (and the Search button) opens the palette; palette navigation + New-proxy + Apply-now + Toggle-theme + Sign-out all work and close the palette.

## Done criteria
- Top bar live with breadcrumbs + sync status + theme toggle + ⌘K trigger; palette opens via ⌘K and the button; all palette actions work.
- `pnpm --dir ui build` + `test run` + `check` all green.
- **Next M0b plan:** Form Foundation (RHF+Zod + login/signup/change-password).
