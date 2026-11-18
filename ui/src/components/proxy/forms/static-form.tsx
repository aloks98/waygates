import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState, type ReactNode } from 'react';
import { useForm, useWatch, useFormContext } from 'react-hook-form';

import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { type StaticFormValues, staticSchema } from '@/lib/form-validation';
import type { CreateStaticRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment } from './acl-selector';
import { AdvancedSettingsSheet } from './shared/advanced-settings-sheet';
import { BasicsFields } from './shared/basics-fields';
import { DescriptionField, ProxyFormLayout } from './shared/proxy-form-layout';
import {
  STATIC_DEFAULTS,
  mapProxyToStaticDefaults,
  mapStaticValuesToRequest,
} from './shared/proxy-form-mappers';
import { StaticFileFields } from './shared/static-file-fields';
import { StaticOptionsFields } from './shared/static-options-fields';
import { TryFilesFields } from './shared/try-files-fields';

interface Props {
  mode: 'create' | 'edit';
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateStaticRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
  typeControl?: ReactNode;
}
const resolver = zodResolver(staticSchema);
export function StaticForm({
  mode,
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
  typeControl,
}: Props) {
  const [acl, setACL] = useState<ACLAssignment[]>(initialACLAssignments ?? []);
  const form = useForm<StaticFormValues>({
    resolver,
    mode: 'onTouched',
    defaultValues: initialData ? mapProxyToStaticDefaults(initialData) : STATIC_DEFAULTS,
  });
  useEffect(() => {
    if (initialData) form.reset(mapProxyToStaticDefaults(initialData));
  }, [initialData, form]);
  useEffect(() => {
    if (initialACLAssignments) setACL(initialACLAssignments);
  }, [initialACLAssignments]);
  const values = useWatch({ control: form.control }) as StaticFormValues;
  const request = mapStaticValuesToRequest(values);
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit((v) => onSubmit(mapStaticValuesToRequest(v), acl))}>
        <ProxyFormLayout
          mode={mode}
          acl={acl}
          onACLChange={setACL}
          loading={loading}
          onCancel={onCancel}
          configRequest={{ ...request, ssl_forced: initialData?.ssl_forced ?? true }}
          configValid={staticSchema.safeParse(values).success}
          advanced={
            <AdvancedSettingsSheet
              form={form}
              resolver={resolver}
              fields={['description', 'index_file', 'browse', 'template_rendering', 'try_files']}
            >
              <StaticIndexField />
              <DescriptionField />
              <StaticOptionsFields />
              <TryFilesFields />
            </AdvancedSettingsSheet>
          }
        >
          {typeControl}
          <BasicsFields autoFocusName={mode === 'create'} />
          <StaticFileFields />
        </ProxyFormLayout>
      </form>
    </Form>
  );
}

function StaticIndexField() {
  const form = useFormContext<StaticFormValues>();
  return (
    <FormField
      control={form.control}
      name="index_file"
      render={({ field }) => (
        <FormItem>
          <FormLabel>Index file</FormLabel>
          <FormControl>
            <Input className="font-mono" {...field} />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}
