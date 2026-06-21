import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

import type { RedirectFormValues } from '@/lib/form-validation';

export function RedirectTargetFields() {
  const form = useFormContext<RedirectFormValues>();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="target"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Target URL</FormLabel>
            <FormControl>
              <Input placeholder="https://new.example.com" {...field} />
            </FormControl>
            <FormDescription>The URL to redirect visitors to</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="status_code"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Redirect Type</FormLabel>
            <Select value={String(field.value)} onValueChange={(v) => field.onChange(Number(v))}>
              <FormControl>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectItem value="301">301 - Permanent</SelectItem>
                <SelectItem value="302">302 - Temporary</SelectItem>
                <SelectItem value="307">307 - Temporary (preserve method)</SelectItem>
                <SelectItem value="308">308 - Permanent (preserve method)</SelectItem>
              </SelectContent>
            </Select>
            <FormDescription>301/308 are cached by browsers, 302/307 are temporary</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
