import { useFormContext, useWatch } from 'react-hook-form';

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
import { TagsInput } from '@/components/ui/tags-input';
import type { L4ProxyFormValues } from '@/lib/form-validation';
import { L4_MATCHER_CONFIG, L4_MATCHER_TYPES } from '@/types/l4-proxy';

export function L4MatcherFields({ routeIndex }: { routeIndex: number }) {
  const form = useFormContext<L4ProxyFormValues>();
  const matcherType = useWatch({
    control: form.control,
    name: `routes.${routeIndex}.matcher_type`,
  });
  const items = L4_MATCHER_TYPES.map((m) => ({ value: m, label: L4_MATCHER_CONFIG[m].label }));
  return (
    <div className="flex flex-col gap-4">
      <FormField
        control={form.control}
        name={`routes.${routeIndex}.matcher_type`}
        render={({ field }) => (
          <FormItem>
            <FormLabel>Connection matcher</FormLabel>
            <Select items={items} value={field.value} onValueChange={field.onChange}>
              <FormControl>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectGroup>
                  {items.map((o) => (
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
      {matcherType === 'tls' && (
        <FormField
          control={form.control}
          name={`routes.${routeIndex}.sni_hostnames`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>SNI hostnames</FormLabel>
              <FormControl>
                <TagsInput
                  value={field.value?.map((v) => v.value) ?? []}
                  onValueChange={(values) => field.onChange(values.map((value) => ({ value })))}
                  placeholder="example.com"
                  className="font-mono"
                  delimiters={['Enter', ',']}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
      {matcherType === 'remote_ip' && (
        <FormField
          control={form.control}
          name={`routes.${routeIndex}.allowed_ip_ranges`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Allowed IP ranges</FormLabel>
              <FormControl>
                <TagsInput
                  value={field.value?.map((v) => v.value) ?? []}
                  onValueChange={(values) => field.onChange(values.map((value) => ({ value })))}
                  placeholder="10.0.0.0/24"
                  className="font-mono"
                  delimiters={['Enter', ',']}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
      {matcherType === 'regexp' && (
        <FormField
          control={form.control}
          name={`routes.${routeIndex}.regex_pattern`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Pattern</FormLabel>
              <FormControl>
                <Input className="font-mono" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
    </div>
  );
}
