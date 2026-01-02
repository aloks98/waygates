export type ProxyType = 'reverse_proxy' | 'redirect' | 'static';

export interface Proxy {
  id: number;
  hostname: string;
  target_url: string;
  proxy_type: ProxyType;
  ssl_enabled: boolean;
  ssl_forced: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  created_by_id: number;
}

export interface CreateProxyRequest {
  hostname: string;
  target_url: string;
  proxy_type: ProxyType;
}

export interface UpdateProxyRequest {
  hostname?: string;
  target_url?: string;
  proxy_type?: ProxyType;
  ssl_enabled?: boolean;
  ssl_forced?: boolean;
  is_active?: boolean;
}
