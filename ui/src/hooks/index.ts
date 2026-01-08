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
  // OAuth Providers (Admin)
  useAdminOAuthProviders,
  useAssignACL,
  // Basic Auth Users
  useBasicAuthUsers,
  useConfigureWaygatesAuth,
  useCreateACLGroup,
  useDeleteACLGroup,
  useDeleteBasicAuthUser,
  useDeleteExternalProvider,
  useDeleteIPRule,
  // External Providers
  useExternalProviders,
  // IP Rules
  useIPRules,
  // OAuth Providers (Public)
  useOAuthProviders,
  // Proxy ACL
  useProxyACL,
  useRemoveACL,
  useUpdateACLBranding,
  useUpdateACLGroup,
  useUpdateBasicAuthUser,
  useUpdateExternalProvider,
  useUpdateIPRule,
  // OAuth Providers (Admin mutation)
  useUpdateOAuthProvider,
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

export { usePermissions } from './use-permissions';
export { useProxies } from './use-proxies';
export { useNotFoundSettings } from './use-settings';
