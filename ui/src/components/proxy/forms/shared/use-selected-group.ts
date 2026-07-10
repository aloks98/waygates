import { useFormContext, useWatch } from 'react-hook-form';

import { useProxyGroups } from '@/hooks/use-proxy-groups';
import type { ProxyGroup } from '@/types/proxy-group';

interface GroupIdFormShape {
  group_id?: number | null;
}

/**
 * Resolves the ProxyGroup currently selected on the form (via the watched
 * `group_id` field) for components that need to show what a proxy would
 * inherit from it — e.g. InheritableSwitch's live "Inherit (on/off)" label.
 * Returns `{ group: null, hasGroup: false }` when ungrouped, so callers
 * re-render automatically whenever the user changes the group selection.
 * Also returns the raw `groupId`, so callers that need to key off it (e.g.
 * forcing InheritableSwitch to remount on group change) don't need a second
 * `useWatch` of their own.
 */
export function useSelectedGroup(): {
  groupId: number | null | undefined;
  group: ProxyGroup | null;
  hasGroup: boolean;
} {
  const form = useFormContext<GroupIdFormShape>();
  const groupId = useWatch({ control: form.control, name: 'group_id' });
  const { data } = useProxyGroups();
  const group = data?.items.find((g) => g.id === groupId) ?? null;
  return { groupId, group, hasGroup: !!group };
}
