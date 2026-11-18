import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState, type ReactNode } from 'react';
import { useForm, useWatch } from 'react-hook-form';

import { Form } from '@/components/ui/form';
import { type RedirectFormValues, redirectSchema } from '@/lib/form-validation';
import type { CreateRedirectRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment } from './acl-selector';
import { AdvancedSettingsSheet } from './shared/advanced-settings-sheet';
import { BasicsFields } from './shared/basics-fields';
import { DescriptionField, ProxyFormLayout } from './shared/proxy-form-layout';
import {
  REDIRECT_DEFAULTS,
  mapProxyToRedirectDefaults,
  mapRedirectValuesToRequest,
} from './shared/proxy-form-mappers';
import { RedirectOptionsFields } from './shared/redirect-options-fields';
import { RedirectTargetFields } from './shared/redirect-target-fields';

interface Props {
  mode: 'create' | 'edit';
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateRedirectRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  typeControl?: ReactNode;
}
const resolver = zodResolver(redirectSchema);
export function RedirectForm({
  mode,
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
  typeControl,
}: Props) {
  const [acl, setACL] = useState<ACLAssignment[]>(initialACLAssignments ?? []);
  const form = useForm<RedirectFormValues>({
    resolver,
    mode: 'onTouched',
    defaultValues: initialData ? mapProxyToRedirectDefaults(initialData) : REDIRECT_DEFAULTS,
  });
  useEffect(() => {
    if (initialData) form.reset(mapProxyToRedirectDefaults(initialData));
  }, [initialData, form]);
  useEffect(() => {
    if (initialACLAssignments) setACL(initialACLAssignments);
  }, [initialACLAssignments]);
  const values = useWatch({ control: form.control }) as RedirectFormValues;
  const request = mapRedirectValuesToRequest(values);
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit((v) => onSubmit(mapRedirectValuesToRequest(v), acl))}>
        <ProxyFormLayout
          mode={mode}
          acl={acl}
          onACLChange={setACL}
          loading={loading}
          onCancel={onCancel}
          configRequest={{ ...request, ssl_forced: initialData?.ssl_forced ?? true }}
          configValid={redirectSchema.safeParse(values).success}
          advanced={
            <AdvancedSettingsSheet
              form={form}
              resolver={resolver}
              fields={['description', 'preserve_path', 'preserve_query']}
            >
              <DescriptionField />
              <RedirectOptionsFields />
            </AdvancedSettingsSheet>
          }
        >
          {typeControl}
          <BasicsFields autoFocusName={mode === 'create'} />
          <RedirectTargetFields />
        </ProxyFormLayout>
      </form>
    </Form>
  );
}
