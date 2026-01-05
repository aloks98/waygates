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
  | 'caddy.reload';

export type AuditStatus = 'success' | 'failure';

export type AuditResourceType = 'proxy' | 'user' | 'settings' | 'system';

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
}

export interface AuditLogListParams {
  page?: number;
  limit?: number;
  search?: string;
  action?: string; // Comma-separated action values (include)
  action_not?: string; // Comma-separated action values (exclude)
  resource_type?: string; // Comma-separated resource type values (include)
  resource_type_not?: string; // Comma-separated resource type values (exclude)
  user_id?: number;
  status?: AuditStatus;
  status_not?: AuditStatus;
  ip_address?: string; // Exact match
  ip_address_contains?: string; // Contains
  ip_address_starts_with?: string; // Starts with
  ip_address_ends_with?: string; // Ends with
  ip_address_not?: string; // Not equal
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
