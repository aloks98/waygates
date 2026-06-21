# M5b — Access (ACL) Forms → React Hook Form — Design

**Date:** 2026-06-21
**Program:** rnui UI redesign (M5, PR 2 of 2; stacks on M5a / PR #29)
**Status:** Approved — ready for implementation plan

## Goal

Migrate every Access (ACL) form's state management to **React Hook Form + Zod**, matching the
established proxy/L4 form pattern, eliminating both `@tanstack/react-form` and ad-hoc `useState`
form state from the Access code. This is a **form-library swap, not a redesign** — validation
rules, API payloads, and user-visible behavior are unchanged.

## Context

- **Reference pattern (proxy/L4):** `ui/src/components/proxy/forms/reverse-proxy-form.tsx` uses
  RHF `useForm<T>({ resolver: zodResolver(schema) })`, `<Form {...form}>`, and the rnui RHF
  components `FormField` / `FormItem` / `FormControl` / `FormLabel` / `FormMessage` /
  `FormDescription`. `react-hook-form` + `@hookform/resolvers/zod` are already dependencies.
- **Current ACL forms** use `@tanstack/react-form` (`form.Field` render-props,
  `field.state.value` / `field.handleChange` / `field.state.meta.errors`) with the rnui
  non-RHF field primitives (`Field` / `FieldLabel` / `FieldError` / `FieldGroup`), and inline
  zod schemas. A few ACL forms use plain `useState`.
- M5a already restructured the tabs (split Waygates into `oauth-sso-tab` + `waygates-account-tab`,
  added the duration picker, the load-merge-save). M5b swaps those (and the rest) to RHF.

## Scope

**In — migrate all of these to RHF + Zod:**

| Component | Today |
|---|---|
| `components/acl/acl-group-form-modal.tsx` | TanStack |
| `components/acl/ip-rules-tab.tsx` (IPRuleFormModal) | TanStack |
| `components/acl/basic-auth-tab.tsx` (AddUserModal) | TanStack |
| `components/acl/external-providers-tab.tsx` (ProviderFormModal) | TanStack |
| `components/acl/oauth-sso-tab.tsx` (provider form) | TanStack |
| `components/acl/oauth-sso-tab.tsx` (`ProviderRestrictionModal`) | useState |
| `components/acl/waygates-account-tab.tsx` (+ duration picker) | TanStack |
| `components/proxy/forms/acl-selector.tsx` (add-assignment inputs) | useState |
| `components/settings/acl-branding-settings.tsx` | useState |

**Out of scope:** `components/acl/acl-login-form.tsx` (end-user login page, not admin) and
`components/settings/catchall-settings.tsx` (M6 Settings) — both remain TanStack for now. No
backend change. No UI redesign — layout/copy stay as M5a left them.

## Migration pattern (uniform)

For each form:
1. Replace the TanStack `useForm` (or `useState` form state) with RHF
   `const form = useForm<T>({ resolver: zodResolver(schema), defaultValues })`.
2. Wrap the form body in `<Form {...form}>` and submit via `form.handleSubmit(onSubmit)`.
3. Replace each `form.Field name="x"` render-prop block with rnui:
   ```tsx
   <FormField control={form.control} name="x" render={({ field }) => (
     <FormItem>
       <FormLabel>…</FormLabel>
       <FormControl>{/* Input/Select/Switch/Checkbox/TagsInput bound to field */}</FormControl>
       <FormDescription>…</FormDescription>
       <FormMessage />
     </FormItem>
   )} />
   ```
   Text inputs bind `field` directly; non-text controls (Select `value`/`onValueChange`, Switch
   `checked`/`onCheckedChange`, Checkbox grids, `TagsInput` `value`/`onValueChange`) bind via the
   `FormField` render-prop `field.value`/`field.onChange` (RHF `Controller` under the hood).
4. Modals seed defaults with `form.reset(defaults)` on open / when the edited entity arrives
   (the M2a/M3a `useEffect(() => form.reset(...), [initialData, form])` pattern).
5. Replace `FieldError` + manual `field.state.meta` error handling with `<FormMessage />`.

## Schema location

Keep each form's zod schema **colocated** with its component (they already exist inline and are
single-use), passed to `zodResolver`. (Not centralized in `lib/form-validation.ts` — that file is
for the shared, multi-consumer proxy/L4 schemas; moving single-use ACL modal schemas there is
churn without benefit for a lib swap.)

## Preserved exactly

- **Validation rules** — reuse each form's existing zod schema verbatim (regexes, min/max,
  required). **API payloads** — the request objects sent to the `use-acl` hooks are byte-identical.
- **Modal behavior** — open/close, create-vs-edit seeding, success-close, toasts unchanged.
- **Two-tabs-one-record load-merge-save** (oauth-sso + waygates-account): each RHF form holds only
  its slice; `form.reset` seeds from `config` on load; the submit handler spreads the current
  `config` and overlays its slice before calling `useConfigureWaygatesAuth`, so neither tab
  clobbers the other (identical merge logic to M5a, just RHF-driven).
- **Session duration picker** in waygates-account: `session_ttl` stays stored as seconds; the
  number+unit control reads/writes it via the `session-duration.ts` helpers inside a `FormField`
  (`Controller`); 60–604800 clamp + zod bound unchanged.
- **`acl-selector`** stays a controlled `value: ACLAssignment[]` / `onChange` widget — its
  internal "add assignment" inputs become a small standalone RHF form (its own `useForm`); it does
  NOT register into the parent proxy form's RHF context (the proxy form passes ACL assignments
  separately). Public props unchanged.
- **`acl-branding-settings`** live `LoginPreview` keeps updating as the admin edits — driven by
  RHF `form.watch()` instead of the manual `useState` diff; the `hasChanges` dirty state uses RHF
  `formState.isDirty`; `sanitizeCSS` on the custom-CSS field is unchanged.
- **`ProviderRestrictionModal`** keeps its enable/emails/domains fields and its
  `useSetOAuthProviderRestriction` save, now via a small RHF form.

## rnui + RHF gotchas to apply (from prior milestones)

- Number fields (IP-rule `priority`, session seconds, any port) use `z.number()` **not**
  `z.coerce.number()` (coerce makes input≠output and breaks `SubmitHandler` typing); convert at the
  input boundary (`valueAsNumber` / parse in `onChange`). Use `z.number({ error: '…' })` for a
  friendly required message.
- `FormItem` is `display:grid` — add `items-start` in multi-column rows.
- rnui controls use `render` not `asChild`; Switch=`checked`/`onCheckedChange`,
  Select=`value`/`onValueChange`, Checkbox indeterminate is a separate boolean prop.

## Architecture & files

Modify in place (no new files expected beyond possibly extracting a schema constant per form):
`acl-group-form-modal.tsx`, `ip-rules-tab.tsx`, `basic-auth-tab.tsx`, `external-providers-tab.tsx`,
`oauth-sso-tab.tsx` (form + `ProviderRestrictionModal`), `waygates-account-tab.tsx`,
`components/proxy/forms/acl-selector.tsx`, `components/settings/acl-branding-settings.tsx`. No
change to `use-acl.ts`, `types/acl.ts`, routes, or the backend.

## Decomposition

One task per form (8 tasks), each an independent, gate-able migration:
1. `acl-group-form-modal` · 2. `ip-rules-tab` · 3. `basic-auth-tab` · 4. `external-providers-tab`
· 5. `oauth-sso-tab` (+ ProviderRestrictionModal) · 6. `waygates-account-tab` (+ duration picker,
the highest-risk: load-merge-save + Controller-wrapped picker/TagsInput/checkbox grid) ·
7. `acl-selector` · 8. `acl-branding-settings` + final verification that
`grep -rn "@tanstack/react-form" ui/src/components/acl ui/src/components/proxy/forms/acl-selector.tsx ui/src/components/settings/acl-branding-settings.tsx` returns nothing.

## Testing

These are component forms — verification is the gate (`pnpm --dir ui build` + `pnpm --dir ui check`
+ `pnpm --dir ui test`, all green; the existing 57 tests must keep passing). No new unit tests
unless a pure helper is extracted. Because behavior must be unchanged, the reviewer's focus per
task is **parity**: same fields, same validation, same payload, same modal/seed behavior — diffed
against the pre-migration component.

## Risks & notes

- **`waygates-account-tab` + `oauth-sso-tab`** are the riskiest (load-merge-save must survive the
  RHF rewrite; the duration picker + TagsInput + checkbox grids need correct `Controller` wiring).
  Review these against M5a's verified merge logic.
- **`acl-selector` nesting:** keep it an independent controlled widget — do not register it into
  the parent proxy RHF form, or proxy create/edit submission could break. Its own small RHF form
  handles only the add-assignment inputs.
- **`acl-branding-settings` live preview:** ensure `form.watch()` drives the preview without
  excessive re-renders; preserve the `isDirty`-based save-enable and `sanitizeCSS`.
- Parity is the bar: if any validation rule, payload field, or behavior changes, that's a defect —
  this milestone changes the form library only.
