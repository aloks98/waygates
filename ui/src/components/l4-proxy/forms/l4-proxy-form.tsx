import { zodResolver } from '@hookform/resolvers/zod';
import { ArrowDown, FileCode2, Network, Server } from 'lucide-react';
import { useEffect } from 'react';
import { useForm, useWatch } from 'react-hook-form';

import { AdvancedSettingsSheet } from '@/components/proxy/forms/shared/advanced-settings-sheet';
import { DescriptionField } from '@/components/proxy/forms/shared/proxy-form-layout';
import { GeneratedConfig } from '@/components/proxy/generated-config';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Form } from '@/components/ui/form';
import { usePermissions } from '@/hooks/use-permissions';
import { type L4ProxyFormValues, l4ProxySchema } from '@/lib/form-validation';
import type { CreateL4ProxyRequest, L4Proxy } from '@/types/l4-proxy';

import { L4BasicsFields } from './shared/l4-basics-fields';
import {
  L4_PROXY_DEFAULTS,
  mapL4FormValuesToRequest,
  mapL4ProxyToDefaults,
} from './shared/l4-proxy-form-mappers';
import { L4RoutesEditor } from './shared/l4-routes-editor';

interface Props {
  mode: 'create' | 'edit';
  initialData?: L4Proxy | null;
  onSubmit: (data: CreateL4ProxyRequest) => void;
  loading: boolean;
  onCancel: () => void;
}
const resolver = zodResolver(l4ProxySchema);
export function L4ProxyForm({ mode, initialData, onSubmit, loading, onCancel }: Props) {
  const form = useForm<L4ProxyFormValues>({
    resolver,
    mode: 'onTouched',
    defaultValues: initialData ? mapL4ProxyToDefaults(initialData) : L4_PROXY_DEFAULTS,
  });
  useEffect(() => {
    if (initialData) form.reset(mapL4ProxyToDefaults(initialData));
  }, [initialData, form]);
  loading = loading || form.formState.isSubmitting;
  const { canReadCaddyConfig } = usePermissions();
  const values = useWatch({ control: form.control }) as L4ProxyFormValues;
  const upstream = values.routes?.[0]?.upstreams?.[0];
  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit((v) => onSubmit(mapL4FormValuesToRequest(v)))}
        className="grid items-start gap-6 xl:grid-cols-[minmax(0,1.8fr)_minmax(280px,1fr)]"
      >
        <div className="flex min-w-0 flex-col gap-6">
          <Card>
            <CardContent className="flex flex-col gap-6">
              <L4BasicsFields autoFocusName={mode === 'create'} />
              <AdvancedSettingsSheet
                form={form}
                resolver={resolver}
                fields={['routes', 'description']}
                title="Routing & TLS"
              >
                <DescriptionField />
                <L4RoutesEditor />
              </AdvancedSettingsSheet>
              {form.formState.errors.routes && (
                <p role="alert" className="text-sm text-destructive">
                  Configure at least one valid route and destination.
                </p>
              )}
              <div className="flex justify-between gap-3">
                <Button type="button" variant="ghost" onClick={onCancel}>
                  Cancel
                </Button>
                <Button type="submit" disabled={loading}>
                  {loading ? 'Saving…' : mode === 'create' ? 'Create proxy' : 'Save changes'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
        <aside className="flex min-w-0 flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle>Connection preview</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <div className="flex flex-col gap-3 rounded-md bg-muted p-4">
                <span className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Network className="size-4" />
                  Listener · {values.protocol?.toUpperCase()}
                </span>
                <span className="font-mono text-sm">:{values.listen_port || 'port'}</span>
              </div>
              <ArrowDown className="mx-auto size-5 text-muted-foreground" />
              <div className="flex flex-col gap-3 rounded-md bg-muted p-4">
                <span className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Server className="size-4" />
                  Internal destination
                </span>
                <span className="break-all font-mono text-sm">
                  {upstream?.host ? `${upstream.host}:${upstream.port}` : 'host:port'}
                </span>
                <span className="text-xs text-muted-foreground">
                  {values.routes?.length ?? 0} routes
                </span>
              </div>
            </CardContent>
          </Card>
          {canReadCaddyConfig && (
            <Card>
              <CardHeader>
                <CardTitle>Generated configuration</CardTitle>
              </CardHeader>
              <CardContent>
                <GeneratedConfig
                  request={mapL4FormValuesToRequest(values)}
                  enabled={l4ProxySchema.safeParse(values).success}
                  trigger={
                    <Button type="button" variant="outline" className="w-full justify-start">
                      <FileCode2 />
                      View generated config
                    </Button>
                  }
                />
              </CardContent>
            </Card>
          )}
        </aside>
      </form>
    </Form>
  );
}
