// ACL Hooks
export {
  // Branding
  useACLBranding,
  useACLGroup,
  // ACL Groups
  useACLGroups,
  useAddBasicAuthUser,
  useAddExternalProvider,
  useAddIPRule,
  useAssignACL,
  // Basic Auth Users
  useBasicAuthUsers,
  useConfigureWaygatesAuth,
  useCreateACLGroup,
  useDeleteACLGroup,
  useDeleteBasicAuthUser,
  useDeleteExternalProvider,
  useDeleteIPRule,
  useDeleteOAuthProviderRestriction,
  // External Providers
  useExternalProviders,
  // IP Rules
  useIPRules,
  // OAuth Provider Restrictions
  useOAuthProviderRestrictions,
  // OAuth Providers
  useOAuthProviders,
  // Proxy ACL
  useProxyACL,
  useRemoveACL,
  useSetOAuthProviderRestriction,
  useUpdateACLBranding,
  useUpdateACLGroup,
  useUpdateBasicAuthUser,
  useUpdateExternalProvider,
  useUpdateIPRule,
  useUpdateProxyACLAssignment,
  // Waygates Auth
  useWaygatesAuth,
} from './use-acl';

// Audit Hooks
export {
  useAuditConfig,
  useAuditEventGroups,
  useAuditLogById,
  useAuditLogs,
  useAuditStats,
  useExportAuditLogs,
} from './use-audit-logs';

// Dashboard Hooks
export {
  useAppStatus,
  useDashboardData,
  useHealthStatus,
  useSyncStatus,
} from './use-dashboard';

export { usePermissions } from './use-permissions';
export { useProxies, useProxy } from './use-proxies';
export { useNotFoundSettings } from './use-settings';
