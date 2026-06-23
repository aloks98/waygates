# Rebrand + Re-theme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Replace the copper theme + monochrome "W" with the synthwave pixel-portal identity — black/light periwinkle themes, square corners, VT323/Space Grotesk/Space Mono type, and the new pixel logo everywhere.

**Architecture:** Theme is driven by CSS custom properties in `ui/src/app.css` (`:root` = light, `.dark` = dark) consumed by rnui-themes/Tailwind v4; components read tokens (`bg-background`, `text-primary`, `rounded-*`→`--radius`, `--chart-*`). So most of the rebrand is a token swap in one file, plus the logo component + a few brand call-sites.

**Tech Stack:** React 19 + TS, Vite, Tailwind 4 + `@e412/rnui-themes`/`@e412/rnui-react`, next-themes, oxlint/oxfmt. Fonts via Google Fonts. No backend.

## Global Constraints

- **Frontend-only.** Branch `feat/rebrand-theme` (off master; spec committed `ee4d867`).
- **Gate per task:** `pnpm --dir ui build` && `pnpm --dir ui check` && `pnpm --dir ui test` — all green (existing tests must stay passing). **There is NO tsc type-check gate (oxlint only)** → verify token/font/route strings with `grep`, not the compiler.
- **Palette (hex, authoritative)** — Dark: bg `#000000`, surface `#0B0B12`, border/input `#22222F`, text `#EDEDF4`, muted-fg `#8A8A99`, primary `#6E72F0`, ring(cyan) `#34E1E0`, destructive `#E5675F`. Light: bg `#F7F7FA`, card `#FFFFFF`, surface/muted `#F1F2F6`, border/input `#E3E4EC`, text `#16171D`, muted-fg `#5C6072`, primary `#5862E0`, ring(cyan) `#18A6AE`, destructive `#D14A42`. `--radius: 0` both.
- **Fonts:** VT323 = wordmark only (`--font-display`); Space Grotesk = UI (`--font-sans`/heading); Space Mono = data (`--font-mono`).
- **Logo:** pixel mark is multi-color (the old `currentColor` contract is dropped — `className` sizes only).
- Commit trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Stage only each task's files; never `git add -A`.
- Theme/CSS changes have no unit tests — each task's "verification" is the gate + the specified `grep` assertions + (final task) a visual smoke. Keep existing vitest suite green.

## File map

- `ui/src/app.css` — all theme tokens (color L+D, fonts, `--radius:0`, dark glow utility). [Task 1]
- `ui/src/assets/waygates-mark.svg` (new), `ui/src/components/layout/waygate-logo.tsx`, `ui/public/favicon.svg`, `ui/index.html`. [Task 2]
- `ui/src/components/layout/sidebar.tsx`, `ui/src/routes/login.tsx`, `ui/src/routes/signup.tsx`. [Task 3]
- `ui/src/routes/auth/acl-login.tsx`, `ui/src/components/settings/acl-branding-settings.tsx`, `ui/src/routes/theme-preview.tsx`. [Task 4]
- Sweep across `ui/src` (rounded-* + stray refs) + visual smoke. [Task 5]

---

## Task 1: Theme tokens in app.css (colors L+D, radius 0, fonts, dark glow)

**Files:** Modify `ui/src/app.css`.

This replaces the copper palette + fonts. Keep the existing non-token blocks (`@import 'tailwindcss'`, `@import 'tw-animate-css'`, `@import '@e412/rnui-themes'`, `@source`, the `@custom-variant dark`, the `@layer base` global/scrollbar blocks, the `container` utility, and the fade/page-enter animations) **unchanged**.

- [ ] **Step 1: Replace the font `@import` lines.** At the top of `app.css`, the current three font `@import url(...)` lines (Clash Grotesk, Bricolage Grotesque, Geist Mono) become one:
```css
/* Font imports must come first */
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@300;400;500;600;700&family=Space+Mono:wght@400;700&family=VT323&display=swap');
```

- [ ] **Step 2: Replace the `:root { ... }` block** (the copper light theme, currently `--text-sans` … through the shadow vars) with this complete light theme:
```css
/** Waygates Synthwave Theme — light (:root) / dark (.dark) **/
:root {
  --text-sans: 'Space Grotesk', system-ui, sans-serif;
  --font-sans: 'Space Grotesk', system-ui, sans-serif;
  --font-heading: 'Space Grotesk', system-ui, sans-serif;
  --font-display: 'VT323', ui-monospace, monospace;

  --primary: #5862E0;
  --primary-foreground: #FFFFFF;
  --ring: #18A6AE;

  --background: #F7F7FA;
  --foreground: #16171D;

  --card: #FFFFFF;
  --card-foreground: #16171D;

  --popover: #FFFFFF;
  --popover-foreground: #16171D;

  --muted: #F1F2F6;
  --muted-foreground: #5C6072;

  --secondary: #F1F2F6;
  --secondary-foreground: #16171D;

  --accent: #ECEDF3;
  --accent-foreground: #16171D;

  --border: #E3E4EC;
  --input: #E3E4EC;

  --destructive: #D14A42;
  --destructive-foreground: #FFFFFF;

  --radius: 0;
  --radius-sm: 0;
  --radius-md: 0;
  --radius-lg: 0;
  --radius-xl: 0;

  --chart-1: #5862E0;
  --chart-2: #18A6AE;
  --chart-3: #8B5CF6;
  --chart-4: #E0408F;
  --chart-5: #6E72F0;

  --shadow-2xs: 0 1px 2px 0 oklch(0 0 0 / 0.04);
  --shadow-xs: 0 1px 2px 0 oklch(0 0 0 / 0.04);
  --shadow-sm: 0 1px 2px 0 oklch(0 0 0 / 0.04);
  --shadow: 0 1px 3px 0 oklch(0 0 0 / 0.06), 0 1px 2px -1px oklch(0 0 0 / 0.04);
  --shadow-md: 0 1px 3px 0 oklch(0 0 0 / 0.06), 0 1px 2px -1px oklch(0 0 0 / 0.04);
  --shadow-lg: 0 2px 6px -1px oklch(0 0 0 / 0.06), 0 1px 4px -2px oklch(0 0 0 / 0.04);
  --shadow-xl: 0 4px 10px -2px oklch(0 0 0 / 0.08), 0 2px 4px -2px oklch(0 0 0 / 0.04);
  --shadow-2xl: 0 8px 20px -4px oklch(0 0 0 / 0.1);
}
```
(The explicit `--radius-sm/md/lg/xl: 0` guard against rnui-themes deriving non-zero values from `--radius`.)

- [ ] **Step 3: Replace the `.dark { ... }` block** with this complete dark (black) theme:
```css
.dark {
  --primary: #6E72F0;
  --primary-foreground: #FFFFFF;
  --ring: #34E1E0;

  --background: #000000;
  --foreground: #EDEDF4;

  --card: #0B0B12;
  --card-foreground: #EDEDF4;

  --popover: #0B0B12;
  --popover-foreground: #EDEDF4;

  --muted: #15151F;
  --muted-foreground: #8A8A99;

  --secondary: #15151F;
  --secondary-foreground: #EDEDF4;

  --accent: #15151F;
  --accent-foreground: #EDEDF4;

  --border: #22222F;
  --input: #22222F;

  --destructive: #E5675F;
  --destructive-foreground: #FFFFFF;

  --chart-1: #6E72F0;
  --chart-2: #34E1E0;
  --chart-3: #A246E8;
  --chart-4: #FF3D9A;
  --chart-5: #6E8AF0;
}
```

- [ ] **Step 4: Update `@theme inline`** font lines. In the existing `@theme inline { ... }` block, change the three font declarations to:
```css
  --font-sans: 'Space Grotesk', system-ui, sans-serif;
  --font-serif: 'Space Grotesk', system-ui, sans-serif;
  --font-mono: 'Space Mono', ui-monospace, monospace;
```
(Leave the `--animate-marquee*` keyframes in that block untouched.)

- [ ] **Step 5: Add the dark-mode glow utility.** After the `@custom-variant dark (&:is(.dark *));` line, add:
```css
/* Subtle primary glow — dark mode only (apply via class on primary CTAs) */
.dark .glow-primary {
  box-shadow: 0 0 18px -8px var(--primary);
}
```

- [ ] **Step 6: Gate + grep verify.**
```bash
pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test
grep -nE "Space Grotesk|Space Mono|VT323|--radius: 0|#6E72F0|#5862E0" ui/src/app.css   # present
grep -ncE "Clash Grotesk|Bricolage|Geist Mono|0.65 0.15 55|b5841a" ui/src/app.css       # expect 0
```
Expected: build/check/test pass; first grep shows the new values; second grep prints `0`.

- [ ] **Step 7: Commit.**
```bash
git add ui/src/app.css
git commit -m "feat(ui): synthwave theme tokens (black + light periwinkle, square, new type)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Pixel logo — WaygateLogo, favicon, index.html

**Files:** Create `ui/src/assets/waygates-mark.svg`; Modify `ui/src/components/layout/waygate-logo.tsx`; overwrite `ui/public/favicon.svg`; Modify `ui/index.html`.

**Interfaces:**
- Produces: `WaygateLogo({ className }: { className?: string })` — now renders the multi-color pixel mark; `className` controls size only (no `currentColor`).

- [ ] **Step 1: Copy the assets.**
```bash
mkdir -p ui/src/assets
cp logo-design/waygates-mark.svg ui/src/assets/waygates-mark.svg
cp logo-design/waygates-favicon.svg ui/public/favicon.svg
```

- [ ] **Step 2: Rewrite `ui/src/components/layout/waygate-logo.tsx`** entirely:
```tsx
import markUrl from '@/assets/waygates-mark.svg';

/**
 * Waygates logo — synthwave pixel portal (multi-color, transparent).
 * NOTE: the mark is fixed multi-color; `className` controls size only.
 * The previous monochrome `currentColor` contract no longer applies.
 */
export function WaygateLogo({ className }: { className?: string }) {
  return <img src={markUrl} alt="" aria-hidden="true" className={className} />;
}
```
(Vite resolves `.svg` imports to a URL string by default. If an editor flags the `.svg` module type, it does NOT fail the oxlint gate; `ui/src/vite-env.d.ts` referencing `vite/client` already declares it.)

- [ ] **Step 3: Add `theme-color` to `ui/index.html`.** After the `<link rel="icon" ... />` line, add:
```html
    <meta name="theme-color" content="#000000" />
```

- [ ] **Step 4: Gate + verify.**
```bash
pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test
grep -q "FF3D9A" ui/public/favicon.svg && echo "favicon swapped"
grep -q "markUrl" ui/src/components/layout/waygate-logo.tsx && echo "logo uses mark"
grep -q 'theme-color' ui/index.html && echo "theme-color set"
```
Expected: gate passes; all three echoes print.

- [ ] **Step 5: Commit.**
```bash
git add ui/src/assets/waygates-mark.svg ui/public/favicon.svg ui/src/components/layout/waygate-logo.tsx ui/index.html
git commit -m "feat(ui): pixel-portal logo + favicon

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Wordmark lockup — sidebar / login / signup

**Files:** Modify `ui/src/components/layout/sidebar.tsx`, `ui/src/routes/login.tsx`, `ui/src/routes/signup.tsx`.

**Interfaces:** Consumes `WaygateLogo` (Task 2, multi-color, size-only) and `--font-display` (Task 1, VT323).

Each call-site currently wraps the logo in a `rounded bg-primary text-primary-foreground` tile (which framed the old monochrome W) and renders "Waygates" in Bricolage. The new mark is self-colored, so drop the tile wrapper; the wordmark uses VT323.

- [ ] **Step 1: sidebar.tsx** — replace the header block (currently lines ~311-321):
```tsx
          <div className="flex items-center gap-2 px-2 py-2">
            <div className="flex size-8 items-center justify-center rounded bg-primary text-primary-foreground">
              <WaygateLogo className="size-5" />
            </div>
            <span
              className="text-lg font-semibold tracking-tight"
              style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}
            >
              Waygates
            </span>
          </div>
```
with:
```tsx
          <div className="flex items-center gap-2.5 px-2 py-2">
            <WaygateLogo className="size-8" />
            <span
              className="text-xl tracking-wide"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              Waygates
            </span>
          </div>
```

- [ ] **Step 2: login.tsx — desktop brand panel** (lines ~74-82): replace
```tsx
          <div className="flex size-14 items-center justify-center rounded bg-primary text-primary-foreground">
            <WaygateLogo className="size-8" />
          </div>
          <h1
            className="mt-6 text-5xl font-bold tracking-tight leading-[1.1]"
            style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}
          >
            Waygates
          </h1>
```
with:
```tsx
          <WaygateLogo className="size-16" />
          <h1
            className="mt-6 text-6xl tracking-wide leading-[1.1]"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Waygates
          </h1>
```

- [ ] **Step 3: login.tsx — mobile brand header** (lines ~94-102): replace
```tsx
            <div className="flex size-12 items-center justify-center rounded bg-primary text-primary-foreground">
              <WaygateLogo className="size-7" />
            </div>
            <h2
              className="text-2xl font-bold tracking-tight"
              style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}
            >
              Waygates
            </h2>
```
with:
```tsx
            <WaygateLogo className="size-12" />
            <h2
              className="text-3xl tracking-wide"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              Waygates
            </h2>
```

- [ ] **Step 4: login.tsx — square the decorative circles + glow the CTA.** Change the three decorative `rounded-full` divs (lines ~69-71) `rounded-full` → `rounded-none`. On the submit button (line ~171) change `className="w-full"` → `className="w-full glow-primary"`.

- [ ] **Step 5: signup.tsx** — apply the SAME four edits as login: desktop brand panel (lines ~90-98) and mobile header (lines ~110-118) → drop the tile wrapper + `var(--font-display)` (use `size-16`/`text-6xl` desktop, `size-12`/`text-3xl` mobile); the three decorative `rounded-full` → `rounded-none` (lines ~85-87); the submit button (line ~232) `className="w-full"` → `className="w-full glow-primary"`.

- [ ] **Step 6: Gate + verify.**
```bash
pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test
grep -rnc "Bricolage" ui/src/components/layout/sidebar.tsx ui/src/routes/login.tsx ui/src/routes/signup.tsx   # expect 0 each
grep -rn "var(--font-display)" ui/src/components/layout/sidebar.tsx ui/src/routes/login.tsx ui/src/routes/signup.tsx   # present
grep -rc "bg-primary text-primary-foreground" ui/src/routes/login.tsx ui/src/routes/signup.tsx   # expect 0 (logo tile removed)
```
Expected: gate passes; no Bricolage refs remain in these three files; `var(--font-display)` present; the logo `bg-primary` tile gone.

- [ ] **Step 7: Commit.**
```bash
git add ui/src/components/layout/sidebar.tsx ui/src/routes/login.tsx ui/src/routes/signup.tsx
git commit -m "feat(ui): VT323 wordmark lockup + pixel mark in sidebar/login/signup

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: ACL login default branding + stray font ref

**Files:** Modify `ui/src/routes/auth/acl-login.tsx`, `ui/src/components/settings/acl-branding-settings.tsx`, `ui/src/routes/theme-preview.tsx`.

The public ACL login is tenant-configurable but its built-in defaults still carry the copper `#b5841a`; align them to periwinkle. (Admins can still override per-tenant.)

- [ ] **Step 1: acl-login.tsx** — line 18, change `primary_color: '#b5841a',` → `primary_color: '#6E72F0',`.

- [ ] **Step 2: acl-branding-settings.tsx** — line ~53, change `primary_color: '#b5841a',` → `primary_color: '#6E72F0',`. Line ~138, change the hint text `Please enter a valid hex color (e.g., #b5841a)` → `Please enter a valid hex color (e.g., #6E72F0)`.

- [ ] **Step 3: theme-preview.tsx** — line ~249, change `style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}` → `style={{ fontFamily: 'var(--font-display)' }}`.

- [ ] **Step 4: Gate + verify.**
```bash
pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test
grep -rnc "b5841a" ui/src   # expect 0
grep -rnc "Bricolage" ui/src   # expect 0 (app.css + all inline refs now gone)
```
Expected: gate passes; both greps print `0` (no copper hex, no Bricolage anywhere).

- [ ] **Step 5: Commit.**
```bash
git add ui/src/routes/auth/acl-login.tsx ui/src/components/settings/acl-branding-settings.tsx ui/src/routes/theme-preview.tsx
git commit -m "feat(ui): align ACL login defaults + theme-preview to new identity

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Square-corner sweep + visual verification

**Files:** Potentially small edits across `ui/src` (squaring stray rounded surfaces). No new files.

`--radius: 0` (Task 1) squares everything that uses `rounded`/`rounded-sm/md/lg/xl` (they map to `--radius`). What remains are **fixed** Tailwind radii — `rounded-full` and `rounded-[…]` — on surface elements that should also be square per "no rounded borders" (avatars, status pills, chips). Loading spinners / genuinely circular icons stay as-is.

- [ ] **Step 1: Inventory the fixed-radius usages.**
```bash
grep -rnE "rounded-full|rounded-\[" ui/src --include=*.tsx | grep -viE "node_modules"
```
For each hit, decide: **square it** if it's a container/surface/badge/avatar border (change `rounded-full` → `rounded-none`); **leave it** if it's a spinner, a genuinely circular status dot, or chart geometry. Known targets to square: the user avatars in `sidebar.tsx` (`Avatar className="size-8"` / `size-16`) and any `rounded-full` status pills/badges. (The login/signup decorative circles were already squared in Task 3.)

- [ ] **Step 2: Apply the squaring edits** decided in Step 1 (add `rounded-none` to the avatar `Avatar`/inner `div` className and any surface pills). Keep edits minimal and surface-only.

- [ ] **Step 3: Confirm no stale theme remnants remain.**
```bash
grep -rncE "Clash Grotesk|Bricolage|Geist Mono|b5841a" ui/src   # expect 0
grep -rn "currentColor" ui/src/components/layout/waygate-logo.tsx   # expect no output (contract dropped)
```

- [ ] **Step 4: Visual smoke (both themes).** Run `pnpm --dir ui dev` (or `pnpm --dir ui build && pnpm --dir ui preview`) and confirm in the browser, toggling light/dark:
  - Dashboard: black (dark) / `#F7F7FA` (light) canvas; periwinkle primary buttons; cyan focus ring on inputs; square cards/inputs/badges; charts use the logo palette.
  - Sidebar: pixel mark + VT323 "Waygates"; active nav in periwinkle.
  - Login + signup: pixel mark + VT323 wordmark; squared brand panel; `glow-primary` on the submit button (dark).
  - A data table + a chart render legibly; status badges (Space Mono) readable.
  - Favicon shows the portal in the browser tab; logo crisp at small sizes.
  - (No live Caddy needed — this is pure UI.)

- [ ] **Step 5: Final gate + commit.**
```bash
pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test
git add -p ui/src   # stage only the squaring edits you made
git commit -m "feat(ui): square stray rounded surfaces; finalize rebrand sweep

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```
(If Step 1 found nothing to change, skip the commit and note the sweep was clean.)

---

## Self-Review

**Spec coverage:** dark+light tokens (T1); square `--radius:0` (T1 + sweep T5); periwinkle primary + cyan ring + chart palette (T1); fonts VT323/Space Grotesk/Space Mono imports+tokens, copper fonts dropped (T1, verified T4); dark glow utility (T1) applied to CTAs (T3); pixel mark in WaygateLogo + favicon + index theme-color (T2); VT323 wordmark in sidebar/login/signup + tile-wrapper removal (T3); ACL login default branding (T4); sweep for rounded/stale refs + both-theme visual smoke (T5). All spec sections map to a task.

**Placeholder scan:** every code step has concrete code or exact old→new strings + line anchors; gate commands have expected output; no TBD/TODO.

**Consistency:** `--font-display` defined in T1, consumed in T3/T4; `WaygateLogo({className})` defined T2, consumed T3; palette hexes match the spec table verbatim; `glow-primary` defined T1, applied T3; grep assertions (`Bricolage`/`b5841a` → 0) are consistent across T3/T4/T5.
