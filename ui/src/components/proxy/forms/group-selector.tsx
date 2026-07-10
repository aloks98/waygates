import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
} from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { ExternalLink, Layers } from 'lucide-react';
import { useFormContext } from 'react-hook-form';

import { useProxyGroups } from '@/hooks/use-proxy-groups';

const NONE_VALUE = 'none';

interface GroupSelectorFormShape {
  group_id?: number | null;
}

/**
 * Assigns the proxy to a ProxyGroup (config inheritance) — never an ACLGroup
 * (an auth grouping; see acl-selector.tsx, which this mirrors structurally:
 * a self-contained field backed by its own fetch, with an empty-state link
 * to create the referenced resource first).
 */
export function GroupSelector() {
  const form = useFormContext<GroupSelectorFormShape>();
  const { data, isLoading } = useProxyGroups();
  const groups = data?.items ?? [];

  return (
    <FormField
      control={form.control}
      name="group_id"
      render={({ field }) => (
        <FormItem>
          <FormLabel>Proxy Group</FormLabel>
          {isLoading ? (
            <Skeleton className="h-9 w-full" />
          ) : (
            <Select
              value={field.value ? String(field.value) : NONE_VALUE}
              onValueChange={(next) => field.onChange(next === NONE_VALUE ? null : Number(next))}
            >
              <FormControl>
                <SelectTrigger>
                  <SelectValue placeholder="No group" />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectItem value={NONE_VALUE}>No group</SelectItem>
                {groups.map((group) => (
                  <SelectItem key={group.id} value={String(group.id)}>
                    <div className="flex items-center gap-2">
                      <Layers className="size-4 text-muted-foreground" />
                      <span>{group.name}</span>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          <FormDescription>
            Members inherit HTTPS, security, header, and access settings the group has an opinion
            on; anything left to "Inherit" here falls through to what the group sets.
            {!isLoading && groups.length === 0 && (
              <>
                {' '}
                <Link
                  to="/proxy-groups"
                  className="text-primary hover:underline inline-flex items-center gap-1"
                >
                  Create a proxy group
                  <ExternalLink className="size-3" />
                </Link>
              </>
            )}
          </FormDescription>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}
