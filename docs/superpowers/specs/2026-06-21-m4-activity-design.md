# M4 — Activity (Audit) Redesign — Design

**Date:** 2026-06-21
**Program:** rnui UI redesign (see `2026-06-19-rnui-redesign-program-design.md` §11, milestone M4)
**Status:** Approved — ready for implementation plan

## Goal

Redesign the Activity page (the audit-log viewer) on top of the existing 1:1-migrated
screen: add a **Timeline view** (with a Table toggle), a **field-level change diff** in
the detail view, a **date-range filter**, and keep CSV export. Frontend-only — the audit
backend API already supports everything needed.

## Context — current state

- **Backend (no changes needed).** `audit_logs` API supports: list with `operator:value`
  filters (`action`, `status`, `resource_type`, `ip_address`, `user_id`, `search`,
  `date_from`, `date_to`, `sort`, `order`), get-by-id, stats, config get/update,
  event-groups, and CSV export. The `details` JSON field, for **update** events, is shaped
  `{ "<field>": { "old": <v>, "new": <v> }, ... }` (`buildProxyChanges` in
  `backend/internal/api/handlers/proxy.go`). For create/delete/other events `details` is an
  arbitrary snapshot/object.
- **IA already renamed.** `/dashboard/activity` resolves to the page; `/dashboard/audit-logs`
  redirects to it; sidebar/breadcrumb/command-palette say "Activity". Only the page *heading*
  still says "Audit Logs".
- **Current page** (`routes/_dashboard/audit-logs.tsx`, exports `AuditLogsPage`): search +
  rnui `Filters` (4 fields: action, status, resource_type, ip_address) + `AuditDataGrid`
  (table) + CSV export. Row click opens `AuditLogDetailSheet`.
- **Config panel** (`AuditConfigPanel`, event-logging toggles) is rendered on the **Settings**
  page (`routes/_dashboard/settings.tsx`), NOT on Activity. It is out of scope for M4 and is
  left untouched (Settings redesign is M6).
- **Hooks** (`hooks/use-audit-logs.ts`, reused as-is): `useAuditLogs`, `useAuditLogById`,
  `useAuditStats`, `useAuditConfig`, `useAuditEventGroups`, `useExportAuditLogs`.
- **Types** (`types/audit.ts`, reused as-is): `AuditLog`, `AuditAction`, `AuditStatus`,
  `AuditResourceType`, `AuditLogListParams` (already includes `date_from`/`date_to`).

## Locked decisions (from brainstorming)

1. **View model:** Timeline + Table **toggle**. Timeline is the default/headline; toggle
   switches to the table. Both share the same toolbar (search/filters/export).
2. **Config panel:** stays on Settings (already there); not touched in M4.
3. **Diff rendering:** **field-level old→new diff** for update events; JsonViewer for
   create/delete/other event details.
4. **View toggle persists in the URL** (`?view=timeline|table`) via the route's search params.

## Scope

**In:** Activity page redesign — toolbar (search + filters + date-range + view toggle +
export), Timeline view, reworked Table view, enhanced detail sheet with field diff, shared
action-metadata map, page renamed to "Activity", URL-persisted view toggle.

**Out:** any backend change; the audit config panel (Settings/M6); a stats/summary header
(the Dashboard already shows audit charts); JSON export (CSV stays); ACL/Settings screens.

## Architecture & files

### Route
- **Rename** `ui/src/routes/_dashboard/audit-logs.tsx` → `ui/src/routes/_dashboard/activity.tsx`,
  export `ActivityPage` (was `AuditLogsPage`).
- Update `ui/src/lib/router.tsx`: the `/activity` route imports `ActivityPage` from
  `@/routes/_dashboard/activity`; it validates a search param `view` (`'timeline' | 'table'`,
  default `'timeline'`). The `/audit-logs` → `/activity` redirect is unchanged.
- Page `<h1>` reads **"Activity"**.

### New components — `ui/src/components/activity/`
- **`activity-actions.ts`** — single source of truth mapping each `AuditAction` →
  `{ label: string; icon: LucideIcon; tone: 'create' | 'update' | 'delete' | 'auth' | 'system' | 'neutral' }`.
  `label` is a humanized phrase (e.g. `proxy.create` → "Created proxy"). Exposes a helper
  `getActionMeta(action: string)` returning a safe fallback for unknown actions. Used by the
  timeline, table, and detail header. (Mirrors the `L4_MATCHER_CONFIG` pattern.)
- **`activity-toolbar.tsx`** — `ActivityToolbar` props: current search, filters, date range,
  `view`, and change callbacks. Renders the search `Input`, rnui `Filters` (the 4 existing
  fields **+ a date-range field**), the Timeline/Table toggle (rnui `Tabs` or segmented
  control), and the Export CSV `Button`. Owns no data fetching — pure controlled component.
- **`activity-timeline.tsx`** — `ActivityTimeline` props: `logs`, `isLoading`, `onSelect(id)`.
  Groups logs by calendar day (newest first) with sticky day headers ("Today" / "Yesterday" /
  formatted date); renders one `ActivityEventRow` per log = action icon (tone-colored) +
  humanized label + resource name (links to the proxy route when `resource_type==='proxy'` &&
  `resource_id`) + user (username or "system") + relative time + status badge. **Failure events
  use a destructive-toned marker.** Loading skeleton; empty state. Row click → `onSelect(id)`.
- **`activity-table.tsx`** — `ActivityTable` reworked from the existing `AuditDataGrid`:
  `DataGrid` columns (Time, Action [icon+label], Resource, User, IP, Status), manual
  pagination, row click → select. Uses `activity-actions` for the Action cell.
- **`activity-detail-sheet.tsx`** — `ActivityDetailSheet` props: `logId`, `open`,
  `onOpenChange`. Fetches via `useAuditLogById`. Renders: header (action label + status
  badge), metadata grid (user, IP, user-agent, timestamp, resource), the **change section**
  (see below), and an `error_message` callout (destructive) when status is failure.
- **`activity-diff.ts`** — pure helper `extractFieldChanges(details)` → `FieldChange[]`
  (`{ field, old, new }`), detecting the `{field:{old,new}}` shape; returns `null`/empty when
  details is not an update-diff so the sheet falls back to the JsonViewer. Keeps the diff
  parsing testable in isolation.

### Reused unchanged
`hooks/use-audit-logs.ts`, `types/audit.ts`. The toolbar wires `date_from`/`date_to` into the
existing `AuditLogListParams` (no hook change).

### Removed after the new ones land
`components/audit-logs/audit-data-grid.tsx` and `components/audit-logs/audit-log-detail-sheet.tsx`
(superseded). `components/audit-logs/audit-config-panel.tsx` **stays** (used by Settings); the
`components/audit-logs/index.ts` barrel is updated to export only what Settings still needs.

## Data flow

`ActivityPage` owns state: `search` (debounced), `filters` (rnui `Filter[]`, debounced),
`dateRange`, `pagination`, `view` (from URL search param), `selectedLogId` + `sheetOpen`. It
builds `AuditLogListParams` (same operator-mapping logic as today, plus `date_from`/`date_to`)
and calls `useAuditLogs(params)`. It renders `ActivityToolbar` (always) then either
`ActivityTimeline` or `ActivityTable` per `view`, and the `ActivityDetailSheet`. Changing the
toggle calls `navigate({ search: { view } })` so the view is bookmarkable/back-button aware;
the page reads `view` from `useSearch`. Filter/search/date changes reset to page 1.

Pagination applies to both views (timeline paginates the same as the table — page size
selector in the toolbar or a "load page" control; default 20, matching today).

## Detail view — change rendering

1. Compute `changes = extractFieldChanges(log.details)`.
2. If `changes` is non-empty (update event) → render a **field diff list**: each row =
   field name, `old` value (muted + line-through) → `new` value (emphasized/success tone).
   Values rendered via a small formatter (stringify objects/arrays compactly).
3. Else if `log.details` is a non-empty object → render a collapsible **JsonViewer**
   (rnui `JsonViewer`/`CodeBlock`) of the raw details.
4. Else → "No additional details."
5. Always: metadata grid + (on failure) the `error_message` callout.

## Filters & export

- **Filters:** the 4 existing fields kept verbatim (action multiselect, status select,
  resource_type multiselect, ip_address text). **Add** a date-range field that sets
  `date_from`/`date_to` (rnui date-range input or two date inputs in the `Filters` config; if
  rnui `Filters` can't host a range field cleanly, render a small dedicated date-range control
  beside `Filters`). Decided at implementation time based on the `Filters` API.
- **Export:** unchanged — `useExportAuditLogs` (CSV), passes the active filter params (minus
  pagination). Button stays in the toolbar.

## URL state

The `/activity` route declares `validateSearch` for `view: 'timeline' | 'table'` (default
`'timeline'`, invalid → default). The toggle writes via `navigate`; the page reads via
`useSearch`. Filters/search/date stay in component state (not URL) — consistent with the other
list pages this program shipped.

## Testing

Unit tests (Vitest) for the **pure helpers only**:
- `activity-actions`: `getActionMeta` returns the right label/tone for representative actions
  and a safe fallback for an unknown action.
- `activity-diff` `extractFieldChanges`: update-shaped details → field rows; non-diff details
  (snapshot / scalar / empty / null) → empty so the sheet falls back to JsonViewer.
- the day-grouping helper (in `activity-timeline` or extracted): groups/labels Today /
  Yesterday / older correctly given a fixed "now".

No component-render or backend tests (frontend-only redesign; logic lives in pure helpers).
Gate: `pnpm --dir ui build` + `pnpm --dir ui check` (oxlint/oxfmt) + `pnpm --dir ui test`.

## Risks & notes

- **rnui `Filters` date-range support** — if the `Filters` component can't host a range field,
  fall back to a dedicated date-range control beside it (the API just needs `date_from`/`date_to`).
  Doesn't affect the rest of the design.
- **rnui `Timeline`/`JsonViewer` availability** — confirm the exact rnui exports during
  implementation; the dashboard already renders a timeline (`components/dashboard/activity-timeline.tsx`)
  as a reference. If a primitive is missing, build the small piece locally (consistent with the
  program's "capture gaps, wrap locally" stance).
- **Day grouping across timezones** — group by the browser's local day; keep the helper pure
  and inject "now" for testability.
