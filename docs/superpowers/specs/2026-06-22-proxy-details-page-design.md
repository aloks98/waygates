# Proxy Details (Overview) Page — Design

**Date:** 2026-06-22
**Status:** Approved — ready for implementation plan
**Context:** First post-program backlog feature (the rnui redesign M0–M6 is fully merged). Flagged during M4 (Activity): the dashboard/activity timeline and the ACL group-usage tab deep-link to a proxy expecting an overview, but the route is the edit form.

## Goal

Add a **read-only overview page** for each proxy (HTTP and L4/TCP-UDP) at its detail URL, and move the edit form to a child `/edit` route — so deep-links land on an overview and the app follows a standard **list → detail → edit** flow. Frontend-only; reuses existing data (no new backend endpoint).

## Context (current state)

- **HTTP:** `/dashboard/proxies/$proxyId` renders the **edit form** (`ProxyDetailPage` in `ui/src/routes/_dashboard/proxies/$proxyId.tsx`; one of `ReverseProxyForm`/`RedirectForm`/`StaticForm` by `proxy.type`). Loads via `useProxy(id)` (`GET /api/proxies/{id}` → full `ProxyConfig`); ACL via `useProxyACL(id)` (`GET /api/proxies/{id}/acl` → `ProxyACLAssignment[]`).
- **L4:** `/dashboard/proxies/tcp-udp/$l4ProxyId` renders the **edit form** (`L4ProxyDetailPage`; `L4ProxyForm` mode=edit). Loads via `useL4Proxy(id)`. L4 has **no ACL**.
- Routes are **code-based** in `ui/src/lib/router.tsx` (flat children of `dashboardRoute`, lazy-loaded by named export).
- **Deep-link sites** that target `/dashboard/proxies/$proxyId` today (all land on the edit form): the proxies list row-click/edit action (`routes/_dashboard/proxies/index.tsx`), create-success (`proxies/new.tsx`), and the ACL group-usage tab (`components/acl/group-usage-tab.tsx`).
- All overview data is available client-side. `GET /api/proxies/{id}` returns the full config (but NOT `acl_group_count`/`acl_group_names` — those are list-only); ACL detail comes from `useProxyACL`. **No new backend endpoint is required.**
- rnui detail-layout references: `routes/_dashboard/acl/$groupId.tsx` (cards/sections) and `components/activity/activity-detail-sheet.tsx` (read-only `MetaRow` label/value pattern).

## Scope

**In:**
- HTTP proxy **overview** page at `/dashboard/proxies/$proxyId`; move the existing edit form to `/dashboard/proxies/$proxyId/edit`.
- L4 proxy **overview** at `/dashboard/proxies/tcp-udp/$l4ProxyId`; move the L4 edit form to `/dashboard/proxies/tcp-udp/$l4ProxyId/edit`.
- Repoint navigation: list row-click → overview; list row **Edit** action → `…/edit`; create-success → overview; overview **Edit** button → `…/edit`; edit-page **back** → overview. (ACL group-usage already targets `$proxyId`, now the overview — no change needed.)

**Out of scope:**
- Any backend/endpoint/type change.
- **B1 (generated config preview)** and **B2 (health)** — NOT built; the overview layout reserves section slots so they can be added later.
- ACL for L4 (L4 has none).
- Editing from the overview (it is read-only; all mutation stays in the edit form, except the Duplicate/Delete row-style actions below).

## Routing

Code-based, flat sibling routes under `dashboardRoute` (no shared layout — overview and edit are separate full pages):

| Route | Path | Component (named export) |
|---|---|---|
| HTTP overview | `/proxies/$proxyId` | `ProxyOverviewPage` (new) |
| HTTP edit | `/proxies/$proxyId/edit` | `ProxyEditPage` (the current `$proxyId.tsx` content, moved/renamed) |
| L4 overview | `/proxies/tcp-udp/$l4ProxyId` | `L4ProxyOverviewPage` (new) |
| L4 edit | `/proxies/tcp-udp/$l4ProxyId/edit` | `L4ProxyEditPage` (current L4 detail content, moved/renamed) |

The edit-page components are the existing edit pages with their internal "back" navigation pointed at the overview. The route-param parsing (`useParams`) is unchanged.

## HTTP overview content

Single scrollable page of rnui `Card`s (read-only; lighter than the tabbed ACL group page).

- **Header:** back button, type icon, proxy name, status badges (Active/Inactive; SSL enabled / forced), hostname subtitle. Actions: **Edit** (primary, → `…/edit`) + a kebab `DropdownMenu` with **Duplicate** and **Delete** (mirroring the list row actions; Delete uses the existing confirm dialog + `useProxies` delete; Duplicate uses the existing `?duplicate=<id>` create flow). The edit page keeps its own delete.
- **Routing / Config card** (by `proxy.type`):
  - `reverse_proxy`: upstreams (host:port:scheme list), load-balancing strategy, `block_exploits`, `tls_insecure_skip_verify`, custom headers (request/response).
  - `redirect`: target, status code, preserve path / preserve query.
  - `static`: root path, index file, browse, template rendering, try_files.
- **HTTPS / TLS card:** `ssl_enabled`, `ssl_forced`.
- **Access — "What protects this" card** (HTTP only): from `useProxyACL(id)` — list each assignment's ACL group name (link to `/dashboard/access/$groupId`), path pattern, priority, enabled state; empty state = "Unprotected" with a CTA linking to Access. Reuses the `ProxyAclCell` semantics conceptually but rendered as a full card.
- **Details card:** description, id, created_at, updated_at, created_by.
- **Reserved slots (not built):** the card layout anticipates a future **Generated Config** card (B1) and **Health** card (B2); they are not implemented now.

## L4 overview content

Simpler (no ACL): header (protocol icon, name, protocol + Active/Inactive badges, listen-port subtitle) → **Config card** (listen port, protocol, routes → upstreams summary, TLS terminate/passthrough) → **Details card** (description, id, timestamps). Actions: **Edit** + kebab (Duplicate, Delete) mirroring the L4 list row actions.

## Components & data

- `components/proxy/overview/` — `proxy-overview.tsx` (HTTP overview body), per-type config card(s), the Access card, and a shared read-only `DetailRow` (label/value grid; extract from or mirror the Activity sheet's `MetaRow`).
- `components/l4-proxy/overview/` — `l4-proxy-overview.tsx` + L4 config card, reusing `DetailRow`.
- The new route page files (`ProxyOverviewPage`, `L4ProxyOverviewPage`) own data fetching (`useProxy`/`useProxyACL`, `useL4Proxy`), loading skeleton, and not-found — reusing the existing skeleton/not-found patterns from the current detail pages.
- No change to hooks, types, or the backend.

## Decisions

- **Layout:** single scrollable page of cards (not tabs). *(User-approved.)*
- **Actions:** overview owns **Edit (primary) + Duplicate + Delete** (kebab), mirroring the list row's `DropdownMenu`; the edit page keeps its existing delete. *(User-approved.)*
- **No new backend endpoint** — existing `useProxy`/`useProxyACL`/`useL4Proxy` suffice.
- **B1/B2 are reserved slots**, not built.
- **HTTP + L4 parity** — both get overview→edit so navigation is consistent across the unified Proxies area. *(User-approved.)*

## Architecture & files

- New: `routes/_dashboard/proxies/$proxyId` overview page + `…/$proxyId/edit` (moved edit); same for L4. (Exact filenames finalized in the plan — the router is code-based, so route files just export named components.)
- New: `components/proxy/overview/*`, `components/l4-proxy/overview/*`, a shared `DetailRow`.
- Modify: `ui/src/lib/router.tsx` (add the two `/edit` routes; repoint the detail routes to the overview components), `routes/_dashboard/proxies/index.tsx` + the L4 list (row-click → overview, Edit action → `…/edit`), `proxies/new.tsx` + L4 `new.tsx` (success → overview), the moved edit pages' back-nav.

## Decomposition (for the plan)

Subagent-driven, each independently gate-able:
1. **HTTP routing split** — move edit to `…/$proxyId/edit`, add a minimal `ProxyOverviewPage` (header + Edit button, data load + skeleton + not-found), wire `router.tsx` + repoint the HTTP list/new nav. Deliverable: HTTP overview reachable, edit at `/edit`, navigation correct.
2. **HTTP overview content** — the config-by-type card(s), HTTPS/TLS card, Details card, + the kebab Duplicate/Delete actions + `DetailRow`.
3. **HTTP Access card** — "what protects this" via `useProxyACL` (+ unprotected empty state, links to groups).
4. **L4 routing split + overview** — mirror (1)+(2) for L4 (simpler; no ACL): move edit to `…/$l4ProxyId/edit`, add `L4ProxyOverviewPage` with config + details cards + actions, wire router + L4 list/new nav.

## Testing

Frontend-only; verification is the gate (`pnpm --dir ui build` + `check` + `test`, all green; the existing **57 tests** stay green). Add unit tests only for any pure formatting helper extracted (e.g. an upstream/`DetailRow` formatter). Forms convention is unaffected (overview is read-only); any inline action reuses existing mutation hooks.

## Risks & notes

- **Route move correctness:** moving the edit form to `…/edit` must keep `useParams` param names (`proxyId`/`l4ProxyId`) and update every nav call site; verify no deep-link still assumes `$proxyId` is the edit form (the list **Edit** action must now go to `…/edit`). A missed call site = landing on the wrong page.
- **Duplicate/Delete on the overview** must reuse the exact existing flows (`?duplicate=<id>` seed; delete confirm + `useProxies`/`useL4Proxies` delete + post-delete navigate to the list) — no new mutation logic.
- **Reserved B1/B2 slots** are layout-only; do not stub fake data.
- Read-only parity: the overview must render the same config the edit form shows, just non-editable — no field omitted that the user would expect to see.
