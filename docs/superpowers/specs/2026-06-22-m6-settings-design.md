# M6 — Settings Redesign — Design

**Date:** 2026-06-22
**Program:** rnui UI redesign (M6; follows M5 / PRs #29–#30)
**Status:** Approved — ready for implementation plan

## Goal

Restructure the admin **Settings** area from a single tabbed page into a **sectioned, left-nav layout with one deep-linkable route per section**, matching the rnui design language and the multi-route pattern established by Access (M5). Along the way, migrate the one remaining admin TanStack form (`catchall-settings.tsx`) to React Hook Form. **Frontend-only — no backend change.**

## Context

Unlike M1–M5, Settings is **already on rnui** (`@e412/rnui-react`) — there is no `@e412/titanium` left here. So M6 is an **information-architecture + UX restructure**, not a component-library migration.

Today (`ui/src/routes/_dashboard/settings.tsx`): a single page (`space-y-6` container, `<h1>Settings</h1>`) with a 3-tab `Tabs` control:

| Tab (today) | Component | State / form lib | Backing hook → endpoint |
|---|---|---|---|
| Default Page | `components/settings/catchall-settings.tsx` (185 lines) | **`@tanstack/react-form`** | `useNotFoundSettings()` → `GET/PUT /settings/404` |
| Audit Logs | `components/audit-logs/audit-config-panel.tsx` (435 lines) | bespoke `useState` (grouped-checkbox tree, master toggle, dirty tracking) | `useAuditConfig()` → `GET/PUT /audit-logs/config`; `useAuditEventGroups()` → `GET /audit-logs/event-groups` |
| Login Branding | `components/settings/acl-branding-settings.tsx` (566 lines) | **react-hook-form** (migrated in M5b) | `useACLBranding()`/`useUpdateACLBranding()` → `GET/PUT /acl/branding` |

All endpoints are stable; no new endpoints are needed.

The codebase already uses layout routes with `<Outlet/>` (`_dashboard/route.tsx`) and directory-based child routes (`_dashboard/acl/index.tsx` + `$groupId.tsx`).

## Scope

**In:**
- Replace the tabbed `settings.tsx` with a **settings layout route + child route per section** (left-nav shell + `<Outlet/>`), with an index redirect.
- A reusable **`SettingsNav`** vertical nav (icon + label + short description, active-route highlight), responsive to a horizontal strip / Select on narrow viewports.
- Three section routes: **Login Branding**, **Default Page**, **Audit Logs**.
- **Migrate `catchall-settings.tsx` `@tanstack/react-form` → RHF + `zodResolver`** (parity: same fields, validation, payload), per the project form convention.
- Re-home the existing Branding and Audit panels into their section routes with light visual polish for consistency.

**Out of scope:**
- New settings surfaces (TLS/ACME, server config, account/profile) — these are env-var config with no DB endpoints; adding them would require backend work. The shell is built so they can be added later (one nav entry + one child route).
- Any backend change.
- `acl-login-form.tsx` (end-user login page, still TanStack) — not part of Settings.
- Migrating the Audit panel's bespoke `useState` to RHF (decision below).

## IA & routing

Replace `_dashboard/settings.tsx` with a `settings/` route directory:

```
_dashboard/settings/route.tsx          # layout: "Settings" header + <SettingsNav> + <Outlet/>
_dashboard/settings/index.tsx          # redirect → /dashboard/settings/default-page
_dashboard/settings/default-page.tsx   # renders <CatchallSettings/> (now RHF)
_dashboard/settings/login-branding.tsx # renders <ACLBrandingSettings/>
_dashboard/settings/audit-logs.tsx     # renders <AuditConfigPanel/>
```

- **Nav order:** Default Page · Login Branding · Audit Logs.
- Each section is its own deep-linkable URL (`/dashboard/settings/<section>`).
- The index route redirects to the first section (**Default Page**, preserving today's default landing — the current page opens on `defaultValue="catchall"`) so `/dashboard/settings` always lands somewhere.
- The sidebar "Settings" entry continues to point at `/dashboard/settings` (the redirect resolves it).
- Layout container keeps the existing convention (`space-y-6`, `<h1 className="text-2xl font-bold">Settings</h1>`); no extra `max-w`/`mx-auto`, consistent with proxies/activity/access.

## Components

- **`components/settings/settings-nav.tsx`** (new) — the left vertical nav. A small static list of `{ to, label, description, icon }`; renders rnui nav items with active-route styling (via the router's active state). Responsive: below a breakpoint it renders as a horizontal scrollable strip (or Select) above the content.
- **`components/settings/settings-layout.tsx`** (new, optional) — the two-column shell (`SettingsNav` + content `<Outlet/>` slot); may live inline in `route.tsx` if small.
- **`catchall-settings.tsx`** — migrated to RHF (`useForm` + `zodResolver`, `Form`/`FormField`/`FormItem`/`FormControl`/`FormMessage`); same `mode` radio (default 404 vs redirect) + redirect-URL input, same validation, same `useNotFoundSettings` payload. The `Field*` primitives it currently uses are replaced by `Form*` (binding) — `RadioGroup` binds via the `FormField` render-prop (`value`/`onValueChange`).
- **`acl-branding-settings.tsx`** — unchanged logic (already RHF); re-homed; optional light polish (form + live preview layout within the pane).
- **`audit-config-panel.tsx`** — unchanged state model; re-homed; visual polish only.

## Decisions preserved / made

- **Form convention:** all forms bind via RHF `Form`/`FormField` (not shadcn `Field` primitives for binding) — see the `forms-use-rhf-form-component` convention. `catchall` joins Branding on RHF.
- **Audit panel keeps its bespoke `useState`** (grouped-checkbox tree with master toggle, indeterminate groups, dirty-tracked save). It is not a TanStack form, so it doesn't conflict with the "no TanStack" goal; modeling a dynamic checkbox tree in RHF is risk for no real gain. Restyle only. *(User-approved.)*
- **`catchall` migration is parity** — validation rules, the `mode`/`redirect_url` payload, and behavior are unchanged; library swap only.

## Decomposition

Subagent-driven (SDD), one independently gate-able task each:
1. **Settings layout route + `SettingsNav` shell + index redirect** — the new shell wired to the three child routes (rendering the existing panels as-is), old `settings.tsx` removed. Deliverable: navigable left-nav settings with all 3 sections reachable by URL.
2. **Login Branding section** — re-home `ACLBrandingSettings` into its route + light polish.
3. **Default Page section + `catchall` → RHF** — migrate the form (parity) and mount it in its route.
4. **Audit Logs section** — re-home `AuditConfigPanel` + visual polish.
5. **Responsive nav + final sweep** — responsive `SettingsNav`, spacing/consistency pass, confirm no dead code / no `@tanstack/react-form` in `components/settings`.

(The plan finalizes exact task boundaries; tasks 2/4 are light re-homing, task 3 carries the form migration.)

## Testing

Frontend-only; verification is the gate: `pnpm --dir ui build` + `pnpm --dir ui check` + `pnpm --dir ui test`, all green (the existing 57 tests must keep passing). No new unit tests unless a pure helper is extracted (e.g. the nav-section list). Per-task reviewer focus: for the `catchall` migration, **parity** (same fields/validation/payload); for the rest, that re-homing preserved behavior and the routing/redirect works.

## Risks & notes

- **Route conversion** (`settings.tsx` → `settings/` directory with `route.tsx` + children): ensure the index redirect and the sidebar link resolve correctly, and TanStack's generated route tree picks up the new files (run the dev/build so the routegen updates).
- **`catchall` RHF parity** — the `mode` `RadioGroup` + conditional redirect-URL field must keep the same validation (URL required only when mode = redirect) and the same payload shape.
- **Responsive nav** — the left rail must not crush the form panes on small screens; degrade to a horizontal/Select nav.
- Parity is the bar for the `catchall` form; the rest is re-home + restyle with behavior unchanged.
