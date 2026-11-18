import { formatDistanceToNow } from 'date-fns';

import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useSyncStatus } from '@/hooks/use-dashboard';

export function SyncStatus() {
  const { syncStatus, isSyncing, triggerSync } = useSyncStatus();

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
            aria-label={label}
            variant="ghost"
            size="sm"
            className="gap-2"
            disabled={isSyncing}
            onClick={() => triggerSync()}
          />
        }
      >
        <span
          className={`size-1.5 rounded-full ${syncStatus?.last_error ? 'bg-destructive' : isSyncing ? 'bg-warning' : syncStatus?.last_sync_success ? 'bg-success' : 'bg-muted-foreground'}`}
        />
        <span className="hidden text-xs text-muted-foreground sm:inline">{label}</span>
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}
