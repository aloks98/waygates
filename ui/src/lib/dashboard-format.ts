import { formatDuration, intervalToDuration } from 'date-fns';

import type { ProxyStats } from '@/types/api';
import type { AuditLog } from '@/types/audit';
import type { L4ProxyStats } from '@/types/l4-proxy';

export function formatUptime(raw: string): string {
  const hours = raw.match(/(\d+)h/);
  const minutes = raw.match(/(\d+)m/);
  const seconds = raw.match(/([\d.]+)s/);
  const totalSeconds =
    (hours ? Number.parseInt(hours[1], 10) * 3600 : 0) +
    (minutes ? Number.parseInt(minutes[1], 10) * 60 : 0) +
    (seconds ? Math.floor(Number.parseFloat(seconds[1])) : 0);
  const duration = intervalToDuration({ start: 0, end: totalSeconds * 1000 });
  return (
    formatDuration(duration, { format: ['days', 'hours', 'minutes'], delimiter: ' ' }) ||
    'less than a minute'
  );
}

export function getActionLabel(action: string): string {
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
    'acl_ip_rule.add': 'added an IP rule',
    'acl_ip_rule.update': 'updated an IP rule',
    'acl_ip_rule.delete': 'deleted an IP rule',
    'acl_basic_auth.add': 'added a basic-auth user',
    'acl_basic_auth.update': 'updated basic auth',
    'acl_basic_auth.delete': 'deleted a basic-auth user',
    'acl_waygates_auth.update': 'updated Waygates auth',
    'acl_assignment.create': 'assigned ACL to a proxy',
    'acl_assignment.update': 'updated an ACL assignment',
    'acl_assignment.delete': 'removed ACL from a proxy',
    'acl_branding.update': 'updated branding',
    'acl_session.revoke': 'revoked a session',
    'acl_oauth_restriction.set': 'configured an OAuth provider',
    'acl_oauth_restriction.delete': 'removed an OAuth provider',
  };
  return labels[action] ?? action.replace(/[._]/g, ' ');
}

export function getActionColor(action: string): string {
  if (action.includes('delete') || action.includes('failed')) return 'text-destructive';
  if (action.includes('create') || action.includes('enable') || action === 'sync.completed')
    return 'text-green-600 dark:text-green-500';
  if (action.includes('disable')) return 'text-amber-600 dark:text-amber-500';
  return 'text-muted-foreground';
}

export function getActivityLink(log: AuditLog): string | null {
  if (!log.resource_id || log.action.includes('delete')) return null;
  switch (log.resource_type) {
    case 'proxy':
      return `/proxies/${log.resource_id}`;
    case 'acl':
      return `/access/${log.resource_id}`;
    default:
      return null;
  }
}

export function buildCompositionData(
  proxyStats?: ProxyStats,
  l4Stats?: L4ProxyStats,
): { name: string; value: number }[] {
  const t = proxyStats?.by_type ?? {};
  return [
    { name: 'Reverse', value: t.reverse_proxy ?? 0 },
    { name: 'Redirect', value: t.redirect ?? 0 },
    { name: 'Static', value: t.static ?? 0 },
    { name: 'TCP', value: l4Stats?.tcp_proxies ?? 0 },
    { name: 'UDP', value: l4Stats?.udp_proxies ?? 0 },
  ].filter((d) => d.value > 0);
}
