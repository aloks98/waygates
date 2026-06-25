# User Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Admin-managed user accounts (Settings → Users) with roles, password lifecycle, activate/deactivate, last-login, and a default-off open-registration toggle replacing forced self-signup.

**Architecture:** New `user_service` (CRUD + guards + roles via goauth + audit) over the existing `user_repository`, exposed by a new `users_handler` under `/api/users` (RBAC `users:read`/`users:manage`). Auth flow gains inactive-rejection, last-login, must-change-password, and registration gating on a DB setting. Frontend adds a Settings → Users section (rnui DataGrid + RHF dialogs) plus login/signup adjustments.

**Tech Stack:** Go 1.25 (chi, GORM, goauth, zap, testify), React 19 + `@e412/rnui-react` + RHF/Zod + TanStack Query, oxlint/oxfmt.

## Global Constraints

- **Part 1 of 2** (SSO is Part 2). Branch `feat/user-management` (off master `d27fb87`; spec committed).
- **Roles:** one per user — `admin`/`operator`/`viewer` — stored in goauth's `goauth_user_permissions` table, read via `GetUserPermissions(...).RoleLabel`, written via `AssignRole(ctx, userID, role)`. There is NO list/single-role getter — read per-user.
- **RBAC:** `users:read`/`users:manage` already exist in `rbac.yaml`; `admin` has `*` (covers them). Do NOT grant them to operator/viewer. **No rbac.yaml change is required.**
- **Guards (service-level, return 4xx not 500):** a user cannot delete, deactivate, or demote-from-admin **themselves** or the **last remaining admin**.
- **Username immutable** after create; admin types the initial password; bcrypt cost = `cfg.Security.BcryptCost`.
- **Bootstrap stays:** `createDefaultUserIfNeeded` (main.go) already provisions a `DEFAULT_USER_*` admin when the table is empty — keep it. Additionally, the first-ever UI signup (empty table) bootstraps as admin even when registration is closed (prevents lockout).
- **Per-task gate** — backend: `cd backend && gofmt -l .` (empty) + `go build ./...` + `go test ./... -short`; golangci-lint is CI-only (handle every returned error incl. `_ = x.Close()`, no gocritic `appendAssign`, US spelling). Frontend: `pnpm --dir ui build` + `pnpm --dir ui check` + `pnpm --dir ui test` (existing tests green; NO tsc gate → grep-verify token/route strings).
- Commit trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; stage only each task's files (never `git add -A`).

## Shared shapes (used across tasks)

User DTO (API responses): `{ id, name, username, email, role, active, must_change_password, last_login_at, created_at }`.
- Create body: `{ name, username, email, role, password, must_change_password }`
- Update body: `{ name, email, role, active }`
- Reset-password body: `{ password, must_change_password }`
- `LoginResponse` gains `must_change_password bool`.
- Registration status: `{ open: bool }`.

---

## Task 1: Migration + user model fields + repo methods

**Files:** Create `backend/migrations/000013_add_user_management_fields.up.sql` + `.down.sql`; Modify `backend/internal/models/user.go`; `backend/internal/repository/user_repository.go` + `backend/internal/repository/interfaces.go`; Test `backend/internal/repository/user_repository_test.go`.

- [ ] **Step 1: Migration.** `000013_add_user_management_fields.up.sql`:
```sql
ALTER TABLE users ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMP NULL;
```
`.down.sql`:
```sql
ALTER TABLE users DROP COLUMN last_login_at;
ALTER TABLE users DROP COLUMN must_change_password;
ALTER TABLE users DROP COLUMN active;
```

- [ ] **Step 2: Model.** In `models/user.go` add to the `User` struct (after `Email`):
```go
	Active             bool       `json:"active" gorm:"not null;default:true"`
	MustChangePassword bool       `json:"must_change_password" gorm:"not null;default:false"`
	LastLoginAt        *time.Time `json:"last_login_at"`
```

- [ ] **Step 3: Repo interface.** In `interfaces.go` add to `UserRepositoryInterface`:
```go
	List() ([]models.User, error)
	Update(user *models.User) error
	UpdateLastLogin(id int, t time.Time) error
```

- [ ] **Step 4: Repo impl.** In `user_repository.go`:
```go
func (r *UserRepository) List() ([]models.User, error) {
	var users []models.User
	if err := r.db.Order("id ASC").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// Update writes the mutable profile fields (NOT username or password).
// Select is required so a false `Active` is persisted (GORM skips zero-values otherwise).
func (r *UserRepository) Update(user *models.User) error {
	if err := r.db.Model(user).
		Select("Name", "Email", "Active", "MustChangePassword").
		Updates(user).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateLastLogin(id int, t time.Time) error {
	if err := r.db.Model(&models.User{}).Where("id = ?", id).
		Update("last_login_at", t).Error; err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}
```
(Add `"time"` + `"fmt"` imports if missing.)

- [ ] **Step 5: Test** (follow the existing `user_repository_test.go` testcontainers pattern; if that file skips under `-short`, keep yours consistent): create a user, `List()` returns it; `Update` flips `Active`→false and changes `Name`/`Email` (and a subsequent `GetByID` reflects false); `UpdateLastLogin` sets the timestamp.

- [ ] **Step 6: Gate** (gofmt/build/test-short) + **Commit:** `feat(users): add active/must_change_password/last_login columns + repo methods`.

---

## Task 2: user_service (CRUD + guards + roles + audit)

**Files:** Create `backend/internal/service/user_service.go` + `backend/internal/service/user_service_test.go`; add the service interface to `backend/internal/service/interfaces.go`; add user audit action constants to `backend/internal/models/audit_log.go`.

**Interfaces:**
- Consumes: `repository.UserRepositoryInterface` (Task 1), `models.User`, `AuditServiceInterface.LogEvent`, and a role manager.
- Produces: `UserService` + `UserWithRole{ User models.User; Role string }`, `CreateUserInput`, `UpdateUserInput`.

- [ ] **Step 1: Role-manager interface (avoid import cycle).** In `user_service.go`:
```go
// RoleManager is the subset of goauth used to read/assign a user's role.
type RoleManager interface {
	AssignRole(ctx context.Context, userID, role string) error
	GetUserPermissions(ctx context.Context, userID string) (*store.UserPermissions, error)
}
```
(import `"github.com/aloks98/goauth/store"` — same module the handlers use.)

- [ ] **Step 2: Audit action constants.** In `models/audit_log.go`, in the action-constants block, add:
```go
	// User-management actions
	AuditActionUserCreate       = "user.create"
	AuditActionUserUpdate       = "user.update"
	AuditActionUserDelete       = "user.delete"
	AuditActionUserRoleChange   = "user.role_change"
	AuditActionUserPasswordReset = "user.password_reset"
	AuditActionUserActivate     = "user.activate"
	AuditActionUserDeactivate   = "user.deactivate"
```

- [ ] **Step 3: Types + struct + constructor + sentinel errors.**
```go
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidRole       = errors.New("invalid role")
	ErrCannotModifySelf  = errors.New("you cannot perform this action on your own account")
	ErrLastAdmin         = errors.New("cannot remove or demote the last administrator")
	ErrUsernameTaken     = errors.New("username already in use")
	ErrEmailTaken        = errors.New("email already in use")
)

var validRoles = map[string]bool{"admin": true, "operator": true, "viewer": true}

type UserWithRole struct {
	models.User
	Role string `json:"role"`
}
type CreateUserInput struct {
	Name, Username, Email, Role, Password string
	MustChangePassword                    bool
}
type UpdateUserInput struct {
	Name, Email, Role string
	Active            bool
}

type userService struct {
	repo       repository.UserRepositoryInterface
	roles      RoleManager
	audit      AuditServiceInterface
	bcryptCost int
	logger     *zap.Logger
}

func NewUserService(repo repository.UserRepositoryInterface, roles RoleManager, audit AuditServiceInterface, bcryptCost int, logger *zap.Logger) UserService {
	return &userService{repo: repo, roles: roles, audit: audit, bcryptCost: bcryptCost, logger: logger}
}
```
Add `UserService` to `interfaces.go`:
```go
type UserService interface {
	List(ctx context.Context) ([]UserWithRole, error)
	Get(ctx context.Context, id int) (*UserWithRole, error)
	Create(ctx context.Context, in CreateUserInput, actorID int, ip, ua string) (*UserWithRole, error)
	Update(ctx context.Context, id int, in UpdateUserInput, actorID int, ip, ua string) (*UserWithRole, error)
	ResetPassword(ctx context.Context, id int, password string, mustChange bool, actorID int, ip, ua string) error
	Delete(ctx context.Context, id, actorID int, ip, ua string) error
}
```

- [ ] **Step 4: role read + admin-count helpers.**
```go
func (s *userService) roleOf(ctx context.Context, userID int) string {
	perms, err := s.roles.GetUserPermissions(ctx, strconv.Itoa(userID))
	if err != nil || perms == nil {
		return ""
	}
	return perms.RoleLabel
}

// withRoles attaches each user's role (one GetUserPermissions call per user).
func (s *userService) withRoles(ctx context.Context, users []models.User) []UserWithRole {
	out := make([]UserWithRole, 0, len(users))
	for i := range users {
		out = append(out, UserWithRole{User: users[i], Role: s.roleOf(ctx, users[i].ID)})
	}
	return out
}

func (s *userService) adminCount(ctx context.Context) (int, error) {
	users, err := s.repo.List()
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range users {
		if s.roleOf(ctx, users[i].ID) == "admin" {
			n++
		}
	}
	return n, nil
}
```

- [ ] **Step 5: List/Get/Create.**
```go
func (s *userService) List(ctx context.Context) ([]UserWithRole, error) {
	users, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	return s.withRoles(ctx, users), nil
}

func (s *userService) Get(ctx context.Context, id int) (*UserWithRole, error) {
	u, err := s.repo.GetByID(id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}
	return &UserWithRole{User: *u, Role: s.roleOf(ctx, id)}, nil
}

func (s *userService) Create(ctx context.Context, in CreateUserInput, actorID int, ip, ua string) (*UserWithRole, error) {
	if !validRoles[in.Role] {
		return nil, ErrInvalidRole
	}
	u := &models.User{Name: in.Name, Username: in.Username, Email: in.Email, Active: true, MustChangePassword: in.MustChangePassword}
	if err := u.SetPassword(in.Password, s.bcryptCost); err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.Create(u); err != nil {
		return nil, mapUniqueErr(err) // returns ErrUsernameTaken/ErrEmailTaken on unique-violation, else err
	}
	if err := s.roles.AssignRole(ctx, strconv.Itoa(u.ID), in.Role); err != nil {
		_ = s.repo.Delete(u.ID) // rollback
		return nil, fmt.Errorf("assign role: %w", err)
	}
	s.logUser(ctx, models.AuditActionUserCreate, u, actorID, ip, ua)
	return &UserWithRole{User: *u, Role: in.Role}, nil
}
```
`mapUniqueErr` inspects the GORM error string for the unique constraint (username vs email) and returns the matching sentinel; otherwise returns the wrapped error. `logUser` is a helper (Step 8).

- [ ] **Step 6: Update (with guards).**
```go
func (s *userService) Update(ctx context.Context, id int, in UpdateUserInput, actorID int, ip, ua string) (*UserWithRole, error) {
	if !validRoles[in.Role] {
		return nil, ErrInvalidRole
	}
	u, err := s.repo.GetByID(id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}
	oldRole := s.roleOf(ctx, id)

	// Guard: demoting or deactivating the last admin / self.
	losingAdmin := oldRole == "admin" && in.Role != "admin"
	deactivating := u.Active && !in.Active
	if (losingAdmin || deactivating) && oldRole == "admin" {
		n, err := s.adminCount(ctx)
		if err != nil {
			return nil, err
		}
		if n <= 1 {
			return nil, ErrLastAdmin
		}
	}
	if id == actorID && (losingAdmin || deactivating) {
		return nil, ErrCannotModifySelf
	}

	u.Name, u.Email, u.Active = in.Name, in.Email, in.Active
	if err := s.repo.Update(u); err != nil {
		return nil, mapUniqueErr(err)
	}
	if in.Role != oldRole {
		if err := s.roles.AssignRole(ctx, strconv.Itoa(id), in.Role); err != nil {
			return nil, fmt.Errorf("assign role: %w", err)
		}
		s.logUser(ctx, models.AuditActionUserRoleChange, u, actorID, ip, ua)
	}
	if deactivating {
		s.logUser(ctx, models.AuditActionUserDeactivate, u, actorID, ip, ua)
	} else if !oldUserActive(u, in) { /* activating */ }
	s.logUser(ctx, models.AuditActionUserUpdate, u, actorID, ip, ua)
	return &UserWithRole{User: *u, Role: in.Role}, nil
}
```
(Keep activation/deactivation audit simple: log `AuditActionUserActivate` when `!u.Active→true`, `AuditActionUserDeactivate` when true→false, plus a general `user.update`. Drop the placeholder `oldUserActive` helper — compute the activation transition inline from the pre-update `u.Active` captured before mutation.)

- [ ] **Step 7: ResetPassword + Delete (with guards).**
```go
func (s *userService) ResetPassword(ctx context.Context, id int, password string, mustChange bool, actorID int, ip, ua string) error {
	u, err := s.repo.GetByID(id)
	if err != nil || u == nil {
		return ErrUserNotFound
	}
	if err := u.SetPassword(password, s.bcryptCost); err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.UpdatePassword(id, u.PasswordHash); err != nil {
		return err
	}
	u.MustChangePassword = mustChange
	if err := s.repo.Update(u); err != nil {
		return err
	}
	s.logUser(ctx, models.AuditActionUserPasswordReset, u, actorID, ip, ua)
	return nil
}

func (s *userService) Delete(ctx context.Context, id, actorID int, ip, ua string) error {
	if id == actorID {
		return ErrCannotModifySelf
	}
	u, err := s.repo.GetByID(id)
	if err != nil || u == nil {
		return ErrUserNotFound
	}
	if s.roleOf(ctx, id) == "admin" {
		n, err := s.adminCount(ctx)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.logUser(ctx, models.AuditActionUserDelete, u, actorID, ip, ua)
	return nil
}
```

- [ ] **Step 8: audit helper** (mirror how `audit_service.go` builds `models.AuditEvent`):
```go
func (s *userService) logUser(ctx context.Context, action string, u *models.User, actorID int, ip, ua string) {
	aid := actorID
	uid := u.ID
	_ = s.audit.LogEvent(ctx, models.AuditEvent{
		UserID: &aid, Action: action, ResourceType: "user", ResourceID: &uid,
		ResourceName: u.Username, IPAddress: ip, UserAgent: ua, Status: "success",
	})
}
```

- [ ] **Step 9: Tests** (testify, mock `UserRepositoryInterface` + a fake `RoleManager` + the audit mock; follow `service/mocks` if present). Cover: create assigns role + audits; **Delete self → ErrCannotModifySelf**; **Delete last admin → ErrLastAdmin** (fake roles: one admin); delete non-last admin succeeds; **Update demote last admin → ErrLastAdmin**; **Update deactivate self → ErrCannotModifySelf**; invalid role → ErrInvalidRole; ResetPassword sets must_change. The admin-count uses the fake RoleManager's per-id role.

- [ ] **Step 10: Gate + Commit:** `feat(users): user service with role mgmt, guards, and audit`.

---

## Task 3: users_handler + routes

**Files:** Create `backend/internal/api/handlers/users_handler.go` + `users_handler_test.go`; Modify `backend/internal/api/routes/routes.go`.

**Interfaces:** Consumes `service.UserService` (Task 2). Produces the `/api/users` route group.

- [ ] **Step 1: Handler.** `UsersHandler{ svc service.UserService; logger *zap.Logger }` + `NewUsersHandler`. Methods: `List`, `Get`, `Create`, `Update`, `ResetPassword`, `Delete`. Use `chimw.UserID(r)` for the actor id (as in `settings_handler.go`), `getClientIP(r)` + `r.UserAgent()` for ip/ua. Map service sentinels to status: `ErrUserNotFound`→`utils.NotFound`; `ErrInvalidRole`/`ErrUsernameTaken`/`ErrEmailTaken`→`utils.BadRequest`; `ErrCannotModifySelf`/`ErrLastAdmin`→`utils.BadRequest` (clear message); else `utils.InternalError`. Validate request bodies (required fields, role in set, password length — reuse `validation` rules used by register/change-password). Responses via `utils.Success`. Return the `UserWithRole` DTO.

- [ ] **Step 2: Routes.** In `routes.go`, construct `usersHandler := handlers.NewUsersHandler(userService, logger)` (and build `userService` where the other services are built — Task 2's `NewUserService(userRepo, goauthInstance, auditService, cfg.Security.BcryptCost, logger)`; `goauthInstance` satisfies `RoleManager`). Inside the PROTECTED group, add:
```go
r.Route("/api/users", func(r chi.Router) {
	r.With(chimw.RequirePermission(authAdapter, "users:read", mwConfig)).Get("/", usersHandler.List)
	r.With(chimw.RequirePermission(authAdapter, "users:read", mwConfig)).Get("/{id}", usersHandler.Get)
	r.With(chimw.RequirePermission(authAdapter, "users:manage", mwConfig)).Post("/", usersHandler.Create)
	r.With(chimw.RequirePermission(authAdapter, "users:manage", mwConfig)).Put("/{id}", usersHandler.Update)
	r.With(chimw.RequirePermission(authAdapter, "users:manage", mwConfig)).Post("/{id}/password", usersHandler.ResetPassword)
	r.With(chimw.RequirePermission(authAdapter, "users:manage", mwConfig)).Delete("/{id}", usersHandler.Delete)
})
```

- [ ] **Step 3: Tests** — handler tests with a mock `UserService`: List/Create happy paths; Create with invalid role → 400; Delete returning `ErrLastAdmin` → 400; Get missing → 404. (Mirror `settings_handler_test.go` harness.)

- [ ] **Step 4: Gate + Commit:** `feat(api): user management endpoints (users:read/users:manage)`.

---

## Task 4: Auth flow — inactive reject, last-login, must-change-password

**Files:** Modify `backend/internal/api/handlers/auth_handler.go` (+ `auth_handler_test.go`).

- [ ] **Step 1: Login changes.** In `Login`, after fetching the user and verifying the password: if `!user.Active` → return `utils.Forbidden`/`BadRequest` with "account is disabled" (and `LogLoginFailed` reason "disabled"); on success call `h.userRepo.UpdateLastLogin(user.ID, time.Now())` (log+continue on error); add `MustChangePassword bool json:"must_change_password"` to `LoginResponse` and set it from `user.MustChangePassword`.

- [ ] **Step 2: ChangePassword clears the flag.** After a successful password change, set `must_change_password=false`: load the user, set the field, `h.userRepo.Update(user)` (or extend the existing update path). Keep the existing `UpdatePassword` call for the hash.

- [ ] **Step 3: GetMe** — include `must_change_password` in the returned map (so the SPA can re-check after a token refresh).

- [ ] **Step 4: Tests** — login of an inactive user is rejected; successful login sets `last_login_at` (mock repo asserts `UpdateLastLogin` called) and returns `must_change_password` matching the user; change-password clears the flag.

- [ ] **Step 5: Gate + Commit:** `feat(auth): reject inactive logins, record last login, surface must-change-password`.

---

## Task 5: Registration control — open-registration setting + gated register + public status

**Files:** Modify `backend/internal/models/setting.go` (key const), `backend/internal/api/handlers/auth_handler.go` (Register gating + new `RegistrationStatus` handler + inject settings), `backend/internal/api/routes/routes.go` (public status route + pass settings to AuthHandler). Test: `auth_handler_test.go`.

- [ ] **Step 1: Setting key.** In `setting.go` add `SettingOpenRegistration = "auth.open_registration"`.

- [ ] **Step 2: Inject settings into AuthHandler.** Add a `settings repository.SettingsRepositoryInterface` field to `AuthHandler` + `NewAuthHandler` param; pass it at the routes.go construction site.

- [ ] **Step 3: Gate registration.** Replace the `createUserAndAssignRole` first-user logic with:
```go
open := h.settings.GetValue(models.SettingOpenRegistration, "false") == "true"
count, _ := h.userRepo.Count()
switch {
case count == 0:
	// first-ever user bootstraps as admin (prevents lockout when DEFAULT_USER_* unset)
	role = "admin"
case open:
	role = "viewer"
default:
	utils.Forbidden(w, "registration is disabled") // or BadRequest; pick the existing 403 helper
	return
}
```
Keep the `sync.Mutex` + create + AssignRole(role) + rollback-on-failure. (The startup `createDefaultUserIfNeeded` bootstrap is unchanged.)

- [ ] **Step 4: Public status endpoint.** Add `func (h *AuthHandler) RegistrationStatus(w, r)` returning `utils.Success(w, map[string]bool{"open": open}, "")` where `open` is read from the setting. Register it in the PUBLIC group in routes.go: `r.Get("/api/auth/registration-status", authHandler.RegistrationStatus)`.

- [ ] **Step 5: Tests** — register when closed + table non-empty → 403; register when open → user gets `viewer`; register on empty table → `admin` (bootstrap); `RegistrationStatus` reflects the setting. (Use a mock settings repo returning the value.)

- [ ] **Step 6: Gate + Commit:** `feat(auth): default-off open-registration setting + gated signup + public status`.

---

## Task 6: Frontend — Settings → Users page

**Files:** Create `ui/src/types/user.ts`, `ui/src/hooks/use-users.ts`, `ui/src/components/settings/user-management.tsx` (page + dialogs), `ui/src/routes/_dashboard/settings/users.tsx` (route component export); Modify `ui/src/components/settings/settings-nav.tsx`, `ui/src/lib/router.tsx`, `ui/src/hooks/use-permissions.ts`.

**Patterns to mirror (read first):** `components/settings/metrics-publish-settings.tsx` (RHF + settings hook + Switch + Card footer), `hooks/use-settings.ts` (`useMetricsPublishSettings` query/mutation + `ApiResponse` unwrap + toasts), `components/proxy/proxy-data-grid.tsx` (rnui `DataGrid` + `useReactTable`), `components/acl/basic-auth-tab.tsx` `AddUserModal` (rnui `Dialog` + RHF), `components/settings/settings-nav.tsx` + `lib/router.tsx` (`/settings/metrics` registration).

- [ ] **Step 1: Types** (`types/user.ts`): `Role = 'admin'|'operator'|'viewer'`; `ManagedUser { id; name; username; email; role: Role; active: boolean; must_change_password: boolean; last_login_at: string|null; created_at: string }`; `CreateUserRequest { name; username; email; role: Role; password: string; must_change_password: boolean }`; `UpdateUserRequest { name; email; role: Role; active: boolean }`; `ResetPasswordRequest { password: string; must_change_password: boolean }`.

- [ ] **Step 2: Hook** (`use-users.ts`): `useUsers()` — `useQuery(['users'], () => api.get('users').json<ApiResponse<ManagedUser[]>>().data)` + mutations `createUser` (`api.post('users',{json})`), `updateUser` (`api.put(\`users/${id}\`,{json})`), `resetPassword` (`api.post(\`users/${id}/password\`,{json})`), `deleteUser` (`api.delete(\`users/${id}\`)`), each invalidating `['users']` + toast (mirror use-settings). Add `useRegistrationSetting()` — read `publicApi.get('auth/registration-status').json<...>()` → `{open}`, and a setter that `api.put('settings/auth.open_registration',{json:{value: String(open)}})` then invalidates. (The generic settings update body is `{ value: string }` per `UpdateSettingRequest`.)

- [ ] **Step 3: Permissions** — in `use-permissions.ts` add `canManageUsers: hasPermission('users:manage')` to the returned object.

- [ ] **Step 4: Page** (`components/settings/user-management.tsx`): an "Open registration" `Switch` card at the top (wired to `useRegistrationSetting`), then a `DataGrid` of users (columns: name, username, email, role badge, status badge active/inactive, last login formatted, created) with a toolbar "Add user" button and a per-row actions menu (Edit / Reset password / Activate-Deactivate / Delete). Use local pagination (`getPaginationRowModel`, no `manualPagination`). Loading skeletons + empty state.

- [ ] **Step 5: Dialogs** (in the same file or colocated): **AddUserModal** (RHF+Zod: name, username, email, role select, password, "must change on first login" switch → `createUser`); **EditUserModal** (name, email, role, active; username shown read-only → `updateUser`); **ResetPasswordModal** (password + must-change → `resetPassword`); **Delete** confirm (rnui `AlertDialog` like the proxy delete) → `deleteUser`. Surface guard errors from the API (toast the message, e.g. "cannot remove the last administrator"). Mirror `basic-auth-tab.tsx`'s modal structure.

- [ ] **Step 6: Route export** (`routes/_dashboard/settings/users.tsx`): `export function SettingsUsersPage() { return <UserManagement /> }`.

- [ ] **Step 7: Register nav + route.** Add to `SETTINGS_NAV_ITEMS` (settings-nav.tsx): `{ to: '/settings/users', label: 'Users', description: 'Manage user accounts & access', icon: <Users className="size-4" /> }` (import `Users` from lucide). Gate its visibility on `canManageUsers` (filter the nav items, or conditionally include). In `router.tsx` add `settingsUsersRoute` (path `/users`, component `lazyRouteComponent(() => import('@/routes/_dashboard/settings/users'), 'SettingsUsersPage')`) and include it in `settingsRoute.addChildren([...])`.

- [ ] **Step 8: Gate + grep verify.** `pnpm --dir ui build` + `check` + `test`; `grep -rn "users:manage\|/settings/users\|use-users" ui/src` shows the wiring; field names match the API DTO (snake_case `must_change_password`, `last_login_at`).

- [ ] **Step 9: Commit:** `feat(ui): Settings → Users management page`.

---

## Task 7: Frontend — login & signup adjustments

**Files:** Modify `ui/src/routes/login.tsx`, `ui/src/routes/signup.tsx`. (Reuse the existing change-password UI in `sidebar.tsx`'s `ChangePasswordDialog` pattern, or route to a dedicated change screen — see Step 2.)

- [ ] **Step 1: Login response type + must-change handling.** Extend the login response type with `must_change_password?: boolean`. In `login.tsx onSubmit`, after `setTokens(response.data)`: if `response.data.must_change_password` → navigate to a forced change-password flow (e.g. `/settings` change-password dialog auto-opened, or a minimal `/change-password` step) instead of `/`. Keep it simple: navigate to `/` but pass a flag, OR open the existing change-password dialog. (Pick the lightest path that blocks normal use until changed; document the choice.)

- [ ] **Step 2: Disabled-account error.** In the `catch`, inspect the error response body; if the backend returned the "account is disabled" message (HTTP 403), show that specific message rather than the generic "wrong username or password".

- [ ] **Step 3: Hide signup link when registration closed.** On mount, query `publicApi.get('auth/registration-status')`; render the "Sign up" `<Link>` only when `open === true`.

- [ ] **Step 4: Signup refuses when closed.** In `signup.tsx`, query registration-status on mount; if `open === false`, show a "Registration is disabled — contact your administrator" message (and hide the form) instead of allowing submit.

- [ ] **Step 5: Gate + Commit:** `feat(ui): force password change, disabled-account error, gate signup on registration setting`.

---

## Self-Review

**Spec coverage:** data model (T1); roles via goauth + guards + audit (T2); CRUD API + RBAC (T3); login active/last-login/must-change + change-password clear (T4); open-registration setting + gated register + public status + bootstrap-safe (T5, plus existing `createDefaultUserIfNeeded`); Settings → Users page + dialogs + toggle + nav/route + `canManageUsers` (T6); login/signup adjustments (T7). Username immutable (T6 read-only; repo `Update` excludes it). Admin-only (routes `users:manage`, nav gated on `canManageUsers`, admin `*`). All spec sections map to a task.

**Placeholder scan:** code is concrete; the one prose simplification (activation/deactivation audit transition in T2 Step 6) is called out with the exact rule (capture `u.Active` before mutation; log activate/deactivate accordingly) rather than left vague — the implementer must drop the illustrative `oldUserActive`/`losingAdmin`-only sketch and compute the transition inline. T7 Step 1 leaves the must-change UX to the implementer's lightest option but states the requirement (block normal use until changed).

**Type consistency:** `UserWithRole`/DTO field names (`role`, `active`, `must_change_password`, `last_login_at`) are identical across service (T2), handler (T3), and FE types (T6). Sentinels (`ErrLastAdmin`, `ErrCannotModifySelf`, `ErrUserNotFound`, `ErrInvalidRole`) defined T2, mapped T3. `SettingOpenRegistration` defined T5, read in T5, written via generic settings in T6. `RoleManager` (T2) satisfied by the goauth instance (T3 wiring).
