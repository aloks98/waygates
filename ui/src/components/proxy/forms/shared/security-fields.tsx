import { useFormContext } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel } from '@/components/ui/form';
import { Switch } from '@/components/ui/switch';
import type { ReverseProxyFormValues } from '@/lib/form-validation';

export function SecurityFields() {
  const form = useFormContext<ReverseProxyFormValues>();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="block_exploits"
        render={({ field }) => (
          <FormItem
            orientation="horizontal"
            className="flex flex-row items-center justify-between gap-4"
          >
            <div className="space-y-0.5">
              <FormLabel>Block Common Exploits</FormLabel>
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
          <FormItem
            orientation="horizontal"
            className="flex flex-row items-center justify-between gap-4"
          >
            <div className="space-y-0.5">
              <FormLabel>Allow Self-Signed Certificates</FormLabel>
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
