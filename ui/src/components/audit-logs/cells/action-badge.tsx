import { Badge } from '@e412/titanium';
import type { AuditAction } from '@/types/audit';

interface ActionBadgeProps {
  action: AuditAction;
}

const actionConfig: Record<
  string,
  { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }
> = {
  'proxy.create': { label: 'Create', variant: 'default' },
  'proxy.update': { label: 'Update', variant: 'secondary' },
  'proxy.delete': { label: 'Delete', variant: 'destructive' },
  'proxy.enable': { label: 'Enable', variant: 'default' },
  'proxy.disable': { label: 'Disable', variant: 'secondary' },
  'auth.login': { label: 'Login', variant: 'default' },
  'auth.logout': { label: 'Logout', variant: 'secondary' },
  'auth.register': { label: 'Register', variant: 'default' },
  'auth.password_change': { label: 'Password', variant: 'secondary' },
  'auth.login_failed': { label: 'Failed Login', variant: 'destructive' },
  'settings.update': { label: 'Settings', variant: 'secondary' },
  'sync.started': { label: 'Sync Start', variant: 'outline' },
  'sync.completed': { label: 'Sync Done', variant: 'default' },
  'sync.failed': { label: 'Sync Fail', variant: 'destructive' },
  'system.startup': { label: 'Startup', variant: 'outline' },
  'caddy.reload': { label: 'Reload', variant: 'outline' },
};

const categoryIcons: Record<string, string> = {
  proxy: '🔀',
  auth: '🔐',
  settings: '⚙️',
  sync: '🔄',
  system: '💻',
  caddy: '🌐',
};

export function ActionBadge({ action }: ActionBadgeProps) {
  const config = actionConfig[action] || { label: action, variant: 'outline' as const };
  const category = action.split('.')[0];
  const icon = categoryIcons[category] || '📋';

  return (
    <div className="flex items-center gap-2">
      <span className="text-sm">{icon}</span>
      <Badge variant={config.variant} className="font-mono text-xs">
        {config.label}
      </Badge>
    </div>
  );
}
