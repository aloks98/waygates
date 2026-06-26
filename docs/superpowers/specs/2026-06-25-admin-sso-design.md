# Admin SSO (OIDC) — Design

**Status:** Approved (brainstorm) — ready for implementation plan
**Date:** 2026-06-25
**Part:** 2 of 2 (Part 1 = User Management, merged in PR #40)

## Overview

Add Single Sign-On for the Waygates **admin panel** via a generic OpenID Connect
(OIDC) identity provider. An admin clicks "Sign in with SSO" on the login page,
authenticates at the corporate IdP (Keycloak, Authentik, Okta, Azure AD, …), and
is returned to Waygates authenticated as a real Waygates user — receiving the
**same goauth JWT access/refresh pair** that password login issues. Everything
downstream (the Zustand auth store, `/auth/me`, RBAC middleware, route gating) is
unchanged; SSO is simply another way to obtain those tokens.

This authenticates the pre-provisioned accounts from Part 1. By default a user
must already exist (matched by verified email); an optional, off-by-default
toggle enables just-in-time provisioning.

## Key decisions (from brainstorm)

1. **Provisioning — hybrid.** Match an existing, active user by **verified
   email** and issue a JWT. If no match: refuse login *unless* an
   `sso.auto_provision` setting (default **off**) is on, in which case create the
   user with a configurable default role.
2. **Provider — generic OIDC.** A single corporate IdP configured by issuer URL +
   client id/secret. Uses `coreos/go-oidc` for discovery and ID-token
   verification, layered on the existing `golang.org/x/oauth2`. (The 4 social
   providers used by the ACL OAuth flow are *not* reused here.)
3. **Config — runtime, in the Settings UI.** All SSO config lives in the existing
   `settings` table and is edited in a new Settings → SSO panel; the client
   secret is a sensitive key (never returned by `GET /settings`). No redeploy to
   change. Matches the open-registration / metrics-publish / branding patterns.

### Reuse vs. avoid

- **Reuse:** goauth token generation (`GenerateTokenPair`, identical to password
  login), the `users` table + `GetByEmail`, RBAC roles (`AssignRole`), the
  settings store + sensitive-key exclusion, `golang.org/x/oauth2`, and the ACL
  OAuth flow's **state/PKCE cookie helpers** (mechanism only).
- **Do NOT reuse:** the ACL `oauth_handler` session creation — it makes an
  email-only ACL session with no Waygates user. Admin SSO is a distinct identity
  model (real user + JWT) and gets its own handler/service.

### New dependency

- `github.com/coreos/go-oidc/v3` — OIDC discovery (`/.well-known/openid-configuration`),
  JWKS fetch, and ID-token signature/issuer/audience/expiry verification.

## Architecture

A self-contained module:

- `backend/internal/service/sso_service.go` — `SSOService`: holds a lazily-built,
  cached OIDC provider + `oauth2.Config` derived from settings; performs the
  match/provision logic; mints the JWT pair; owns the one-time-code store.
- `backend/internal/api/handlers/sso_handler.go` — HTTP endpoints (status, lookup,
  login, callback, exchange, test).
- Frontend: a login-page button, a `/auth/sso/callback` route, and a
  Settings → SSO panel.

ID-token verification sits behind a small interface (`idTokenVerifier`) so the
service/handler is unit-testable with injected claims, without standing up a live
JWKS server.

## Settings (runtime, `settings` table, `sso.` namespace)

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `sso.enabled` | bool string | `false` | master on/off |
| `sso.oidc_issuer` | string | `""` | e.g. `https://idp.example.com/realms/main` |
| `sso.oidc_client_id` | string | `""` | |
| `sso.oidc_client_secret` | string | `""` | **sensitive** — excluded from `GET /settings`; blank write keeps existing value |
| `sso.auto_provision` | bool string | `false` | JIT-create on no match |
| `sso.default_role` | string | `viewer` | role for auto-provisioned users (`viewer`/`operator`/`admin`) |
| `sso.button_label` | string | `Sign in with SSO` | login-page button text |
| `sso.base_url` | string | `""` | optional redirect-URI base override; blank = derive from request |

`sso.oidc_client_secret` is appended to the existing sensitive-keys list in the
settings repository so it is never returned by `GET /settings` / `GET
/settings/{key}`.

The `SSOService` builds and caches the `*oidc.Provider` and `oauth2.Config` from
these values. Saving any `sso.*` key invalidates the cache so the next login
re-runs discovery. Issuer-unreachable / discovery failure is handled gracefully:
login fails with a logged, friendly error rather than panicking.

## Data model — no migration

The existing `users` table is reused as-is.

- An **auto-provisioned** SSO user is created with `password_hash = ""`. bcrypt
  verification of an empty hash always fails, so password login is impossible for
  that account; it can only sign in via SSO. An admin can later use the existing
  "reset password" flow to grant a password.
- Matching is by **verified email** (`GetByEmail`), which is already the account
  key established in Part 1. No `sub`/issuer columns are stored. *(Stricter
  `sub`-based linking is deliberately out of scope; can be added later via a
  small migration if email reassignment becomes a concern.)*

## Endpoints

All public (no JWT) except as noted. Registered alongside the existing
`/api/auth/*` public routes.

- `GET /api/auth/sso/status` → `{ "enabled": bool, "label": string }`. The login
  page reads this. Never exposes issuer/secret.
- `POST /api/auth/sso/lookup` → body `{ "email": string }` → `{ "method":
  "sso" | "password" }`. Powers the two-step login (below). Returns `sso` only
  when `sso.enabled` **and** a user with that email exists **and** that user has
  no usable password (empty `password_hash`); otherwise `password` (has-password
  account, non-email input, **or no such account**). Rate-limited per IP like
  login/register.
- `GET /api/auth/sso/login` → if `sso.enabled`: generate `state` + PKCE verifier,
  store both in HttpOnly short-TTL cookies (reusing the ACL flow's cookie
  helpers), redirect to the IdP authorize URL (`scope=openid email profile`),
  forwarding an optional `?login_hint=<email>` query param as the OIDC
  `login_hint` so the IdP pre-fills the address. If disabled: `404`.
- `GET /api/auth/sso/callback` → validate `state`/PKCE, exchange the code, verify
  the ID token, run match/provision (below), mint the JWT pair, store it under a
  one-time code, and redirect the browser to the SPA route
  `/auth/sso/callback?code=…`. On any failure: redirect to
  `/login?sso_error=<code>` (see Error handling).
- `POST /api/auth/sso/exchange` → body `{ "code": string }`; returns the JWT pair
  (`{ access_token, refresh_token }`). Single-use, ~60 s TTL.
- `POST /api/auth/sso/test` (**protected**, `settings:write`) → attempts OIDC
  discovery against the currently-saved (or supplied) issuer and reports
  success/error. Powers the Settings panel "Test connection" button.

### Token delivery — one-time code

The callback is a browser GET, but the SPA stores **bearer** tokens in
`localStorage` (Zustand), so the JWT pair must reach the SPA without sitting in
the URL/history. The callback stores the freshly minted pair in an in-memory,
single-use, ~60 s-TTL map keyed by an opaque random code, and redirects to
`/auth/sso/callback?code=…`. The SPA POSTs the code to `/exchange` and stores the
returned tokens. Tokens never appear in the URL fragment or query. Waygates runs
as a single backend instance, so an in-memory store is sufficient (consistent
with the existing `firstUserRegistrationMu` single-instance assumption).

## Match / provision rules (callback core)

From the verified ID token, read `email`, `email_verified`, `sub`, `name`:

1. Require `email` present and `email_verified == true`; else reject
   (`email_unverified`).
2. `GetByEmail(email)`:
   - **Found + active** → update `last_login_at`, mint JWT. `must_change_password`
     is ignored for SSO (no password is involved).
   - **Found + inactive** → reject (`disabled`).
   - **Not found + `sso.auto_provision` on** → create
     `{ name, email, username = unique(email-local-part), active = true,
     password_hash = "", must_change_password = false }`, `AssignRole(default_role)`,
     mint JWT.
   - **Not found + `auto_provision` off** → reject (`no_account`).
3. Audit-log the outcome (success and each failure reason) via a new
   `auth.sso_login` audit action, registered in `GetAuditEventGroups` (auth group).

Username uniqueness: derive from the email local-part; on collision append a
numeric suffix until unique.

## Redirect URI

The OIDC `redirect_uri` must exactly match the IdP registration. It resolves to
`{origin}/api/auth/sso/callback`, where `origin` is derived from the request
(scheme + host) honoring the trusted-proxy `X-Forwarded-*` headers the app
already applies for Caddy. The optional `sso.base_url` setting overrides the
origin for proxies where derivation is wrong. The Settings panel displays the
final resolved redirect URI (with a copy button) so the admin registers the exact
value.

## Frontend

- **Login page (`routes/login.tsx`)** — fetch `GET /api/auth/sso/status` via the
  public client (like `registration-status`) and read a `sso_error` query param
  to show a friendly alert. The form is **identifier-first (two-step) when SSO is
  enabled**, and the classic single-step form when SSO is disabled.

  **Two-step flow (SSO enabled):**
  - **Step 1** — an identifier field + "Continue", plus a separate button
    labelled `sso.button_label` ("Login with SSO").
  - On **Continue**: if the input is an email, POST it to
    `/api/auth/sso/lookup`. `method === "sso"` → full navigation to
    `/api/auth/sso/login?login_hint=<email>`. Otherwise (or non-email input) →
    reveal the password field (step 2), keeping the entered identifier, and submit
    to the existing password login.
  - On **"Login with SSO"** → full navigation to `/api/auth/sso/login` (no hint).

  **Single-step (SSO disabled):** today's identifier + password form, no SSO
  button, no lookup call.
- **SSO callback route (`/auth/sso/callback`, unauthenticated, new in
  `lib/router.tsx`)** — read `?code`, POST to `/api/auth/sso/exchange`, on success
  `setTokens(...)` then navigate `/`; on missing/expired code redirect to
  `/login?sso_error=sso_failed`. Shows a "Signing you in…" spinner.
- **Settings → SSO panel (`components/settings/sso-settings.tsx`)** — gated on
  `canWriteSettings`, added to `settings-nav`. RHF form: enabled (Switch), issuer
  URL, client ID, client secret (password field, **write-only**: shows
  "configured" when set, blank keeps stored value), auto-provision (Switch),
  default-role (Select), button label; a read-only **Redirect URI** field with a
  copy button; a **Test connection** button. Save buttons disabled when
  `!canWriteSettings`. A `useSsoSettings` hook reads current non-secret values;
  saves go through `PUT /api/settings/{key}`.

## Error handling

Every callback failure redirects the browser to `/login?sso_error=<code>`:

| code | meaning | login-page message (example) |
|------|---------|------------------------------|
| `no_account` | no Waygates user, auto-provision off | "No Waygates account for this identity — contact an administrator." |
| `disabled` | matched user is inactive | "Your account is disabled. Contact your administrator." |
| `email_unverified` | IdP did not assert a verified email | "Your identity provider did not provide a verified email." |
| `state_mismatch` | state/PKCE/CSRF failure | "Sign-in could not be verified. Please try again." |
| `sso_disabled` | SSO turned off mid-flow | "SSO is not enabled." |
| `sso_failed` | discovery/exchange/token-verify/other | "Single sign-on failed. Please try again or use a password." |

Detailed causes are logged server-side via zap; the client sees only the friendly
message. `/exchange` returns a generic error for a bad/expired/used code.

## Security considerations

- Client secret stored as a sensitive setting, never serialized to clients.
- ID token fully verified (signature via JWKS, issuer, audience = client id,
  expiry) by go-oidc — not just decoded.
- `state` + PKCE protect against CSRF / code interception.
- JWT pair delivered via single-use, short-TTL one-time code — never in URL.
- `email_verified` required before trusting the email for account matching.
- Inactive accounts rejected (after a match) — same posture as password login.
- Password login always remains available (cannot be disabled), preventing
  lockout if the IdP is misconfigured or down; the bootstrap admin always has a
  password.
- The `sso/lookup` endpoint is an account-routing oracle: a `sso` response
  reveals "this email is a passwordless SSO account." It is rate-limited per IP
  (same limiter as login/register) and never distinguishes a password account
  from a non-existent one (both return `password`).

## Testing

- **Backend unit** — match/provision rules, table-driven (found-active /
  found-inactive / not-found+auto-provision / not-found+no-provision /
  email-unverified / missing-email) against a mock user repo + role assigner with
  ID-token claims injected (verification behind an interface). Settings → OIDC
  config build + cache-invalidation-on-save. One-time-code store: single-use +
  TTL expiry.
- **Backend handler** — status (enabled/disabled), lookup (passwordless account
  → `sso`; has-password account → `password`; unknown email → `password`;
  SSO disabled → `password`), login (disabled → 404, enabled → redirect sets
  cookies, `login_hint` forwarded), callback happy path (verifier mocked)
  asserting a JWT pair is minted and `auth.sso_login` is audited, exchange
  (valid → tokens; reused/expired → error).
- **Frontend** — `sso_error` code → message mapping unit-tested; the step-1
  routing decision (email → lookup → sso/password branch) unit-tested. Gate
  green: `pnpm --dir ui build` + `check` + the existing test suite stays green.
- **Human smoke test (not CI-able)** — real round trip against an actual IdP:
  button → IdP → callback → dashboard; auto-provision on and off; inactive user
  rejected; client secret absent from `GET /settings`; redirect-URI value shown
  in the panel matches the IdP registration.

## Out of scope / future

- Multiple simultaneous IdPs / the 4 social providers for admin login.
- `sub`-based account linking and email-change handling.
- SSO-only mode (disabling password login).
- SCIM / group-to-role mapping from IdP claims.

## Rough task decomposition (for the plan)

1. Settings keys + sensitive-key exclusion + `SSOService` config build/cache.
2. OIDC client (`coreos/go-oidc`) + match/provision service + unit tests.
3. One-time-code store (single-use, TTL) + tests.
4. `sso_handler` + routes (`status`, `login` w/ `login_hint`, `callback`,
   `exchange`, `lookup`, `test`) + `auth.sso_login` audit action.
5. Login page: identifier-first two-step form (`status` fetch, `lookup` routing,
   "Login with SSO" button, `sso_error` alerts; single-step when disabled).
6. SPA `/auth/sso/callback` route + `/exchange` wiring.
7. Settings → SSO panel (form, redirect-URI display, test connection).
