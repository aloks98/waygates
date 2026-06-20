import { Button, StatusIndicator, Tooltip, TooltipContent, TooltipTrigger } from '@e412/rnui-react';
import { formatDistanceToNow } from 'date-fns';

import { useSyncStatus } from '@/hooks/use-dashboard';

export function SyncStatus() {
  const { syncStatus, isSyncing, triggerSync } = useSyncStatus();

  const state = isSyncing
    ? 'fixing'
    : syncStatus?.last_error
      ? 'down'
      : syncStatus?.last_sync_success
        ? 'active'
        : 'idle';

  const label = isSyncing
    ? 'Syncing…'
    : syncStatus?.last_error
      ? 'Sync failed'
      : syncStatus?.last_sync_success
        ? 'Synced'
        : 'Not synced';

  const tooltip = isSyncing
    ? 'Applying configuration to Caddy…'
    : syncStatus?.last_error
      ? `Last sync failed: ${syncStatus.last_error}`
      : syncStatus?.last_sync_time
        ? `Last synced ${formatDistanceToNow(new Date(syncStatus.last_sync_time), { addSuffix: true })}. Click to re-apply.`
        : syncStatus?.last_sync_success
          ? 'Synced. Click to re-apply.'
          : 'Not yet synced. Click to apply now.';

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className="gap-2"
            disabled={isSyncing}
            onClick={() => triggerSync()}
          />
        }
      >
        <StatusIndicator state={state} size="sm" label={label} />
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}
