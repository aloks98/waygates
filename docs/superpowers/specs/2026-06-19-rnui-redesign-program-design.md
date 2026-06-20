# Waygates UI Redesign + rnui Migration — Program Design

**Date:** 2026-06-19
**Status:** Approved (design)
**Author:** Alok Kumar Sahoo (with Claude)

## Problem

The Waygates UI is a competent but conventional admin-CRUD app built on
`@e412/titanium` (89 distinct components in use across ~40 files). It works, but:

- It is **under-designed for the one thing a reverse-proxy manager must do well:
  give the operator confidence that a change took effect and the thing behind it
  is reachable.** Changes apply through a 60s sync; a bad upstream or a failed
  Caddy reload can silently leave you broken. There is no per-proxy health, no
  visibility into the generated config, and no prominent applied/syncing/failed
  signal.
- **Power-user speed is missing** — no command palette, no keyboard shortcuts, no
  bulk actions, no clone, no config inspection.
- **The IA is jargon-first** ("L4 Proxies", "ACL", "catchall", "bypass") and
  fragments one mental model ("things I expose") across two nav items.
- The complex flows (reverse-proxy form, ACL's six tabs) **dump all complexity
  up front** with no progressive disclosure or inline help.

A UX critique of the current app scored it **26/40** on Nielsen's heuristics —
functional, but weak on system status (P0), flexibility/efficiency (P1), and
help/error-recovery (P2). Full critique summary in [§5](#5-ux-critique-summary).

At the same time, `@e412/rnui-react` + `@e412/rnui-themes` (the author's
successor to titanium: Base UI + shadcn patterns + Tailwind v4, ~85 exports) is
ready and wants real-world dogfooding. It ships exactly the primitives this
product needs — `Command`, `CodeBlock`/`JsonViewer`, `StatCard`/charts,
`Timeline`, `StatusIndicator`, `EmptyState`, `Stepper` — plus a theming system.

## Goal

Migrate the UI from `@e412/titanium` to `@e412/rnui-react`, and in the same pass
**rethink the experience around trust, speed, and legibility** — while
deliberately exercising rnui's breadth so the migration also hardens the library.

This is an **integrated redesign**: each screen is migrated *and* redesigned
together, not a mechanical swap followed by a separate redesign.

### Non-goals

- **Theme switcher / multi-theme.** Single branded theme only (light + dark).
- **Full ACL redesign this round.** Access/ACL is migrated 1:1 (component swap);
  its IA redesign is deferred to a later pass (see [§11 M5](#11-milestone-roadmap)).
- **No new product domains.** We redesign and add QoL to what exists; we do not
  add features like multi-node clustering, DNS management, etc.
- **No auth/RBAC model changes.** The permission model and goauth wiring stay as-is.

## Locked decisions

| # | Decision | Choice |
|---|----------|--------|
| 1 | Primary goal | Better UX **and** broad rnui dogfooding (equal weight) |
| 2 | Approach | Integrated redesign (migrate + redesign + features per screen) |
| 3 | Form stack | Migrate to **React Hook Form + Zod** (`@hookform/resolvers` zodResolver), using rnui's `Form`/`FormField` helpers |
| 4 | Theming | **Single branded theme** — port the existing "copper" OKLCH palette into a custom rnui preset; light + dark via `next-themes`; **no picker** |
| 5 | Backend trust features | **All three in scope**: config preview (easy), per-proxy health (moderate), traffic metrics + charts (large) |
| 6 | Information architecture | **Restructure + rename** — unify L7+L4 under one Proxies section; ACL→Access; Audit→Activity; add breadcrumbs |
| 7 | Complex-screen scope | Redesign Dashboard / Proxies / Activity + the proxy forms; **migrate Access/ACL 1:1**, redesign later |
| 8 | Docs | One master program spec (this doc) + a per-milestone spec/plan as each milestone is reached |

## Personas

- **Sam — homelabber, first proxy.** Semi-technical, comes from
  nginx-proxy-manager. Needs to get the first proxy working fast, trust HTTPS,
  and *know it worked*. Allergic to jargon.
- **Devon — sysadmin, 30+ proxies.** Power user. Wants keyboard speed, bulk ops,
  clone, and to inspect exactly what config got generated. Will drop to the
  Caddyfile if the UI is slower than editing it directly.
- **Riley — security-minded.** Wants an at-a-glance answer to "what is exposed,
  and what auth guards each path?" Today that answer is spread across six ACL
  tabs.

## 5. UX critique summary

The rethink that motivates this program. Anti-pattern check: the copper identity
is genuinely distinctive (not AI slop); the *layouts* are conventional and are
the real opportunity.

| Heuristic | Score | Issue driving the redesign |
|-----------|:----:|----------------------------|
| Visibility of system status | 3/4 | No per-proxy health, no applied/pending signal, no generated-config view |
| Match real world | 2/4 | "L4 Proxies", "ACL", "catchall", "bypass" |
| User control & freedom | 3/4 | No clone, undo, or bulk ops |
| Consistency | 3/4 | Mixed create patterns (ACL modal vs proxy full-page); "Proxies" vs "L4 Proxies" |
| Error prevention | 3/4 | Hostname uniqueness only; no path/route conflict detection |
| Recognition vs recall | 3/4 | No command palette, no breadcrumbs in deep pages |
| Flexibility & efficiency | 2/4 | No shortcuts, palette, bulk, or templates |
| Aesthetic & minimalist | 3/4 | Forms expose everything at once |
| Error recovery | 2/4 | Failed Caddy reload likely surfaced generically |
| Help & documentation | 2/4 | No inline help/onboarding |
| **Total** | **26/40** | **Moderate — under-designed for trust & speed** |

**Priority themes:** trust/verification (P0), speed (P1), legibility/onboarding
(P1–P2). The roadmap is organized to deliver these.

## 6. New information architecture

```
Dashboard      control-plane overview (status, counts, recent activity, honest charts)
Proxies        tabs:  HTTP  ·  TCP/UDP      (unify L7+L4; "TCP/UDP" replaces "L4")
Access         (was "ACL") groups + per-proxy assignments
Activity       (was "Audit Logs") timeline + config diffs
Settings
———
global:  ⌘K command palette  ·  breadcrumbs  ·  sync/apply status in the top bar
```

Renames are UI-only (labels + route paths where worth it); the API and RBAC
permission keys (`proxies:*`, `acl:*`, `audit:read`, …) are unchanged.

## 7. Theming

- **Source of truth:** the existing copper tokens in `ui/src/app.css` (primary
  `oklch(0.65 0.15 55)`, warm-cream background, tight `0.25rem` radius; fonts
  Clash Grotesk / Bricolage Grotesque / Geist Mono).
- **Port** those values into rnui's CSS variable names (`--primary`,
  `--background`, `--radius`, `--font-sans`, `--font-heading`, the `--chart-*`
  ramp, `--sidebar-*`, etc.) as a custom theme layered on rnui's base, with a
  matching `.dark` override block. CSS setup:
  ```css
  @import 'tailwindcss';
  @import '@e412/rnui-themes';            /* base + light + dark */
  @source "../node_modules/@e412/rnui-react/dist";
  /* custom copper theme variables defined after the import */
  ```
- **Dark mode** via `next-themes` (`attribute="class"`), preserving today's
  dark-first feel. Fonts are loaded by the app (Google Fonts/self-host); rnui
  only declares the variable names.
- **No theme picker.** The 8 rnui presets are not shipped to users.

## 8. Cross-cutting foundations

### Forms (RHF + Zod)
- Adopt rnui's `Form`/`FormField`/`FormItem`/`FormControl`/`FormMessage` (shadcn
  RHF pattern) with `useForm({ resolver: zodResolver(schema) })`.
- **Reuse the existing Zod schemas** in `ui/src/lib/form-validation.ts`; they are
  framework-agnostic. Build a small set of reusable field wrappers
  (text, select, switch, tags, number) over rnui primitives so domain forms stay
  declarative.
- **Exception:** Access/ACL forms (M5, migrate-only) keep TanStack Form
  temporarily — only their *components* swap to rnui primitives. They move to RHF
  when ACL is redesigned later. This is an accepted transitional state.

### App providers (root)
```tsx
<ThemeProvider attribute="class" defaultTheme="dark" enableSystem>
  <TooltipProvider>
    <SidebarProvider>
      <Toaster />
      <App />
```
React Query + TanStack Router setup are unchanged.

### Component migration map (titanium → rnui)
Most primitives map ~1:1: `Button`, `Card*`, `Dialog*`, `Sheet*`,
`AlertDialog*`, `Tabs*`, `Badge`, `Select*`, `Input`, `Textarea`, `Checkbox`,
`Switch`, `Skeleton`, `DropdownMenu*`, `Tooltip*`, `Alert*`, `Separator`,
`Avatar`, `Label`, `ScrollArea`, `Sidebar*`. Toasts stay on `sonner`
(`toast.*` + rnui `Toaster`). Notable differences:

| Titanium | rnui | Note |
|----------|------|------|
| `DataGrid` (built-in) | `DataGrid` + `DataGridContainer` + `DataGridTable` + `DataGridPagination` | Compositional; same TanStack Table instance pattern we already use — near-mechanical |
| `Filters` | `DataGridColumnFilter` / `filters` | Re-implement the audit/proxy filters |
| `TagsInput*` | `Combobox` (multi) or small custom tags field | No direct rnui `TagsInput`; used in headers/allow-lists |
| `ThemeProvider` (titanium) | `next-themes` `ThemeProvider` | Theming moves to CSS + next-themes |
| `Spinner` | `Spinner` | direct |

Gaps to resolve in **M0** before domain migration: `Filters` and `TagsInput`.

### Migration strategy
- **No long-lived dual-library state.** M0 replaces titanium globally with rnui
  equivalents in the *current* layouts (so the app builds and runs), then domain
  milestones redesign on top. titanium is removed in M0.
- **Incremental & shippable.** Every milestone leaves the app compiling and
  green. After each milestone run the project checklist: `make lint-ui`,
  `pnpm --dir ui build`, `make backend-test` (for backend pipelines), `make check`.
- Prefer git-worktree isolation for parallel milestone/pipeline work.

## 9. QoL feature catalog

**Baseline (UI-only; backend already supports — included across milestones):**

| Feature | rnui | Lands in |
|---------|------|----------|
| ⌘K command palette (nav + actions) | `Command` | M0 |
| Sync/apply confidence bar ("applied / syncing / failed: {error}" + Apply now) | `StatusIndicator`, `Alert` | M0 (`/sync/status`+`/sync/trigger`) |
| Theme toggle (light/dark) | next-themes | M0 |
| Breadcrumbs on deep pages | `Breadcrumb` | M0 |
| Copy buttons (hostname, URL, snippets) | `CopyButton` | M0+ |
| Empty states + first-run checklist (`/status.user_setup_complete`) | `EmptyState`, `Stepper` | M1/M2 |
| Honest charts (proxy-type mix `by_type`; audit action/status mix) | `PieChart`/`BarChart` | M1 |
| Bulk enable/disable/delete (row-select → N calls + progress) | `DataGrid` selection | M2/M3 |
| Duplicate proxy (prefill create form from existing) | — | M2/M3 |
| Export/Import proxies as JSON (GET all → file; import = loop create) | `CodeBlock` | M2 |
| "What protects this proxy" summary (`/proxies/{id}/acl`) | `Badge`, `HoverCard` | M2 |
| Progressive disclosure on complex forms (advanced collapsed) | `Collapsible`/`Accordion` | M2/M3 |
| First-proxy guided flow | `Stepper` | M2 |

**Backend-gated (the marquee trust features — see §10):** generated config
preview, per-proxy/upstream health, traffic metrics + time-series charts.

## 10. Backend pipelines

Each is independent, runs parallel to the UI work, and gets its own spec/plan.

### B1 — Config preview *(easy)*
The generated Caddy JSON already lives on disk (`FileManager.ReadJSONConfig()`).
Expose read endpoints; UI renders with `CodeBlock`/`JsonViewer`.
- `GET /api/config/current` → full generated `caddy.json`
- `GET /api/proxies/{id}/config/preview` → built JSON for one proxy
- RBAC: `proxies:read` (and a `settings:read`/`sync:read` gate for the full doc).

### B2 — Per-proxy health *(moderate)*
Caddy probes upstreams but the backend never reads results. Add a poller that
queries Caddy's admin API for upstream health, cache it, and expose:
- `GET /api/proxies/{id}/health` → per-upstream up/down + last-checked
- (optional) `GET /api/proxies/health` → batch for the list view
UI: `StatusIndicator` badges in the Proxies list and Dashboard.

### B3 — Traffic metrics *(large)*
There is **zero traffic data today**; without this, dashboard graphs would be
dishonest. Build the pipeline:
- Enable/scrape Caddy's Prometheus `/metrics` (or Caddy admin metrics).
- Store time-series (new table(s) / lightweight rollups; choice deferred to B3's
  own spec — evaluate a TSDB-lite table vs. embedded store).
- Expose `GET /api/metrics/overview` and `GET /api/proxies/{id}/metrics?range=…`
  (requests, bandwidth, status-code mix, latency).
UI: real charts on Dashboard + per-proxy detail.

## 11. Milestone roadmap

**M0 — Foundation + App Shell** *(blocks everything)*
rnui+themes install, titanium removal, Tailwind v4 CSS wiring, branded copper
theme + light/dark, root providers, RHF+Zod form foundation + field wrappers,
gap fills (`Filters`, `TagsInput`), new sidebar (new IA) + top bar (sync/apply
status, ⌘K palette, theme toggle, user menu, breadcrumbs), command palette.
*Outcome: app builds & runs on rnui with the new shell; titanium gone.*

**M1 — Dashboard** — status (`StatusIndicator`), counts (`StatCard`), recent
activity (`Timeline`), honest categorical charts, empty/first-run state. Wires
B1/B2/B3 widgets as those land.

**M2 — Proxies: HTTP** — unified list (bulk actions, duplicate, "what protects
this", config-preview + health badges, export/import); forms → RHF + progressive
disclosure + first-proxy stepper + inline help.

**M3 — Proxies: TCP/UDP (L4)** — same patterns under the TCP/UDP tab.

**M4 — Activity (Audit)** — `Timeline` + config-diff (`JsonViewer`), filters,
export.

**M5 — Access (ACL)** — *migrate-only*: rnui component swap, keep TanStack Form
temporarily; full redesign deferred.

**M6 — Settings** — migrate + tidy.

**Sequence:** M0 → (B1 + M1 + M2 in parallel) → B2 → (M3 + M4) → B3 → (M5 + M6).
Backend pipelines feed UI screens as they complete; screens ship with existing
data first and gain the richer widgets when the pipeline lands.

```
M0 ─┬─ B1 ─┐
    ├─ M1 ←┼← B2 ←──── B3
    ├─ M2 ←┘
    ├─ M3
    ├─ M4
    ├─ M5
    └─ M6
```

## 12. Risks & mitigations

- **Big-bang foundation diff (M0).** Mitigate: M0 keeps current layouts (parity
  swap), so behavior is unchanged while the platform moves; redesign lands
  per-screen afterward.
- **Two form libraries during transition** (RHF everywhere, TanStack in ACL).
  Accepted and bounded; removed when ACL is redesigned.
- **rnui gaps surface mid-migration** (e.g., `TagsInput`). Expected — this is
  dogfooding. Capture gaps; fix in rnui or wrap locally.
- **B3 scope creep.** Time-series storage is the heaviest item; its own spec
  decides the storage approach before any UI depends on it. Dashboard works
  without it.
- **Caddy admin API coupling (B2).** Health polling depends on Caddy admin API
  shape; isolate behind a service with its own tests.

## 13. Success criteria

- App fully on `@e412/rnui-react`; `@e412/titanium` removed from `package.json`.
- New IA live (Proxies HTTP/TCP-UDP tabs, Access, Activity); breadcrumbs + ⌘K +
  sync/apply status + theme toggle present.
- All baseline QoL features shipped; the three backend pipelines delivered with
  their UI surfaces.
- A re-run of `/critique` materially improves the heuristic score (target ≥ 33/40),
  especially system status, flexibility/efficiency, and help.
- `make check` green; `pnpm --dir ui build` clean; existing E2E/integration suites pass.

## 14. Out of scope

Multi-theme picker; ACL IA redesign (deferred); new product domains; auth/RBAC
model changes; performance/load work; real bulk-API endpoints and import parsers
for non-Waygates formats (UI-side loop is sufficient this round).

## 15. Document map

- **This doc** — program-level design (vision, decisions, IA, QoL catalog,
  component map, theming, pipelines, roadmap).
- **Per-milestone specs/plans** — each milestone (M0–M6) and pipeline (B1–B3)
  gets its own `docs/superpowers/specs/<date>-<milestone>-design.md` + an
  implementation plan when we reach it. **Next:** M0 (Foundation + App Shell).
