import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardHeading,
  CardTitle,
  CardToolbar,
  Skeleton,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/titanium';
import { Link } from '@tanstack/react-router';
import { formatDistanceToNow, formatDuration, intervalToDuration } from 'date-fns';
import {
  Activity,
  ArrowRight,
  CheckCircle2,
  Globe,
  Network,
  Plus,
  RefreshCw,
  Server,
  XCircle,
} from 'lucide-react';
import { useAppStatus, useDashboardData, useHealthStatus, useSyncStatus } from '@/hooks';

function formatUptime(raw: string): string {
  // Parse Go duration string (e.g., "173h33m10.048s") into total seconds
  const hours = raw.match(/(\d+)h/);
  const minutes = raw.match(/(\d+)m/);
  const seconds = raw.match(/([\d.]+)s/);
  const totalSeconds =
    (hours ? Number.parseInt(hours[1], 10) * 3600 : 0) +
    (minutes ? Number.parseInt(minutes[1], 10) * 60 : 0) +
    (seconds ? Math.floor(Number.parseFloat(seconds[1])) : 0);

  const duration = intervalToDuration({ start: 0, end: totalSeconds * 1000 });
  return formatDuration(duration, { format: ['days', 'hours', 'minutes'], delimiter: ' ' });
}

import { useAuthStore } from '@/stores/auth';
import type { AuditLog } from '@/types/audit';

function getActionLabel(action: string): string {
  const labels: Record<string, string> = {
    'proxy.create': 'created a proxy',
    'proxy.update': 'updated a proxy',
    'proxy.delete': 'deleted a proxy',
    'proxy.enable': 'enabled a proxy',
    'proxy.disable': 'disabled a proxy',
    'auth.login': 'signed in',
    'auth.logout': 'signed out',
    'auth.register': 'registered',
    'auth.password_change': 'changed password',
    'auth.login_failed': 'failed login attempt',
    'settings.update': 'updated settings',
    'sync.started': 'sync started',
    'sync.completed': 'sync completed',
    'sync.failed': 'sync failed',
    'system.startup': 'system started',
    'caddy.reload': 'proxy server reloaded',
    'acl_group.create': 'created ACL group',
    'acl_group.update': 'updated ACL group',
    'acl_group.delete': 'deleted ACL group',
  };
  return labels[action] ?? action.replace(/[._]/g, ' ');
}

function getActionColor(action: string): string {
  if (action.includes('delete') || action.includes('failed')) return 'text-destructive';
  if (action.includes('create') || action.includes('enable') || action === 'sync.completed')
    return 'text-green-600 dark:text-green-500';
  if (action.includes('disable')) return 'text-amber-600 dark:text-amber-500';
  return 'text-muted-foreground';
}

function StatusDot({ status }: { status: 'healthy' | 'unhealthy' | 'degraded' | 'unknown' }) {
  const colors = {
    healthy: 'bg-green-500',
    unhealthy: 'bg-destructive',
    degraded: 'bg-amber-500',
    unknown: 'bg-muted-foreground',
  };
  return (
    <span className="relative flex size-2.5">
      {status === 'healthy' && (
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
      )}
      <span className={`relative inline-flex size-2.5 rounded-full ${colors[status]}`} />
    </span>
  );
}

function SystemStatusStrip() {
  const { syncStatus, isLoading: isSyncLoading, triggerSync, isSyncing } = useSyncStatus();
  const { health, isLoading: isHealthLoading } = useHealthStatus();
  const { appStatus, isLoading: isAppLoading } = useAppStatus();

  const isLoading = isSyncLoading || isHealthLoading || isAppLoading;

  if (isLoading) {
    return (
      <div className="flex items-center gap-5">
        <Skeleton className="h-5 w-24" />
        <Skeleton className="h-5 w-20" />
        <Skeleton className="h-5 w-28" />
        <Skeleton className="h-5 w-16" />
      </div>
    );
  }

  const caddyHealthy = appStatus?.caddy_status === 'healthy';
  const dbHealthy = health?.components?.database === 'healthy';
  const lastSync = syncStatus?.last_sync_time
    ? formatDistanceToNow(new Date(syncStatus.last_sync_time), { addSuffix: true })
    : 'never';

  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
      {/* Proxy server status */}
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-2">
            <StatusDot status={caddyHealthy ? 'healthy' : 'unhealthy'} />
            <span className="text-muted-foreground">Proxy Server</span>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          {caddyHealthy ? 'Proxy server is running (Caddy)' : 'Proxy server is down (Caddy)'}
        </TooltipContent>
      </Tooltip>

      {/* Database status */}
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-2">
            <StatusDot status={dbHealthy ? 'healthy' : 'unhealthy'} />
            <span className="text-muted-foreground">Database</span>
          </div>
        </TooltipTrigger>
        <TooltipContent>{dbHealthy ? 'Database connected' : 'Database unreachable'}</TooltipContent>
      </Tooltip>

      {/* Sync status */}
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={() => triggerSync()}
            disabled={isSyncing}
            className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`size-3.5 ${isSyncing ? 'animate-spin' : ''}`} />
            <span>{isSyncing ? 'Syncing...' : `Synced ${lastSync}`}</span>
            {syncStatus && !syncStatus.last_sync_success && (
              <XCircle className="size-3.5 text-destructive" />
            )}
          </button>
        </TooltipTrigger>
        <TooltipContent>
          {isSyncing
            ? 'Applying proxy config to the server...'
            : syncStatus?.last_sync_success
              ? 'Click to apply proxy config now'
              : `Last sync failed${syncStatus?.last_error ? `: ${syncStatus.last_error}` : ''}`}
        </TooltipContent>
      </Tooltip>

      {/* Uptime */}
      {health?.uptime && (
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Server className="size-3.5" />
              <span>Up {formatUptime(health.uptime)}</span>
            </div>
          </TooltipTrigger>
          <TooltipContent>Service uptime</TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}

function ProxyOverview({
  proxyStats,
  l4ProxyStats,
  recentProxies,
  isLoading,
}: {
  proxyStats: ReturnType<typeof useDashboardData>['proxyStats'];
  l4ProxyStats: ReturnType<typeof useDashboardData>['l4ProxyStats'];
  recentProxies: ReturnType<typeof useDashboardData>['recentProxies'];
  isLoading: boolean;
}) {
  const totalL7 = proxyStats?.total ?? 0;
  const activeL7 = proxyStats?.active ?? 0;
  const totalL4 = l4ProxyStats?.total_proxies ?? 0;
  const activeL4 = l4ProxyStats?.active_proxies ?? 0;
  const totalAll = totalL7 + totalL4;

  if (isLoading) {
    return (
      <Card className="flex-1">
        <CardHeader>
          <Skeleton className="h-5 w-24" />
        </CardHeader>
        <CardContent className="space-y-2">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </CardContent>
      </Card>
    );
  }

  if (totalAll === 0) {
    return (
      <Card className="flex-1">
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <div className="rounded bg-muted p-4">
            <Globe className="size-8 text-muted-foreground" />
          </div>
          <h3 className="mt-4 text-base font-medium">No proxies yet</h3>
          <p className="mt-1.5 text-sm text-muted-foreground max-w-[260px]">
            Create your first proxy to start routing traffic through Waygates.
          </p>
          <div className="mt-4 flex gap-2">
            <Button size="sm" asChild>
              <Link to="/dashboard/proxies/new">
                <Plus className="size-4" />
                HTTP Proxy
              </Link>
            </Button>
            <Button size="sm" variant="outline" asChild>
              <Link to="/dashboard/l4-proxies/new">
                <Plus className="size-4" />
                TCP/UDP Proxy
              </Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="flex-1">
      <CardHeader className="pb-3">
        <CardHeading>
          <CardTitle className="text-base">Proxies</CardTitle>
        </CardHeading>
        <CardToolbar>
          <span className="text-sm text-muted-foreground">
            <span className="font-medium text-foreground">{activeL7 + activeL4}</span> active of{' '}
            <span className="font-medium text-foreground">{totalAll}</span>
          </span>
        </CardToolbar>
      </CardHeader>
      <CardContent className="space-y-1 pt-0">
        {/* HTTP proxies summary */}
        {totalL7 > 0 && (
          <Link
            to="/dashboard/proxies"
            className="flex items-center justify-between rounded-md px-3 py-2.5 hover:bg-muted/50 transition-colors group"
          >
            <div className="flex items-center gap-3">
              <Globe className="size-4 text-muted-foreground" />
              <div>
                <span className="text-sm font-medium">HTTP Proxies</span>
                <span className="text-xs text-muted-foreground ml-2">
                  {activeL7} active, {totalL7 - activeL7} inactive
                </span>
              </div>
            </div>
            <ArrowRight className="size-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
          </Link>
        )}

        {/* L4 proxies summary */}
        {totalL4 > 0 && (
          <Link
            to="/dashboard/l4-proxies"
            className="flex items-center justify-between rounded-md px-3 py-2.5 hover:bg-muted/50 transition-colors group"
          >
            <div className="flex items-center gap-3">
              <Network className="size-4 text-muted-foreground" />
              <div>
                <span className="text-sm font-medium">TCP/UDP Proxies</span>
                <span className="text-xs text-muted-foreground ml-2">
                  {activeL4} active, {totalL4 - activeL4} inactive
                </span>
              </div>
            </div>
            <ArrowRight className="size-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
          </Link>
        )}

        {/* Recent proxies list */}
        {recentProxies.length > 0 && (
          <>
            <div className="border-t my-2" />
            <p className="px-3 pt-1 pb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Recently added
            </p>
            {recentProxies.slice(0, 5).map((proxy) => (
              <Link
                key={proxy.id}
                to="/dashboard/proxies/$proxyId"
                params={{ proxyId: String(proxy.id) }}
                className="flex items-center justify-between rounded-md px-3 py-2 hover:bg-muted/50 transition-colors group"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span
                    className={`size-2 rounded-full flex-shrink-0 ${proxy.is_active ? 'bg-green-500' : 'bg-muted-foreground/40'}`}
                  />
                  <span className="text-sm truncate">{proxy.name}</span>
                  <span className="text-xs text-muted-foreground truncate hidden sm:inline">
                    {proxy.hostname}
                  </span>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <Badge variant="outline" className="text-xs font-normal">
                    {proxy.type === 'reverse_proxy' ? 'reverse' : proxy.type}
                  </Badge>
                </div>
              </Link>
            ))}
          </>
        )}

        {/* Footer link */}
        <div className="border-t pt-2 mt-2">
          <Button variant="ghost" size="sm" asChild>
            <Link to="/dashboard/proxies">
              View all proxies
              <ArrowRight className="ml-1 size-3.5" />
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function getActivityLink(log: AuditLog): string | null {
  if (!log.resource_id || log.action.includes('delete')) return null;

  switch (log.resource_type) {
    case 'proxy':
      return `/dashboard/proxies/${log.resource_id}`;
    case 'acl':
      return `/dashboard/acl/${log.resource_id}`;
    default:
      return null;
  }
}

function ActivityItem({ log }: { log: AuditLog }) {
  const timeAgo = formatDistanceToNow(new Date(log.created_at), { addSuffix: true });
  const actor = log.user?.username ?? 'system';
  const actionLabel = getActionLabel(log.action);
  const actionColor = getActionColor(log.action);
  const isFailed = log.status === 'failure';
  const link = getActivityLink(log);

  const content = (
    <>
      <div className="mt-0.5 flex-shrink-0">
        {isFailed ? (
          <XCircle className="size-4 text-destructive" />
        ) : log.action.includes('create') || log.action.includes('enable') ? (
          <CheckCircle2 className="size-4 text-green-500" />
        ) : (
          <Activity className={`size-4 ${actionColor}`} />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm leading-snug">
          <span className="font-medium">{actor}</span>{' '}
          <span className={actionColor}>{actionLabel}</span>
          {log.resource_name && (
            <span className="text-muted-foreground"> &middot; {log.resource_name}</span>
          )}
        </p>
        <p className="text-xs text-muted-foreground mt-0.5">{timeAgo}</p>
      </div>
    </>
  );

  if (link) {
    return (
      <Link
        to={link}
        className="flex gap-3 px-3 py-2 rounded-md hover:bg-muted/50 transition-colors"
      >
        {content}
      </Link>
    );
  }

  return <div className="flex gap-3 px-3 py-2">{content}</div>;
}

function RecentActivity({ activity, isLoading }: { activity: AuditLog[]; isLoading: boolean }) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-28" />
        </CardHeader>
        <CardContent className="space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="flex gap-3 px-3 py-2">
              <Skeleton className="size-4 rounded-sm flex-shrink-0" />
              <div className="space-y-1.5 flex-1">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardHeading>
          <CardTitle className="text-base">Recent Activity</CardTitle>
        </CardHeading>
        <CardToolbar>
          <Button variant="ghost" size="sm" asChild>
            <Link to="/dashboard/audit-logs">
              View all
              <ArrowRight className="ml-1 size-3.5" />
            </Link>
          </Button>
        </CardToolbar>
      </CardHeader>
      <CardContent className="pt-0">
        {activity.length === 0 ? (
          <div className="flex flex-col items-center py-6 text-center">
            <Activity className="size-5 text-muted-foreground/40" />
            <p className="text-sm text-muted-foreground mt-2">No recent activity</p>
            <p className="text-xs text-muted-foreground/70 mt-1 max-w-[200px]">
              Actions like creating proxies and changing settings will appear here.
            </p>
          </div>
        ) : (
          <div className="space-y-1">
            {activity.map((log) => (
              <ActivityItem key={log.id} log={log} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function QuickActions() {
  return (
    <div className="flex flex-wrap gap-2">
      <Button size="sm" asChild>
        <Link to="/dashboard/proxies/new">
          <Plus className="size-4" />
          New Proxy
        </Link>
      </Button>
      <Button size="sm" variant="outline" asChild>
        <Link to="/dashboard/l4-proxies/new">
          <Network className="size-4" />
          New TCP/UDP Proxy
        </Link>
      </Button>
    </div>
  );
}

export function DashboardIndex() {
  const { user } = useAuthStore();
  const { proxyStats, l4ProxyStats, recentProxies, recentActivity, isLoading } = useDashboardData();

  return (
    <div className="space-y-8">
      {/* Header: greeting + quick actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Welcome back, {user?.name || 'User'}</h1>
          <div className="mt-2">
            <SystemStatusStrip />
          </div>
        </div>
        <QuickActions />
      </div>

      {/* Main content: asymmetric two-column layout */}
      <div className="grid gap-6 lg:grid-cols-5">
        {/* Left column — wider — proxy overview */}
        <div
          className="lg:col-span-3 flex flex-col animate-fade-up"
          style={{ animationDelay: '100ms' }}
        >
          <ProxyOverview
            proxyStats={proxyStats}
            l4ProxyStats={l4ProxyStats}
            recentProxies={recentProxies}
            isLoading={isLoading}
          />
        </div>

        {/* Right column — narrower — activity feed */}
        <div className="lg:col-span-2 animate-fade-up" style={{ animationDelay: '200ms' }}>
          <RecentActivity activity={recentActivity} isLoading={isLoading} />
        </div>
      </div>
    </div>
  );
}
