# M0a — rnui Platform Swap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the Waygates UI from `@e412/titanium` to `@e412/rnui-react` at behavioral parity (current layouts unchanged), port the copper theme into rnui's theming model, and remove `@e412/titanium`.

**Architecture:** Add rnui alongside titanium, swap the global theme/providers, build a small interim `TagsInput` shim (rnui lacks one), mechanically migrate all 43 titanium-importing files domain-by-domain (import-source rewrite + three component renames), then delete titanium. Each step keeps the app compiling. The new sidebar/top-bar redesign and the React-Hook-Form foundation are **out of scope** — they belong to M0b.

**Tech Stack:** React 19, TypeScript 5.7, Vite 8 (rolldown) + `@tailwindcss/vite` (Tailwind 4), pnpm, `@e412/rnui-react`/`@e412/rnui-themes`, `next-themes`, `sonner`, `lucide-react`. Tests: Vitest + React Testing Library (added in this plan).

## Global Constraints

- Package manager is **pnpm**; the UI lives in `ui/`. Run UI commands as `pnpm --dir ui <script>`.
- rnui from npm: **`@e412/rnui-react@^0.1.0`**, **`@e412/rnui-themes@^0.1.0`**. (Dev convenience: `pnpm --dir ui link ~/projects/rnui/packages/react` to test local rnui fixes; do not commit the link.)
- Theme is the **single branded copper theme**, light + dark via `next-themes` (`attribute="class"`, default dark). **No theme picker.**
- Toasts stay on **`sonner`** (`import { toast } from 'sonner'`); the `Toaster` element comes from `@e412/rnui-react`.
- **Parity only** — do not change behavior, layout, copy, or routes in this plan.
- **Per-task verification gates (run all that apply):**
  - Build (primary correctness gate): `pnpm --dir ui build` → expect success. The bundler errors on missing/renamed exports, which is the main migration risk.
  - Lint/format: `pnpm --dir ui check:fix` then `pnpm --dir ui check` → expect clean.
  - Unit tests (after Task 2): `pnpm --dir ui test run` → expect pass.
  - Smoke: deferred to a final `verify` pass (subagents cannot drive a browser); for each task, confirm the build succeeds and the affected files render in that pass.
  - **NOTE — no `tsc` gate:** the project has ~20 pre-existing type errors (zod-v4 leftovers, union narrowing, etc.) and `vite build` does not type-check, so a clean-`tsc` gate is not achievable in this parity milestone. Do **not** attempt to fix unrelated pre-existing type errors here. You may run `pnpm --dir ui exec tsc --noEmit -p tsconfig.json 2>&1` informationally to sanity-check files you changed, but it is **not** a pass/fail gate.
- Keep the app compiling at **every** commit. `@e412/titanium` stays installed until Task 9.
- Branch: `feat/rnui-redesign-program`.
- Commit messages use Conventional Commits and end with the project trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

### Component delta table (authoritative — every migration task references this)

| titanium import | rnui replacement | Action |
|---|---|---|
| from `@e412/titanium` | from `@e412/rnui-react` | Rewrite import source |
| `ThemeProvider` | `ThemeProvider` from `next-themes` | Provider swap (Task 4) |
| `Toaster` | `Toaster` from `@e412/rnui-react` | Import swap |
| `CardHeading` | `CardTitle` | Rename element + import (13 files) |
| `CardToolbar` | `CardAction` | Rename element + import (8 files) |
| `DialogBody` | `<div className="grid gap-4 py-2">…</div>` | Replace element, drop import (6 files) |
| `TagsInput`, `TagsInputTag`, `TagsInputTagText`, `TagsInputTagRemove`, `TagsInputInput` | local `@/components/ui/tags-input` (Task 5) | Import swap (1 file) |
| `Filters`, `Filter`, `FilterFieldsConfig` | same names from `@e412/rnui-react` | Import swap (2 files) |
| `DataGrid`, `DataGridColumnHeader`, `DataGridContainer`, `DataGridPagination`, `DataGridTable` | same names from `@e412/rnui-react` (`emptyMessage` supported) | Import swap |
| `Sidebar*`, `Field*`, `Dialog*` (except Body), `Sheet*`, `AlertDialog*`, `Select*`, `DropdownMenu*`, `Tabs*`, `Tooltip*`, `Alert*`, `Badge`, `Button`, `Input`, `Textarea`, `Checkbox`, `Switch`, `Skeleton`, `Separator`, `Avatar`, `Label`, `ScrollArea`, `Spinner` | same names from `@e412/rnui-react` | Import swap |

If the build (or an informational `tsc` run) surfaces any name/prop not in this table, resolve against rnui's exports in `node_modules/@e412/rnui-react/dist/index.d.ts` and add a row.

---

## Task 1: Add rnui + next-themes (keep titanium)

**Files:**
- Modify: `ui/package.json` (dependencies)

**Interfaces:**
- Produces: `@e412/rnui-react`, `@e412/rnui-themes`, `next-themes` resolvable for all later tasks.

- [ ] **Step 1: Install**

```bash
pnpm --dir ui add @e412/rnui-react@^0.1.0 @e412/rnui-themes@^0.1.0 next-themes
```

- [ ] **Step 2: Verify resolution**

Run: `pnpm --dir ui exec node -e "require('@e412/rnui-react/package.json')&&require('@e412/rnui-themes/package.json')&&require('next-themes/package.json')&&console.log('ok')"`
Expected: prints `ok`. (`@e412/titanium` is still present — that is correct.)

- [ ] **Step 3: Commit**

```bash
git add ui/package.json ui/pnpm-lock.yaml
git commit -m "build(ui): add @e412/rnui-react, rnui-themes, next-themes

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Add Vitest + React Testing Library

**Files:**
- Create: `ui/vitest.config.ts`
- Create: `ui/src/test/setup.ts`
- Create: `ui/src/test/smoke.test.tsx`
- Modify: `ui/package.json` (add `test` script)

**Interfaces:**
- Produces: `pnpm --dir ui test run` test runner; `render`/`screen` available for component tests (Task 5).

- [ ] **Step 1: Install dev deps**

```bash
pnpm --dir ui add -D vitest jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event
```

- [ ] **Step 2: Add the `test` script**

In `ui/package.json` `scripts`, add:

```json
"test": "vitest"
```

- [ ] **Step 3: Create `ui/vitest.config.ts`**

```ts
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  resolve: { tsconfigPaths: true },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
});
```

- [ ] **Step 4: Create `ui/src/test/setup.ts`**

```ts
import '@testing-library/jest-dom/vitest';
```

- [ ] **Step 5: Write the smoke test `ui/src/test/smoke.test.tsx`**

```tsx
import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';

test('test harness renders DOM', () => {
  render(<button>hello</button>);
  expect(screen.getByText('hello')).toBeInTheDocument();
});
```

- [ ] **Step 6: Run it**

Run: `pnpm --dir ui test run`
Expected: 1 passed.

- [ ] **Step 7: Lint + commit**

```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/package.json ui/pnpm-lock.yaml ui/vitest.config.ts ui/src/test/setup.ts ui/src/test/smoke.test.tsx
git commit -m "test(ui): add vitest + react testing library harness

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Port the copper theme to rnui's CSS model

**Files:**
- Modify: `ui/src/app.css`

**Interfaces:**
- Consumes: nothing.
- Produces: copper tokens defined on `:root` / `.dark` (unscoped, no `.default` class needed), rnui theme CSS imported, both `@source` lines present.

Replace the titanium theme wiring. Keep titanium's `@source` line for now (titanium components must still be scanned until Task 9) and add rnui's. Move the copper variables out of the titanium `.default.default` / `.default.default.dark` selectors onto `:root` / `.dark`. Rely on `@e412/rnui-themes` for the `@theme inline` color mapping; keep only the app-specific extras (marquee, container, scrollbar, fade animations).

- [ ] **Step 1: Rewrite the top of `ui/src/app.css`**

Replace lines 1–9 (the font + framework + titanium imports) with:

```css
/* Font imports must come first */
@import url('https://api.fontshare.com/v2/css?f[]=clash-grotesk@200,300,400,500,600,700&display=swap');
@import url('https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,200..800&display=swap');
@import url('https://fonts.googleapis.com/css2?family=Geist+Mono:wght@100..900&display=swap');

@import 'tailwindcss';
@import 'tw-animate-css';
@import '@e412/rnui-themes';
@source "../node_modules/@e412/rnui-react/dist";
@source "../node_modules/@e412/titanium";
```

- [ ] **Step 2: Move copper light tokens to `:root`**

Change the selector `.default.default {` (currently line 18) to `:root {`. Leave its body (the `--primary`, `--background`, … `--shadow-*` declarations) unchanged. Add `--font-heading` alongside `--text-sans`:

```css
:root {
  --text-sans: 'Clash Grotesk', system-ui, sans-serif;
  --font-sans: 'Clash Grotesk', system-ui, sans-serif;
  --font-heading: 'Bricolage Grotesque', 'Clash Grotesk', sans-serif;

  --primary: oklch(0.65 0.15 55);
  /* …rest of the existing light declarations unchanged… */
}
```

- [ ] **Step 3: Move copper dark tokens to `.dark`**

Change the selector `.default.default.dark {` (currently line 67) to `.dark {`. Leave its body unchanged.

- [ ] **Step 4: Trim the custom `@theme inline` block**

In the `@theme inline { … }` block (currently lines 105–159), **delete** the `--color-*` mapping lines and the `--radius-*` lines (rnui-themes provides these). **Keep** `--font-sans`, `--font-serif`, `--font-mono`, the `--animate-marquee*` lines and both `@keyframes marquee*`. Result:

```css
@custom-variant dark (&:is(.dark *));

@theme inline {
  --font-sans: 'Clash Grotesk', system-ui, sans-serif;
  --font-serif: var(--text-serif);
  --font-mono: var(--text-mono);

  --animate-marquee: marquee var(--duration) infinite linear;
  --animate-marquee-vertical: marquee-vertical var(--duration) linear infinite;

  @keyframes marquee {
    from { transform: translateX(0); }
    to { transform: translateX(calc(-100% - var(--gap))); }
  }
  @keyframes marquee-vertical {
    from { transform: translateY(0); }
    to { transform: translateY(calc(-100% - var(--gap))); }
  }
}
```

Leave everything from `/** Global Styles **/` (line 161) to end of file unchanged.

- [ ] **Step 5: Build + smoke**

Run: `pnpm --dir ui build`
Expected: success.
Run: `pnpm --dir ui dev`, load `/login` — copper primary + cream background visible, dark mode renders (titanium components still in place, still styled because they read the same `--*` variables).

- [ ] **Step 6: Lint + commit**

```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/app.css
git commit -m "style(ui): port copper theme onto rnui-themes (:root/.dark)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Swap root providers to next-themes + rnui

**Files:**
- Modify: `ui/src/main.tsx`
- Possibly modify: the dashboard layout that renders `SidebarProvider` (find in Step 1)

**Interfaces:**
- Consumes: `@e412/rnui-react`, `next-themes`.
- Produces: app wrapped in `next-themes` `ThemeProvider` (class strategy, default dark), `TooltipProvider`, and rnui `Toaster`.

- [ ] **Step 1: Locate the SidebarProvider usage**

Run: `grep -rn "SidebarProvider" ui/src`
Note the file(s); they will need their import switched to `@e412/rnui-react` (handled here if it's a layout route, otherwise in Task 6h).

- [ ] **Step 2: Rewrite `ui/src/main.tsx`**

```tsx
import { Toaster, TooltipProvider } from '@e412/rnui-react';
import { QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { ThemeProvider } from 'next-themes';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { queryClient } from './lib/query-client';
import { router } from './lib/router';

import './app.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false}>
      <TooltipProvider>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
          <Toaster richColors position="top-right" />
        </QueryClientProvider>
      </TooltipProvider>
    </ThemeProvider>
  </StrictMode>,
);
```

- [ ] **Step 3: Build**

Run: `pnpm --dir ui build` → success (no "not exported" errors for `Toaster`/`TooltipProvider`/`ThemeProvider`).

- [ ] **Step 4: Lint + commit**

```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/main.tsx
git commit -m "feat(ui): swap root providers to next-themes + rnui Toaster/TooltipProvider

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Interim TagsInput shim (+ tests)

**Files:**
- Create: `ui/src/components/ui/tags-input.tsx`
- Create: `ui/src/components/ui/tags-input.test.tsx`
- Modify: `ui/src/components/acl/waygates-auth-tab.tsx` (import only)

**Interfaces:**
- Produces (compound API mirroring titanium so rnui's future `TagsInput` is a drop-in swap):
  - `TagsInput({ value: string[]; onValueChange: (v: string[]) => void; placeholder?: string; delimiters?: string[]; validation?: { pattern: RegExp }; children: ReactNode })`
  - `TagsInputTag({ index: number; children })`, `TagsInputTagText({ children })`, `TagsInputTagRemove()`, `TagsInputInput({ placeholder? })`

> Temporary shim. Replace with rnui's `TagsInput` once published; the public API matches so consumers won't change.

- [ ] **Step 1: Write the failing tests `ui/src/components/ui/tags-input.test.tsx`**

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { expect, test, vi } from 'vitest';

import {
  TagsInput,
  TagsInputInput,
  TagsInputTag,
  TagsInputTagRemove,
  TagsInputTagText,
} from './tags-input';

function Harness({ onChange }: { onChange?: (v: string[]) => void }) {
  const [value, setValue] = useState<string[]>([]);
  return (
    <TagsInput
      value={value}
      onValueChange={(v) => {
        setValue(v);
        onChange?.(v);
      }}
      delimiters={['Enter', ',']}
      validation={{ pattern: /^[^@\s]+@[^@\s]+\.[^@\s]+$/ }}
    >
      {value.map((email, index) => (
        <TagsInputTag key={email} index={index}>
          <TagsInputTagText>{email}</TagsInputTagText>
          <TagsInputTagRemove />
        </TagsInputTag>
      ))}
      <TagsInputInput placeholder="Add email..." />
    </TagsInput>
  );
}

test('adds a valid tag on Enter', async () => {
  const onChange = vi.fn();
  render(<Harness onChange={onChange} />);
  await userEvent.type(screen.getByPlaceholderText('Add email...'), 'a@b.com{Enter}');
  expect(onChange).toHaveBeenLastCalledWith(['a@b.com']);
  expect(screen.getByText('a@b.com')).toBeInTheDocument();
});

test('rejects an invalid tag', async () => {
  const onChange = vi.fn();
  render(<Harness onChange={onChange} />);
  await userEvent.type(screen.getByPlaceholderText('Add email...'), 'not-an-email{Enter}');
  expect(onChange).not.toHaveBeenCalled();
});

test('dedupes existing tags', async () => {
  const onChange = vi.fn();
  render(<Harness onChange={onChange} />);
  const input = screen.getByPlaceholderText('Add email...');
  await userEvent.type(input, 'a@b.com{Enter}');
  await userEvent.type(input, 'a@b.com{Enter}');
  expect(onChange).toHaveBeenCalledTimes(1);
});

test('removes a tag via its remove button', async () => {
  render(<Harness />);
  await userEvent.type(screen.getByPlaceholderText('Add email...'), 'a@b.com{Enter}');
  await userEvent.click(screen.getByRole('button', { name: /remove/i }));
  expect(screen.queryByText('a@b.com')).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `pnpm --dir ui test run src/components/ui/tags-input.test.tsx`
Expected: FAIL — cannot resolve `./tags-input`.

- [ ] **Step 3: Implement `ui/src/components/ui/tags-input.tsx`**

```tsx
import { Badge, Input, cn } from '@e412/rnui-react';
import { X } from 'lucide-react';
import { createContext, type ReactNode, useContext, useState } from 'react';

interface TagsInputContextValue {
  value: string[];
  addTag: (raw: string) => void;
  removeAt: (index: number) => void;
  placeholder?: string;
  delimiters: string[];
}

const TagsInputContext = createContext<TagsInputContextValue | null>(null);
const TagIndexContext = createContext<number>(-1);

function useTagsInput() {
  const ctx = useContext(TagsInputContext);
  if (!ctx) throw new Error('TagsInput.* must be used within <TagsInput>');
  return ctx;
}

interface TagsInputProps {
  value: string[];
  onValueChange: (value: string[]) => void;
  placeholder?: string;
  delimiters?: string[];
  validation?: { pattern: RegExp };
  className?: string;
  children: ReactNode;
}

export function TagsInput({
  value,
  onValueChange,
  placeholder,
  delimiters = ['Enter'],
  validation,
  className,
  children,
}: TagsInputProps) {
  const addTag = (raw: string) => {
    const tag = raw.trim();
    if (!tag) return;
    if (value.includes(tag)) return;
    if (validation?.pattern && !validation.pattern.test(tag)) return;
    onValueChange([...value, tag]);
  };
  const removeAt = (index: number) => {
    onValueChange(value.filter((_, i) => i !== index));
  };
  return (
    <TagsInputContext.Provider value={{ value, addTag, removeAt, placeholder, delimiters }}>
      <div
        className={cn(
          'border-input focus-within:ring-ring/50 flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border bg-transparent px-2 py-1.5 text-sm focus-within:ring-2',
          className,
        )}
      >
        {children}
      </div>
    </TagsInputContext.Provider>
  );
}

export function TagsInputTag({ index, children }: { index: number; children: ReactNode }) {
  return (
    <TagIndexContext.Provider value={index}>
      <Badge variant="secondary" className="gap-1">
        {children}
      </Badge>
    </TagIndexContext.Provider>
  );
}

export function TagsInputTagText({ children }: { children: ReactNode }) {
  return <span>{children}</span>;
}

export function TagsInputTagRemove() {
  const { removeAt } = useTagsInput();
  const index = useContext(TagIndexContext);
  return (
    <button
      type="button"
      aria-label="Remove tag"
      className="hover:text-destructive ml-0.5 inline-flex"
      onClick={() => removeAt(index)}
    >
      <X className="size-3" />
    </button>
  );
}

export function TagsInputInput({ placeholder }: { placeholder?: string }) {
  const { addTag, delimiters, placeholder: rootPlaceholder } = useTagsInput();
  const [draft, setDraft] = useState('');
  return (
    <Input
      value={draft}
      placeholder={placeholder ?? rootPlaceholder}
      className="h-6 flex-1 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0"
      onChange={(e) => setDraft(e.target.value)}
      onKeyDown={(e) => {
        const isDelim =
          delimiters.includes(e.key) ||
          (delimiters.includes(',') && e.key === ',') ||
          (delimiters.includes(' ') && e.key === ' ');
        if (isDelim) {
          e.preventDefault();
          if (draft.trim()) {
            addTag(draft);
            setDraft('');
          }
        }
      }}
    />
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `pnpm --dir ui test run src/components/ui/tags-input.test.tsx`
Expected: 4 passed. (If `cn` is not exported by `@e412/rnui-react`, import it from `@e412/rnui-react` per its index; it is exported as `cn`.)

- [ ] **Step 5: Point waygates-auth-tab at the shim**

In `ui/src/components/acl/waygates-auth-tab.tsx`, remove `TagsInput, TagsInputInput, TagsInputTag, TagsInputTagRemove, TagsInputTagText` from the `@e412/titanium` import and add:

```tsx
import {
  TagsInput,
  TagsInputInput,
  TagsInputTag,
  TagsInputTagRemove,
  TagsInputTagText,
} from '@/components/ui/tags-input';
```

(The three usages already pass `value`/`onValueChange`/`placeholder`/`delimiters`/`validation` — no JSX changes needed.)

- [ ] **Step 6: Build + tests + commit**

Run: `pnpm --dir ui build` → success.
Run: `pnpm --dir ui test run` → all pass.

```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/ui/tags-input.tsx ui/src/components/ui/tags-input.test.tsx ui/src/components/acl/waygates-auth-tab.tsx
git commit -m "feat(ui): interim TagsInput shim over rnui primitives + tests

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Tasks 6a–6h: Mechanical domain migration (titanium → rnui)

Each sub-task migrates one domain group. The procedure is identical:

1. In each listed file, change the import source `'@e412/titanium'` → `'@e412/rnui-react'`.
2. Apply the renames from the **Component delta table**: `CardHeading`→`CardTitle`, `CardToolbar`→`CardAction`, and replace `<DialogBody>…</DialogBody>` with `<div className="grid gap-4 py-2">…</div>` (removing `DialogBody` from imports).
3. `pnpm --dir ui build`. The bundler errors on any unresolved/renamed export — fix those against `node_modules/@e412/rnui-react/dist/index.d.ts` and the delta table. Do **not** restructure, restyle, or fix unrelated pre-existing type errors.
4. `pnpm --dir ui check:fix && pnpm --dir ui check` → clean.
5. Commit the group. (Interactive smoke happens in the final `verify` pass.)

> A whole-tree find/replace of the import source is acceptable as a starting point, but commit per group and verify each group's routes individually so failures are isolated.

### Task 6a: Proxy (HTTP) domain
**Files:** `ui/src/components/proxy/cells/proxy-actions-cell.tsx`, `proxy-status-badge.tsx`, `proxy-target-cell.tsx`, `proxy-type-badge.tsx`, `ui/src/components/proxy/proxy-data-grid.tsx`, `ui/src/components/proxy/forms/acl-selector.tsx`, `redirect-form.tsx`, `reverse-proxy-form.tsx`, `static-form.tsx`, `ui/src/routes/_dashboard/proxies/index.tsx` (also contains `Filters` — same import swap), `new.tsx`, `$proxyId.tsx`.
- [ ] Apply procedure steps 1–5. Smoke: `/dashboard/proxies`, `/dashboard/proxies/new`, edit a proxy. Verify DataGrid (sorting, pagination, skeleton, empty state) and the Filters bar work.
- [ ] Commit: `refactor(ui): migrate proxy (HTTP) screens to rnui`

### Task 6b: L4 (TCP/UDP) domain
**Files:** `ui/src/components/l4-proxy/forms/l4-proxy-form.tsx`, `ui/src/routes/_dashboard/l4-proxies/index.tsx`, `new.tsx`, `$l4ProxyId.tsx`.
- [ ] Apply procedure. Smoke: `/dashboard/l4-proxies`, create + edit.
- [ ] Commit: `refactor(ui): migrate L4 (TCP/UDP) screens to rnui`

### Task 6c: Audit / Activity domain
**Files:** `ui/src/components/audit-logs/audit-config-panel.tsx`, `audit-data-grid.tsx`, `audit-log-detail-sheet.tsx`, `cells/action-badge.tsx`, `cells/status-badge.tsx`, `ui/src/routes/_dashboard/audit-logs.tsx` (also contains `Filters`).
- [ ] Apply procedure. Smoke: `/dashboard/audit-logs` — grid, Filters, detail sheet open/close.
- [ ] Commit: `refactor(ui): migrate audit-log screens to rnui`

### Task 6d: Settings domain
**Files:** `ui/src/components/settings/acl-branding-settings.tsx`, `catchall-settings.tsx`, `ui/src/routes/_dashboard/settings.tsx`.
- [ ] Apply procedure. Smoke: `/dashboard/settings` — all tabs.
- [ ] Commit: `refactor(ui): migrate settings screens to rnui`

### Task 6e: Access (ACL) domain
**Files:** `ui/src/components/acl/acl-group-form-modal.tsx`, `basic-auth-tab.tsx`, `external-providers-tab.tsx`, `group-usage-tab.tsx`, `ip-rules-tab.tsx`, `waygates-auth-tab.tsx` (TagsInput already swapped in Task 5 — migrate its remaining titanium imports here), `ui/src/routes/_dashboard/acl/index.tsx`, `$groupId.tsx`.
- [ ] Apply procedure. Smoke: `/dashboard/acl`, open a group, walk every tab.
- [ ] Commit: `refactor(ui): migrate access/ACL screens to rnui (component swap only)`

### Task 6f: Auth pages
**Files:** `ui/src/routes/login.tsx`, `signup.tsx`, `ui/src/routes/auth/acl-login.tsx`, `acl-forbidden.tsx`, `ui/src/components/acl/acl-login-form.tsx`, `oauth-provider-button.tsx`.
- [ ] Apply procedure. Smoke: `/login`, `/signup`, `/auth/login`, `/auth/forbidden`.
- [ ] Commit: `refactor(ui): migrate auth pages to rnui`

### Task 6g: Dashboard page
**Files:** `ui/src/routes/_dashboard/index.tsx` (contains `CardHeading`/`CardToolbar`).
- [ ] Apply procedure. Smoke: `/dashboard` — status strip, overview, recent activity, quick actions.
- [ ] Commit: `refactor(ui): migrate dashboard page to rnui`

### Task 6h: Layout sidebar
**Files:** `ui/src/components/layout/sidebar.tsx` (contains `DialogBody`), plus the `SidebarProvider` host found in Task 4 Step 1 if not already migrated.
- [ ] Apply procedure. Note: `Sidebar*` names match; verify `SidebarProvider`/`SidebarInset`/`SidebarMenuButton` compose the same. Smoke: navigate the whole app via the sidebar; open the user menu + change-password dialog.
- [ ] Commit: `refactor(ui): migrate layout sidebar to rnui`

---

## Task 7: Verify zero remaining titanium imports

**Files:** none (verification).

- [ ] **Step 1: Search**

Run: `grep -rn "@e412/titanium" ui/src`
Expected: **no matches.** If any remain, migrate them using the Task 6 procedure and commit per their domain.

---

## Task 8: Update Vite vendor chunking

**Files:**
- Modify: `ui/vite.config.ts:34`

**Interfaces:**
- Consumes: none.
- Produces: vendor-ui chunk references rnui, not titanium.

- [ ] **Step 1: Edit the `vendor-ui` test regex**

Change line 34 from:

```ts
test: /node_modules\/(@e412\/titanium|lucide-react|sonner)/,
```

to:

```ts
test: /node_modules\/(@e412\/rnui-react|lucide-react|sonner)/,
```

- [ ] **Step 2: Build + commit**

Run: `pnpm --dir ui build` → success.

```bash
git add ui/vite.config.ts
git commit -m "build(ui): point vendor-ui chunk at rnui

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 9: Remove @e412/titanium

**Files:**
- Modify: `ui/package.json` (remove dependency)
- Modify: `ui/src/app.css` (remove titanium `@source`)

**Interfaces:**
- Consumes: Task 7 (no remaining imports).
- Produces: titanium fully gone; M0a complete.

- [ ] **Step 1: Remove the titanium `@source` from `ui/src/app.css`**

Delete the line: `@source "../node_modules/@e412/titanium";`

- [ ] **Step 2: Uninstall titanium**

```bash
pnpm --dir ui remove @e412/titanium
```

- [ ] **Step 3: Full automated verification**

Run: `pnpm --dir ui test run` → all pass.
Run: `pnpm --dir ui build` → success.
Run: `pnpm --dir ui check` → clean.

Interactive smoke of every route (`/login`, `/signup`, `/auth/login`, `/auth/forbidden`, `/dashboard`, `/dashboard/proxies` +new/edit, `/dashboard/l4-proxies` +new/edit, `/dashboard/acl` +group tabs, `/dashboard/audit-logs` +detail sheet, `/dashboard/settings` all tabs) — confirming copper theme + dark default, DataGrids, Filters, dialogs/sheets, toasts, and the TagsInput shim — is performed in the **final `verify` pass** by the controller, not in this task.

- [ ] **Step 4: Commit**

```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/package.json ui/pnpm-lock.yaml ui/src/app.css
git commit -m "build(ui): remove @e412/titanium — M0a platform swap complete

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Done criteria (M0a)

- `grep -rn "@e412/titanium" ui` returns nothing (incl. package.json).
- `pnpm --dir ui test run`, `pnpm --dir ui build`, `pnpm --dir ui check` all green.
- App runs on rnui at visual/behavioral parity with copper theme + dark default.
- Interim `TagsInput` shim in place (tracked for replacement by rnui's `TagsInput`).
- **Next:** M0b — App Shell + Global QoL + RHF/Zod form foundation.
