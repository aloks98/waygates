import { useFormContext } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import {
  Select,
  SelectGroup,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { L4ProxyFormValues } from '@/lib/form-validation';

// single source of truth driving both the trigger label and the dropdown options;
// "none" is the sentinel that maps to/from undefined (see onValueChange below)
const PROXY_PROTOCOLS = [
  { value: 'none', label: 'None' },
  { value: 'v1', label: 'Version 1' },
  { value: 'v2', label: 'Version 2' },
] as const;

interface L4TlsFieldsProps {
  routeIndex: number;
}

export function L4TlsFields({ routeIndex }: L4TlsFieldsProps) {
  const form = useFormContext<L4ProxyFormValues>();

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name={`routes.${routeIndex}.tls_terminate` as never}
        render={({ field }) => (
          <FormItem
            orientation="horizontal"
            className="flex flex-row items-center justify-between gap-4"
          >
            <div className="space-y-0.5">
              <FormLabel>TLS Termination</FormLabel>
            </div>
            <FormControl>
              <Switch
                checked={field.value}
                onCheckedChange={(v) => {
                  field.onChange(v);
                  if (v) {
                    form.setValue(`routes.${routeIndex}.tls_passthrough` as never, false as never, {
                      shouldValidate: true,
                    });
                  }
                }}
              />
            </FormControl>
            {/* The TLS-conflict superRefine error lands on tls_terminate */}
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name={`routes.${routeIndex}.tls_passthrough` as never}
        render={({ field }) => (
          <FormItem
            orientation="horizontal"
            className="flex flex-row items-center justify-between gap-4"
          >
            <div className="space-y-0.5">
              <FormLabel>TLS Passthrough</FormLabel>
            </div>
            <FormControl>
              <Switch
                checked={field.value}
                onCheckedChange={(v) => {
                  field.onChange(v);
                  if (v) {
                    form.setValue(`routes.${routeIndex}.tls_terminate` as never, false as never, {
                      shouldValidate: true,
                    });
                  }
                }}
              />
            </FormControl>
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name={`routes.${routeIndex}.proxy_protocol_version` as never}
        render={({ field }) => (
          <FormItem>
            <FormLabel>Proxy Protocol</FormLabel>
            <Select
              items={PROXY_PROTOCOLS}
              value={field.value ?? 'none'}
              onValueChange={(v) => field.onChange(v === 'none' ? undefined : v)}
            >
              <FormControl>
                <SelectTrigger className="w-48">
                  <SelectValue placeholder="None" />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectGroup>
                  {PROXY_PROTOCOLS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>

            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
