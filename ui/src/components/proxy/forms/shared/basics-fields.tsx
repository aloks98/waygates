import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
} from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

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
              <FormLabel>Hostname</FormLabel>
              <FormControl>
                <Input placeholder="app.example.com" {...field} />
              </FormControl>
              <FormDescription>The domain visitors will use to reach this service.</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
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
    </div>
  );
}
