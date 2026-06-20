# M0b — Form Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the React Hook Form + Zod foundation using rnui's `Form`/`FormField` helpers, and migrate the three auth forms (login, signup, change-password) to it as the reference implementation.

**Architecture:** rnui's `Form` is the standard shadcn RHF pattern (`Form = FormProvider`, `FormField` wraps `Controller`, `FormMessage` auto-renders the field's RHF error). Each form becomes `useForm({ resolver: zodResolver(schema), defaultValues })` + `<Form {...form}><form onSubmit={form.handleSubmit(onSubmit)}>…<FormField .../></form></Form>`. The existing Zod schemas are reused verbatim. Server/submit errors keep their existing `Alert` banner (separate from field validation). `@tanstack/react-form` stays installed — domain forms (proxy/L4/ACL) migrate to RHF in M1–M3.

**Tech Stack:** `react-hook-form`, `@hookform/resolvers` (zodResolver), `zod` (already v4.3.4), rnui `Form`/`FormField`/`FormItem`/`FormControl`/`FormLabel`/`FormMessage`/`Input`/`Button`/`Alert`, Vitest + RTL.

## Global Constraints

- pnpm; UI in `ui/`. Run `pnpm --dir ui <script>`. Branch `feat/rnui-redesign-program`.
- **Reuse the existing Zod schemas verbatim** (login/signup schemas live in their route files; change-password schema is in `sidebar.tsx`). Do not change validation rules or messages.
- **Preserve each form's submit behavior exactly** (API calls, navigation, token storage, mutation, success/error `Alert`, dialog reset/close). Only the form *plumbing* changes (TanStack Form → RHF).
- `@hookform/resolvers` must be a version compatible with **zod 4** (resolvers ≥ 3.10 / latest). If install resolves to something that errors against zod 4, pick a working version and note it.
- Use RHF `mode: 'onTouched'` so field errors appear after first interaction (matches the prior UX where errors showed once a field was touched).
- Verification gates: `pnpm --dir ui build` (success) + `pnpm --dir ui check:fix && pnpm --dir ui check` (clean; pre-existing oxlint warnings OK) + `pnpm --dir ui test run` (pass). **No `tsc` gate.**
- Tests one-shot: `pnpm --dir ui test run <path>`. Commits: Conventional Commits + trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

### The RHF field block template (apply per field)
```tsx
<FormField
  control={form.control}
  name="FIELD_NAME"
  render={({ field }) => (
    <FormItem>
      <FormLabel>LABEL</FormLabel>
      <FormControl>
        <Input type="TYPE" placeholder="PLACEHOLDER" autoComplete="AUTOCOMPLETE" {...field} />
      </FormControl>
      <FormMessage />
    </FormItem>
  )}
/>
```
`FormMessage` renders the RHF error automatically (including a cross-field `.refine` error placed on its `path`). No manual `hasError`/`aria-invalid` wiring is needed — rnui's `FormControl`/`FormMessage` handle it.

---

## Task 1: Add react-hook-form + resolvers

**Files:**
- Modify: `ui/package.json`

**Interfaces:**
- Produces: `react-hook-form` and `@hookform/resolvers` (`zodResolver`) resolvable.

- [ ] **Step 1: Install**

```bash
pnpm --dir ui add react-hook-form @hookform/resolvers
```

- [ ] **Step 2: Verify the resolver imports against zod 4**

Run: `pnpm --dir ui exec node -e "require('react-hook-form');require('@hookform/resolvers/zod');console.log('ok')"`
Expected: prints `ok`. (Peer-dep WARNINGS are acceptable; a hard failure is not — if `@hookform/resolvers` rejects zod 4, install the latest version explicitly and note it in the report.)

- [ ] **Step 3: Commit**

```bash
git add ui/package.json ui/pnpm-lock.yaml
git commit -m "build(ui): add react-hook-form + @hookform/resolvers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Migrate login.tsx to RHF (reference) + test

**Files:**
- Modify: `ui/src/routes/login.tsx`
- Create: `ui/src/routes/login.test.tsx`

**Interfaces:**
- Consumes: rnui `Form`/`FormField`/…, `zodResolver`, `react-hook-form` `useForm`.
- Produces: the canonical RHF form pattern other tasks copy.

- [ ] **Step 1: Write the failing test `ui/src/routes/login.test.tsx`**

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';

const navigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigate,
  useSearch: () => ({}),
  Link: ({ to, children, ...props }: any) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));
const postJson = vi.fn();
vi.mock('@/lib/api', () => ({ publicApi: { post: () => ({ json: postJson }) } }));
vi.mock('../lib/api', () => ({ publicApi: { post: () => ({ json: postJson }) } }));

import { LoginPage } from './login';

test('shows validation errors on empty submit and does not call the API', async () => {
  render(<LoginPage />);
  await userEvent.click(screen.getByRole('button', { name: /sign in/i }));
  expect(await screen.findByText(/username or email is required/i)).toBeInTheDocument();
  expect(screen.getByText(/password is required/i)).toBeInTheDocument();
  expect(postJson).not.toHaveBeenCalled();
});

test('submits valid credentials to the API', async () => {
  postJson.mockResolvedValueOnce({ success: true, data: { access_token: 'a', refresh_token: 'b' } });
  render(<LoginPage />);
  await userEvent.type(screen.getByLabelText(/username or email/i), 'admin');
  await userEvent.type(screen.getByLabelText(/password/i), 'secret');
  await userEvent.click(screen.getByRole('button', { name: /sign in/i }));
  await waitFor(() => expect(postJson).toHaveBeenCalled());
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm --dir ui test run src/routes/login.test.tsx`
Expected: FAIL — current login uses TanStack Form; either the errors don't render the same way or (for the mock to resolve) the component differs. (It must end GREEN after Step 3.)

- [ ] **Step 3: Rewrite `ui/src/routes/login.tsx`**

Replace the imports + the form logic. Keep the entire visual layout (the brand panel, headings, spacing, the `justRegistered`/`error` `Alert`s, and the Sign-up link) byte-for-byte — only swap the form plumbing. New top + form:

```tsx
import {
  Alert,
  AlertDescription,
  Button,
  FieldGroup,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { Link, useNavigate, useSearch } from '@tanstack/react-router';
import { CheckCircle2, XCircle } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { WaygateLogo } from '@/components/layout/waygate-logo';

import { publicApi } from '../lib/api';
import { useAuthStore } from '../stores/auth';
import type { ApiResponse, TokenPair } from '../types/api';

const loginSchema = z.object({
  identifier: z.string().min(1, 'Username or email is required'),
  password: z.string().min(1, 'Password is required'),
});
type LoginValues = z.infer<typeof loginSchema>;

export function LoginPage() {
  const navigate = useNavigate();
  const searchParams = useSearch({ strict: false }) as { registered?: string };
  const { setTokens } = useAuthStore();
  const [error, setError] = useState<string | null>(null);
  const justRegistered = searchParams.registered === 'true';

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    mode: 'onTouched',
    defaultValues: { identifier: '', password: '' },
  });
  const isSubmitting = form.formState.isSubmitting;

  const onSubmit = async (value: LoginValues) => {
    setError(null);
    try {
      const response = await publicApi
        .post('auth/login', { json: value })
        .json<ApiResponse<TokenPair>>();
      if (response.success && response.data) {
        setTokens(response.data);
        navigate({ to: '/dashboard' });
      } else {
        setError(response.message || 'Login failed');
      }
    } catch {
      setError('Wrong username or password. Check your details and try again.');
    }
  };

  return (
    /* …unchanged layout up to the <form>… */
  );
}
```

Then the `<form>` block becomes (replacing the old `<form onSubmit={...form.handleSubmit()}>` + `form.Field`/`form.Subscribe` JSX), keeping the surrounding `FieldGroup` and the two alerts and the Sign-up link unchanged:

```tsx
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)}>
              <FieldGroup>
                {justRegistered && !error && (
                  <Alert variant="success">
                    <CheckCircle2 className="size-4" />
                    <AlertDescription>Account created! Sign in to get started.</AlertDescription>
                  </Alert>
                )}

                {error && (
                  <Alert variant="destructive">
                    <XCircle className="size-4" />
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}

                <FormField
                  control={form.control}
                  name="identifier"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Username or Email</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="Enter your username or email"
                          autoComplete="username"
                          autoFocus
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Password</FormLabel>
                      <FormControl>
                        <Input
                          type="password"
                          placeholder="Enter your password"
                          autoComplete="current-password"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button type="submit" disabled={isSubmitting} className="w-full">
                  {isSubmitting ? 'Signing in...' : 'Sign in'}
                </Button>

                <p className="text-center text-sm text-muted-foreground">
                  Don&apos;t have an account?{' '}
                  <Link to="/signup" className="text-primary hover:underline">
                    Sign up
                  </Link>
                </p>
              </FieldGroup>
            </form>
          </Form>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm --dir ui test run src/routes/login.test.tsx`
Expected: 2 passed.

- [ ] **Step 5: Build + commit**

Run: `pnpm --dir ui build` → success. `pnpm --dir ui test run` → all pass.
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/routes/login.tsx ui/src/routes/login.test.tsx
git commit -m "refactor(ui): migrate login form to react-hook-form

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Migrate signup.tsx to RHF

**Files:**
- Modify: `ui/src/routes/signup.tsx`

**Interfaces:**
- Consumes: the login RHF pattern (Task 2).

Apply the **exact same migration pattern as login** (Task 2) to `ui/src/routes/signup.tsx`. Read the current file first to preserve its layout and submit behavior verbatim; only swap the form plumbing.

- Schema (unchanged): `signupSchema = z.object({ name, username, email, password, confirmPassword }).refine(data => data.password === data.confirmPassword, { message: "Passwords don't match", path: ['confirmPassword'] })`. Add `type SignupValues = z.infer<typeof signupSchema>;`.
- `useForm<SignupValues>({ resolver: zodResolver(signupSchema), mode: 'onTouched', defaultValues: { name: '', username: '', email: '', password: '', confirmPassword: '' } })`.
- Keep the existing `onSubmit` body (POST `auth/register`, then navigate to `/login?registered=true` — copy whatever the current submit does, just move it into `onSubmit(value)` and wrap the `<form>` with `<Form {...form}><form onSubmit={form.handleSubmit(onSubmit)}>`).
- Replace each `form.Field` block with a `FormField` block per the template — five fields:
  - `name` → label "Full Name", autoComplete "name"
  - `username` → label "Username", autoComplete "username"
  - `email` → label "Email", type "email", autoComplete "email"
  - `password` → label "Password", type "password", autoComplete "new-password"
  - `confirmPassword` → label "Confirm Password", type "password", autoComplete "new-password"
- The `.refine` "Passwords don't match" error appears on the `confirmPassword` field's `<FormMessage />` automatically.
- Replace `form.Subscribe` submit button with `disabled={form.formState.isSubmitting}` and keep the same button label logic.
- Update imports the same way login did (drop `@tanstack/react-form`, `Field`/`FieldError`/`FieldLabel`; add `Form`/`FormField`/`FormItem`/`FormControl`/`FormLabel`/`FormMessage`, `useForm` from `react-hook-form`, `zodResolver`). Keep `FieldGroup` and the `Alert` server-error pattern.

- [ ] **Step 1: Read `ui/src/routes/signup.tsx`; apply the migration above.**
- [ ] **Step 2: Build + verify.** Run: `pnpm --dir ui build` → success; `pnpm --dir ui test run` → all pass.
- [ ] **Step 3: Lint + commit.**
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/routes/signup.tsx
git commit -m "refactor(ui): migrate signup form to react-hook-form

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Migrate the Change-Password dialog to RHF

**Files:**
- Modify: `ui/src/components/layout/sidebar.tsx` (the `ChangePasswordDialog` component, ~lines 100-308)

**Interfaces:**
- Consumes: the login RHF pattern (Task 2).

Apply the same RHF migration to `ChangePasswordDialog` in `ui/src/components/layout/sidebar.tsx`. Read it first; preserve the dialog, the `useMutation` call, the success/error `Alert` (`status` state), the reset-on-close, and the 1.5s success-close behavior exactly — only swap the form plumbing.

- Schema (unchanged): `passwordSchema = z.object({ current_password, new_password, confirm_password }).refine(d => d.new_password === d.confirm_password, { message: "Passwords don't match", path: ['confirm_password'] })`. Add `type PasswordValues = z.infer<typeof passwordSchema>;`.
- `useForm<PasswordValues>({ resolver: zodResolver(passwordSchema), mode: 'onTouched', defaultValues: { current_password: '', new_password: '', confirm_password: '' } })`.
- `onSubmit(value)`: keep the current body (calls `mutation.mutate({ current_password, new_password })`). Wrap with `<Form {...form}><form onSubmit={form.handleSubmit(onSubmit)}>`.
- Three `FormField` blocks (all `type="password"`):
  - `current_password` → "Current Password", autoComplete "current-password"
  - `new_password` → "New Password", autoComplete "new-password"
  - `confirm_password` → "Confirm New Password", autoComplete "new-password"
- On dialog close/reset, call `form.reset()` (replace the TanStack `form.reset()` call in `handleOpenChange`/onSuccess).
- Update imports in sidebar.tsx: remove `@tanstack/react-form` `useForm` and the `Field`/`FieldError`/`FieldGroup`/`FieldLabel` used ONLY by this dialog (verify they aren't used elsewhere in sidebar.tsx before removing); add the rnui `Form`/`FormField`/… and `react-hook-form` `useForm` + `zodResolver`. Keep `FieldGroup` if other parts of the file use it.

- [ ] **Step 1: Read the `ChangePasswordDialog`; apply the migration above.**
- [ ] **Step 2: Build + verify.** Run: `pnpm --dir ui build` → success; `pnpm --dir ui test run` → all pass.
- [ ] **Step 3: Lint + commit.**
```bash
pnpm --dir ui check:fix && pnpm --dir ui check
git add ui/src/components/layout/sidebar.tsx
git commit -m "refactor(ui): migrate change-password dialog to react-hook-form

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Verification

**Files:** none.

- [ ] **Step 1: No TanStack Form left in the auth forms**

Run: `grep -rn "@tanstack/react-form" ui/src/routes/login.tsx ui/src/routes/signup.tsx ui/src/components/layout/sidebar.tsx` → expect **no matches**. (`@tanstack/react-form` remains used by proxy/L4/ACL forms — that's expected; do not remove the dependency.)

- [ ] **Step 2: Gates**

Run: `pnpm --dir ui build` → success · `pnpm --dir ui test run` → all pass · `pnpm --dir ui check` → clean.

- [ ] **Step 3: Note for the controller's `verify` pass**

Interactive smoke: login shows "required" errors on empty submit and logs in with valid creds; signup validates all 5 fields incl. "Passwords don't match" under Confirm Password, and registers; change-password validates + submits + shows success and closes.

## Done criteria
- login, signup, change-password run on react-hook-form + zodResolver via rnui `Form`/`FormField`; validation + submit behavior preserved; login has tests.
- `@tanstack/react-form` no longer imported by the 3 auth files (still a dep for domain forms).
- `pnpm --dir ui build` + `test run` + `check` all green.
- **M0b complete** after this plan (IA restructure ✓, Shell QoL ✓, Form Foundation). Next program milestone: **M1 — Dashboard** (or backend pipeline B1 — config preview).
