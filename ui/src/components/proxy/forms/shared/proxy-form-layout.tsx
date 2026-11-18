import { ArrowDown, FileCode2, Globe, Server } from 'lucide-react';
import type { ReactNode } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';

import { GeneratedConfig } from '@/components/proxy/generated-config';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { usePermissions } from '@/hooks/use-permissions';
import { formatUpstream } from '@/lib/upstream-url';
import type { CreateProxyRequest } from '@/types/proxy';

import { ACLSelector, type ACLAssignment } from '../acl-selector';

export function DescriptionField() {
  const form = useFormContext();
  return (
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
  );
}

export function ProxyFormLayout({
  children,
  advanced,
  acl,
  onACLChange,
  loading,
  mode,
  onCancel,
  configRequest,
  configValid,
}: {
  children: ReactNode;
  advanced: ReactNode;
  acl: ACLAssignment[];
  onACLChange: (acl: ACLAssignment[]) => void;
  loading: boolean;
  mode: 'create' | 'edit';
  onCancel: () => void;
  configRequest: CreateProxyRequest & { ssl_forced: boolean };
  configValid: boolean;
}) {
  const form = useFormContext();
  loading = loading || form.formState.isSubmitting;
  const { canReadCaddyConfig } = usePermissions();
  const values = useWatch({ control: form.control });
  const destination = values.target || values.root_path || formatUpstream(values.upstreams?.[0]);
  return (
    <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1.8fr)_minmax(280px,1fr)]">
      <div className="flex min-w-0 flex-col gap-6">
        <Card>
          <CardContent className="flex flex-col gap-6">
            {children}
            <div className="border-t pt-6">
              <FormField
                control={form.control}
                name="ssl_enabled"
                render={({ field }) => (
                  <FormItem
                    orientation="horizontal"
                    className="flex flex-row items-center justify-between"
                  >
                    <FormLabel>HTTPS</FormLabel>
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>
            <ACLSelector value={acl} onChange={onACLChange} disabled={loading} />
            <div className="border-t pt-5">{advanced}</div>
            {Object.keys(form.formState.errors).length > 0 && (
              <p role="alert" className="text-sm text-destructive">
                Check the highlighted fields and advanced settings before saving.
              </p>
            )}
            <div className="mt-1 flex justify-between gap-3">
              <Button type="button" variant="ghost" onClick={onCancel} disabled={loading}>
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
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Globe className="size-4 text-accent-foreground" />
                Public address · {values.ssl_enabled ? 'HTTPS' : 'HTTP'}
              </div>
              <p className="break-all font-mono text-sm">{values.hostname || 'app.example.com'}</p>
            </div>
            <ArrowDown aria-hidden="true" className="mx-auto size-5 text-muted-foreground" />
            <div className="flex flex-col gap-3 rounded-md bg-muted p-4">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Server className="size-4 text-accent-foreground" />
                {values.target
                  ? 'Redirect destination'
                  : values.root_path
                    ? 'File directory'
                    : 'Internal destination'}
              </div>
              <p className="break-all font-mono text-sm">{destination || 'http://service:3000'}</p>
              {values.upstreams?.length > 1 && (
                <span className="text-xs text-muted-foreground">
                  +{values.upstreams.length - 1} more destinations
                </span>
              )}
            </div>
          </CardContent>
        </Card>
        {canReadCaddyConfig && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileCode2 className="size-4 text-muted-foreground" />
                Generated configuration
              </CardTitle>
            </CardHeader>
            <CardContent>
              <GeneratedConfig
                request={configRequest}
                enabled={configValid}
                acl={acl}
                trigger={
                  <Button type="button" variant="outline" className="w-full justify-start">
                    <FileCode2 data-icon="inline-start" />
                    View generated config
                  </Button>
                }
              />
            </CardContent>
          </Card>
        )}
      </aside>
    </div>
  );
}
