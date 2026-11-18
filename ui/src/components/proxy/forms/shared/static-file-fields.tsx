import { useFormContext } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import type { StaticFormValues } from '@/lib/form-validation';

export function StaticFileFields() {
  const form = useFormContext<StaticFormValues>();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="root_path"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Root Path</FormLabel>
            <FormControl>
              <Input className="font-mono" placeholder="/var/www/html" {...field} />
            </FormControl>

            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
