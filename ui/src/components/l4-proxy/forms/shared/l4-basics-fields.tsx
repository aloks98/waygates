import { useFormContext } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { L4ProxyFormValues } from '@/lib/form-validation';

export function L4BasicsFields({ autoFocusName = false }: { autoFocusName?: boolean }) {
  const form = useFormContext<L4ProxyFormValues>();
  const protocols = [
    { value: 'tcp', label: 'TCP' },
    { value: 'udp', label: 'UDP' },
  ];
  return (
    <div className="flex flex-col gap-6">
      <FormField
        control={form.control}
        name="name"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Name</FormLabel>
            <FormControl>
              <Input placeholder="PostgreSQL" autoFocus={autoFocusName} {...field} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          control={form.control}
          name="protocol"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Protocol</FormLabel>
              <Select items={protocols} value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectGroup>
                    {protocols.map((p) => (
                      <SelectItem key={p.value} value={p.value}>
                        {p.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="listen_port"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Listen port</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  min={1}
                  max={65535}
                  className="font-mono"
                  {...field}
                  value={field.value ?? ''}
                  onChange={(e) =>
                    field.onChange(e.target.value === '' ? undefined : e.target.valueAsNumber)
                  }
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          control={form.control}
          name="routes.0.upstreams.0.host"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Destination host</FormLabel>
              <FormControl>
                <Input
                  className="font-mono"
                  placeholder="postgres"
                  {...field}
                  value={field.value ?? ''}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="routes.0.upstreams.0.port"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Destination port</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  min={1}
                  max={65535}
                  className="font-mono"
                  {...field}
                  value={field.value ?? ''}
                  onChange={(e) =>
                    field.onChange(e.target.value === '' ? undefined : e.target.valueAsNumber)
                  }
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
      <FormField
        control={form.control}
        name="is_active"
        render={({ field }) => (
          <FormItem orientation="horizontal" className="justify-between border-t pt-6">
            <FormLabel>Enabled</FormLabel>
            <FormControl>
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </FormItem>
        )}
      />
    </div>
  );
}
