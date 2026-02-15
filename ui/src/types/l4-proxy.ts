// L4 Protocol constants
export const L4_PROTOCOLS = ['tcp', 'udp'] as const;
export type L4Protocol = (typeof L4_PROTOCOLS)[number];

// L4 Matcher type constants
export const L4_MATCHER_TYPES = [
  'any',
  'tls',
  'ssh',
  'postgres',
  'http',
  'rdp',
  'socks5',
  'remote_ip',
  'regexp',
] as const;
export type L4MatcherType = (typeof L4_MATCHER_TYPES)[number];

// L4 Load balancing policy constants
export const L4_LOAD_BALANCING_POLICIES = [
  'round_robin',
  'least_conn',
  'random',
  'first',
  'ip_hash',
] as const;
export type L4LoadBalancingPolicy = (typeof L4_LOAD_BALANCING_POLICIES)[number];

// L4 Proxy protocol version constants
export const L4_PROXY_PROTOCOL_VERSIONS = ['v1', 'v2'] as const;
export type L4ProxyProtocolVersion = (typeof L4_PROXY_PROTOCOL_VERSIONS)[number];

export interface L4Upstream {
  host: string;
  port: number;
  weight?: number;
}

export interface L4Route {
  id: number;
  l4_proxy_id: number;
  priority: number;
  matcher_type: L4MatcherType;
  sni_hostnames?: string[];
  allowed_ip_ranges?: string[];
  regex_pattern?: string;
  upstreams: L4Upstream[];
  load_balancing_policy: L4LoadBalancingPolicy;
  tls_terminate: boolean;
  tls_passthrough: boolean;
  proxy_protocol_version?: L4ProxyProtocolVersion;
  created_at: string;
  updated_at: string;
}

export interface L4Proxy {
  id: number;
  name: string;
  description?: string;
  listen_port: number;
  protocol: L4Protocol;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  routes?: L4Route[];
}

export interface L4ProxyListResponse {
  items: L4Proxy[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface L4ProxyStats {
  total_proxies: number;
  active_proxies: number;
  tcp_proxies: number;
  udp_proxies: number;
  total_routes: number;
  total_upstreams: number;
}

export interface CreateL4UpstreamRequest {
  host: string;
  port: number;
  weight?: number;
}

export interface CreateL4RouteRequest {
  priority: number;
  matcher_type: L4MatcherType;
  sni_hostnames?: string[];
  allowed_ip_ranges?: string[];
  regex_pattern?: string;
  upstreams: CreateL4UpstreamRequest[];
  load_balancing_policy: L4LoadBalancingPolicy;
  tls_terminate: boolean;
  tls_passthrough: boolean;
  proxy_protocol_version?: L4ProxyProtocolVersion;
}

export interface CreateL4ProxyRequest {
  name: string;
  description?: string;
  listen_port: number;
  protocol: L4Protocol;
  is_active: boolean;
  routes?: CreateL4RouteRequest[];
}

export interface UpdateL4ProxyRequest {
  name: string;
  description?: string;
  listen_port: number;
  protocol: L4Protocol;
  is_active: boolean;
  routes?: CreateL4RouteRequest[];
}
