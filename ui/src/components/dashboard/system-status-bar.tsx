import {
  Button,
  Skeleton,
  StatusIndicator,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/rnui-react';
import { formatDistanceToNow } from 'date-fns';
import { RefreshCw } from 'lucide-react';

import { useAppStatus, useHealthStatus, useSyncStatus } from '@/hooks';
import { formatUptime } from '@/lib/dashboard-format';

export function SystemStatusBar() {
  const { syncStatus, triggerSync, isSyncing, isLoading: sl } = useSyncStatus();
  const { health, isLoading: hl } = useHealthStatus();
  const { appStatus, isLoading: al } = useAppStatus();

  if (sl || hl || al) {
    return (
      <div className="flex flex-wrap items-center gap-4">
        <Skeleton className="h-5 w-28" />
        <Skeleton className="h-5 w-28" />
        <Skeleton className="h-5 w-32" />
      </div>
    );
  }

  const caddy = appStatus?.caddy_status === 'healthy' ? 'active' : 'down';
  const db = health?.components?.database === 'healthy' ? 'active' : 'down';
  const syncState = isSyncing
    ? 'fixing'
    : syncStatus?.last_sync_success === false
      ? 'down'
      : 'active';
  const lastSync = syncStatus?.last_sync_time
    ? formatDistanceToNow(new Date(syncStatus.last_sync_time), { addSuffix: true })
    : 'never';

  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
      <StatusIndicator state={caddy} size="sm" label="Caddy" />
      <StatusIndicator state={db} size="sm" label="Database" />
      <Tooltip>
        <TooltipTrigger render={<span />}>
          <StatusIndicator
            state={syncState}
            size="sm"
            label={isSyncing ? 'Syncing…' : `Synced ${lastSync}`}
          />
        </TooltipTrigger>
        <TooltipContent>
          {isSyncing
            ? 'Applying configuration to Caddy…'
            : syncStatus?.last_sync_success === false
              ? `Last sync failed${syncStatus?.last_error ? `: ${syncStatus.last_error}` : ''}`
              : 'Configuration is applied'}
        </TooltipContent>
      </Tooltip>
      {health?.uptime && (
        <span className="text-muted-foreground">Up {formatUptime(health.uptime)}</span>
      )}
      {health?.version && <span className="text-muted-foreground">v{health.version}</span>}
      <Button
        variant="outline"
        size="sm"
        className="ml-auto gap-1.5"
        disabled={isSyncing}
        onClick={() => triggerSync()}
      >
        <RefreshCw className={`size-3.5 ${isSyncing ? 'animate-spin' : ''}`} />
        Apply now
      </Button>
    </div>
  );
}
