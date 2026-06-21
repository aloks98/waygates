# M2a — HTTP Proxy Forms Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the three HTTP proxy forms (reverse proxy, redirect, static) from `@tanstack/react-form` to `react-hook-form` + `zodResolver`, and redesign them so **create is a guided Stepper wizard** and **edit is a single sectioned form with progressive disclosure**, both rendering the same shared field-group components.

**Architecture:** Each proxy type has one form component taking a `mode: 'create' | 'edit'` prop. The component owns the RHF form (`useForm` + `zodResolver`) and ACL state, wraps everything in `<Form {...form}>`, and renders either a `Stepper` wizard (create) or an `Accordion`/`Collapsible` sectioned layout (edit). Both layouts render the same shared field-group subcomponents, which read form state via `useFormContext()` / `useFieldArray()` / `useWatch()`. All pure value↔request mapping lives in a separate, unit-tested module. Arrays (upstreams, request/response headers, try_files) move from local `useState` to RHF `useFieldArray`. ACL stays a separate controlled component (`acl-selector.tsx`), unchanged.

**Tech Stack:** React 19, `react-hook-form` 7.79, `@hookform/resolvers` 5.4, `zod` 4, `@e412/rnui-react` (Base UI — composes via `render`, NOT `asChild`), TanStack Router, Vitest 4 + React Testing Library.

## Global Constraints

- **rnui composes with `render`, NOT `asChild`.** `<Button render={<Link to="/x" />}>…</Button>`. Base UI silently ignores `asChild`.
- **RHF context hooks come from `react-hook-form` directly** (`useFormContext`, `useFieldArray`, `useWatch`). rnui only re-exports the `Form`/`FormField`/`FormItem`/`FormControl`/`FormLabel`/`FormDescription`/`FormMessage` UI wrappers.
- **rnui `Switch` uses `checked` / `onCheckedChange`** (not `value`/`onChange`): `<Switch checked={field.value} onCheckedChange={field.onChange} />`.
- **Numeric inputs use `z.coerce.number()` in the schema.** `zodResolver` passes the *parsed/coerced* values to the valid-submit handler, so `port` and `status_code` arrive as numbers. Bind `<Input type="number" value={field.value} onChange={field.onChange} />` and `<Select value={String(field.value)} onValueChange={field.onChange} />` — coercion happens at validation.
- **No `tsc` gate.** `vite build` does not type-check (~20 pre-existing type errors). Gate every task on: `pnpm --dir ui build` + `pnpm --dir ui check` (oxlint/oxfmt) + `pnpm --dir ui test run` (vitest). The bundler still errors on missing/renamed exports.
- **`form.reset(...)` in edit mode** when async `initialData` arrives; omit `form` from the effect deps (stable ref).
- **ACL is never in the RHF schema.** It is separate `useState<ACLAssignment[]>`, passed to `onSubmit` as the 2nd argument, exactly as today.
- **Submit shapes (do not change):** reverse → `CreateReverseProxyRequest` (`upstreams`/`load_balancing`/`custom_headers` at top level); redirect → `CreateRedirectRequest` (nested `redirect: {...}`); static → `CreateStaticRequest` (nested `static: {...}`). Page-level `onSubmit`/`handleUpdate`/`handleSubmit` logic is unchanged — only the JSX call sites gain `mode=`.
- **Files NOT to touch:** `acl-selector.tsx`, `use-proxies.ts`, `types/proxy.ts`, the page-level submit/update logic in `new.tsx`/`$proxyId.tsx` (only their form JSX call sites change).
- **Run from repo root** `/home/aloks98/projects/waygates`. UI commands use `pnpm --dir ui …`.

---

## File Structure

**Create:**
- `ui/src/components/proxy/forms/shared/proxy-form-mappers.ts` — pure defaults + value↔request mappers (unit-tested).
- `ui/src/components/proxy/forms/shared/proxy-form-mappers.test.ts` — vitest for the mappers.
- `ui/src/components/proxy/forms/shared/form-section.tsx` — `Collapsible` section wrapper (`open`/`onOpenChange`/`hasError`).
- `ui/src/components/proxy/forms/shared/basics-fields.tsx` — name + hostname + description (all 3 types).
- `ui/src/components/proxy/forms/shared/backend-fields.tsx` — upstreams `useFieldArray` (reverse).
- `ui/src/components/proxy/forms/shared/load-balancing-fields.tsx` — lb_strategy + health check, shown when upstreams > 1 (reverse).
- `ui/src/components/proxy/forms/shared/security-fields.tsx` — ssl_enabled + block_exploits + tls_insecure_skip_verify (reverse).
- `ui/src/components/proxy/forms/shared/custom-headers-fields.tsx` — request/response header `useFieldArray` (reverse).
- `ui/src/components/proxy/forms/shared/redirect-target-fields.tsx` — target + status_code (redirect).
- `ui/src/components/proxy/forms/shared/redirect-options-fields.tsx` — ssl_enabled + preserve_path + preserve_query (redirect).
- `ui/src/components/proxy/forms/shared/static-file-fields.tsx` — root_path + index_file (static).
- `ui/src/components/proxy/forms/shared/static-options-fields.tsx` — ssl_enabled + browse + template_rendering (static).
- `ui/src/components/proxy/forms/shared/try-files-fields.tsx` — try_files `useFieldArray` (static).
- `ui/src/components/proxy/forms/shared/proxy-wizard.tsx` — small shared wizard primitives (stepper nav + actions + review row helpers) reused by all 3 forms.

**Modify:**
- `ui/src/lib/form-validation.ts` — add the 3 HTTP schemas + value types (additive; L4 schemas untouched).
- `ui/src/components/proxy/forms/reverse-proxy-form.tsx` — replace with `mode`-prop implementation.
- `ui/src/components/proxy/forms/redirect-form.tsx` — replace with `mode`-prop implementation.
- `ui/src/components/proxy/forms/static-form.tsx` — replace with `mode`-prop implementation.
- `ui/src/routes/_dashboard/proxies/new.tsx` — pass `mode="create"`.
- `ui/src/routes/_dashboard/proxies/$proxyId.tsx` — pass `mode="edit"`.

---

## Reference: the canonical RHF field pattern (used everywhere)

Every field group renders inside `<Form {...form}>`, so it reads the form from context. The single repeated pattern:

```tsx
import { useFormContext } from 'react-hook-form';
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage, Input } from '@e412/rnui-react';

const form = useFormContext<SomeFormValues>();

<FormField
  control={form.control}
  name="hostname"
  render={({ field }) => (
    <FormItem>
      <FormLabel>Hostname</FormLabel>
      <FormControl>
        <Input placeholder="app.example.com" {...field} />
      </FormControl>
      <FormDescription>The domain visitors will use to reach this service.</FormDescription>
      <FormMessage />
    </FormItem>
  )}
/>
```

Variants:
- **Switch:** `<FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl>`
- **Select:** `<Select value={String(field.value)} onValueChange={field.onChange}><SelectTrigger render inside FormControl>…`
- **Numeric Input:** `<Input type="number" {...field} />` (schema uses `z.coerce.number()`).

> When a task says "port the labels/descriptions/placeholders from the current `<file>`", read that file and reuse its exact copy — only the field plumbing changes (TanStack `Field`/`form.Field` → RHF `FormField`). The current file is the source of truth for copy.

---

## Task 1: HTTP proxy Zod schemas

**Files:**
- Modify: `ui/src/lib/form-validation.ts` (append after the L4 schemas)

**Interfaces:**
- Produces: `upstreamSchema`, `headerPairSchema`, `tryFileSchema`, `reverseProxySchema`, `redirectSchema`, `staticSchema` and value types `ReverseProxyFormValues`, `RedirectFormValues`, `StaticFormValues`, `UpstreamFormValues`, `HeaderPairFormValues`.

- [ ] **Step 1: Append the schemas**

Add to the end of `ui/src/lib/form-validation.ts`:

```ts
// ============================================================================
// HTTP Proxy Validation Schemas (M2a)
// ============================================================================

export const upstreamSchema = z.object({
  host: z.string().min(1, 'Host is required'),
  port: z.coerce
    .number()
    .min(1, 'Port must be at least 1')
    .max(65535, 'Port must be at most 65535'),
  scheme: z.enum(['http', 'https']),
});
export type UpstreamFormValues = z.infer<typeof upstreamSchema>;

export const headerPairSchema = z.object({
  name: z.string(),
  value: z.string(),
});
export type HeaderPairFormValues = z.infer<typeof headerPairSchema>;

// useFieldArray needs object items; try_files (string[]) is wrapped as { value }.
export const tryFileSchema = z.object({ value: z.string() });

export const reverseProxySchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname must be at most 253 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  upstreams: z.array(upstreamSchema).min(1, 'Add at least one backend server'),
  ssl_enabled: z.boolean(),
  block_exploits: z.boolean(),
  tls_insecure_skip_verify: z.boolean(),
  lb_strategy: z.enum(['round_robin', 'least_conn', 'ip_hash', 'random']),
  health_check_enabled: z.boolean(),
  health_check_path: z.string(),
  health_check_interval: z.string(),
  health_check_timeout: z.string(),
  request_headers: z.array(headerPairSchema),
  response_headers: z.array(headerPairSchema),
});
export type ReverseProxyFormValues = z.infer<typeof reverseProxySchema>;

export const redirectSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname must be at most 253 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  ssl_enabled: z.boolean(),
  target: z.string().min(1, 'Target URL is required').url('Target must be a valid URL'),
  status_code: z.coerce.number().refine((val) => [301, 302, 307, 308].includes(val), {
    message: 'Status code must be 301, 302, 307, or 308',
  }),
  preserve_path: z.boolean(),
  preserve_query: z.boolean(),
});
export type RedirectFormValues = z.infer<typeof redirectSchema>;

export const staticSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname must be at most 253 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  ssl_enabled: z.boolean(),
  root_path: z.string().min(1, 'Root path is required'),
  index_file: z.string().min(1, 'Index file is required'),
  browse: z.boolean(),
  template_rendering: z.boolean(),
  try_files: z.array(tryFileSchema),
});
export type StaticFormValues = z.infer<typeof staticSchema>;
```

- [ ] **Step 2: Verify build + lint**

Run: `pnpm --dir ui build && pnpm --dir ui check`
Expected: build succeeds, no lint/format errors. (No runtime usage yet; this confirms the schemas compile and are exported.)

- [ ] **Step 3: Commit**

```bash
git add ui/src/lib/form-validation.ts
git commit -m "feat(ui): add RHF HTTP proxy zod schemas (M2a)"
```

---

## Task 2: Pure form mappers + unit tests

**Files:**
- Create: `ui/src/components/proxy/forms/shared/proxy-form-mappers.ts`
- Test: `ui/src/components/proxy/forms/shared/proxy-form-mappers.test.ts`

**Interfaces:**
- Consumes: value types from Task 1; `ProxyConfig`, `CreateReverseProxyRequest`, `CreateRedirectRequest`, `CreateStaticRequest` from `@/types/proxy`.
- Produces:
  - `REVERSE_PROXY_DEFAULTS: ReverseProxyFormValues`, `REDIRECT_DEFAULTS: RedirectFormValues`, `STATIC_DEFAULTS: StaticFormValues`
  - `mapProxyToReverseDefaults(data: ProxyConfig): ReverseProxyFormValues`
  - `mapReverseValuesToRequest(values: ReverseProxyFormValues): CreateReverseProxyRequest`
  - `mapProxyToRedirectDefaults(data: ProxyConfig): RedirectFormValues`
  - `mapRedirectValuesToRequest(values: RedirectFormValues): CreateRedirectRequest`
  - `mapProxyToStaticDefaults(data: ProxyConfig): StaticFormValues`
  - `mapStaticValuesToRequest(values: StaticFormValues): CreateStaticRequest`

- [ ] **Step 1: Write the failing tests**

Create `ui/src/components/proxy/forms/shared/proxy-form-mappers.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import type { ProxyConfig } from '@/types/proxy';

import {
  mapProxyToRedirectDefaults,
  mapProxyToReverseDefaults,
  mapProxyToStaticDefaults,
  mapRedirectValuesToRequest,
  mapReverseValuesToRequest,
  mapStaticValuesToRequest,
  REVERSE_PROXY_DEFAULTS,
} from './proxy-form-mappers';

const baseProxy = (over: Partial<ProxyConfig>): ProxyConfig => ({
  id: 1,
  type: 'reverse_proxy',
  name: 'svc',
  hostname: 'svc.example.com',
  ssl_enabled: true,
  ssl_forced: false,
  is_active: true,
  created_at: '',
  updated_at: '',
  ...over,
});

describe('reverse proxy mappers', () => {
  it('defaults seed one empty upstream and HTTPS on', () => {
    expect(REVERSE_PROXY_DEFAULTS.upstreams).toHaveLength(1);
    expect(REVERSE_PROXY_DEFAULTS.ssl_enabled).toBe(true);
    expect(REVERSE_PROXY_DEFAULTS.request_headers).toEqual([]);
  });

  it('maps a proxy with multiple upstreams + headers to defaults', () => {
    const d = mapProxyToReverseDefaults(
      baseProxy({
        upstreams: [
          { host: 'a', port: 80, scheme: 'http' },
          { host: 'b', port: 443, scheme: 'https' },
        ],
        load_balancing: {
          strategy: 'least_conn',
          health_checks: {
            enabled: true,
            path: '/up',
            interval: '10s',
            timeout: '2s',
            unhealthy_threshold: 3,
            healthy_threshold: 2,
          },
        },
        custom_headers: { request: { 'X-A': '1' }, response: { 'X-B': '2' } },
      }),
    );
    expect(d.upstreams).toHaveLength(2);
    expect(d.lb_strategy).toBe('least_conn');
    expect(d.health_check_enabled).toBe(true);
    expect(d.health_check_path).toBe('/up');
    expect(d.request_headers).toEqual([{ name: 'X-A', value: '1' }]);
    expect(d.response_headers).toEqual([{ name: 'X-B', value: '2' }]);
  });

  it('omits load_balancing + custom_headers when single upstream and no headers', () => {
    const req = mapReverseValuesToRequest({
      ...REVERSE_PROXY_DEFAULTS,
      name: 'svc',
      hostname: 'svc.example.com',
      upstreams: [{ host: 'a', port: 80, scheme: 'http' }],
    });
    expect(req.type).toBe('reverse_proxy');
    expect(req.load_balancing).toBeUndefined();
    expect(req.custom_headers).toBeUndefined();
  });

  it('includes load_balancing with health_checks when >1 upstream', () => {
    const req = mapReverseValuesToRequest({
      ...REVERSE_PROXY_DEFAULTS,
      name: 'svc',
      hostname: 'svc.example.com',
      upstreams: [
        { host: 'a', port: 80, scheme: 'http' },
        { host: 'b', port: 81, scheme: 'http' },
      ],
      lb_strategy: 'ip_hash',
      health_check_enabled: true,
      health_check_path: '/h',
      health_check_interval: '15s',
      health_check_timeout: '3s',
    });
    expect(req.load_balancing?.strategy).toBe('ip_hash');
    expect(req.load_balancing?.health_checks?.path).toBe('/h');
  });

  it('drops empty/blank header names', () => {
    const req = mapReverseValuesToRequest({
      ...REVERSE_PROXY_DEFAULTS,
      name: 'svc',
      hostname: 'svc.example.com',
      request_headers: [
        { name: ' ', value: 'x' },
        { name: 'X-Keep', value: 'y' },
      ],
    });
    expect(req.custom_headers?.request).toEqual({ 'X-Keep': 'y' });
  });
});

describe('redirect mappers', () => {
  it('round-trips nested redirect config', () => {
    const d = mapProxyToRedirectDefaults(
      baseProxy({
        type: 'redirect',
        redirect: { target: 'https://x', status_code: 308, preserve_path: true, preserve_query: false },
      }),
    );
    expect(d.target).toBe('https://x');
    expect(d.status_code).toBe(308);
    const req = mapRedirectValuesToRequest(d);
    expect(req.type).toBe('redirect');
    expect(req.redirect.status_code).toBe(308);
    expect(req.redirect.preserve_path).toBe(true);
  });
});

describe('static mappers', () => {
  it('wraps/unwraps try_files and drops blanks', () => {
    const d = mapProxyToStaticDefaults(
      baseProxy({
        type: 'static',
        static: {
          root_path: '/srv',
          index_file: 'index.html',
          browse: true,
          template_rendering: false,
          try_files: ['{path}', 'index.html'],
        },
      }),
    );
    expect(d.try_files).toEqual([{ value: '{path}' }, { value: 'index.html' }]);
    const req = mapStaticValuesToRequest({ ...d, try_files: [{ value: '{path}' }, { value: '' }] });
    expect(req.type).toBe('static');
    expect(req.static.try_files).toEqual(['{path}']);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `pnpm --dir ui test run proxy-form-mappers`
Expected: FAIL — module `./proxy-form-mappers` not found.

- [ ] **Step 3: Implement the mappers**

Create `ui/src/components/proxy/forms/shared/proxy-form-mappers.ts`:

```ts
import type {
  RedirectFormValues,
  ReverseProxyFormValues,
  StaticFormValues,
} from '@/lib/form-validation';
import type {
  CreateRedirectRequest,
  CreateReverseProxyRequest,
  CreateStaticRequest,
  HealthCheck,
  ProxyConfig,
} from '@/types/proxy';

// ---------- shared helpers ----------

function normalizeScheme(scheme: string | undefined): 'http' | 'https' {
  return String(scheme ?? '').toLowerCase() === 'https' ? 'https' : 'http';
}

function recordToPairs(rec?: Record<string, string>): { name: string; value: string }[] {
  return Object.entries(rec ?? {}).map(([name, value]) => ({ name, value }));
}

function pairsToRecord(pairs: { name: string; value: string }[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of pairs) {
    const name = p.name.trim();
    if (name) out[name] = p.value;
  }
  return out;
}

// ---------- reverse proxy ----------

export const REVERSE_PROXY_DEFAULTS: ReverseProxyFormValues = {
  name: '',
  hostname: '',
  description: '',
  upstreams: [{ host: '', port: 8080, scheme: 'http' }],
  ssl_enabled: true,
  block_exploits: true,
  tls_insecure_skip_verify: false,
  lb_strategy: 'round_robin',
  health_check_enabled: false,
  health_check_path: '/health',
  health_check_interval: '30s',
  health_check_timeout: '5s',
  request_headers: [],
  response_headers: [],
};

export function mapProxyToReverseDefaults(data: ProxyConfig): ReverseProxyFormValues {
  const hc = data.load_balancing?.health_checks;
  return {
    name: data.name,
    hostname: data.hostname,
    description: data.description ?? '',
    upstreams:
      data.upstreams?.length
        ? data.upstreams.map((u) => ({
            host: u.host || '',
            port: u.port || 8080,
            scheme: normalizeScheme(u.scheme),
          }))
        : [{ host: '', port: 8080, scheme: 'http' }],
    ssl_enabled: data.ssl_enabled ?? true,
    block_exploits: data.block_exploits ?? true,
    tls_insecure_skip_verify: data.tls_insecure_skip_verify ?? false,
    lb_strategy: data.load_balancing?.strategy ?? 'round_robin',
    health_check_enabled: hc?.enabled ?? false,
    health_check_path: hc?.path ?? '/health',
    health_check_interval: hc?.interval ?? '30s',
    health_check_timeout: hc?.timeout ?? '5s',
    request_headers: recordToPairs(data.custom_headers?.request),
    response_headers: recordToPairs(data.custom_headers?.response),
  };
}

export function mapReverseValuesToRequest(
  values: ReverseProxyFormValues,
): CreateReverseProxyRequest {
  const request = pairsToRecord(values.request_headers);
  const response = pairsToRecord(values.response_headers);
  const hasHeaders = Object.keys(request).length > 0 || Object.keys(response).length > 0;
  const multiUpstream = values.upstreams.length > 1;

  const healthChecks: HealthCheck | undefined = values.health_check_enabled
    ? {
        enabled: true,
        path: values.health_check_path,
        interval: values.health_check_interval,
        timeout: values.health_check_timeout,
        unhealthy_threshold: 3,
        healthy_threshold: 2,
      }
    : undefined;

  return {
    type: 'reverse_proxy',
    name: values.name,
    hostname: values.hostname,
    description: values.description || undefined,
    ssl_enabled: values.ssl_enabled,
    upstreams: values.upstreams,
    block_exploits: values.block_exploits,
    tls_insecure_skip_verify: values.tls_insecure_skip_verify,
    ...(multiUpstream
      ? { load_balancing: { strategy: values.lb_strategy, health_checks: healthChecks } }
      : {}),
    ...(hasHeaders
      ? {
          custom_headers: {
            ...(Object.keys(request).length ? { request } : {}),
            ...(Object.keys(response).length ? { response } : {}),
          },
        }
      : {}),
  };
}

// ---------- redirect ----------

export const REDIRECT_DEFAULTS: RedirectFormValues = {
  name: '',
  hostname: '',
  description: '',
  ssl_enabled: true,
  target: '',
  status_code: 301,
  preserve_path: true,
  preserve_query: true,
};

export function mapProxyToRedirectDefaults(data: ProxyConfig): RedirectFormValues {
  const r = data.redirect;
  return {
    name: data.name,
    hostname: data.hostname,
    description: data.description ?? '',
    ssl_enabled: data.ssl_enabled ?? true,
    target: r?.target ?? '',
    status_code: r?.status_code ?? 301,
    preserve_path: r?.preserve_path ?? true,
    preserve_query: r?.preserve_query ?? true,
  };
}

export function mapRedirectValuesToRequest(values: RedirectFormValues): CreateRedirectRequest {
  return {
    type: 'redirect',
    name: values.name,
    hostname: values.hostname,
    description: values.description || undefined,
    ssl_enabled: values.ssl_enabled,
    redirect: {
      target: values.target,
      status_code: values.status_code as 301 | 302 | 307 | 308,
      preserve_path: values.preserve_path,
      preserve_query: values.preserve_query,
    },
  };
}

// ---------- static ----------

export const STATIC_DEFAULTS: StaticFormValues = {
  name: '',
  hostname: '',
  description: '',
  ssl_enabled: true,
  root_path: '/var/www/html',
  index_file: 'index.html',
  browse: false,
  template_rendering: false,
  try_files: [],
};

export function mapProxyToStaticDefaults(data: ProxyConfig): StaticFormValues {
  const s = data.static;
  return {
    name: data.name,
    hostname: data.hostname,
    description: data.description ?? '',
    ssl_enabled: data.ssl_enabled ?? true,
    root_path: s?.root_path ?? '/var/www/html',
    index_file: s?.index_file ?? 'index.html',
    browse: s?.browse ?? false,
    template_rendering: s?.template_rendering ?? false,
    try_files: (s?.try_files ?? []).map((value) => ({ value })),
  };
}

export function mapStaticValuesToRequest(values: StaticFormValues): CreateStaticRequest {
  return {
    type: 'static',
    name: values.name,
    hostname: values.hostname,
    description: values.description || undefined,
    ssl_enabled: values.ssl_enabled,
    static: {
      root_path: values.root_path,
      index_file: values.index_file,
      browse: values.browse,
      template_rendering: values.template_rendering,
      try_files: values.try_files.map((f) => f.value.trim()).filter(Boolean),
    },
  };
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `pnpm --dir ui test run proxy-form-mappers`
Expected: PASS (all cases green).

- [ ] **Step 5: Verify build + lint**

Run: `pnpm --dir ui build && pnpm --dir ui check`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add ui/src/components/proxy/forms/shared/proxy-form-mappers.ts ui/src/components/proxy/forms/shared/proxy-form-mappers.test.ts
git commit -m "feat(ui): add tested proxy form mappers (M2a)"
```

---

## Task 3: FormSection wrapper + shared wizard primitives

**Files:**
- Create: `ui/src/components/proxy/forms/shared/form-section.tsx`
- Create: `ui/src/components/proxy/forms/shared/proxy-wizard.tsx`

**Interfaces:**
- Produces:
  - `FormSection({ title, description?, hasError?, open, onOpenChange, children })`
  - `WizardStepNav({ steps, activeStep, completedSteps, onStepClick })` where `steps: { step: number; title: string }[]`, `completedSteps: Set<number>`
  - `WizardActions({ activeStep, lastStep, onBack, onNext, onCancel, submitting, submitLabel })` — renders Back/Cancel + (Next or submit). On the last step the primary button is `type="submit"` with `submitLabel`; otherwise it is a `type="button"` "Continue" calling `onNext`.
  - `ReviewSection({ title, children })`, `ReviewRow({ label, value })`

- [ ] **Step 1: Implement `form-section.tsx`**

```tsx
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@e412/rnui-react';
import { ChevronDown } from 'lucide-react';
import type { ReactNode } from 'react';

interface FormSectionProps {
  title: string;
  description?: string;
  hasError?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}

export function FormSection({
  title,
  description,
  hasError,
  open,
  onOpenChange,
  children,
}: FormSectionProps) {
  return (
    <Collapsible open={open} onOpenChange={onOpenChange} className="rounded-lg border">
      <CollapsibleTrigger className="flex w-full items-center justify-between gap-2 px-4 py-3 text-left">
        <div className="flex items-center gap-2">
          <span className="font-medium">{title}</span>
          {hasError && (
            <span
              aria-label="Section has errors"
              className="inline-block size-2 rounded-full bg-destructive"
            />
          )}
        </div>
        <ChevronDown
          className={`size-4 text-muted-foreground transition-transform ${open ? 'rotate-180' : ''}`}
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="space-y-4 px-4 pb-4">
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
          {children}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
```

- [ ] **Step 2: Implement `proxy-wizard.tsx`**

```tsx
import {
  Button,
  Stepper,
  StepperIndicator,
  StepperItem,
  StepperNav,
  StepperSeparator,
  StepperTitle,
  StepperTrigger,
} from '@e412/rnui-react';
import { Check } from 'lucide-react';
import type { ReactNode } from 'react';

export interface WizardStep {
  step: number;
  title: string;
}

export function WizardStepNav({
  steps,
  activeStep,
  completedSteps,
  onStepClick,
}: {
  steps: WizardStep[];
  activeStep: number;
  completedSteps: Set<number>;
  onStepClick: (step: number) => void;
}) {
  return (
    <Stepper value={activeStep} onValueChange={onStepClick}>
      <StepperNav>
        {steps.map((s, i) => (
          <StepperItem
            key={s.step}
            step={s.step}
            completed={completedSteps.has(s.step)}
            disabled={s.step > activeStep && !completedSteps.has(s.step)}
          >
            <StepperTrigger>
              <StepperIndicator>
                {completedSteps.has(s.step) ? <Check className="size-4" /> : s.step}
              </StepperIndicator>
              <StepperTitle className="hidden sm:block">{s.title}</StepperTitle>
            </StepperTrigger>
            {i < steps.length - 1 && <StepperSeparator />}
          </StepperItem>
        ))}
      </StepperNav>
    </Stepper>
  );
}

export function WizardActions({
  activeStep,
  lastStep,
  onBack,
  onNext,
  onCancel,
  submitting,
  submitLabel,
}: {
  activeStep: number;
  lastStep: number;
  onBack: () => void;
  onNext: () => void;
  onCancel: () => void;
  submitting: boolean;
  submitLabel: string;
}) {
  const isLast = activeStep === lastStep;
  return (
    <div className="flex items-center justify-between">
      <Button type="button" variant="ghost" onClick={onCancel} disabled={submitting}>
        Cancel
      </Button>
      <div className="flex items-center gap-2">
        {activeStep > 1 && (
          <Button type="button" variant="outline" onClick={onBack} disabled={submitting}>
            Back
          </Button>
        )}
        {isLast ? (
          <Button type="submit" disabled={submitting}>
            {submitting ? 'Saving…' : submitLabel}
          </Button>
        ) : (
          <Button type="button" onClick={onNext}>
            Continue
          </Button>
        )}
      </div>
    </div>
  );
}

export function ReviewSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <h4 className="text-sm font-medium text-muted-foreground">{title}</h4>
      <dl className="divide-y rounded-lg border">{children}</dl>
    </div>
  );
}

export function ReviewRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 px-4 py-2 text-sm">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}
```

> If any imported name (`Stepper*`, `Collapsible*`) is not exported by `@e412/rnui-react`, the build will fail — confirm exact export names against `node_modules/@e412/rnui-react/dist/index.d.ts` and fix imports.

- [ ] **Step 3: Verify build + lint**

Run: `pnpm --dir ui build && pnpm --dir ui check`
Expected: clean (components compile; unused-export warnings are acceptable until consumed).

- [ ] **Step 4: Commit**

```bash
git add ui/src/components/proxy/forms/shared/form-section.tsx ui/src/components/proxy/forms/shared/proxy-wizard.tsx
git commit -m "feat(ui): add FormSection + wizard primitives (M2a)"
```

---

## Task 4: Shared field groups — reverse proxy set

**Files:**
- Create: `ui/src/components/proxy/forms/shared/basics-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/backend-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/load-balancing-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/security-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/custom-headers-fields.tsx`

**Interfaces:**
- Consumes: `ReverseProxyFormValues` from `@/lib/form-validation`; canonical RHF field pattern (top of plan).
- Produces: `BasicsFields({ autoFocusName? })`, `BackendFields()`, `LoadBalancingFields()`, `SecurityFields()`, `CustomHeadersFields()`. Each reads the form via `useFormContext()`; no value props.

> Port the exact labels, descriptions, placeholders, and select option copy from the current `ui/src/components/proxy/forms/reverse-proxy-form.tsx` (read it). Only the plumbing changes (`form.Field`/`Field*` → RHF `FormField`/`FormItem`/`FormControl`/`FormLabel`/`FormDescription`/`FormMessage`).

- [ ] **Step 1: `basics-fields.tsx`** — `name` + `hostname` in a `grid sm:grid-cols-2`, `description` full width. Generic over the three value types (it only touches shared fields), so type it loosely:

```tsx
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage, Input } from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

export function BasicsFields({ autoFocusName = false }: { autoFocusName?: boolean }) {
  // BasicsFields only touches name/hostname/description, present on all 3 schemas.
  const form = useFormContext();
  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input autoFocus={autoFocusName} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="hostname"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Hostname</FormLabel>
              <FormControl>
                <Input placeholder="app.example.com" {...field} />
              </FormControl>
              <FormDescription>The domain visitors will use to reach this service.</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
      <FormField
        control={form.control}
        name="description"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Description</FormLabel>
            <FormControl>
              <Input {...field} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
```

- [ ] **Step 2: `backend-fields.tsx`** — upstreams `useFieldArray` (scheme Select / host Input / port number Input / remove button per row; "Add Server" appends `{ host: '', port: 8080, scheme: 'http' }`; an array-level `FormMessage` for the `.min(1)` error):

```tsx
import {
  Button,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@e412/rnui-react';
import { Plus, Trash2 } from 'lucide-react';
import { useFieldArray, useFormContext } from 'react-hook-form';

import type { ReverseProxyFormValues } from '@/lib/form-validation';

export function BackendFields() {
  const form = useFormContext<ReverseProxyFormValues>();
  const { fields, append, remove } = useFieldArray({ control: form.control, name: 'upstreams' });

  return (
    <div className="space-y-3">
      {fields.map((item, index) => (
        <div key={item.id} className="flex items-start gap-2">
          <FormField
            control={form.control}
            name={`upstreams.${index}.scheme`}
            render={({ field }) => (
              <FormItem className="w-28">
                <FormControl>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">http</SelectItem>
                      <SelectItem value="https">https</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`upstreams.${index}.host`}
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormControl>
                  <Input placeholder="10.0.0.5 or backend.internal" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`upstreams.${index}.port`}
            render={({ field }) => (
              <FormItem className="w-28">
                <FormControl>
                  <Input type="number" placeholder="8080" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          {fields.length > 1 && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Remove server"
              onClick={() => remove(index)}
            >
              <Trash2 className="size-4" />
            </Button>
          )}
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => append({ host: '', port: 8080, scheme: 'http' })}
      >
        <Plus className="size-4" /> Add Server
      </Button>
      <FormField
        control={form.control}
        name="upstreams"
        render={() => (
          <FormItem>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
```

- [ ] **Step 3: `load-balancing-fields.tsx`** — read `upstreams` + `health_check_enabled` via `useWatch`; `if (upstreams.length <= 1) return null;`. Render `lb_strategy` Select (4 options w/ the descriptions from the current file), a `health_check_enabled` Switch, and — when enabled — `health_check_path` / `health_check_interval` / `health_check_timeout` Inputs. Use the canonical FormField + Switch + Select patterns.

- [ ] **Step 4: `security-fields.tsx`** — three Switch rows: `ssl_enabled` ("Enable HTTPS"), `block_exploits` ("Block Common Exploits"), `tls_insecure_skip_verify` ("Allow Self-Signed Certificates"). Reuse the descriptions from the current file.

- [ ] **Step 5: `custom-headers-fields.tsx`** — two `useFieldArray` instances (`request_headers`, `response_headers`); per section a label + "Add Header" button, then rows of `name`/`value` Inputs + remove. Append `{ name: '', value: '' }`.

- [ ] **Step 6: Verify build + lint**

Run: `pnpm --dir ui build && pnpm --dir ui check`
Expected: clean (groups compile; not yet consumed).

- [ ] **Step 7: Commit**

```bash
git add ui/src/components/proxy/forms/shared/basics-fields.tsx ui/src/components/proxy/forms/shared/backend-fields.tsx ui/src/components/proxy/forms/shared/load-balancing-fields.tsx ui/src/components/proxy/forms/shared/security-fields.tsx ui/src/components/proxy/forms/shared/custom-headers-fields.tsx
git commit -m "feat(ui): add reverse-proxy shared field groups (M2a)"
```

---

## Task 5: Shared field groups — redirect + static sets

**Files:**
- Create: `ui/src/components/proxy/forms/shared/redirect-target-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/redirect-options-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/static-file-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/static-options-fields.tsx`
- Create: `ui/src/components/proxy/forms/shared/try-files-fields.tsx`

**Interfaces:**
- Consumes: `RedirectFormValues`, `StaticFormValues`.
- Produces: `RedirectTargetFields()`, `RedirectOptionsFields()`, `StaticFileFields()`, `StaticOptionsFields()`, `TryFilesFields()`.

> Port labels/descriptions/help text from the current `redirect-form.tsx` and `static-form.tsx`.

- [ ] **Step 1: `redirect-target-fields.tsx`** — `target` Input (URL) + `status_code` Select. Status code select: bind `value={String(field.value)} onValueChange={field.onChange}` with options 301/302/307/308 (use the current file's labels, e.g. "301 — Permanent"); schema coerces back to number.

- [ ] **Step 2: `redirect-options-fields.tsx`** — Switch rows: `ssl_enabled`, `preserve_path`, `preserve_query` (descriptions from current file).

- [ ] **Step 3: `static-file-fields.tsx`** — `root_path` Input + `index_file` Input (descriptions from current file).

- [ ] **Step 4: `static-options-fields.tsx`** — Switch rows: `ssl_enabled`, `browse`, `template_rendering`.

- [ ] **Step 5: `try-files-fields.tsx`** — `useFieldArray({ name: 'try_files' })`; rows bind `name={`try_files.${index}.value`}` to a text Input + remove; "Add File" appends `{ value: '' }`; keep the `{path}` help text from the current file; show the "No try_files configured" empty hint when `fields.length === 0`.

```tsx
import { Button, FormControl, FormField, FormItem, FormMessage, Input } from '@e412/rnui-react';
import { Plus, Trash2 } from 'lucide-react';
import { useFieldArray, useFormContext } from 'react-hook-form';

import type { StaticFormValues } from '@/lib/form-validation';

export function TryFilesFields() {
  const form = useFormContext<StaticFormValues>();
  const { fields, append, remove } = useFieldArray({ control: form.control, name: 'try_files' });

  return (
    <div className="space-y-3">
      {fields.length === 0 && (
        <p className="text-sm text-muted-foreground">No try_files configured.</p>
      )}
      {fields.map((item, index) => (
        <div key={item.id} className="flex items-start gap-2">
          <FormField
            control={form.control}
            name={`try_files.${index}.value`}
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormControl>
                  <Input placeholder="{path}" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Remove file"
            onClick={() => remove(index)}
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={() => append({ value: '' })}>
        <Plus className="size-4" /> Add File
      </Button>
    </div>
  );
}
```

- [ ] **Step 6: Verify build + lint**

Run: `pnpm --dir ui build && pnpm --dir ui check`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add ui/src/components/proxy/forms/shared/redirect-target-fields.tsx ui/src/components/proxy/forms/shared/redirect-options-fields.tsx ui/src/components/proxy/forms/shared/static-file-fields.tsx ui/src/components/proxy/forms/shared/static-options-fields.tsx ui/src/components/proxy/forms/shared/try-files-fields.tsx
git commit -m "feat(ui): add redirect + static shared field groups (M2a)"
```

---

## Task 6: ReverseProxyForm — mode-prop wizard + sectioned edit

**Files:**
- Modify (replace): `ui/src/components/proxy/forms/reverse-proxy-form.tsx`

**Interfaces:**
- Consumes: Task 1 schema (`reverseProxySchema`, `ReverseProxyFormValues`), Task 2 mappers (`REVERSE_PROXY_DEFAULTS`, `mapProxyToReverseDefaults`, `mapReverseValuesToRequest`), Task 3 (`FormSection`, `WizardStepNav`, `WizardActions`, `ReviewSection`, `ReviewRow`), Task 4 field groups, `ACLSelector`/`ACLAssignment` from `./acl-selector`.
- Produces: `ReverseProxyForm({ mode, initialData?, initialACLAssignments?, onSubmit, loading, onCancel })` where `onSubmit: (data: CreateReverseProxyRequest, acl?: ACLAssignment[]) => void` (unchanged from today).

**Wizard steps (reverse):** `1 Basics` → `2 Backend` → `3 Security & Load Balancing` → `4 Custom Headers` → `5 Access Control` → `6 Review`.

**Step → fields for `form.trigger` validation:**
```ts
const STEP_FIELDS: Record<number, (keyof ReverseProxyFormValues)[]> = {
  1: ['name', 'hostname', 'description'],
  2: ['upstreams'],
  3: ['ssl_enabled', 'block_exploits', 'tls_insecure_skip_verify', 'lb_strategy', 'health_check_enabled', 'health_check_path', 'health_check_interval', 'health_check_timeout'],
  4: ['request_headers', 'response_headers'],
  5: [],
};
```

- [ ] **Step 1: Write the component**

Structure (fill the field-group placements; reuse the patterns above):

```tsx
import { Card, CardContent, CardHeader, CardTitle, Form } from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { type FieldErrors, useForm, useFormContext } from 'react-hook-form';

import { type ReverseProxyFormValues, reverseProxySchema } from '@/lib/form-validation';
import type { CreateReverseProxyRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment, ACLSelector } from './acl-selector';
import { BasicsFields } from './shared/basics-fields';
import { BackendFields } from './shared/backend-fields';
import { CustomHeadersFields } from './shared/custom-headers-fields';
import { FormSection } from './shared/form-section';
import { LoadBalancingFields } from './shared/load-balancing-fields';
import {
  mapProxyToReverseDefaults,
  mapReverseValuesToRequest,
  REVERSE_PROXY_DEFAULTS,
} from './shared/proxy-form-mappers';
import { ReviewRow, ReviewSection, WizardActions, WizardStepNav } from './shared/proxy-wizard';
import { SecurityFields } from './shared/security-fields';

interface ReverseProxyFormProps {
  mode: 'create' | 'edit';
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateReverseProxyRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
}

const WIZARD_STEPS = [
  { step: 1, title: 'Basics' },
  { step: 2, title: 'Backend' },
  { step: 3, title: 'Security' },
  { step: 4, title: 'Headers' },
  { step: 5, title: 'Access' },
  { step: 6, title: 'Review' },
];

const STEP_FIELDS: Record<number, (keyof ReverseProxyFormValues)[]> = {
  1: ['name', 'hostname', 'description'],
  2: ['upstreams'],
  3: ['ssl_enabled', 'block_exploits', 'tls_insecure_skip_verify', 'lb_strategy', 'health_check_enabled', 'health_check_path', 'health_check_interval', 'health_check_timeout'],
  4: ['request_headers', 'response_headers'],
  5: [],
};

export function ReverseProxyForm({
  mode,
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
}: ReverseProxyFormProps) {
  const [aclAssignments, setAclAssignments] = useState<ACLAssignment[]>(
    initialACLAssignments ?? [],
  );

  const form = useForm<ReverseProxyFormValues>({
    resolver: zodResolver(reverseProxySchema),
    mode: 'onTouched',
    defaultValues:
      mode === 'edit' && initialData ? mapProxyToReverseDefaults(initialData) : REVERSE_PROXY_DEFAULTS,
  });

  // ACL arrives async on edit
  useEffect(() => {
    if (initialACLAssignments) setAclAssignments(initialACLAssignments);
  }, [initialACLAssignments]);

  // Proxy data arrives async on edit
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (mode === 'edit' && initialData) form.reset(mapProxyToReverseDefaults(initialData));
  }, [initialData]);

  const submit = (values: ReverseProxyFormValues) => {
    onSubmit(mapReverseValuesToRequest(values), aclAssignments.length ? aclAssignments : undefined);
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(submit)} className="space-y-6">
        {mode === 'create' ? (
          <ReverseWizard
            acl={aclAssignments}
            onAclChange={setAclAssignments}
            loading={loading}
            onCancel={onCancel}
          />
        ) : (
          <ReverseEdit
            acl={aclAssignments}
            onAclChange={setAclAssignments}
            loading={loading}
            onCancel={onCancel}
          />
        )}
      </form>
    </Form>
  );
}
```

- [ ] **Step 2: Implement `ReverseWizard`** (private, same file). State: `activeStep`, `completedSteps: Set<number>`. `advance()` runs `await form.trigger(STEP_FIELDS[activeStep] ?? [])`; on success add to `completedSteps` and `setActiveStep(s => s + 1)`. `goTo(step)` allows back (`step < activeStep`) or forward into an already-completed step. Render `<WizardStepNav steps={WIZARD_STEPS} … onStepClick={goTo} />`, then a `Card` whose content switches on `activeStep`:
  - 1 → `<BasicsFields autoFocusName />`
  - 2 → `<BackendFields />`
  - 3 → `<SecurityFields />` + `<LoadBalancingFields />`
  - 4 → `<CustomHeadersFields />`
  - 5 → `<ACLSelector value={acl} onChange={onAclChange} disabled={loading} />`
  - 6 → `<ReverseReview acl={acl} />`
  - then `<WizardActions activeStep={activeStep} lastStep={6} onBack={() => setActiveStep(s => s - 1)} onNext={advance} onCancel={onCancel} submitting={loading} submitLabel="Create Proxy" />`.

  Use `useFormContext<ReverseProxyFormValues>()` inside the wizard to access `form.trigger`/`getValues`.

- [ ] **Step 3: Implement `ReverseReview`** — read `form.getValues()`; render `ReviewSection`/`ReviewRow` summarizing Basics, Backend (one row per upstream `scheme://host:port`), Security (HTTPS/exploits/self-signed yes/no), Load balancing (only if >1 upstream), Headers count, and Access (assignment count) when `acl.length`.

- [ ] **Step 4: Implement `ReverseEdit`** — `openSections` state `{ backend: true, security: false, headers: false }`. The form's submit is wrapped so invalid submits expand offending sections. Because the outer `<form onSubmit={form.handleSubmit(submit)}>` only fires on valid, add an `onInvalid` handler via a second handleSubmit bound to the Save button is unnecessary — instead, render the Save button as `type="submit"` and ALSO pass an invalid handler by replacing the outer handler with `form.handleSubmit(submit, onInvalid)` is owned by the parent. To keep the expand logic local, in `ReverseEdit` use `form.formState.errors` after a submit attempt:

  Simpler, deterministic approach — define the invalid handler in the **parent** and pass section state down. Restructure: lift `openSections` into the parent component and pass `openSections`/`setOpenSections` to `ReverseEdit`, and change the parent's form element to:

```tsx
const onInvalid = (errors: FieldErrors<ReverseProxyFormValues>) => {
  setOpenSections((prev) => ({
    backend: prev.backend || !!errors.upstreams,
    security:
      prev.security ||
      !!(errors.ssl_enabled || errors.block_exploits || errors.tls_insecure_skip_verify || errors.lb_strategy || errors.health_check_path || errors.health_check_interval || errors.health_check_timeout),
    headers: prev.headers || !!(errors.request_headers || errors.response_headers),
  }));
};
// ...
<form onSubmit={form.handleSubmit(submit, onInvalid)} className="space-y-6">
```

  Guard `onInvalid` so it is only meaningful in edit mode (in create mode `openSections` is unused). `ReverseEdit` renders:
  - `<Card><CardHeader><CardTitle>Basics</CardTitle></CardHeader><CardContent><BasicsFields /></CardContent></Card>`
  - `<FormSection title="Backend Servers" open={openSections.backend} onOpenChange={…} hasError={!!form.formState.errors.upstreams}><BackendFields /></FormSection>`
  - `<FormSection title="Security & Load Balancing" open={openSections.security} onOpenChange={…} hasError={…}><SecurityFields /><LoadBalancingFields /></FormSection>`
  - `<FormSection title="Custom Headers" open={openSections.headers} onOpenChange={…}><CustomHeadersFields /></FormSection>`
  - `<ACLSelector value={acl} onChange={onAclChange} disabled={loading} />`
  - actions row: `<Button variant="outline" onClick={onCancel}>Cancel</Button><Button type="submit" disabled={loading}>{loading ? 'Saving…' : 'Save Changes'}</Button>`

- [ ] **Step 5: Verify build + lint**

Run: `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run`
Expected: clean; mapper tests still pass.

- [ ] **Step 6: Manual verification**

Run `pnpm --dir ui dev` (port 8008). Verify: `/dashboard/proxies/new?type=reverse_proxy` shows the wizard; cannot advance past Basics with empty name/hostname (errors show); Backend step blocks empty host; Headers/Access are skippable; Review shows a summary; Create works. Open an existing reverse proxy at `/dashboard/proxies/$id`: sectioned form with Basics visible, sections collapsible, prefilled; submitting with a cleared hostname inside a collapsed section expands the section. Document results in the report.

- [ ] **Step 7: Commit**

```bash
git add ui/src/components/proxy/forms/reverse-proxy-form.tsx
git commit -m "feat(ui): RHF reverse-proxy form — wizard create + sectioned edit (M2a)"
```

---

## Task 7: RedirectForm — mode-prop wizard + sectioned edit

**Files:**
- Modify (replace): `ui/src/components/proxy/forms/redirect-form.tsx`

**Interfaces:**
- Consumes: `redirectSchema`/`RedirectFormValues`, `REDIRECT_DEFAULTS`/`mapProxyToRedirectDefaults`/`mapRedirectValuesToRequest`, Task 3 primitives, `RedirectTargetFields`/`RedirectOptionsFields`, `ACLSelector`.
- Produces: `RedirectForm({ mode, initialData?, initialACLAssignments?, onSubmit, loading, onCancel })` with `onSubmit: (data: CreateRedirectRequest, acl?) => void`.

**Wizard steps:** `1 Basics` → `2 Redirect Target` → `3 Options` → `4 Access Control` → `5 Review`. `STEP_FIELDS`: `1:['name','hostname','description']`, `2:['target','status_code']`, `3:['ssl_enabled','preserve_path','preserve_query']`, `4:[]`.

- [ ] **Step 1: Implement** following the exact structure of Task 6 (same parent shell, `RedirectWizard`, `RedirectReview`, `RedirectEdit`). Edit sections: Basics card (visible) + `FormSection "Redirect Target"` (open by default; `hasError` = `target`/`status_code`) + `FormSection "Options"`. `onInvalid` expands the Target section on `errors.target || errors.status_code`. Submit label "Create Proxy" / "Save Changes".

- [ ] **Step 2: Verify build + lint + tests**

Run: `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run`
Expected: clean.

- [ ] **Step 3: Manual verification** — `/dashboard/proxies/new?type=redirect` wizard: target must be a valid URL to advance step 2; status code select works; create works; edit prefills and saves. Document in report.

- [ ] **Step 4: Commit**

```bash
git add ui/src/components/proxy/forms/redirect-form.tsx
git commit -m "feat(ui): RHF redirect form — wizard create + sectioned edit (M2a)"
```

---

## Task 8: StaticForm — mode-prop wizard + sectioned edit

**Files:**
- Modify (replace): `ui/src/components/proxy/forms/static-form.tsx`

**Interfaces:**
- Consumes: `staticSchema`/`StaticFormValues`, `STATIC_DEFAULTS`/`mapProxyToStaticDefaults`/`mapStaticValuesToRequest`, Task 3 primitives, `StaticFileFields`/`StaticOptionsFields`/`TryFilesFields`, `ACLSelector`.
- Produces: `StaticForm({ mode, initialData?, initialACLAssignments?, onSubmit, loading, onCancel })` with `onSubmit: (data: CreateStaticRequest, acl?) => void`.

**Wizard steps:** `1 Basics` → `2 File Server` (StaticFileFields + TryFilesFields) → `3 Options` → `4 Access Control` → `5 Review`. `STEP_FIELDS`: `1:['name','hostname','description']`, `2:['root_path','index_file','try_files']`, `3:['ssl_enabled','browse','template_rendering']`, `4:[]`.

- [ ] **Step 1: Implement** following Task 6's structure. Edit sections: Basics card + `FormSection "File Server"` (StaticFileFields + TryFilesFields; open by default; `hasError` = `root_path`/`index_file`/`try_files`) + `FormSection "Options"`. `onInvalid` expands File Server on those errors.

- [ ] **Step 2: Verify build + lint + tests**

Run: `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run`
Expected: clean.

- [ ] **Step 3: Manual verification** — `/dashboard/proxies/new?type=static`: File Server step requires root_path + index_file; try_files add/remove works; create works; edit prefills try_files and saves (blank try_files rows dropped). Document in report.

- [ ] **Step 4: Commit**

```bash
git add ui/src/components/proxy/forms/static-form.tsx
git commit -m "feat(ui): RHF static form — wizard create + sectioned edit (M2a)"
```

---

## Task 9: Wire pages (create/edit modes)

**Files:**
- Modify: `ui/src/routes/_dashboard/proxies/new.tsx`
- Modify: `ui/src/routes/_dashboard/proxies/$proxyId.tsx`

**Interfaces:**
- Consumes: the three forms now require a `mode` prop.

- [ ] **Step 1: `new.tsx`** — add `mode="create"` to each of the three `<ReverseProxyForm />` / `<RedirectForm />` / `<StaticForm />` call sites. Leave the type pill selector and `handleSubmit` (createProxy + ACL loop + navigate) unchanged.

- [ ] **Step 2: `$proxyId.tsx`** — add `mode="edit"` to each form call site; keep existing `initialData={proxy}`, `initialACLAssignments={aclAssignments}`, `onSubmit={handleUpdate}`, `loading={isUpdating}`, `onCancel`. Leave the diff/update + ACL add/remove logic unchanged.

- [ ] **Step 3: Verify full gate**

Run: `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test run`
Expected: clean; tests pass.

- [ ] **Step 4: Manual end-to-end** — create one proxy of each type via the wizard; edit each via the sectioned form; confirm list reflects changes and ACL assignment persists on a reverse proxy. Document in report.

- [ ] **Step 5: Commit**

```bash
git add ui/src/routes/_dashboard/proxies/new.tsx ui/src/routes/_dashboard/proxies/\$proxyId.tsx
git commit -m "feat(ui): wire proxy pages to create/edit form modes (M2a)"
```

---

## Self-Review (run after writing, before execution)

1. **Spec coverage:** stepper-create ✓ (Tasks 6–8 wizards), sectioned-edit ✓ (edit layouts + FormSection), shared field groups ✓ (Tasks 3–5), RHF + zodResolver ✓ (Task 1–2, 6–8), useFieldArray for upstreams/headers/try_files ✓ (Tasks 4–5), per-step validation ✓ (`STEP_FIELDS` + `form.trigger`), expand-on-error ✓ (`onInvalid` + `FormSection.hasError`), ACL unchanged/separate ✓, submit shapes preserved ✓ (mappers + tests), defaultValues mapping ✓ (mappers + `form.reset`).
2. **Placeholder scan:** field-group copy is sourced from the named current files (real source, not a cross-task reference) — implementers read them; all tricky code (mappers, useFieldArray, wizard, onInvalid) is shown in full.
3. **Type consistency:** mapper names (`mapProxyToReverseDefaults`, `mapReverseValuesToRequest`, `REVERSE_PROXY_DEFAULTS`, redirect/static analogues) are identical in Task 2 definitions and Tasks 6–8 consumers; schema/value-type names match Task 1 exports; `onSubmit` signatures match `types/proxy.ts` request types.

---

## Execution notes

- The reverse-proxy form (Task 6) is the heaviest and establishes the wizard/edit/onInvalid pattern that Tasks 7–8 copy structurally — review it carefully before proceeding.
- Deferred to **M2b** (separate plan): list-side features — bulk enable/disable/delete, duplicate-via-prefill, "what protects this" ACL summary, export/import.
- Deferred to backend pipelines: config-preview (B1) and per-proxy health (B2) badges are NOT part of these forms yet.
