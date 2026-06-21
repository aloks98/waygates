# M5b — Access (ACL) Forms → React Hook Form Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all 8 Access (ACL) forms from `@tanstack/react-form` / ad-hoc `useState` to React Hook Form + Zod, matching the proxy/L4 pattern — a form-library swap with behavior, validation, and payloads unchanged.

**Architecture:** Each form becomes `useForm<T>({ resolver: zodResolver(schema) })` + `<Form {...form}>` with rnui `FormField`/`FormItem`/`FormControl`/`FormLabel`/`FormMessage`/`FormDescription`. Schemas stay colocated. One task per form; parity (same fields/validation/payload/behavior) is the bar.

**Tech Stack:** React 19, `@e412/rnui-react` (RHF `Form*` components), `react-hook-form`, `@hookform/resolvers/zod`, `zod` 4. Existing deps — nothing new.

## Global Constraints

- **PARITY, not redesign.** Reuse each form's existing zod schema verbatim (same regexes/min/max/required). The request object passed to the `use-acl` hooks must be byte-identical. Modal open/close, create-vs-edit seeding, success-close, and toasts unchanged. Layout/copy unchanged from M5a.
- **Schemas colocated** with each component (do NOT move to `lib/form-validation.ts`).
- **RHF wiring (rnui):** `<Form {...form}>` wrapping `<form onSubmit={form.handleSubmit(onSubmit)}>`; each field via `<FormField control={form.control} name="…" render={({ field }) => (<FormItem><FormLabel/><FormControl>…</FormControl><FormDescription/><FormMessage/></FormItem>)} />`. Replace `FieldError`/`field.state.meta` with `<FormMessage />`.
- **Control bindings** (see Migration Pattern below): text `<Input {...field} />`; Select `value={field.value} onValueChange={field.onChange}`; Switch `checked={field.value} onCheckedChange={field.onChange}`; number `z.number()` + `onChange={(e)=>field.onChange(e.target.value===''?undefined:e.target.valueAsNumber)} value={field.value ?? ''}`; `TagsInput value={field.value ?? []} onValueChange={field.onChange}`; checkbox-grid via `field.value`/`field.onChange` array toggling.
- **Number fields:** `z.number()` NOT `z.coerce.number()`; use `z.number({ error: '…' })` for friendly required messages. `FormItem` is `display:grid` → add `items-start` in multi-column rows.
- **Modals seed via `form.reset(defaults)`** on open / when the edited entity changes (`useEffect(() => { if (open) form.reset(defaults); }, [open, entity, form])`).
- **Two-tabs-one-record (oauth-sso + waygates-account):** each RHF form holds only its slice; `form.reset` from `config` on load; the submit handler spreads the current `config` and overlays its slice before calling `useConfigureWaygatesAuth` — identical merge logic to M5a, just RHF-driven. Neither tab may clobber the other.
- Frontend-only; no backend, no `use-acl.ts`/`types/acl.ts`/route changes.
- Per-task gate (repo root): `pnpm --dir ui build` && `pnpm --dir ui check` && `pnpm --dir ui test` — all green; the existing **57 tests must keep passing**. No new unit tests unless a pure helper is extracted. Stage only the task's file(s); never `git add -A`.
- Branch: `feat/rnui-m5b-access-forms` (off `feat/rnui-m5-access`, stacked on PR #29).
- Conventional Commits; end every commit with `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

## Migration Pattern (reference — every task uses this)

Reference implementation: `ui/src/components/proxy/forms/shared/{basics-fields,redirect-options-fields,load-balancing-fields,backend-fields}.tsx`. Read one before starting if unsure.

Imports: drop `useForm` from `@tanstack/react-form` and the rnui `Field`/`FieldLabel`/`FieldContent`/`FieldError`/`FieldGroup` primitives; add:
```tsx
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
```

Form setup + submit:
```tsx
const form = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues });
// modal: seed on open
useEffect(() => { if (open) form.reset(defaults); }, [open, /* entity */, form]);
// ...
<Form {...form}>
  <form onSubmit={form.handleSubmit(onSubmit)} className="…">
    {/* fields */}
  </form>
</Form>
```

Per-control transforms (TanStack `form.Field` → RHF `FormField`):

```tsx
// TEXT
<FormField control={form.control} name="name" render={({ field }) => (
  <FormItem>
    <FormLabel>Name</FormLabel>
    <FormControl><Input placeholder="…" {...field} /></FormControl>
    <FormMessage />
  </FormItem>
)} />

// SELECT
<FormField control={form.control} name="rule_type" render={({ field }) => (
  <FormItem>
    <FormLabel>…</FormLabel>
    <Select value={field.value} onValueChange={field.onChange}>
      <FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl>
      <SelectContent>{/* SelectItem value="allow" etc. */}</SelectContent>
    </Select>
    <FormMessage />
  </FormItem>
)} />

// SWITCH (horizontal row)
<FormField control={form.control} name="enabled" render={({ field }) => (
  <FormItem className="flex flex-row items-center justify-between">
    <div className="space-y-0.5"><FormLabel>…</FormLabel><FormDescription>…</FormDescription></div>
    <FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl>
  </FormItem>
)} />

// NUMBER (z.number(), convert at boundary)
<FormField control={form.control} name="priority" render={({ field }) => (
  <FormItem>
    <FormLabel>Priority</FormLabel>
    <FormControl>
      <Input type="number" min={0} max={1000}
        value={field.value ?? ''}
        onChange={(e) => field.onChange(e.target.value === '' ? undefined : e.target.valueAsNumber)} />
    </FormControl>
    <FormMessage />
  </FormItem>
)} />

// TAGSINPUT
<FormField control={form.control} name="allowed_emails" render={({ field }) => (
  <FormItem>
    <FormLabel>…</FormLabel>
    <FormControl><TagsInput value={field.value ?? []} onValueChange={field.onChange} delimiters={['Enter', ',']} validation={…} /></FormControl>
    <FormDescription>…</FormDescription>
    <FormMessage />
  </FormItem>
)} />

// CHECKBOX GRID (array field)
<FormField control={form.control} name="allowed_roles" render={({ field }) => {
  const selected: string[] = field.value ?? [];
  const toggle = (v: string, c: boolean) => field.onChange(c ? [...selected, v] : selected.filter((x) => x !== v));
  return (<FormItem>{/* map options to <Checkbox checked={selected.includes(v)} onCheckedChange={(c)=>toggle(v, !!c)} /> */}<FormMessage /></FormItem>);
}} />
```

Parity rule per task: **diff the field set, validation, and the submit payload against the pre-migration component** — they must match exactly.

---

### Task 1: acl-group-form-modal → RHF

**Files:** Modify `ui/src/components/acl/acl-group-form-modal.tsx`

Fields: `name` (text, required), `description` (textarea, optional), `combination_mode` (Select: any/all/ip_bypass, rendered via the existing `getModeLabel` options — keep that). Create vs edit: seeds from the group on edit; on create, navigates to the new group's detail after save (keep). Toast copy unchanged (already "access group" after M5a).

- [ ] **Step 1:** Read the current component; note its zod schema, the 3 fields, defaultValues, and the create/edit submit logic.
- [ ] **Step 2:** Migrate to RHF per the pattern — `useForm` + `zodResolver(schema)`; `<Form {...form}>`; the 3 fields via `FormField`; `form.reset` seed from the group on open (edit) / empty (create); submit calls the same create/update hook with the same payload; preserve the navigate-on-create.
- [ ] **Step 3:** Remove the `@tanstack/react-form` import and the rnui `Field`/`FieldLabel`/`FieldError`/`FieldGroup` imports.
- [ ] **Step 4: Gate** — `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` green.
- [ ] **Step 5: Commit** — `refactor(ui): migrate ACL group form modal to RHF (M5b)`.

---

### Task 2: ip-rules-tab (IPRuleFormModal) → RHF

**Files:** Modify `ui/src/components/acl/ip-rules-tab.tsx`

Fields: `cidr` (text, regex-validated), `rule_type` (Select allow/deny/bypass — render labels via the existing `getRuleTypeLabel`/inline descriptions from M5a; keep option values), `priority` (**number**, 0–1000 — use `z.number()` + boundary conversion), `description` (textarea, optional). Add + Edit both use the modal. Only the modal's form migrates; the table/list rendering stays.

- [ ] **Step 1:** Read the component; note the IPRuleFormModal's schema + 4 fields + add-vs-edit seeding.
- [ ] **Step 2:** Migrate the modal to RHF per the pattern; `priority` uses the number-field transform (`z.number()`, not coerce); `form.reset` seeds the edited rule / empty for add; submit calls the same add/update hook with the same payload.
- [ ] **Step 3:** Remove TanStack + rnui `Field*` imports (only those the modal used).
- [ ] **Step 4: Gate** green.
- [ ] **Step 5: Commit** — `refactor(ui): migrate ACL IP-rule form to RHF (M5b)`.

---

### Task 3: basic-auth-tab (AddUserModal) → RHF

**Files:** Modify `ui/src/components/acl/basic-auth-tab.tsx`

Fields: `username` (text, alphanumeric/underscore/hyphen), `password` (password input with show/hide toggle, 8+ chars). Add-only modal. Keep the show/hide toggle behavior.

- [ ] **Step 1:** Read the component; note the schema + 2 fields + the password show/hide local state (keep that `useState` — it's UI state, not form state).
- [ ] **Step 2:** Migrate the modal form to RHF; `password` input keeps the show/hide toggle (local `useState` for `showPassword`, bound to input `type`); submit calls the same add hook with the same payload.
- [ ] **Step 3:** Remove TanStack + rnui `Field*` imports.
- [ ] **Step 4: Gate** green.
- [ ] **Step 5: Commit** — `refactor(ui): migrate ACL basic-auth form to RHF (M5b)`.

---

### Task 4: external-providers-tab (ProviderFormModal) → RHF

**Files:** Modify `ui/src/components/acl/external-providers-tab.tsx`

Fields: `provider_type` (Select: authelia/authentik/custom), `name` (text), `verify_url` (URL, required), `auth_redirect_url` (URL, optional), `headers_to_copy` (TagsInput / comma-list parsing). "Forward Auth" labels from M5a stay. Add + Edit modal.

- [ ] **Step 1:** Read the component; note the schema + 5 fields + how `headers_to_copy` is parsed (keep the same parsing → array) + add-vs-edit seeding.
- [ ] **Step 2:** Migrate to RHF; `headers_to_copy` via the TagsInput transform (`field.value`/`field.onChange`); `form.reset` seeds the edited provider / empty for add; submit payload unchanged.
- [ ] **Step 3:** Remove TanStack + rnui `Field*` imports.
- [ ] **Step 4: Gate** green.
- [ ] **Step 5: Commit** — `refactor(ui): migrate ACL forward-auth provider form to RHF (M5b)`.

---

### Task 5: oauth-sso-tab (provider form + ProviderRestrictionModal) → RHF

**Files:** Modify `ui/src/components/acl/oauth-sso-tab.tsx`

Two forms in this file:
1. **Main form** — `allowed_providers: string[]` (the provider checkbox grid). Migrate to RHF; the grid uses the checkbox-grid transform. **Load-merge-save:** `form.reset({ allowed_providers: config?.allowed_providers ?? [] })` on load; on submit, spread the current `config` account fields and overlay `allowed_providers` (the exact merge from M5a — `enabled`/`allowed_users`/`allowed_roles`/`allowed_email_patterns`/`require_2fa`/`session_ttl` from `config?.*`, `allowed_providers` from the form) and call `useConfigureWaygatesAuth`.
2. **ProviderRestrictionModal** (currently `useState`) — fields `enabled` (Switch), `allowed_emails` (TagsInput), `allowed_domains` (TagsInput). Migrate to its own small RHF form; `form.reset` from the provider's existing restriction on open; submit calls `useSetOAuthProviderRestriction` with the same payload.

- [ ] **Step 1:** Read the component; note the main form's `allowed_providers` handling + the exact load-merge-save submit, and the ProviderRestrictionModal's 3 fields + open-seed + save.
- [ ] **Step 2:** Migrate the main form to RHF (checkbox grid; keep the per-provider "Configure Restrictions" button + restriction-summary badge logic). Preserve the load-merge-save submit exactly.
- [ ] **Step 3:** Migrate ProviderRestrictionModal to its own RHF form (Switch + 2 TagsInput); seed via `form.reset` on open; same save call.
- [ ] **Step 4:** Remove TanStack + rnui `Field*` imports; remove the modal's `useState` form state.
- [ ] **Step 5: Gate** green.
- [ ] **Step 6: Commit** — `refactor(ui): migrate ACL OAuth/SSO tab to RHF (M5b)`.

---

### Task 6: waygates-account-tab (+ duration picker) → RHF  — highest risk

**Files:** Modify `ui/src/components/acl/waygates-account-tab.tsx`

Fields: `enabled` (Switch), `allowed_roles` (checkbox grid), `allowed_email_patterns` (TagsInput), `require_2fa` (carried in payload; no UI today — keep it in defaults/payload), `session_ttl` (**duration picker** — number + unit Select, stored as seconds via `session-duration.ts`). `allowed_users` has no UI but is seeded/forwarded — keep in defaults + payload.

- [ ] **Step 1:** Read the component; note the schema, the load-merge-save submit, the duration picker (number `Input` + unit `Select` driven by `secondsToDuration`/`durationToSeconds`), the roles checkbox grid, and the email-patterns TagsInput.
- [ ] **Step 2:** Migrate to RHF. `enabled` Switch; `allowed_roles` checkbox grid; `allowed_email_patterns` TagsInput — all via `FormField`. The conditional sub-section still renders when `enabled` (use `form.watch('enabled')`).
- [ ] **Step 3: Duration picker via `FormField` for `session_ttl`** (the form field stays seconds): the number `Input` shows `secondsToDuration(field.value).value`, the unit `Select` uses local `useState` unit (seeded from `secondsToDuration(config.session_ttl).unit`); both call `field.onChange(durationToSeconds(value, unit))`. Keep the 60–604800 bound (schema `z.number().min(SESSION_TTL_MIN).max(SESSION_TTL_MAX)`).
- [ ] **Step 4: Load-merge-save** — `form.reset` seeds all account fields from `config` (incl. `allowed_users`); submit spreads `config?.allowed_providers` (preserve the OAuth slice) and sends the account fields from the form. Identical to M5a's verified merge.
- [ ] **Step 5:** Remove TanStack + rnui `Field*` imports.
- [ ] **Step 6: Gate** green.
- [ ] **Step 7: Commit** — `refactor(ui): migrate ACL Waygates Account tab to RHF (M5b)`.

---

### Task 7: acl-selector (proxy-form widget) → RHF

**Files:** Modify `ui/src/components/proxy/forms/acl-selector.tsx`

This is a **controlled widget**: props `value: ACLAssignment[]` + `onChange(next)` (the parent proxy form passes ACL assignments separately — NOT through the proxy RHF context). Today its internal "add assignment" inputs use `useState`. Migrate ONLY those add-assignment inputs to their own small standalone RHF form (`group` Select, `path_pattern` text, `priority` number, `enabled` switch — match the current fields). On "Add", append to `value` via `onChange` and `form.reset()` the add-form. **Do NOT** register this widget into the parent proxy form's RHF context — its public `value`/`onChange` contract is unchanged.

- [ ] **Step 1:** Read the component; note its public props, the assignment list rendering, and the add-assignment input fields + their `useState`.
- [ ] **Step 2:** Replace the add-assignment `useState` inputs with a small `useForm` + `FormField`s; on submit append to `value` (via `onChange`) and `form.reset()`. Keep the list rendering + per-assignment enable/remove behavior unchanged.
- [ ] **Step 3:** Confirm the widget still takes/emits the same `value`/`onChange` and is NOT wired into any parent form context.
- [ ] **Step 4: Gate** green (build exercises the proxy create/edit forms that embed this widget).
- [ ] **Step 5: Commit** — `refactor(ui): migrate ACL selector add-form to RHF (M5b)`.

---

### Task 8: acl-branding-settings → RHF + final verification

**Files:** Modify `ui/src/components/settings/acl-branding-settings.tsx`

Migrate the branding panel from `useState` to RHF. Fields: `logo_url`, `primary_color` (color picker), `background_color`, `title`, `subtitle`, `footer_text`, `custom_css` (textarea, `sanitizeCSS` on the value before save/preview — keep). The **live `LoginPreview`** updates as the admin edits — drive it from `form.watch()`. The save button enables on changes — use `form.formState.isDirty` (replaces the manual `hasChanges` diff). Submit calls `useUpdateACLBranding` with the same payload; on success `form.reset(values)` to clear dirty.

- [ ] **Step 1:** Read the component; note its fields, the `hasChanges` diff, the `errors` memo, `LoginPreview`, the color `<input type="color">`, and `sanitizeCSS`.
- [ ] **Step 2:** Migrate to RHF — `useForm` seeded from the fetched branding (`form.reset` when it loads); each field via `FormField`; `LoginPreview` reads `form.watch()`; save-enable via `formState.isDirty`; keep `sanitizeCSS` applied to the custom-CSS value for preview + payload; same update hook + payload; `form.reset(saved)` on success.
- [ ] **Step 3:** Remove the manual `useState`/`hasChanges`/`errors` form scaffolding.
- [ ] **Step 4: Verify TanStack is gone** — run and include the output:
  ```bash
  grep -rn "@tanstack/react-form" ui/src/components/acl ui/src/components/proxy/forms/acl-selector.tsx ui/src/components/settings/acl-branding-settings.tsx
  ```
  Expected: **no output** (the only remaining `@tanstack/react-form` in the app is `acl-login-form.tsx` (end-user, out of scope) and `catchall-settings.tsx` (M6) — confirm those two are the ONLY remaining hits via `grep -rn "@tanstack/react-form" ui/src`).
- [ ] **Step 5: Gate** green.
- [ ] **Step 6: Commit** — `refactor(ui): migrate ACL branding settings to RHF; remove TanStack from Access admin (M5b)`.

---

## Self-Review

**Spec coverage:**
- All 8 forms migrated → Tasks 1–8. ✅
- Parity (same fields/validation/payload/behavior) → Global Constraints + per-task "diff against pre-migration" rule. ✅
- Colocated schemas → Global Constraints. ✅
- Two-tabs-one-record load-merge-save preserved → Tasks 5, 6 (exact M5a merge). ✅
- Duration picker preserved → Task 6 (session-duration helpers, seconds in the field). ✅
- acl-selector stays controlled, not nested in parent RHF → Task 7. ✅
- branding live preview via watch + isDirty + sanitizeCSS → Task 8. ✅
- rnui+RHF gotchas (z.number not coerce, friendly error, items-start, Switch/Select bindings) → Global Constraints + Migration Pattern. ✅
- TanStack removed from Access admin (verified) → Task 8 Step 4. ✅
- Out of scope (acl-login-form, catchall) → Global Constraints + Task 8 Step 4 note. ✅
- No backend change → Global Constraints. ✅

**Placeholder scan:** The Migration Pattern section carries the full worked transforms for every control type (text/Select/Switch/number/TagsInput/checkbox-grid); each task lists its concrete field set + special handling and directs transformation of the named source file with parity diffing. No "TBD"/"add validation"/"similar to" placeholders. (Per-form full code is not duplicated because the schemas already exist in the source and the transform is uniform — the pattern section + field inventory is the spec.)

**Type consistency:** `FormField`/`FormItem`/`FormControl`/`FormLabel`/`FormMessage`/`FormDescription`, `form.control`, `form.reset`, `form.watch`, `form.formState.isDirty`, `form.handleSubmit` used consistently; the load-merge-save fields (`allowed_providers` vs the account fields) match M5a's `ConfigureWaygatesAuthRequest`; `secondsToDuration`/`durationToSeconds` (from `session-duration.ts`) used as in M5a.
