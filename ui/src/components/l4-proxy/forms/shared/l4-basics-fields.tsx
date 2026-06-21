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
  Switch,
} from '@e412/rnui-react';
import { useFormContext } from 'react-hook-form';

import type { L4ProxyFormValues } from '@/lib/form-validation';
import { L4_PROTOCOLS } from '@/types/l4-proxy';

interface L4BasicsFieldsProps {
  autoFocusName?: boolean;
}

export function L4BasicsFields({ autoFocusName = false }: L4BasicsFieldsProps) {
  const form = useFormContext<L4ProxyFormValues>();

  return (
    <div className="space-y-4">
      {/* items-start: FormItem is display:grid; without it the shorter cell stretches
          to the taller sibling and the inputs misalign. */}
      <div className="grid items-start gap-4 sm:grid-cols-2">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input placeholder="MySQL Proxy" autoFocus={autoFocusName} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="description"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Description (optional)</FormLabel>
              <FormControl>
                <Input placeholder="Proxy for database connections" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>

      <div className="grid items-start gap-4 sm:grid-cols-3">
        <FormField
          control={form.control}
          name="listen_port"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Listen Port</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  placeholder="3306"
                  {...field}
                  value={field.value ?? ''}
                  onChange={(e) =>
                    field.onChange(e.target.value === '' ? undefined : e.target.valueAsNumber)
                  }
                />
              </FormControl>
              <FormDescription>Port to listen on (1-65535)</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="protocol"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Protocol</FormLabel>
              <FormControl>
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {L4_PROTOCOLS.map((protocol) => (
                      <SelectItem key={protocol} value={protocol}>
                        {protocol.toUpperCase()}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormControl>
              <FormDescription>TCP or UDP</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="is_active"
          render={({ field }) => (
            <FormItem className="flex flex-row items-center justify-between gap-4 sm:pt-6">
              <div className="space-y-0.5">
                <FormLabel>Active</FormLabel>
                <FormDescription>Enable this proxy</FormDescription>
              </div>
              <FormControl>
                <Switch checked={field.value} onCheckedChange={field.onChange} />
              </FormControl>
            </FormItem>
          )}
        />
      </div>
    </div>
  );
}
