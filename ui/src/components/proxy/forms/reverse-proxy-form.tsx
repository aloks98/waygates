import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState, type ReactNode } from 'react';
import { useForm, useWatch } from 'react-hook-form';

import { Form } from '@/components/ui/form';
import { type ReverseProxyFormValues, reverseProxySchema } from '@/lib/form-validation';
import type { CreateReverseProxyRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment } from './acl-selector';
import { AdvancedSettingsSheet } from './shared/advanced-settings-sheet';
import { BasicsFields } from './shared/basics-fields';
import { CustomHeadersFields } from './shared/custom-headers-fields';
import { DestinationField } from './shared/destination-field';
import { LoadBalancingFields } from './shared/load-balancing-fields';
import { DescriptionField, ProxyFormLayout } from './shared/proxy-form-layout';
import {
  REVERSE_PROXY_DEFAULTS,
  mapProxyToReverseDefaults,
  mapReverseValuesToRequest,
} from './shared/proxy-form-mappers';
import { SecurityFields } from './shared/security-fields';

interface Props {
  mode: 'create' | 'edit';
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateReverseProxyRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  typeControl?: ReactNode;
}
const resolver = zodResolver(reverseProxySchema);
export function ReverseProxyForm({
  mode,
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
  typeControl,
}: Props) {
  const [acl, setACL] = useState<ACLAssignment[]>(initialACLAssignments ?? []);
  const form = useForm<ReverseProxyFormValues>({
    resolver,
    mode: 'onTouched',
    defaultValues: initialData ? mapProxyToReverseDefaults(initialData) : REVERSE_PROXY_DEFAULTS,
  });
  useEffect(() => {
    if (initialData) form.reset(mapProxyToReverseDefaults(initialData));
  }, [initialData, form]);
  useEffect(() => {
    if (initialACLAssignments) setACL(initialACLAssignments);
  }, [initialACLAssignments]);
  const values = useWatch({ control: form.control }) as ReverseProxyFormValues;
  const request = mapReverseValuesToRequest(values);
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit((v) => onSubmit(mapReverseValuesToRequest(v), acl))}>
        <ProxyFormLayout
          mode={mode}
          acl={acl}
          onACLChange={setACL}
          loading={loading}
          onCancel={onCancel}
          configRequest={{ ...request, ssl_forced: initialData?.ssl_forced ?? true }}
          configValid={reverseProxySchema.safeParse(values).success}
          advanced={
            <AdvancedSettingsSheet
              form={form}
              resolver={resolver}
              fields={[
                'description',
                'block_exploits',
                'tls_insecure_skip_verify',
                'lb_strategy',
                'health_check_enabled',
                'health_check_path',
                'health_check_interval',
                'health_check_timeout',
                'request_headers',
                'response_headers',
              ]}
            >
              <DescriptionField />
              <SecurityFields />
              <LoadBalancingFields />
              <CustomHeadersFields />
            </AdvancedSettingsSheet>
          }
        >
          {typeControl}
          <BasicsFields autoFocusName={mode === 'create'} />
          <DestinationField />
        </ProxyFormLayout>
      </form>
    </Form>
  );
}
