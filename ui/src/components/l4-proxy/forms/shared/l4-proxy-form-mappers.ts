import type { L4ProxyFormValues, L4RouteFormValues } from '@/lib/form-validation';
import type { CreateL4ProxyRequest, CreateL4RouteRequest, L4Proxy } from '@/types/l4-proxy';

export function createDefaultRoute(): L4RouteFormValues {
  return {
    priority: 0,
    matcher_type: 'any',
    sni_hostnames: [],
    allowed_ip_ranges: [],
    regex_pattern: '',
    upstreams: [{ host: '', port: 8080 }],
    load_balancing_policy: 'round_robin',
    tls_terminate: false,
    tls_passthrough: false,
    proxy_protocol_version: undefined,
  };
}

export const L4_PROXY_DEFAULTS: L4ProxyFormValues = {
  name: '',
  description: '',
  listen_port: 8080,
  protocol: 'tcp',
  is_active: true,
  routes: [createDefaultRoute()],
};

export function mapL4ProxyToDefaults(data: L4Proxy): L4ProxyFormValues {
  const routes: L4RouteFormValues[] =
    data.routes && data.routes.length > 0
      ? data.routes.map((r) => ({
          priority: r.priority ?? 0,
          matcher_type: r.matcher_type,
          sni_hostnames: (r.sni_hostnames ?? []).map((v) => ({ value: v })),
          allowed_ip_ranges: (r.allowed_ip_ranges ?? []).map((v) => ({ value: v })),
          regex_pattern: r.regex_pattern ?? '',
          upstreams: r.upstreams?.length
            ? r.upstreams.map((u) => ({ host: u.host, port: u.port, weight: u.weight }))
            : [{ host: '', port: 8080 }],
          load_balancing_policy: r.load_balancing_policy ?? 'round_robin',
          tls_terminate: r.tls_terminate ?? false,
          tls_passthrough: r.tls_passthrough ?? false,
          proxy_protocol_version: r.proxy_protocol_version ?? undefined,
        }))
      : [createDefaultRoute()];

  return {
    name: data.name,
    description: data.description ?? '',
    listen_port: data.listen_port,
    protocol: data.protocol,
    is_active: data.is_active,
    routes,
  };
}

export function mapL4FormValuesToRequest(values: L4ProxyFormValues): CreateL4ProxyRequest {
  return {
    name: values.name,
    description: values.description || undefined,
    listen_port: values.listen_port,
    protocol: values.protocol,
    is_active: values.is_active,
    routes: (values.routes ?? []).map(
      (r): CreateL4RouteRequest => ({
        priority: r.priority,
        matcher_type: r.matcher_type,
        upstreams: r.upstreams.map((u) => ({
          host: u.host,
          port: u.port,
          ...(u.weight != null ? { weight: u.weight } : {}),
        })),
        load_balancing_policy: r.load_balancing_policy,
        tls_terminate: r.tls_terminate,
        tls_passthrough: r.tls_passthrough,
        ...(r.proxy_protocol_version ? { proxy_protocol_version: r.proxy_protocol_version } : {}),
        ...(r.matcher_type === 'tls'
          ? { sni_hostnames: (r.sni_hostnames ?? []).map((i) => i.value.trim()).filter(Boolean) }
          : {}),
        ...(r.matcher_type === 'remote_ip'
          ? {
              allowed_ip_ranges: (r.allowed_ip_ranges ?? [])
                .map((i) => i.value.trim())
                .filter(Boolean),
            }
          : {}),
        ...(r.matcher_type === 'regexp'
          ? { regex_pattern: r.regex_pattern?.trim() || undefined }
          : {}),
      }),
    ),
  };
}
