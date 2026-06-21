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
              <Input placeholder="/var/www/html" {...field} />
            </FormControl>
            <FormDescription>The directory path to serve files from</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="index_file"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Index File</FormLabel>
            <FormControl>
              <Input placeholder="index.html" {...field} />
            </FormControl>
            <FormDescription>Default file to serve for directory requests</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
