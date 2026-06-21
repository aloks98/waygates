import { Button, Card, CardContent, CardHeader, CardTitle, Form } from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { type FieldErrors, useForm, useFormContext } from 'react-hook-form';

import { FormSection } from '@/components/proxy/forms/shared/form-section';
import {
  ReviewRow,
  ReviewSection,
  WizardActions,
  WizardStepNav,
} from '@/components/proxy/forms/shared/proxy-wizard';
import { type L4ProxyFormValues, l4ProxySchema } from '@/lib/form-validation';
import { L4_MATCHER_CONFIG } from '@/types/l4-proxy';
import type { CreateL4ProxyRequest, L4LoadBalancingPolicy, L4Proxy } from '@/types/l4-proxy';

import { L4BasicsFields } from './shared/l4-basics-fields';
import {
  L4_PROXY_DEFAULTS,
  mapL4FormValuesToRequest,
  mapL4ProxyToDefaults,
} from './shared/l4-proxy-form-mappers';
import { L4RoutesEditor } from './shared/l4-routes-editor';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface L4ProxyFormProps {
  mode: 'create' | 'edit';
  initialData?: L4Proxy | null;
  onSubmit: (data: CreateL4ProxyRequest) => void;
  loading: boolean;
  onCancel: () => void;
}

interface OpenSections {
  routes: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const WIZARD_STEPS = [
  { step: 1, title: 'Basics' },
  { step: 2, title: 'Routes' },
  { step: 3, title: 'Review' },
];

const STEP_FIELDS: Record<number, (keyof L4ProxyFormValues)[]> = {
  1: ['name', 'description', 'listen_port', 'protocol', 'is_active'],
  2: ['routes'],
};

const LB_POLICY_LABELS: Record<L4LoadBalancingPolicy, string> = {
  round_robin: 'Round Robin',
  least_conn: 'Least Connections',
  random: 'Random',
  first: 'First Available',
  ip_hash: 'Sticky (IP Hash)',
};

// ---------------------------------------------------------------------------
// L4Review — step 3 summary
// ---------------------------------------------------------------------------

function L4Review() {
  const form = useFormContext<L4ProxyFormValues>();
  const values = form.getValues();
  const routes = values.routes ?? [];

  return (
    <div className="space-y-4">
      <ReviewSection title="Basics">
        <ReviewRow label="Name" value={values.name || '—'} />
        <ReviewRow
          label="Listen"
          value={`${values.listen_port} (${values.protocol.toUpperCase()})`}
        />
        {values.description && <ReviewRow label="Description" value={values.description} />}
        <ReviewRow label="Active" value={values.is_active ? 'Yes' : 'No'} />
      </ReviewSection>

      {routes.map((route, i) => (
        <ReviewSection key={i} title={`Route ${i + 1}`}>
          <ReviewRow
            label="Matcher"
            value={L4_MATCHER_CONFIG[route.matcher_type]?.label ?? route.matcher_type}
          />

          {/* Matcher-specific values */}
          {route.matcher_type === 'tls' &&
            route.sni_hostnames &&
            route.sni_hostnames.length > 0 && (
              <ReviewRow
                label="SNI Hostnames"
                value={route.sni_hostnames.map((h) => h.value).join(', ')}
              />
            )}
          {route.matcher_type === 'remote_ip' &&
            route.allowed_ip_ranges &&
            route.allowed_ip_ranges.length > 0 && (
              <ReviewRow
                label="IP Ranges"
                value={route.allowed_ip_ranges.map((r) => r.value).join(', ')}
              />
            )}
          {route.matcher_type === 'regexp' && route.regex_pattern && (
            <ReviewRow label="Regex Pattern" value={route.regex_pattern} />
          )}

          <ReviewRow
            label="Upstreams"
            value={route.upstreams.map((u) => `${u.host}:${u.port}`).join(', ') || '—'}
          />

          {route.upstreams.length > 1 && (
            <ReviewRow
              label="Load Balancing"
              value={LB_POLICY_LABELS[route.load_balancing_policy] ?? route.load_balancing_policy}
            />
          )}

          {(route.tls_terminate || route.tls_passthrough) && (
            <ReviewRow label="TLS" value={route.tls_terminate ? 'Terminate' : 'Passthrough'} />
          )}
        </ReviewSection>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// L4Wizard — create mode
// ---------------------------------------------------------------------------

interface L4WizardProps {
  loading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

function L4Wizard({ loading, onCancel, onSubmit }: L4WizardProps) {
  const form = useFormContext<L4ProxyFormValues>();
  const [activeStep, setActiveStep] = useState(1);
  const [completedSteps, setCompletedSteps] = useState<Set<number>>(new Set());

  const advance = async () => {
    const fields = STEP_FIELDS[activeStep] ?? [];
    const valid = await form.trigger(fields);
    if (valid) {
      setCompletedSteps((prev) => new Set(prev).add(activeStep));
      setActiveStep((s) => s + 1);
    }
  };

  const goTo = (step: number) => {
    if (step < activeStep || completedSteps.has(step)) {
      setActiveStep(step);
    }
  };

  return (
    <div className="space-y-6">
      <WizardStepNav
        steps={WIZARD_STEPS}
        activeStep={activeStep}
        completedSteps={completedSteps}
        onStepClick={goTo}
      />

      <Card>
        <CardContent className="pt-6">
          {activeStep === 1 && <L4BasicsFields autoFocusName />}
          {activeStep === 2 && <L4RoutesEditor />}
          {activeStep === 3 && <L4Review />}
        </CardContent>
      </Card>

      <WizardActions
        activeStep={activeStep}
        lastStep={3}
        onBack={() => setActiveStep((s) => s - 1)}
        onNext={advance}
        onCancel={onCancel}
        onSubmit={onSubmit}
        submitting={loading}
        submitLabel="Create Proxy"
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// L4Edit — edit mode
// ---------------------------------------------------------------------------

interface L4EditProps {
  loading: boolean;
  onCancel: () => void;
  openSections: OpenSections;
  setOpenSections: React.Dispatch<React.SetStateAction<OpenSections>>;
}

function L4Edit({ loading, onCancel, openSections, setOpenSections }: L4EditProps) {
  const form = useFormContext<L4ProxyFormValues>();

  return (
    <div className="space-y-6">
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
        </CardHeader>
        <CardContent>
          <L4BasicsFields />
        </CardContent>
      </Card>

      {/* Routes */}
      <FormSection
        title="Routes"
        open={openSections.routes}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, routes: v }))}
        hasError={!!form.formState.errors.routes}
      >
        <L4RoutesEditor />
      </FormSection>

      {/* Actions */}
      <div className="flex justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving…' : 'Save Changes'}
        </Button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// L4ProxyForm — public export
// ---------------------------------------------------------------------------

export function L4ProxyForm({ mode, initialData, onSubmit, loading, onCancel }: L4ProxyFormProps) {
  const [openSections, setOpenSections] = useState<OpenSections>({ routes: true });

  const form = useForm<L4ProxyFormValues>({
    resolver: zodResolver(l4ProxySchema),
    mode: 'onTouched',
    defaultValues: initialData ? mapL4ProxyToDefaults(initialData) : L4_PROXY_DEFAULTS,
  });

  // Reset form when initialData changes (async load on edit/duplicate)
  useEffect(() => {
    if (initialData) form.reset(mapL4ProxyToDefaults(initialData));
  }, [initialData, form]);

  const submit = (values: L4ProxyFormValues) => {
    onSubmit(mapL4FormValuesToRequest(values));
  };

  const onInvalid = (errors: FieldErrors<L4ProxyFormValues>) => {
    if (mode !== 'edit') return;
    setOpenSections((prev) => ({
      routes: prev.routes || !!errors.routes,
    }));
  };

  return (
    <Form {...form}>
      {/* In create (wizard) mode, native form submission is suppressed entirely —
          the wizard submits only via its explicit Create button (onSubmit below).
          This blocks Enter-in-input and any button-morph from auto-creating. */}
      <form
        onSubmit={
          mode === 'edit' ? form.handleSubmit(submit, onInvalid) : (e) => e.preventDefault()
        }
        className="space-y-6"
      >
        {mode === 'create' ? (
          <L4Wizard
            loading={loading}
            onCancel={onCancel}
            onSubmit={form.handleSubmit(submit, onInvalid)}
          />
        ) : (
          <L4Edit
            loading={loading}
            onCancel={onCancel}
            openSections={openSections}
            setOpenSections={setOpenSections}
          />
        )}
      </form>
    </Form>
  );
}
