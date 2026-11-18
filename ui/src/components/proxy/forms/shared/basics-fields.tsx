import { useFormContext } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';

export function BasicsFields({ autoFocusName = false }: { autoFocusName?: boolean }) {
  // BasicsFields only touches name/hostname/description, present on all 3 schemas.
  const form = useFormContext();
  return (
    <div className="space-y-4">
      {/* items-start: FormItem is display:grid; without this the shorter Name cell
          stretches to the taller Hostname cell (which has a description), misaligning inputs. */}
      <div className="grid items-start gap-4 sm:grid-cols-2">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input autoFocus={autoFocusName} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="hostname"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Domain</FormLabel>
              <FormControl>
                <Input className="font-mono" placeholder="app.example.com" {...field} />
              </FormControl>

              <FormMessage />
            </FormItem>
          )}
        />
      </div>
    </div>
  );
}
