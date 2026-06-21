import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  Switch,
} from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

import type { ReverseProxyFormValues } from '@/lib/form-validation';

export function SecurityFields() {
  const form = useFormContext<ReverseProxyFormValues>();

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
              <Switch checked={field.value} onCheckedChange={field.onChange} />
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
              <Switch checked={field.value} onCheckedChange={field.onChange} />
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
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </FormItem>
        )}
      />
    </div>
  );
}
