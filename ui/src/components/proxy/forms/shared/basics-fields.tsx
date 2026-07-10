import { FormControl, FormField, FormItem, FormLabel, FormMessage, Input } from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

import { GroupSelector } from '../group-selector';
import { HostnameField } from './hostname-field';

export function BasicsFields({ autoFocusName = false }: { autoFocusName?: boolean }) {
  // BasicsFields only touches name/hostname/description/group_id/hostname_label,
  // present on all 3 schemas.
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
        <HostnameField />
      </div>
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
      <GroupSelector />
    </div>
  );
}
