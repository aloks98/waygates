import { FormControl, FormDescription, FormField, FormItem, FormLabel } from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

import type { ReverseProxyFormValues } from '@/lib/form-validation';

import { InheritableSwitch, PROXY_SYSTEM_DEFAULTS } from './inheritable-switch';
import { useSelectedGroup } from './use-selected-group';

export function SecurityFields() {
  const form = useFormContext<ReverseProxyFormValues>();
  const { groupId, group, hasGroup } = useSelectedGroup();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="ssl_enabled"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between gap-4">
            <div className="space-y-0.5">
              <FormLabel>Enable HTTPS</FormLabel>
              <FormDescription>
                Automatically obtain and manage SSL/TLS certificates
              </FormDescription>
            </div>
            <FormControl>
              <InheritableSwitch
                key={groupId ?? 'no-group'}
                value={field.value}
                onChange={field.onChange}
                groupValue={group?.ssl_enabled ?? null}
                systemDefault={PROXY_SYSTEM_DEFAULTS.ssl_enabled}
                hasGroup={hasGroup}
                label="Enable HTTPS"
              />
            </FormControl>
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="ssl_forced"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between gap-4">
            <div className="space-y-0.5">
              <FormLabel>Force HTTPS</FormLabel>
              <FormDescription>Redirect plain HTTP requests to HTTPS</FormDescription>
            </div>
            <FormControl>
              <InheritableSwitch
                key={groupId ?? 'no-group'}
                value={field.value}
                onChange={field.onChange}
                groupValue={group?.ssl_forced ?? null}
                systemDefault={PROXY_SYSTEM_DEFAULTS.ssl_forced}
                hasGroup={hasGroup}
                label="Force HTTPS"
              />
            </FormControl>
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="block_exploits"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between gap-4">
            <div className="space-y-0.5">
              <FormLabel>Block Common Exploits</FormLabel>
              <FormDescription>Block SQL injection, XSS, and other common attacks</FormDescription>
            </div>
            <FormControl>
              <InheritableSwitch
                key={groupId ?? 'no-group'}
                value={field.value}
                onChange={field.onChange}
                groupValue={group?.block_exploits ?? null}
                systemDefault={PROXY_SYSTEM_DEFAULTS.block_exploits}
                hasGroup={hasGroup}
                label="Block Common Exploits"
              />
            </FormControl>
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="tls_insecure_skip_verify"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between gap-4">
            <div className="space-y-0.5">
              <FormLabel>Allow Self-Signed Certificates</FormLabel>
              <FormDescription>
                Trust the backend server even if its certificate isn't from a public authority
              </FormDescription>
            </div>
            <FormControl>
              <InheritableSwitch
                key={groupId ?? 'no-group'}
                value={field.value}
                onChange={field.onChange}
                groupValue={group?.tls_insecure_skip_verify ?? null}
                systemDefault={PROXY_SYSTEM_DEFAULTS.tls_insecure_skip_verify}
                hasGroup={hasGroup}
                label="Allow Self-Signed Certificates"
              />
            </FormControl>
          </FormItem>
        )}
      />
    </div>
  );
}
