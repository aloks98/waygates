export type ProxyType = 'reverse_proxy' | 'redirect' | 'static';

export type LoadBalancingStrategy = 'round_robin' | 'least_conn' | 'ip_hash' | 'random';

export interface Upstream {
  host: string;
  port: number;
  scheme: 'http' | 'https';
}

export interface HealthCheck {
  enabled: boolean;
  path: string;
  interval: string;
  timeout: string;
  unhealthy_threshold: number;
  healthy_threshold: number;
}

export interface LoadBalancing {
  strategy: LoadBalancingStrategy;
  health_checks?: HealthCheck;
}

export interface RedirectConfig {
  target: string;
  status_code: 301 | 302 | 307 | 308;
  preserve_path: boolean;
  preserve_query: boolean;
}

export interface StaticConfig {
  root_path: string;
  index_file: string;
  browse: boolean;
  template_rendering: boolean;
  try_files?: string[];
}

export interface Proxy {
  id: number;
  type: ProxyType;
  name: string;
  hostname: string;
  description?: string;
  ssl_enabled: boolean;
  ssl_forced: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  created_by?: number;
  // Reverse proxy fields
  upstreams?: Upstream[];
  load_balancing?: LoadBalancing;
  block_exploits?: boolean;
  tls_insecure_skip_verify?: boolean;
  custom_headers?: Record<string, string>;
  // Redirect fields
  redirect?: RedirectConfig;
  // Static fields
  static?: StaticConfig;
}

export interface CreateReverseProxyRequest {
  type: 'reverse_proxy';
  name: string;
  hostname: string;
  description?: string;
  upstreams: Upstream[];
  block_exploits?: boolean;
  tls_insecure_skip_verify?: boolean;
  load_balancing?: LoadBalancing;
  custom_headers?: Record<string, string>;
}

export interface CreateRedirectRequest {
  type: 'redirect';
  name: string;
  hostname: string;
  description?: string;
  redirect: RedirectConfig;
}

export interface CreateStaticRequest {
  type: 'static';
  name: string;
  hostname: string;
  description?: string;
  static: StaticConfig;
}

export type CreateProxyRequest =
  | CreateReverseProxyRequest
  | CreateRedirectRequest
  | CreateStaticRequest;

export interface UpdateProxyRequest {
  name?: string;
  hostname?: string;
  description?: string;
  ssl_enabled?: boolean;
  ssl_forced?: boolean;
  is_active?: boolean;
  // Reverse proxy fields
  upstreams?: Upstream[];
  load_balancing?: LoadBalancing;
  block_exploits?: boolean;
  tls_insecure_skip_verify?: boolean;
  custom_headers?: Record<string, string>;
  // Redirect fields
  redirect?: RedirectConfig;
  // Static fields
  static?: StaticConfig;
}
