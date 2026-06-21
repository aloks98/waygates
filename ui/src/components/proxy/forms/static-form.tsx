import { Button, Card, CardContent, CardHeader, CardTitle, Form } from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { type FieldErrors, useForm, useFormContext } from 'react-hook-form';

import { type StaticFormValues, staticSchema } from '@/lib/form-validation';
import type { CreateStaticRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment, ACLSelector } from './acl-selector';
import { BasicsFields } from './shared/basics-fields';
import { FormSection } from './shared/form-section';
import {
  STATIC_DEFAULTS,
  mapProxyToStaticDefaults,
  mapStaticValuesToRequest,
} from './shared/proxy-form-mappers';
import { ReviewRow, ReviewSection, WizardActions, WizardStepNav } from './shared/proxy-wizard';
import { StaticFileFields } from './shared/static-file-fields';
import { StaticOptionsFields } from './shared/static-options-fields';
import { TryFilesFields } from './shared/try-files-fields';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface StaticFormProps {
  mode: 'create' | 'edit';
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateStaticRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
}

interface OpenSections {
  fileServer: boolean;
  options: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const WIZARD_STEPS = [
  { step: 1, title: 'Basics' },
  { step: 2, title: 'File Server' },
  { step: 3, title: 'Options' },
  { step: 4, title: 'Access' },
  { step: 5, title: 'Review' },
];

const STEP_FIELDS: Record<number, (keyof StaticFormValues)[]> = {
  1: ['name', 'hostname', 'description'],
  2: ['root_path', 'index_file', 'try_files'],
  3: ['ssl_enabled', 'browse', 'template_rendering'],
  4: [],
};

// ---------------------------------------------------------------------------
// StaticReview — step 5 summary
// ---------------------------------------------------------------------------

function StaticReview({ acl }: { acl: ACLAssignment[] }) {
  const form = useFormContext<StaticFormValues>();
  const values = form.getValues();

  return (
    <div className="space-y-4">
      <ReviewSection title="Basics">
        <ReviewRow label="Name" value={values.name || '—'} />
        <ReviewRow label="Hostname" value={values.hostname || '—'} />
        {values.description && <ReviewRow label="Description" value={values.description} />}
      </ReviewSection>

      <ReviewSection title="File Server">
        <ReviewRow label="Root Path" value={values.root_path || '—'} />
        <ReviewRow label="Index File" value={values.index_file || '—'} />
        <ReviewRow
          label="Try Files"
          value={values.try_files.length > 0 ? `${values.try_files.length} configured` : 'None'}
        />
      </ReviewSection>

      <ReviewSection title="Options">
        <ReviewRow label="HTTPS" value={values.ssl_enabled ? 'Yes' : 'No'} />
        <ReviewRow label="Directory Browsing" value={values.browse ? 'Yes' : 'No'} />
        <ReviewRow label="Template Rendering" value={values.template_rendering ? 'Yes' : 'No'} />
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
// StaticWizard — create mode
// ---------------------------------------------------------------------------

interface StaticWizardProps {
  acl: ACLAssignment[];
  onAclChange: (acl: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

function StaticWizard({ acl, onAclChange, loading, onCancel, onSubmit }: StaticWizardProps) {
  const form = useFormContext<StaticFormValues>();
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
          {activeStep === 2 && (
            <div className="space-y-6">
              <StaticFileFields />
              <TryFilesFields />
            </div>
          )}
          {activeStep === 3 && <StaticOptionsFields />}
          {activeStep === 4 && (
            <ACLSelector value={acl} onChange={onAclChange} disabled={loading} />
          )}
          {activeStep === 5 && <StaticReview acl={acl} />}
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
// StaticEdit — edit mode
// ---------------------------------------------------------------------------

interface StaticEditProps {
  acl: ACLAssignment[];
  onAclChange: (acl: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  openSections: OpenSections;
  setOpenSections: React.Dispatch<React.SetStateAction<OpenSections>>;
}

function StaticEdit({
  acl,
  onAclChange,
  loading,
  onCancel,
  openSections,
  setOpenSections,
}: StaticEditProps) {
  const form = useFormContext<StaticFormValues>();
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

      {/* File Server */}
      <FormSection
        title="File Server"
        open={openSections.fileServer}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, fileServer: v }))}
        hasError={!!(errors.root_path || errors.index_file || errors.try_files)}
      >
        <div className="space-y-6">
          <StaticFileFields />
          <TryFilesFields />
        </div>
      </FormSection>

      {/* Options */}
      <FormSection
        title="Options"
        open={openSections.options}
        onOpenChange={(v) => setOpenSections((s) => ({ ...s, options: v }))}
        hasError={!!(errors.ssl_enabled || errors.browse || errors.template_rendering)}
      >
        <StaticOptionsFields />
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
// StaticForm — public export
// ---------------------------------------------------------------------------

export function StaticForm({
  mode,
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
}: StaticFormProps) {
  const [aclAssignments, setAclAssignments] = useState<ACLAssignment[]>(
    initialACLAssignments ?? [],
  );
  const [openSections, setOpenSections] = useState<OpenSections>({
    fileServer: true,
    options: false,
  });

  const form = useForm<StaticFormValues>({
    resolver: zodResolver(staticSchema),
    mode: 'onTouched',
    defaultValues: initialData ? mapProxyToStaticDefaults(initialData) : STATIC_DEFAULTS,
  });

  // ACL arrives async on edit
  useEffect(() => {
    if (initialACLAssignments) setAclAssignments(initialACLAssignments);
  }, [initialACLAssignments]);

  // Proxy data arrives async (edit or duplicate) — reset form when it lands.
  // form is a stable ref so this only re-runs when initialData changes.
  useEffect(() => {
    if (initialData) form.reset(mapProxyToStaticDefaults(initialData));
  }, [initialData, form]);

  const submit = (values: StaticFormValues) => {
    onSubmit(mapStaticValuesToRequest(values), aclAssignments.length ? aclAssignments : undefined);
  };

  const onInvalid = (errors: FieldErrors<StaticFormValues>) => {
    if (mode !== 'edit') return;
    setOpenSections((prev) => ({
      fileServer: prev.fileServer || !!(errors.root_path || errors.index_file || errors.try_files),
      options: prev.options || !!(errors.ssl_enabled || errors.browse || errors.template_rendering),
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
          <StaticWizard
            acl={aclAssignments}
            onAclChange={setAclAssignments}
            loading={loading}
            onCancel={onCancel}
            onSubmit={form.handleSubmit(submit, onInvalid)}
          />
        ) : (
          <StaticEdit
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
