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
  proxy_events: boolean;
  auth_events: boolean;
  settings_events: boolean;
  sync_events: boolean;
  system_events: boolean;
}

export interface AuditLogListParams {
  page?: number;
  limit?: number;
  search?: string;
  action?: AuditAction;
  resource_type?: AuditResourceType;
  user_id?: number;
  status?: AuditStatus;
  date_from?: string;
  date_to?: string;
  sort?: 'created_at' | 'action' | 'status';
  order?: 'asc' | 'desc';
}
