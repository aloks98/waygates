# User Management — Design

**Date:** 2026-06-23
**Status:** Approved — ready for implementation plan
**Context:** Part 1 of a two-part effort (Part 2 = admin SSO). Today the app has no user-management surface — only `user_repository.go` (data layer); users self-register via `POST /api/auth/register` (first user becomes admin, rest operator) and the `/signup` page. The `users:read` / `users:manage` RBAC permissions exist in `rbac.yaml` but are unused. This part adds admin-managed users and makes self-registration an opt-in setting, establishing the pre-provisioned-account model that SSO (Part 2) will authenticate against.

## Goal

Give admins a Settings page to manage user accounts (create, edit, delete, assign roles, reset passwords, activate/deactivate, see last login), replace forced self-signup with a default-off "open registration" toggle, and guarantee a bootstrap admin so a fresh install always has a way in.

## Scope

**In:**
- Backend user-management API (CRUD + role assignment + password reset + activate/deactivate), gated by `users:read` / `users:manage`.
- A **Users** section under **Settings** (`/settings/users`) — list + manage users.
- Password lifecycle: admin-set initial password, admin reset, force-change-on-first-login, user self-change (exists).
- `active` (enable/disable) and `last_login_at` tracking.
- Configurable **open registration** setting (default off); gate the register endpoint + signup page on it.
- **Bootstrap admin** on first run via `DEFAULT_USER_*`.
- Audit logging of all user-management actions.

**Out of scope (Part 2 or later):** admin SSO (Part 2); email/SMTP invites; multiple roles per user or custom per-user permissions; self-service profile editing beyond the existing change-password.

## Locked decisions

- **Pre-provisioned model:** admins create accounts; both password and (later) SSO authenticate existing accounts.
- **Roles:** exactly one per user — `admin` / `operator` / `viewer` (existing RBAC roles, stored in the goauth role store, not on the user row).
- **Initial password:** admin types it on create (with a "must change on first login" checkbox).
- **Open registration:** a DB-backed setting, default **off**. When on, self-registered users get the **viewer** role.
- **Optional features (all in):** admin password reset, activate/deactivate, force-change-on-first-login, last-login tracking.
- **Users page lives under Settings**, and is **admin-only** (`users:read`/`users:manage` granted to admin only; operators/viewers don't see the section).
- **Username is immutable** after creation; `name`, `email`, `role`, and `active` are editable.
- **Guards:** a user cannot delete, deactivate, or demote **themselves** or the **last remaining admin**.

## Data model

Migration `000013` adds to `users`:
- `active BOOLEAN NOT NULL DEFAULT true`
- `must_change_password BOOLEAN NOT NULL DEFAULT false`
- `last_login_at TIMESTAMP NULL`

`created_at`/`updated_at` already exist (GORM). Roles are **not** a column — they live in the goauth role store and are read/written via the auth wrapper.

## Roles & RBAC

- The user service assigns roles via `auth.AssignRole(ctx, userID, role)` and reads a user's current role via the auth wrapper (a single-role lookup; if only `GetUserPermissions` exists, add/expose a `GetUserRole`).
- New routes gated by `users:read` (list/detail) and `users:manage` (mutations). Both granted to **admin only** in `rbac.yaml` (admin already has `*`; do not grant to operator/viewer).
- Frontend hides the Users settings section unless the current user has `users:manage` (via `usePermissions`).

## Backend API

New `users_handler.go` + `user_service.go` over the existing `user_repository` + goauth + audit service. Standard `{success,data,error}` envelope.

| Method · Path | Perm | Notes |
|---|---|---|
| `GET /api/users` | `users:read` | List: id, name, username, email, role, active, last_login_at, created_at |
| `GET /api/users/{id}` | `users:read` | Single user |
| `POST /api/users` | `users:manage` | Create: name, username, email, role, password, must_change_password |
| `PUT /api/users/{id}` | `users:manage` | Edit: name, email, role, active (username NOT editable) |
| `POST /api/users/{id}/password` | `users:manage` | Admin reset password (+ sets must_change_password) |
| `DELETE /api/users/{id}` | `users:manage` | Delete |

**Service-level guards** (return a clear 4xx, not 500): block delete / deactivate / role-change-away-from-admin when the target is the **acting user** or the **last admin** (admin count would drop to 0). Validation: unique username/email, password strength (reuse the existing register/change-password rules), role ∈ {admin,operator,viewer}.

**Audit:** each mutation logs via the existing audit service (`user.create`, `user.update`, `user.role_change`, `user.password_reset`, `user.activate`/`user.deactivate`, `user.delete`) with actor + target + before/after where relevant (never the password).

## Auth-flow changes

- **Login** (`auth_handler.go`): reject if `active == false` (clear "account disabled" error); on success set `last_login_at`; surface `must_change_password` in the login/`/me` response so the UI can force a password change.
- **Change-password:** clears `must_change_password` on success.
- **Register** (`POST /api/auth/register`): gated on the open-registration setting → `403` when off. When on, create the user with the **viewer** role (drop the "first user becomes admin" branch — bootstrap handles the first admin).
- **Bootstrap admin:** on startup, if the users table is empty, create an admin from `DEFAULT_USER_NAME/USERNAME/EMAIL/PASSWORD` and assign the `admin` role. (The config fields exist; this wires the actual provisioning.) If the env is unset and no users exist, log a clear warning.

## Open-registration setting

A DB-backed key-value setting (same pattern as the metrics-publish / not-found settings), e.g. `auth.open_registration` (bool, default `false`):
- Read by the register handler to allow/deny self-signup.
- Admin write via the settings layer (`settings:write`).
- A small **public** read (e.g. `GET /api/auth/registration-status` → `{ open: bool }`) so the login page shows/hides the "Sign up" link and the signup page can refuse when closed.

## Frontend

- **Settings → Users** (`/settings/users`, new `SettingsNav` entry "Users", admin-only): an rnui `DataGrid` (name, username, email, role badge, status badge, last login) with row actions + a toolbar "Add user" button. Dialogs (RHF + Zod, matching existing settings/ACL form patterns):
  - **Add user** — name, username, email, role select, password, "must change on first login" checkbox.
  - **Edit user** — name, email, role, active (username read-only).
  - **Reset password** — new password + must-change.
  - **Delete** — confirm dialog.
  - **Activate/Deactivate** — toggle/action.
  - An **Open registration** switch at the top of the section (persisted via the settings API).
- **Login page:** handle a forced password change (route to change-password before the dashboard when `must_change_password`); show the "account disabled" error; hide the "Sign up" link when registration is closed (via the public registration-status read).
- **Signup page:** stays, but refuses (redirect/disabled message) when registration is closed.
- New `use-users` hook(s) (React Query) for the CRUD, mirroring existing hooks.

## Architecture & files

- **Backend:** `migrations/000013_*` (user columns); `models/user.go` (+ new fields); `repository/user_repository.go` (list/update/active/last-login/password methods as needed) + interface; new `service/user_service.go` (CRUD + guards + role via goauth + audit); new `api/handlers/users_handler.go`; `routes.go` (wire `/api/users` + the public registration-status route + bootstrap call); `auth_handler.go` (login active/last-login/must-change, register gating); a startup bootstrap-admin step; settings layer for `auth.open_registration`; `rbac.yaml` (grant `users:*` to admin). Tests alongside.
- **Frontend:** `types/user.ts`; `hooks/use-users.ts`; `components/settings/user-management.tsx` (+ dialogs); `SettingsNav` + settings route registration; `login.tsx` / `signup.tsx` adjustments; an open-registration control.

## Testing

- **Backend (testify):** service guards (last-admin, self-delete/deactivate/demote), create/edit/role-change/reset/active, register gating (open vs closed), login active/must-change/last-login. Pure logic + repo (testcontainers pattern where the project uses it).
- **Frontend:** gate (`pnpm --dir ui build` + `check` + `test`), existing tests green; any extracted pure helper unit-tested.

## Risks & notes

- **Lockout:** the guards + bootstrap admin must be correct — never allow the last admin to be removed/demoted/deactivated, and always provision a bootstrap admin when the table is empty.
- **Role storage split:** roles live in goauth's store, not the user row — listing users requires a role lookup per user; ensure an efficient path (batch or per-row) and a single source of truth.
- **Password never logged/returned;** reuse existing hashing + strength rules.
- **Part 2 (SSO) depends on this:** SSO will authenticate against these pre-provisioned users + the open-registration/role model defined here.
