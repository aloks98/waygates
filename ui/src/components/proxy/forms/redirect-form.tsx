import { Button, Card, CardContent, CardHeader, CardTitle, Form } from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { type FieldErrors, useForm, useFormContext } from 'react-hook-form';

import { type RedirectFormValues, redirectSchema } from '@/lib/form-validation';
import type { CreateRedirectRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment, ACLSelector } from './acl-selector';
import { BasicsFields } from './shared/basics-fields';
import { FormSection } from './shared/form-section';
import {
  REDIRECT_DEFAULTS,
  mapProxyToRedirectDefaults,
  mapRedirectValuesToRequest,
} from './shared/proxy-form-mappers';
import { ReviewRow, ReviewSection, WizardActions, WizardStepNav } from './shared/proxy-wizard';
import { RedirectOptionsFields } from './shared/redirect-options-fields';
import { RedirectTargetFields } from './shared/redirect-target-fields';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface RedirectFormProps {
  mode: 'create' | 'edit';
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateRedirectRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
}

interface OpenSections {
  target: boolean;
  options: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const WIZARD_STEPS = [
  { step: 1, title: 'Basics' },
  { step: 2, title: 'Target' },
  { step: 3, title: 'Options' },
  { step: 4, title: 'Access' },
  { step: 5, title: 'Review' },
];

const STEP_FIELDS: Record<number, (keyof RedirectFormValues)[]> = {
  1: ['name', 'hostname', 'description'],
  2: ['target', 'status_code'],
  3: ['ssl_enabled', 'preserve_path', 'preserve_query'],
  4: [],
};

// ---------------------------------------------------------------------------
// Status code labels for the review screen
// ---------------------------------------------------------------------------

const STATUS_CODE_LABELS: Record<number, string> = {
  301: '301 — Permanent',
  302: '302 — Temporary',
  307: '307 — Temporary (preserve method)',
  308: '308 — Permanent (preserve method)',
};

// ---------------------------------------------------------------------------
// RedirectReview — step 5 summary
// ---------------------------------------------------------------------------

function RedirectReview({ acl }: { acl: ACLAssignment[] }) {
  const form = useFormContext<RedirectFormValues>();
  const values = form.getValues();

  return (
    <div className="space-y-4">
      <ReviewSection title="Basics">
        <ReviewRow label="Name" value={values.name || '—'} />
        <ReviewRow label="Hostname" value={values.hostname || '—'} />
        {values.description && <ReviewRow label="Description" value={values.description} />}
      </ReviewSection>

      <ReviewSection title="Target">
        <ReviewRow label="Target URL" value={values.target || '—'} />
        <ReviewRow
          label="Redirect Type"
          value={STATUS_CODE_LABELS[values.status_code] ?? String(values.status_code)}
        />
      </ReviewSection>

      <ReviewSection title="Options">
        <ReviewRow label="HTTPS" value={values.ssl_enabled ? 'Yes' : 'No'} />
        <ReviewRow label="Preserve Path" value={values.preserve_path ? 'Yes' : 'No'} />
        <ReviewRow label="Preserve Query" value={values.preserve_query ? 'Yes' : 'No'} />
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
// RedirectWizard — create mode
// ---------------------------------------------------------------------------

interface RedirectWizardProps {
  acl: ACLAssignment[];
  onAclChange: (acl: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

function RedirectWizard({ acl, onAclChange, loading, onCancel, onSubmit }: RedirectWizardProps) {
  const form = useFormContext<RedirectFormValues>();
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
          {activeStep === 2 && <RedirectTargetFields />}
          {activeStep === 3 && <RedirectOptionsFields />}
          {activeStep === 4 && (
            <ACLSelector value={acl} onChange={onAclChange} disabled={loading} />
          )}
          {activeStep === 5 && <RedirectReview acl={acl} />}
        </CardContent>
      </Card>

      <WizardActions
        activeStep={activeStep}
        lastStep={5}
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
// RedirectEdit — edit mode
// ---------------------------------------------------------------------------

interface RedirectEditProps {
  acl: ACLAssignment[];
  onAclChange: (acl: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  openSections: OpenSections;
  setOpenSections: React.Dispatch<React.SetStateAction<OpenSections>>;
}

function RedirectEdit({
  acl,
  onAclChange,
  loading,
  onCancel,
  openSections,
  setOpenSections,
}: RedirectEditProps) {
  const form = useFormContext<RedirectFormValues>();
  const errors = form.formState.errors;

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

      {/* Redirect Target */}
      <FormSection
        title="Redirect Target"
        open={openSections.target}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, target: v }))}
        hasError={!!(errors.target || errors.status_code)}
      >
        <RedirectTargetFields />
      </FormSection>

      {/* Options */}
      <FormSection
        title="Options"
        open={openSections.options}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, options: v }))}
        hasError={!!(errors.ssl_enabled || errors.preserve_path || errors.preserve_query)}
      >
        <RedirectOptionsFields />
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
// RedirectForm — public export
// ---------------------------------------------------------------------------

export function RedirectForm({
  mode,
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
}: RedirectFormProps) {
  const [aclAssignments, setAclAssignments] = useState<ACLAssignment[]>(
    initialACLAssignments ?? [],
  );
  const [openSections, setOpenSections] = useState<OpenSections>({
    target: true,
    options: false,
  });

  const form = useForm<RedirectFormValues>({
    resolver: zodResolver(redirectSchema),
    mode: 'onTouched',
    defaultValues: initialData ? mapProxyToRedirectDefaults(initialData) : REDIRECT_DEFAULTS,
  });

  // ACL arrives async on edit
  useEffect(() => {
    if (initialACLAssignments) setAclAssignments(initialACLAssignments);
  }, [initialACLAssignments]);

  // Proxy data arrives async (edit or duplicate) — reset form when it lands.
  // form is a stable ref so this only re-runs when initialData changes.
  useEffect(() => {
    if (initialData) form.reset(mapProxyToRedirectDefaults(initialData));
  }, [initialData, form]);

  const submit = (values: RedirectFormValues) => {
    onSubmit(
      mapRedirectValuesToRequest(values),
      aclAssignments.length ? aclAssignments : undefined,
    );
  };

  const onInvalid = (errors: FieldErrors<RedirectFormValues>) => {
    if (mode !== 'edit') return;
    setOpenSections((prev) => ({
      target: prev.target || !!(errors.target || errors.status_code),
      options:
        prev.options || !!(errors.ssl_enabled || errors.preserve_path || errors.preserve_query),
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
          <RedirectWizard
            acl={aclAssignments}
            onAclChange={setAclAssignments}
            loading={loading}
            onCancel={onCancel}
            onSubmit={form.handleSubmit(submit, onInvalid)}
          />
        ) : (
          <RedirectEdit
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
