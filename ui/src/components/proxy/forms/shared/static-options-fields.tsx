import { useFormContext } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel } from '@/components/ui/form';
import { Switch } from '@/components/ui/switch';
import type { StaticFormValues } from '@/lib/form-validation';

export function StaticOptionsFields() {
  const form = useFormContext<StaticFormValues>();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="browse"
        render={({ field }) => (
          <FormItem orientation="horizontal" className="flex flex-row items-center justify-between">
            <div className="space-y-0.5">
              <FormLabel>Directory Browsing</FormLabel>
            </div>
            <FormControl>
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="template_rendering"
        render={({ field }) => (
          <FormItem orientation="horizontal" className="flex flex-row items-center justify-between">
            <div className="space-y-0.5">
              <FormLabel>Template Rendering</FormLabel>
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
