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

// single source of truth (string values to match value={String(field.value)})
const REDIRECT_STATUSES = [
  { value: '301', label: '301 - Permanent' },
  { value: '302', label: '302 - Temporary' },
  { value: '307', label: '307 - Temporary (preserve method)' },
  { value: '308', label: '308 - Permanent (preserve method)' },
] as const;

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
            <Select
              items={REDIRECT_STATUSES}
              value={String(field.value)}
              onValueChange={(v) => field.onChange(Number(v))}
            >
              <FormControl>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                {REDIRECT_STATUSES.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
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
