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
  };
}
