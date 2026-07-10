import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
} from '@e412/rnui-react';
import { useEffect } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';

import { useProxyGroups } from '@/hooks/use-proxy-groups';

// Moving into a base-domain group: pre-fill the label by stripping the
// suffix when the current hostname already sits under it, so the common
// case is a no-op rather than a retype. A hostname exactly equal to the
// base domain (no separating label) is not a valid label and yields ''.
export function deriveLabel(hostname: string, baseDomain: string): string {
  const suffix = `.${baseDomain}`;
  return hostname.endsWith(suffix) ? hostname.slice(0, -suffix.length) : '';
}

interface HostnameFormShape {
  hostname: string;
  hostname_label?: string | null;
  group_id?: number | null;
}

/**
 * When the selected group has a base_domain, narrows to a single-label
 * input with the base domain rendered as a static suffix; otherwise a
 * normal hostname input. The underlying RHF `hostname` field always holds
 * the full composed value in either mode, so the existing "Hostname is
 * required" validation on it also covers an empty label — no separate
 * schema rule needed. `hostname_label` is kept in sync purely so the wire
 * request carries it explicitly, per the API contract.
 */
export function HostnameField() {
  const form = useFormContext<HostnameFormShape>();
  const groupId = useWatch({ control: form.control, name: 'group_id' });
  const { data, isLoading: groupsLoading } = useProxyGroups();
  const group = data?.items.find((g) => g.id === groupId) ?? null;
  const baseDomain = group?.base_domain || null;

  // Re-derive the label/hostname pairing whenever the selected group (or its
  // base_domain) changes — this is the one place the fields get synced
  // programmatically; user keystrokes sync them directly (see onChange below).
  // Skip while the groups list is still loading: groupId can already be set
  // (e.g. from an async initialData reset) before the list resolves, and
  // `group`/`baseDomain` would be transiently null — clobbering a correctly
  // loaded hostname_label instead of just waiting one more render.
  useEffect(() => {
    if (groupsLoading) return;
    if (!baseDomain) {
      if (form.getValues('hostname_label')) {
        form.setValue('hostname_label', null);
      }
      return;
    }
    const currentHostname = form.getValues('hostname') || '';
    const label = deriveLabel(currentHostname, baseDomain);
    form.setValue('hostname_label', label || null);
    form.setValue('hostname', label ? `${label}.${baseDomain}` : '');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groupId, baseDomain, groupsLoading]);

  if (!baseDomain) {
    return (
      <FormField
        control={form.control}
        name="hostname"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Hostname</FormLabel>
            <FormControl>
              <Input placeholder="app.example.com" {...field} />
            </FormControl>
            <FormDescription>The domain visitors will use to reach this service.</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    );
  }

  return (
    <FormField
      control={form.control}
      name="hostname"
      render={({ field }) => {
        const label = deriveLabel(field.value || '', baseDomain);
        return (
          <FormItem>
            <FormLabel>Hostname</FormLabel>
            <FormControl>
              <div className="flex items-center gap-2">
                <Input
                  value={label}
                  onChange={(e) => {
                    const next = e.target.value;
                    form.setValue('hostname_label', next || null);
                    field.onChange(next ? `${next}.${baseDomain}` : '');
                  }}
                  onBlur={field.onBlur}
                  placeholder="app"
                  className="flex-1"
                />
                <span className="text-sm text-muted-foreground whitespace-nowrap">
                  .{baseDomain}
                </span>
              </div>
            </FormControl>
            <FormDescription>
              Single label — the group supplies the rest of the domain.
            </FormDescription>
            <FormMessage />
          </FormItem>
        );
      }}
    />
  );
}
