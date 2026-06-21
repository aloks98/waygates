import { Button, Card, CardContent, CardHeader, CardTitle, Form } from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { type FieldErrors, useForm, useFormContext } from 'react-hook-form';

import { type ReverseProxyFormValues, reverseProxySchema } from '@/lib/form-validation';
import type { CreateReverseProxyRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment, ACLSelector } from './acl-selector';
import { BackendFields } from './shared/backend-fields';
import { BasicsFields } from './shared/basics-fields';
import { CustomHeadersFields } from './shared/custom-headers-fields';
import { FormSection } from './shared/form-section';
import { LoadBalancingFields } from './shared/load-balancing-fields';
import {
  REVERSE_PROXY_DEFAULTS,
  mapProxyToReverseDefaults,
  mapReverseValuesToRequest,
} from './shared/proxy-form-mappers';
import { ReviewRow, ReviewSection, WizardActions, WizardStepNav } from './shared/proxy-wizard';
import { SecurityFields } from './shared/security-fields';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ReverseProxyFormProps {
  mode: 'create' | 'edit';
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateReverseProxyRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
}

interface OpenSections {
  backend: boolean;
  security: boolean;
  headers: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

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
  3: [
    'ssl_enabled',
    'block_exploits',
    'tls_insecure_skip_verify',
    'lb_strategy',
    'health_check_enabled',
    'health_check_path',
    'health_check_interval',
    'health_check_timeout',
  ],
  4: ['request_headers', 'response_headers'],
  5: [],
};

// ---------------------------------------------------------------------------
// ReverseReview — step 6 summary
// ---------------------------------------------------------------------------

function ReverseReview({ acl }: { acl: ACLAssignment[] }) {
  const form = useFormContext<ReverseProxyFormValues>();
  const values = form.getValues();

  return (
    <div className="space-y-4">
      <ReviewSection title="Basics">
        <ReviewRow label="Name" value={values.name || '—'} />
        <ReviewRow label="Hostname" value={values.hostname || '—'} />
        {values.description && <ReviewRow label="Description" value={values.description} />}
      </ReviewSection>

      <ReviewSection title="Backend Servers">
        {values.upstreams.length === 0 ? (
          <ReviewRow label="Servers" value="None" />
        ) : (
          values.upstreams.map((u, i) => (
            <ReviewRow
              key={`${u.scheme}://${u.host}:${u.port}`}
              label={`Server ${i + 1}`}
              value={`${u.scheme}://${u.host}:${u.port}`}
            />
          ))
        )}
      </ReviewSection>

      <ReviewSection title="Security">
        <ReviewRow label="HTTPS" value={values.ssl_enabled ? 'Yes' : 'No'} />
        <ReviewRow label="Block Exploits" value={values.block_exploits ? 'Yes' : 'No'} />
        <ReviewRow
          label="Allow Self-Signed Certs"
          value={values.tls_insecure_skip_verify ? 'Yes' : 'No'}
        />
      </ReviewSection>

      {values.upstreams.length > 1 && (
        <ReviewSection title="Load Balancing">
          <ReviewRow label="Strategy" value={values.lb_strategy} />
          <ReviewRow
            label="Health Checks"
            value={
              values.health_check_enabled
                ? `${values.health_check_path} every ${values.health_check_interval}`
                : 'Disabled'
            }
          />
        </ReviewSection>
      )}

      <ReviewSection title="Headers">
        <ReviewRow
          label="Request Headers"
          value={
            values.request_headers.length > 0
              ? `${values.request_headers.length} configured`
              : 'None'
          }
        />
        <ReviewRow
          label="Response Headers"
          value={
            values.response_headers.length > 0
              ? `${values.response_headers.length} configured`
              : 'None'
          }
        />
      </ReviewSection>

      {acl.length > 0 && (
        <ReviewSection title="Access Control">
          <ReviewRow label="ACL Assignments" value={`${acl.length} configured`} />
        </ReviewSection>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// ReverseWizard — create mode
// ---------------------------------------------------------------------------

interface ReverseWizardProps {
  acl: ACLAssignment[];
  onAclChange: (acl: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

function ReverseWizard({ acl, onAclChange, loading, onCancel, onSubmit }: ReverseWizardProps) {
  const form = useFormContext<ReverseProxyFormValues>();
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
          {activeStep === 1 && <BasicsFields autoFocusName />}
          {activeStep === 2 && <BackendFields />}
          {activeStep === 3 && (
            <div className="space-y-6">
              <SecurityFields />
              <LoadBalancingFields />
            </div>
          )}
          {activeStep === 4 && <CustomHeadersFields />}
          {activeStep === 5 && (
            <ACLSelector value={acl} onChange={onAclChange} disabled={loading} />
          )}
          {activeStep === 6 && <ReverseReview acl={acl} />}
        </CardContent>
      </Card>

      <WizardActions
        activeStep={activeStep}
        lastStep={6}
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
// ReverseEdit — edit mode
// ---------------------------------------------------------------------------

interface ReverseEditProps {
  acl: ACLAssignment[];
  onAclChange: (acl: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  openSections: OpenSections;
  setOpenSections: React.Dispatch<React.SetStateAction<OpenSections>>;
}

function ReverseEdit({
  acl,
  onAclChange,
  loading,
  onCancel,
  openSections,
  setOpenSections,
}: ReverseEditProps) {
  const form = useFormContext<ReverseProxyFormValues>();
  const errors = form.formState.errors;

  const securityHasError = !!(
    errors.ssl_enabled ||
    errors.block_exploits ||
    errors.tls_insecure_skip_verify ||
    errors.lb_strategy ||
    errors.health_check_path ||
    errors.health_check_interval ||
    errors.health_check_timeout
  );

  return (
    <div className="space-y-6">
      {/* Basics */}
      <Card>
        <CardHeader>
          <CardTitle>Basics</CardTitle>
        </CardHeader>
        <CardContent>
          <BasicsFields />
        </CardContent>
      </Card>

      {/* Backend */}
      <FormSection
        title="Backend Servers"
        open={openSections.backend}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, backend: v }))}
        hasError={!!errors.upstreams}
      >
        <BackendFields />
      </FormSection>

      {/* Security & Load Balancing */}
      <FormSection
        title="Security & Load Balancing"
        open={openSections.security}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, security: v }))}
        hasError={securityHasError}
      >
        <div className="space-y-6">
          <SecurityFields />
          <LoadBalancingFields />
        </div>
      </FormSection>

      {/* Custom Headers */}
      <FormSection
        title="Custom Headers"
        open={openSections.headers}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, headers: v }))}
        hasError={!!(errors.request_headers || errors.response_headers)}
      >
        <CustomHeadersFields />
      </FormSection>

      {/* Access Control */}
      <ACLSelector value={acl} onChange={onAclChange} disabled={loading} />

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
// ReverseProxyForm — public export
// ---------------------------------------------------------------------------

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
  const [openSections, setOpenSections] = useState<OpenSections>({
    backend: true,
    security: false,
    headers: false,
  });

  const form = useForm<ReverseProxyFormValues>({
    resolver: zodResolver(reverseProxySchema),
    mode: 'onTouched',
    defaultValues: initialData ? mapProxyToReverseDefaults(initialData) : REVERSE_PROXY_DEFAULTS,
  });

  // ACL arrives async on edit
  useEffect(() => {
    if (initialACLAssignments) setAclAssignments(initialACLAssignments);
  }, [initialACLAssignments]);

  // Proxy data arrives async (edit or duplicate) — reset form when it lands.
  // form is a stable ref so this only re-runs when initialData changes.
  useEffect(() => {
    if (initialData) form.reset(mapProxyToReverseDefaults(initialData));
  }, [initialData, form]);

  const submit = (values: ReverseProxyFormValues) => {
    onSubmit(mapReverseValuesToRequest(values), aclAssignments.length ? aclAssignments : undefined);
  };

  const onInvalid = (errors: FieldErrors<ReverseProxyFormValues>) => {
    if (mode !== 'edit') return;
    setOpenSections((prev) => ({
      backend: prev.backend || !!errors.upstreams,
      security:
        prev.security ||
        !!(
          errors.ssl_enabled ||
          errors.block_exploits ||
          errors.tls_insecure_skip_verify ||
          errors.lb_strategy ||
          errors.health_check_path ||
          errors.health_check_interval ||
          errors.health_check_timeout
        ),
      headers: prev.headers || !!(errors.request_headers || errors.response_headers),
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
          <ReverseWizard
            acl={aclAssignments}
            onAclChange={setAclAssignments}
            loading={loading}
            onCancel={onCancel}
            onSubmit={form.handleSubmit(submit, onInvalid)}
          />
        ) : (
          <ReverseEdit
            acl={aclAssignments}
            onAclChange={setAclAssignments}
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
