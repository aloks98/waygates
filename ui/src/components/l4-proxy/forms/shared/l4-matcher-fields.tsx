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
import { L4_MATCHER_CONFIG, L4_MATCHER_TYPES } from '@/types/l4-proxy';
import type { L4MatcherType } from '@/types/l4-proxy';

interface L4MatcherFieldsProps {
  routeIndex: number;
}

export function L4MatcherFields({ routeIndex }: L4MatcherFieldsProps) {
  const form = useFormContext<L4ProxyFormValues>();
  const matcherType = useWatch({
    control: form.control,
    name: `routes.${routeIndex}.matcher_type` as never,
  });

  const sni = useFieldArray({
    control: form.control,
    name: `routes.${routeIndex}.sni_hostnames` as never,
  });
  const ip = useFieldArray({
    control: form.control,
    name: `routes.${routeIndex}.allowed_ip_ranges` as never,
  });

  return (
    <div className="space-y-4">
      <FormField
        control={form.control}
        name={`routes.${routeIndex}.matcher_type` as never}
        render={({ field }) => (
          <FormItem>
            <FormLabel>Connection Matcher</FormLabel>
            <FormControl>
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {L4_MATCHER_TYPES.map((m) => (
                    <SelectItem key={m} value={m}>
                      {L4_MATCHER_CONFIG[m].label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormControl>
            {matcherType && (
              <FormDescription>
                {L4_MATCHER_CONFIG[matcherType as L4MatcherType]?.description}
              </FormDescription>
            )}
            <FormMessage />
          </FormItem>
        )}
      />

      {matcherType === 'tls' && (
        <div className="space-y-2 rounded-lg border p-4 bg-muted/30">
          <div className="flex items-center justify-between">
            <FormLabel>SNI Hostnames</FormLabel>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => sni.append({ value: '' })}
            >
              <Plus className="size-4" /> Add Hostname
            </Button>
          </div>
          <FormDescription>
            Match TLS connections by the hostname in the TLS handshake (Server Name Indication)
          </FormDescription>
          {sni.fields.map((item, i) => (
            <div key={item.id} className="flex items-start gap-2">
              <FormField
                control={form.control}
                name={`routes.${routeIndex}.sni_hostnames.${i}.value` as never}
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormControl>
                      <Input placeholder="example.com" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Remove hostname"
                onClick={() => sni.remove(i)}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          ))}
          {/* Array-level sentinel to surface superRefine/min errors */}
          <FormField
            control={form.control}
            name={`routes.${routeIndex}.sni_hostnames` as never}
            render={() => (
              <FormItem>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      )}

      {matcherType === 'remote_ip' && (
        <div className="space-y-2 rounded-lg border p-4 bg-muted/30">
          <div className="flex items-center justify-between">
            <FormLabel>Allowed IP Ranges</FormLabel>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => ip.append({ value: '' })}
            >
              <Plus className="size-4" /> Add IP Range
            </Button>
          </div>
          <FormDescription>
            Only allow connections from these IP addresses or CIDR ranges
          </FormDescription>
          {ip.fields.map((item, i) => (
            <div key={item.id} className="flex items-start gap-2">
              <FormField
                control={form.control}
                name={`routes.${routeIndex}.allowed_ip_ranges.${i}.value` as never}
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormControl>
                      <Input placeholder="10.0.0.0/24 or 192.168.1.1" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Remove IP range"
                onClick={() => ip.remove(i)}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          ))}
          {/* Array-level sentinel to surface superRefine/min errors */}
          <FormField
            control={form.control}
            name={`routes.${routeIndex}.allowed_ip_ranges` as never}
            render={() => (
              <FormItem>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      )}

      {matcherType === 'regexp' && (
        <FormField
          control={form.control}
          name={`routes.${routeIndex}.regex_pattern` as never}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Regex Pattern</FormLabel>
              <FormControl>
                <Input placeholder="^example\\." {...field} />
              </FormControl>
              <FormDescription>
                Regular expression to match against the first bytes of data
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      )}
    </div>
  );
}
