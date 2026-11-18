import { useFormContext } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel } from '@/components/ui/form';
import { Switch } from '@/components/ui/switch';
import type { RedirectFormValues } from '@/lib/form-validation';

export function RedirectOptionsFields() {
  const form = useFormContext<RedirectFormValues>();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="preserve_path"
        render={({ field }) => (
          <FormItem orientation="horizontal" className="flex flex-row items-center justify-between">
            <div className="space-y-0.5">
              <FormLabel>Preserve Path</FormLabel>
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
          <FormItem orientation="horizontal" className="flex flex-row items-center justify-between">
            <div className="space-y-0.5">
              <FormLabel>Preserve Query String</FormLabel>
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
