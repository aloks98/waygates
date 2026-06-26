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

// single source of truth (ported from l4-proxy-form.tsx): concise `label` drives the
// collapsed trigger, the longer `description` is appended in the dropdown rows
const LB_POLICIES = [
  { value: 'round_robin', label: 'Round Robin', description: 'each server takes turns' },
  { value: 'least_conn', label: 'Least Connections', description: 'prefer less busy servers' },
  { value: 'random', label: 'Random' },
  { value: 'first', label: 'First Available', description: 'use the first server that responds' },
  { value: 'ip_hash', label: 'Sticky', description: 'same client always reaches same server' },
] as const;

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
                <Select items={LB_POLICIES} value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {LB_POLICIES.map((o) => (
                      <SelectItem key={o.value} value={o.value}>
                        {'description' in o && o.description
                          ? `${o.label} — ${o.description}`
                          : o.label}
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
