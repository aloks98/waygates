import {
  Button,
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
import { Plus, Trash2 } from 'lucide-react';
import { useFieldArray, useFormContext, useWatch } from 'react-hook-form';

import type { L4ProxyFormValues } from '@/lib/form-validation';
import { L4_LOAD_BALANCING_POLICIES } from '@/types/l4-proxy';
import type { L4LoadBalancingPolicy } from '@/types/l4-proxy';

// Load balancing policy display labels (ported from l4-proxy-form.tsx)
const LB_POLICY_LABELS: Record<L4LoadBalancingPolicy, string> = {
  round_robin: 'Round Robin — each server takes turns',
  least_conn: 'Least Connections — prefer less busy servers',
  random: 'Random',
  first: 'First Available — use the first server that responds',
  ip_hash: 'Sticky — same client always reaches same server',
};

interface L4UpstreamFieldsProps {
  routeIndex: number;
}

export function L4UpstreamFields({ routeIndex }: L4UpstreamFieldsProps) {
  const form = useFormContext<L4ProxyFormValues>();
  const name = `routes.${routeIndex}.upstreams` as const;
  const { fields, append, remove } = useFieldArray({ control: form.control, name });

  const upstreams = useWatch({ control: form.control, name });

  return (
    <div className="space-y-3">
      <FormLabel>Backend Servers</FormLabel>
      <FormDescription>Where to forward matched connections</FormDescription>

      {fields.map((item, i) => (
        <div key={item.id} className="flex items-start gap-2">
          <FormField
            control={form.control}
            name={`routes.${routeIndex}.upstreams.${i}.host` as never}
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormControl>
                  <Input placeholder="10.0.0.5 or backend.internal" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`routes.${routeIndex}.upstreams.${i}.port` as never}
            render={({ field }) => (
              <FormItem className="w-24">
                <FormControl>
                  <Input
                    type="number"
                    placeholder="8080"
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
          {fields.length > 1 && (
            <FormField
              control={form.control}
              name={`routes.${routeIndex}.upstreams.${i}.weight` as never}
              render={({ field }) => (
                <FormItem className="w-20">
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="weight"
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
          )}
          {fields.length > 1 && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Remove server"
              onClick={() => remove(i)}
            >
              <Trash2 className="size-4" />
            </Button>
          )}
        </div>
      ))}

      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => append({ host: '', port: 8080 })}
      >
        <Plus className="size-4" /> Add Server
      </Button>

      {/* Array-level sentinel to surface superRefine/min errors */}
      <FormField
        control={form.control}
        name={name}
        render={() => (
          <FormItem>
            <FormMessage />
          </FormItem>
        )}
      />

      {/* Load Balancing — only shown when multiple upstreams */}
      {upstreams && upstreams.length > 1 && (
        <FormField
          control={form.control}
          name={`routes.${routeIndex}.load_balancing_policy` as never}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Load Balancing Strategy</FormLabel>
              <FormControl>
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {L4_LOAD_BALANCING_POLICIES.map((policy) => (
                      <SelectItem key={policy} value={policy}>
                        {LB_POLICY_LABELS[policy]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormControl>
              <FormDescription>How to distribute connections across your servers</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
    </div>
  );
}
