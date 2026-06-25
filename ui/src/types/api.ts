export interface ApiResponse<T> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
  details?: Record<string, string>;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface ProxyStats {
  total: number;
  active: number;
  inactive: number;
  by_type: Record<string, number>;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  must_change_password?: boolean;
}

export interface SyncStatus {
  last_sync_time: string;
  is_syncing: boolean;
  last_sync_success: boolean;
  last_error?: string;
  sync_count: number;
  last_reload_time?: string;
  reload_count: number;
  config_changed: boolean;
}

export interface HealthStatus {
  status: 'healthy' | 'degraded';
  service: string;
  version: string;
  uptime: string;
  time: string;
  components: {
    database: 'healthy' | 'unhealthy';
  };
}

export interface AppStatus {
  caddy_status: 'healthy' | 'unhealthy';
  user_setup_complete: boolean;
}
