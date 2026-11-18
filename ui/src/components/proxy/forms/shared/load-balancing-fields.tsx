import { useFormContext, useWatch } from 'react-hook-form';

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectGroup,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { ReverseProxyFormValues } from '@/lib/form-validation';

// single source of truth: concise `label` drives the trigger, `description` the dropdown hint
const LB_STRATEGIES = [
  { value: 'round_robin', label: 'Round Robin', description: 'each server takes turns' },
  { value: 'least_conn', label: 'Least Connections', description: 'prefer less busy servers' },
  { value: 'ip_hash', label: 'Sticky', description: 'same visitor always reaches same server' },
  { value: 'random', label: 'Random' },
] as const;

export function LoadBalancingFields() {
  const form = useFormContext<ReverseProxyFormValues>();

  const upstreams = useWatch({ control: form.control, name: 'upstreams' });
  const healthCheckEnabled = useWatch({ control: form.control, name: 'health_check_enabled' });

  if (!upstreams || upstreams.length <= 1) return null;

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name="lb_strategy"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Strategy</FormLabel>
            <Select items={LB_STRATEGIES} value={field.value} onValueChange={field.onChange}>
              <FormControl>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectGroup>
                  {LB_STRATEGIES.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {'description' in o && o.description
                        ? `${o.label} — ${o.description}`
                        : o.label}
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
        name="health_check_enabled"
        render={({ field }) => (
          <FormItem
            orientation="horizontal"
            className="flex flex-row items-center justify-between gap-4"
          >
            <div className="space-y-0.5">
              <FormLabel>Health Checks</FormLabel>
            </div>
            <FormControl>
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </FormItem>
        )}
      />

      {healthCheckEnabled && (
        <div className="space-y-4">
          <FormField
            control={form.control}
            name="health_check_path"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Path</FormLabel>
                <FormControl>
                  <Input placeholder="/health" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="grid grid-cols-2 gap-4">
            <FormField
              control={form.control}
              name="health_check_interval"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Check Every</FormLabel>
                  <FormControl>
                    <Input placeholder="30s" {...field} />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="health_check_timeout"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Timeout</FormLabel>
                  <FormControl>
                    <Input placeholder="5s" {...field} />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
      )}
    </div>
  );
}
