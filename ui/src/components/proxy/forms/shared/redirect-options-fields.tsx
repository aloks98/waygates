import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  Switch,
} from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

import type { RedirectFormValues } from '@/lib/form-validation';

export function RedirectOptionsFields() {
  const form = useFormContext<RedirectFormValues>();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="ssl_enabled"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between">
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
        name="preserve_path"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between">
            <div className="space-y-0.5">
              <FormLabel>Preserve Path</FormLabel>
              <FormDescription>Append the original path to the target URL</FormDescription>
            </div>
            <FormControl>
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="preserve_query"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between">
            <div className="space-y-0.5">
              <FormLabel>Preserve Query String</FormLabel>
              <FormDescription>
                Append the original query parameters to the target URL
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
