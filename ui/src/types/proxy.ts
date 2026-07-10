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

export interface CustomHeaders {
  request?: Record<string, string>;
  response?: Record<string, string>;
}

export interface ProxyConfig {
  id: number;
  type: ProxyType;
  name: string;
  hostname: string;
  description?: string;
  // ssl_enabled / ssl_forced / block_exploits / tls_insecure_skip_verify are
  // tri-state: null means "inherit from the proxy's group, or the system
  // default if ungrouped". Never coerce null to false — that silently turns
  // "inherit" into "explicitly disabled". See `effective` for the resolved
  // value actually served.
  // All four are optional because GET /api/proxies (the list endpoint) can
  // omit them from a given row; only GET /api/proxies/:id is guaranteed to
  // send every field. Treat a missing key the same as null (inherit).
  ssl_enabled?: boolean | null;
  ssl_forced?: boolean | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  created_by?: number;
  // ACL summary (from the list endpoint)
  acl_group_count?: number;
  acl_group_names?: string[];
  // ProxyGroup (config inheritance) — never an ACLGroup (an auth grouping).
  group_id?: number | null;
  group_name?: string | null;
  // Set iff the proxy has a group AND that group has a base_domain. hostname
  // then holds the materialized <hostname_label>.<group.base_domain>.
  hostname_label?: string | null;
  // Reverse proxy fields
  upstreams?: Upstream[];
  load_balancing?: LoadBalancing;
  block_exploits?: boolean | null;
  tls_insecure_skip_verify?: boolean | null;
  custom_headers?: CustomHeaders;
  // Redirect fields
  redirect?: RedirectConfig;
  // Static fields
  static?: StaticConfig;
  // Present on GET /api/proxies/:id only: the resolved (proxygroup.Resolve)
  // view of the inheritable settings, plus where each came from.
  effective?: {
    ssl_enabled: boolean;
    ssl_forced: boolean;
    block_exploits: boolean;
    tls_insecure_skip_verify: boolean;
    custom_headers?: CustomHeaders;
    /** Where each resolved value came from: 'proxy' | 'group' | 'default'. */
    _source: Record<string, 'proxy' | 'group' | 'default'>;
  };
}

export interface CreateReverseProxyRequest {
  type: 'reverse_proxy';
  name: string;
  hostname: string;
  description?: string;
  group_id?: number | null;
  hostname_label?: string | null;
  // null means inherit (from the group, or the system default if
  // ungrouped) — never send `false` for a field the user left on "Inherit".
  ssl_enabled?: boolean | null;
  ssl_forced?: boolean | null;
  upstreams: Upstream[];
  block_exploits?: boolean | null;
  tls_insecure_skip_verify?: boolean | null;
  load_balancing?: LoadBalancing;
  custom_headers?: CustomHeaders;
}

export interface CreateRedirectRequest {
  type: 'redirect';
  name: string;
  hostname: string;
  description?: string;
  group_id?: number | null;
  hostname_label?: string | null;
  ssl_enabled?: boolean | null;
  ssl_forced?: boolean | null;
  block_exploits?: boolean | null;
  redirect: RedirectConfig;
}

export interface CreateStaticRequest {
  type: 'static';
  name: string;
  hostname: string;
  description?: string;
  group_id?: number | null;
  hostname_label?: string | null;
  ssl_enabled?: boolean | null;
  ssl_forced?: boolean | null;
  block_exploits?: boolean | null;
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
  group_id?: number | null;
  hostname_label?: string | null;
  // BREAKING (backend Task 6): an omitted boolean here now means "inherit",
  // not "keep existing" — always send all four explicitly (null for inherit).
  ssl_enabled?: boolean | null;
  ssl_forced?: boolean | null;
  is_active?: boolean;
  // Reverse proxy fields
  upstreams?: Upstream[];
  load_balancing?: LoadBalancing;
  block_exploits?: boolean | null;
  tls_insecure_skip_verify?: boolean | null;
  custom_headers?: CustomHeaders;
  // Redirect fields
  redirect?: RedirectConfig;
  // Static fields
  static?: StaticConfig;
}
