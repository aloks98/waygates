import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
} from '@e412/rnui-react';
import { ChevronDown, ChevronUp, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';

import type { L4ProxyFormValues } from '@/lib/form-validation';
import { L4_MATCHER_CONFIG } from '@/types/l4-proxy';
import type { L4MatcherType } from '@/types/l4-proxy';

import { L4MatcherFields } from './l4-matcher-fields';
import { L4TlsFields } from './l4-tls-fields';
import { L4UpstreamFields } from './l4-upstream-fields';

export interface L4RouteCardProps {
  routeIndex: number;
  totalRoutes: number;
  onRemove: () => void;
  defaultOpen?: boolean;
}

export function L4RouteCard({
  routeIndex,
  totalRoutes,
  onRemove,
  defaultOpen = true,
}: L4RouteCardProps) {
  const [open, setOpen] = useState(defaultOpen);
  const form = useFormContext<L4ProxyFormValues>();

  const matcherType = useWatch({
    control: form.control,
    name: `routes.${routeIndex}.matcher_type` as never,
  }) as L4MatcherType | undefined;

  const upstreams = useWatch({
    control: form.control,
    name: `routes.${routeIndex}.upstreams` as never,
  }) as { host: string; port: number; weight?: number }[] | undefined;

  const upstreamCount = upstreams?.length ?? 0;

  return (
    <Card className="border-dashed">
      <CardHeader className="pb-2">
        <div className="flex items-center gap-2">
          <button
            type="button"
            className="flex flex-1 items-center gap-2 text-left"
            onClick={() => setOpen((prev) => !prev)}
            aria-expanded={open}
          >
            {open ? (
              <ChevronUp className="size-4 shrink-0" />
            ) : (
              <ChevronDown className="size-4 shrink-0" />
            )}
            <span className="text-base font-semibold">Route {routeIndex + 1}</span>
            {matcherType && (
              <Badge variant="outline" className="ml-1">
                {L4_MATCHER_CONFIG[matcherType]?.shortLabel ?? matcherType}
              </Badge>
            )}
            {upstreamCount > 0 && (
              <Badge variant="secondary" className="ml-1">
                {upstreamCount} upstream{upstreamCount !== 1 ? 's' : ''}
              </Badge>
            )}
          </button>
          {totalRoutes > 1 && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Remove route"
              onClick={onRemove}
            >
              <Trash2 className="size-4 text-destructive" />
            </Button>
          )}
        </div>
      </CardHeader>

      {open && (
        <CardContent className="space-y-4 pt-2">
          <L4MatcherFields routeIndex={routeIndex} />

          <L4UpstreamFields routeIndex={routeIndex} />

          {/* Priority — only shown when multiple routes */}
          {totalRoutes > 1 && (
            <FormField
              control={form.control}
              name={`routes.${routeIndex}.priority` as never}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Priority</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="0"
                      className="w-24"
                      {...field}
                      value={field.value ?? ''}
                      onChange={(e) =>
                        field.onChange(e.target.value === '' ? undefined : e.target.valueAsNumber)
                      }
                    />
                  </FormControl>
                  <FormDescription>Lower values are matched first (default: 0)</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          <L4TlsFields routeIndex={routeIndex} />
        </CardContent>
      )}
    </Card>
  );
}
