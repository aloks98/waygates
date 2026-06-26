import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Skeleton,
  StatusIndicator,
} from '@e412/rnui-react';
import { formatDistanceToNow } from 'date-fns';
import type { ReactNode } from 'react';

import { useAppStatus, useHealthStatus, useSyncStatus } from '@/hooks';
import { formatUptime } from '@/lib/dashboard-format';

function StatusRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="text-sm font-medium">{children}</div>
    </div>
  );
}

export function SystemStatus() {
  const { syncStatus, isSyncing, isLoading: sl } = useSyncStatus();
  const { health, isLoading: hl } = useHealthStatus();
  const { appStatus, isLoading: al } = useAppStatus();

  const isLoading = sl || hl || al;

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
    <Card>
      <CardHeader>
        <CardTitle>System Status</CardTitle>
        <CardDescription>
          Live health of the Waygates services. Configuration is applied automatically when you save
          changes; use the sync indicator in the top bar to re-apply manually.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-5 w-2/3" />
            <Skeleton className="h-5 w-1/2" />
          </div>
        ) : (
          <div className="divide-y divide-border">
            <StatusRow label="Caddy">
              <StatusIndicator
                state={caddy}
                size="sm"
                label={caddy === 'active' ? 'Running' : 'Down'}
              />
            </StatusRow>
            <StatusRow label="Database">
              <StatusIndicator
                state={db}
                size="sm"
                label={db === 'active' ? 'Connected' : 'Unavailable'}
              />
            </StatusRow>
            <StatusRow label="Configuration">
              <StatusIndicator
                state={syncState}
                size="sm"
                label={
                  isSyncing
                    ? 'Applying…'
                    : syncStatus?.last_sync_success === false
                      ? `Sync failed${syncStatus?.last_error ? `: ${syncStatus.last_error}` : ''}`
                      : `Synced ${lastSync}`
                }
              />
            </StatusRow>
            {health?.uptime && <StatusRow label="Uptime">{formatUptime(health.uptime)}</StatusRow>}
            {health?.version && (
              <StatusRow label="Version">
                <span className="font-mono">v{health.version}</span>
              </StatusRow>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
