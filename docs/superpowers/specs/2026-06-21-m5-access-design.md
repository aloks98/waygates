# M5 — Access (ACL) Redesign — Design (M5a)

**Date:** 2026-06-21
**Program:** rnui UI redesign (see `2026-06-19-rnui-redesign-program-design.md` §11 M5)
**Status:** Approved — ready for implementation plan

## Goal

Redesign the admin **Access** screens (the ACL group management UI) to match the rest of the
app: clear information architecture, plain-language naming (drop "ACL"/"bypass"/"combination
mode" jargon), a scannable group list, and a group-detail page with one clean tab per auth
method. M5 is split into two PRs; **this spec is M5a**.

## Decomposition (M5 → two PRs)

- **M5a (this spec):** Access IA & structure — group-detail tab restructure, de-jargon naming,
  group-list redesign, and three small UX fixes. Admin forms keep their current TanStack Form
  internals (mechanically split/relocated where a tab splits).
- **M5b (follow-up, separate spec):** migrate every Access admin form from `@tanstack/react-form`
  → React Hook Form + Zod (matching the proxy/L4 form patterns), removing the last TanStack Form
  usage in the app.

Out of the whole M5 effort entirely: the public end-user pages (`routes/auth/acl-login.tsx`,
`acl-forbidden.tsx`) and the branding settings panel — different audience, separate effort.

## Context — current state

- **Routes:** `/dashboard/access` (`acl/index.tsx`, `ACLGroupsPage`) is a clean DataGrid list
  (search, create modal, delete). `/dashboard/access/$groupId` (`acl/$groupId.tsx`,
  `ACLGroupDetailPage`) is a header + `Tabs` shell with an inline Overview tab + 4 imported tabs.
- **Current tabs (5):** Overview, IP Rules, Basic Auth, Waygates Auth, Usage. The
  `ExternalProvidersTab` component exists and is complete but is **not mounted** (ghost — the
  Overview still counts `external_providers`). `WaygatesAuthTab` crams two concerns into one:
  OAuth provider selection (+ a `ProviderRestrictionModal`) AND Waygates account auth (roles,
  email patterns, `session_ttl` in raw seconds).
- **Forms:** all ACL admin forms use `@tanstack/react-form` + Zod (group modal, ip-rule,
  basic-auth, external-provider, waygates-auth). `acl-selector.tsx` and branding use plain
  `useState`.
- **Hooks/types:** `hooks/use-acl.ts` (28 hooks, all endpoints listed there), `types/acl.ts`.
  Key types: `ACLGroup` (relations: `ip_rules[]`, `basic_auth_users[]`, `external_providers[]`,
  `waygates_auth`, `oauth_provider_restrictions[]`), `ACLWaygatesAuth` (carries BOTH
  `allowed_providers[]` for OAuth selection AND account fields `enabled`/`allowed_roles[]`/
  `allowed_email_patterns[]`/`session_ttl`/`require_2fa`), `ACLIPRule` (`rule_type`
  allow|deny|bypass, `cidr`, `priority`), `ACLExternalProvider`, `ACLOAuthProviderRestriction`.
- **Backend:** full `/api/acl/*` admin API (acl:read/create/update/delete). The single
  `PUT /api/acl/groups/{id}/waygates-auth` saves the whole `ACLWaygatesAuth` record (both the
  OAuth selection and the account fields).

## Locked decisions (from brainstorming)

1. **Scope:** admin Access management only; M5 split into M5a (this) + M5b (forms→RHF).
2. **Group-detail IA:** keep a tab-per-method structure (not a consolidated methods page),
   but split and relabel them cleanly.
3. **Naming:** ACL → Access everywhere; de-jargon the method/mode labels.
4. **Include in M5a:** session_ttl duration picker, group-list auth-method pills, Usage→proxy
   link fix. **Defer:** basic-auth password reset.
5. **Backend addition is acceptable** for the list pills (per-method indicators on the list query).

## Scope

**In (M5a):** group-detail tab restructure (7 tabs; split WaygatesAuth, re-mount Forward Auth),
de-jargon naming across the Access screens, group-list method pills + Usage link fix, session_ttl
duration picker, a small backend addition for list per-method flags, and a shared auth-method
metadata module.

**Out (M5a):** forms→RHF (M5b); basic-auth password reset; end-user login/forbidden pages;
branding settings; any change to the ACL evaluation semantics or the auth flow itself.

## Group-detail IA

New tab set on `acl/$groupId.tsx`:

**Overview · IP Rules · Basic Auth · OAuth / SSO · Waygates Account · Forward Auth · Usage**

- **Split `WaygatesAuthTab` → two tabs:**
  - **OAuth / SSO** (`oauth-sso-tab.tsx`): the provider checkbox grid (Google/GitHub/… filtered
    to server-available) + the per-provider `ProviderRestrictionModal` (allowed emails/domains).
    Edits the `allowed_providers[]` slice of the waygates-auth record + the separate
    `oauth-restrictions/{provider}` endpoint.
  - **Waygates Account** (`waygates-account-tab.tsx`): the enable switch + allowed roles +
    allowed email patterns + the new **session duration picker** + `require_2fa`. Edits the
    account fields of the waygates-auth record.
  - **Shared-record coordination:** both tabs read `useWaygatesAuth(groupId)` and save via
    `useConfigureWaygatesAuth` (the single `PUT …/waygates-auth`). Each tab sends the full
    record with only its slice modified (load current → merge its fields → save), so neither
    clobbers the other's data. Document this in both components.
- **Re-mount `ExternalProvidersTab` as "Forward Auth"** (Authelia/Authentik/custom forward-auth):
  add it to the tab list; relabel the component's user-facing copy "External Providers" →
  "Forward Auth". Remove the dead Overview "External Providers" tile→nowhere (make the Overview
  Configuration Summary reflect the 5 real methods, each linking to its tab).
- Forms inside all tabs keep TanStack Form internals in M5a (M5b migrates them).

## Naming / de-jargon

Apply consistently across the Access screens (titles, tab labels, badges, toasts, helper text):

| Today | M5a |
|---|---|
| "ACL" / "ACL group" (titles, toasts) | "Access" / "access group" |
| Page title "Access Control Groups" | "Access Groups" |
| IP rule type **Bypass** | **Trusted — skip auth** (badge + select label; keep API value `bypass`) |
| IP rule type Allow / Deny | **Allow** / **Block** |
| **Combination Mode** (field) | **How methods combine** |
| mode `any` / `all` / `ip_bypass` | **Any match** / **All required** / **Trusted IPs first** (keep API values) |
| Tab **Waygates Auth** | split → **OAuth / SSO** + **Waygates Account** |
| **External Providers** | **Forward Auth** |

API field/enum values are unchanged — only display labels change. A shared label map
(`access-labels.ts`, the `L4_MATCHER_CONFIG` pattern) is the single source for rule-type and
mode labels, used by the list, the badges, and the forms.

## Group list redesign

- **Auth-method pills:** replace the single check/X "Auth" column with small pills showing which
  methods a group uses — **IP · Basic · OAuth · Account · Forward** (only the configured ones).
  Keep the existing Mode badge and IP-rules count.
- **Backend addition:** the group-list response items must carry per-method indicators. Add
  computed `gorm:"-"` boolean flags to the `ACLGroup` model — `has_ip_rules`, `has_basic_auth`,
  `has_oauth` (allowed_providers non-empty), `has_waygates_account` (waygates_auth.enabled),
  `has_forward_auth` (external_providers non-empty) — populated in the repository `List` query
  (one batched pass over the page's group ids, mirroring how M2b added ACL counts to the proxy
  list — no migration). The single-group GET already returns full relations, so detail views are
  unaffected.
- **Usage tab fix:** "View proxy" links to the proxy's own page
  (`/dashboard/proxies/$proxyId`) using the row's `proxy.id`, not the proxy list.

## Small UX — session duration picker

- Replace the raw `session_ttl` seconds `<input type="number">` in the Waygates Account tab with
  a **number + unit** control (minutes / hours / days). A pure, unit-tested converter pair —
  `secondsToDuration(s) → {value, unit}` and `durationToSeconds(value, unit) → number` — keeps
  the backend contract (`session_ttl` seconds, 60–604800 range) intact and the conversion
  testable. Pick the largest whole unit when displaying an existing value.

## Architecture & files

- `ui/src/routes/_dashboard/acl/$groupId.tsx` — new 7-tab list + labels; Overview summary fixed.
- `ui/src/components/acl/`:
  - **create** `oauth-sso-tab.tsx`, `waygates-account-tab.tsx` (split from `waygates-auth-tab.tsx`,
    which is removed); both keep TanStack Form.
  - **create** `access-labels.ts` (rule-type + mode label maps; auth-method pill metadata) +
    `access-labels.test.ts`.
  - **create** `session-duration.ts` (converter pair) + `session-duration.test.ts`.
  - `external-providers-tab.tsx` — relabel to "Forward Auth", mount in the route.
  - `ip-rules-tab.tsx` — use `access-labels` for rule-type display ("Trusted — skip auth", "Block").
  - `acl-group-form-modal.tsx` — relabel "Combination Mode" → "How methods combine" via `access-labels`.
  - `group-usage-tab.tsx` — fix the View-proxy link.
  - `index.ts` — update barrel (drop `WaygatesAuthTab`, add the two new tabs).
- `ui/src/routes/_dashboard/acl/index.tsx` — auth-method pills column.
- `ui/src/types/acl.ts` — add the optional per-method flag fields to `ACLGroup`.
- **Backend:** `models/acl.go` (`gorm:"-"` flag fields on the group model), the ACL repository
  `List`/group-list query (populate the flags), handler/service unchanged otherwise. Tests for
  the list-flag population. Docs: update `docs/` ACL API doc for the new list fields.
- Naming: sweep `components/acl/*` + the two routes for "ACL"→"Access" user-facing strings/toasts.

## Testing

- **Unit (Vitest):** `session-duration` converter (round-trips, largest-unit display, clamp
  bounds); `access-labels` (rule-type/mode/method labels + a safe fallback); auth-method pill
  derivation (group flags → pill list).
- **Backend:** repository test that the group-list query populates the per-method flags
  correctly (mixed groups), mirroring the M2b ACL-count test. `make backend-test` (+ testcontainers
  where the existing ACL repo tests run).
- Gate: `pnpm --dir ui build` + `pnpm --dir ui check` + `pnpm --dir ui test`; backend
  `make format-backend` + `make lint-backend` + `make backend-test`.

## Risks & notes

- **Two tabs, one waygates-auth record:** the OAuth/SSO and Waygates Account tabs both persist
  to the single `PUT …/waygates-auth`. Each must load-merge-save to avoid clobbering the other's
  fields. Mitigation: both read `useWaygatesAuth` and spread the current record before applying
  their slice; only one tab is active at a time and the query refetches on save.
- **List-flags backend pass:** keep it a single batched query over the page's group ids (not
  N+1) — follow the GetByIDs/ACL-count precedent from the proxy list work.
- **Tab count (7):** the user accepted the tab-per-method structure; the win is each tab now does
  one thing (vs the old Waygates tab doing three) plus no ghost tab.
- M5b will rewrite these forms to RHF; keep the M5a tab split clean so the migration is localized.
