import { Badge } from '@e412/titanium';
import type { AuditAction } from '@/types/audit';

interface ActionBadgeProps {
  action: AuditAction;
}

const actionConfig: Record<
  string,
  {
    label: string;
    variant: 'primary' | 'secondary' | 'destructive' | 'outline' | 'success' | 'info' | 'warning';
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
  // ACL Group actions
  'acl_group.create': { label: 'ACL Group Create', variant: 'success' },
  'acl_group.update': { label: 'ACL Group Update', variant: 'info' },
  'acl_group.delete': { label: 'ACL Group Delete', variant: 'destructive' },
  // ACL IP Rule actions
  'acl_ip_rule.add': { label: 'IP Rule Add', variant: 'success' },
  'acl_ip_rule.update': { label: 'IP Rule Update', variant: 'info' },
  'acl_ip_rule.delete': { label: 'IP Rule Delete', variant: 'destructive' },
  // ACL Basic Auth actions
  'acl_basic_auth.add': { label: 'Basic Auth Add', variant: 'success' },
  'acl_basic_auth.update': { label: 'Basic Auth Update', variant: 'info' },
  'acl_basic_auth.delete': { label: 'Basic Auth Delete', variant: 'destructive' },
  // ACL Waygates Auth action
  'acl_waygates_auth.update': { label: 'Waygates Auth Update', variant: 'info' },
  // ACL Assignment actions
  'acl_assignment.create': { label: 'ACL Assign', variant: 'success' },
  'acl_assignment.update': { label: 'ACL Assign Update', variant: 'info' },
  'acl_assignment.delete': { label: 'ACL Unassign', variant: 'destructive' },
  // ACL Branding action
  'acl_branding.update': { label: 'ACL Branding Update', variant: 'info' },
  // ACL Session action
  'acl_session.revoke': { label: 'Session Revoke', variant: 'warning' },
  // ACL OAuth Restriction actions
  'acl_oauth_restriction.set': { label: 'OAuth Restriction Set', variant: 'info' },
  'acl_oauth_restriction.delete': { label: 'OAuth Restriction Delete', variant: 'destructive' },
};

export function ActionBadge({ action }: ActionBadgeProps) {
  const config = actionConfig[action] || { label: action, variant: 'outline' as const };

  return (
    <Badge variant={config.variant} className="font-mono text-xs">
      {config.label}
    </Badge>
  );
}
