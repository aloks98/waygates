import { Badge } from '@e412/titanium';
import type { AuditAction } from '@/types/audit';

interface ActionBadgeProps {
  action: AuditAction;
}

const actionConfig: Record<
  string,
  {
    label: string;
    variant: 'primary' | 'secondary' | 'destructive' | 'outline' | 'success' | 'info';
  }
> = {
  'proxy.create': { label: 'Create', variant: 'success' },
  'proxy.update': { label: 'Update', variant: 'info' },
  'proxy.delete': { label: 'Delete', variant: 'destructive' },
  'proxy.enable': { label: 'Enable', variant: 'success' },
  'proxy.disable': { label: 'Disable', variant: 'secondary' },
  'auth.login': { label: 'Login', variant: 'primary' },
  'auth.logout': { label: 'Logout', variant: 'secondary' },
  'auth.register': { label: 'Register', variant: 'success' },
  'auth.password_change': { label: 'Password', variant: 'info' },
  'auth.login_failed': { label: 'Failed Login', variant: 'destructive' },
  'settings.update': { label: 'Settings', variant: 'info' },
  'sync.started': { label: 'Sync Start', variant: 'outline' },
  'sync.completed': { label: 'Sync Done', variant: 'success' },
  'sync.failed': { label: 'Sync Fail', variant: 'destructive' },
  'system.startup': { label: 'Startup', variant: 'outline' },
  'caddy.reload': { label: 'Reload', variant: 'outline' },
};

export function ActionBadge({ action }: ActionBadgeProps) {
  const config = actionConfig[action] || { label: action, variant: 'outline' as const };

  return (
    <Badge variant={config.variant} className="font-mono text-xs">
      {config.label}
    </Badge>
  );
}
