// Audit log action types
export type AuditAction =
  | 'proxy.create'
  | 'proxy.update'
  | 'proxy.delete'
  | 'proxy.enable'
  | 'proxy.disable'
  | 'auth.login'
  | 'auth.logout'
  | 'auth.register'
  | 'auth.password_change'
  | 'auth.login_failed'
  | 'settings.update'
  | 'sync.started'
  | 'sync.completed'
  | 'sync.failed'
  | 'system.startup'
  | 'caddy.reload'
  | 'acl_group.create'
  | 'acl_group.update'
  | 'acl_group.delete'
  | 'acl_ip_rule.add'
  | 'acl_ip_rule.update'
  | 'acl_ip_rule.delete'
  | 'acl_basic_auth.add'
  | 'acl_basic_auth.update'
  | 'acl_basic_auth.delete'
  | 'acl_waygates_auth.update'
  | 'acl_assignment.create'
  | 'acl_assignment.update'
  | 'acl_assignment.delete'
  | 'acl_branding.update'
  | 'acl_session.revoke';

export type AuditStatus = 'success' | 'failure';

export type AuditResourceType = 'proxy' | 'user' | 'settings' | 'system' | 'acl';

export interface AuditLog {
  id: number;
  user_id?: number;
  action: AuditAction;
  resource_type?: AuditResourceType;
  resource_id?: number;
  resource_name?: string;
  details?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  status: AuditStatus;
  error_message?: string;
  created_at: string;
  user?: {
    id: number;
    username: string;
    email: string;
  };
}

export interface AuditLogListResponse {
  items: AuditLog[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface AuditLogStats {
  total_logs: number;
  by_action: Record<string, number>;
  by_status: Record<string, number>;
  by_resource_type: Record<string, number>;
  recent_activity: AuditLog[];
}

export interface AuditConfig {
  // Proxy events
  proxy_create: boolean;
  proxy_update: boolean;
  proxy_delete: boolean;
  proxy_enable: boolean;
  proxy_disable: boolean;

  // Auth events
  auth_login: boolean;
  auth_logout: boolean;
  auth_register: boolean;
  auth_password_change: boolean;
  auth_login_failed: boolean;

  // Settings events
  settings_update: boolean;

  // Sync events
  sync_started: boolean;
  sync_completed: boolean;
  sync_failed: boolean;

  // System events
  system_startup: boolean;
  caddy_reload: boolean;

  // ACL events
  acl_group_create: boolean;
  acl_group_update: boolean;
  acl_group_delete: boolean;
  acl_ip_rule_add: boolean;
  acl_ip_rule_update: boolean;
  acl_ip_rule_delete: boolean;
  acl_basic_auth_add: boolean;
  acl_basic_auth_update: boolean;
  acl_basic_auth_delete: boolean;
  acl_waygates_auth_update: boolean;
  acl_assignment_create: boolean;
  acl_assignment_update: boolean;
  acl_assignment_delete: boolean;
  acl_branding_update: boolean;
  acl_session_revoke: boolean;
}

// Filter format: field=operator:value
// Supported operators: eq (default), not, in, not_in, contains, starts_with, ends_with
// Examples:
//   - action=in:proxy.create,proxy.update
//   - status=not:failure
//   - ip_address=ends_with:121
//   - ip_address=192.168.1.1 (defaults to eq)
export interface AuditLogListParams {
  page?: number;
  limit?: number;
  search?: string;
  action?: string; // operator:value format (e.g., "in:proxy.create,proxy.update" or "not_in:auth.login")
  resource_type?: string; // operator:value format
  user_id?: number;
  status?: string; // operator:value format (e.g., "success" or "not:failure")
  ip_address?: string; // operator:value format (e.g., "ends_with:121" or "contains:192")
  date_from?: string;
  date_to?: string;
  sort?: 'created_at' | 'action' | 'status';
  order?: 'asc' | 'desc';
}

export interface AuditEventDefinition {
  key: keyof AuditConfig;
  label: string;
}

export interface AuditEventGroup {
  key: string;
  label: string;
  description: string;
  events: AuditEventDefinition[];
}
