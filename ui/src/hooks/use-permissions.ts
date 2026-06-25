import { useAuthStore } from '../stores/auth';

export function usePermissions() {
  const { user } = useAuthStore();
  const permissions = user?.permissions || [];

  const hasPermission = (permission: string): boolean => {
    return permissions.includes(permission);
  };

  const hasAnyPermission = (...requiredPermissions: string[]): boolean => {
    return requiredPermissions.some((p) => permissions.includes(p));
  };

  const hasAllPermissions = (...requiredPermissions: string[]): boolean => {
    return requiredPermissions.every((p) => permissions.includes(p));
  };

  return {
    permissions,
    hasPermission,
    hasAnyPermission,
    hasAllPermissions,
    canReadProxies: hasPermission('proxies:read'),
    canCreateProxies: hasPermission('proxies:create'),
    canUpdateProxies: hasPermission('proxies:update'),
    canDeleteProxies: hasPermission('proxies:delete'),
    canManageUsers: hasPermission('users:manage'),
    canReadAccess: hasPermission('acl:read'),
    canCreateAccess: hasPermission('acl:create'),
    canUpdateAccess: hasPermission('acl:update'),
    canDeleteAccess: hasPermission('acl:delete'),
    canReadAuditLogs: hasPermission('audit_logs:read'),
    canReadCaddyLogs: hasPermission('caddy_logs:read'),
    canReadCaddyConfig: hasPermission('caddy_config:read'),
    canReadSettings: hasPermission('settings:read'),
    canWriteSettings: hasPermission('settings:write'),
    canReadMetrics: hasPermission('metrics:read'),
  };
}
