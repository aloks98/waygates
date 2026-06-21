import type {
  RedirectFormValues,
  ReverseProxyFormValues,
  StaticFormValues,
} from '@/lib/form-validation';
import type {
  CreateRedirectRequest,
  CreateReverseProxyRequest,
  CreateStaticRequest,
  HealthCheck,
  ProxyConfig,
} from '@/types/proxy';

// ---------- shared helpers ----------

function normalizeScheme(scheme: string | undefined): 'http' | 'https' {
  return String(scheme ?? '').toLowerCase() === 'https' ? 'https' : 'http';
}

function recordToPairs(rec?: Record<string, string>): { name: string; value: string }[] {
  return Object.entries(rec ?? {}).map(([name, value]) => ({ name, value }));
}

function pairsToRecord(pairs: { name: string; value: string }[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of pairs) {
    const name = p.name.trim();
    if (name) out[name] = p.value;
  }
  return out;
}

// ---------- reverse proxy ----------

export const REVERSE_PROXY_DEFAULTS: ReverseProxyFormValues = {
  name: '',
  hostname: '',
  description: '',
  upstreams: [{ host: '', port: 8080, scheme: 'http' }],
  ssl_enabled: true,
  block_exploits: true,
  tls_insecure_skip_verify: false,
  lb_strategy: 'round_robin',
  health_check_enabled: false,
  health_check_path: '/health',
  health_check_interval: '30s',
  health_check_timeout: '5s',
  request_headers: [],
  response_headers: [],
};

export function mapProxyToReverseDefaults(data: ProxyConfig): ReverseProxyFormValues {
  const hc = data.load_balancing?.health_checks;
  return {
    name: data.name,
    hostname: data.hostname,
    description: data.description ?? '',
    upstreams: data.upstreams?.length
      ? data.upstreams.map((u) => ({
          host: u.host || '',
          port: u.port || 8080,
          scheme: normalizeScheme(u.scheme),
        }))
      : [{ host: '', port: 8080, scheme: 'http' }],
    ssl_enabled: data.ssl_enabled ?? true,
    block_exploits: data.block_exploits ?? true,
    tls_insecure_skip_verify: data.tls_insecure_skip_verify ?? false,
    lb_strategy: data.load_balancing?.strategy ?? 'round_robin',
    health_check_enabled: hc?.enabled ?? false,
    health_check_path: hc?.path ?? '/health',
    health_check_interval: hc?.interval ?? '30s',
    health_check_timeout: hc?.timeout ?? '5s',
    request_headers: recordToPairs(data.custom_headers?.request),
    response_headers: recordToPairs(data.custom_headers?.response),
  };
}

export function mapReverseValuesToRequest(
  values: ReverseProxyFormValues,
): CreateReverseProxyRequest {
  const request = pairsToRecord(values.request_headers);
  const response = pairsToRecord(values.response_headers);
  const hasHeaders = Object.keys(request).length > 0 || Object.keys(response).length > 0;
  const multiUpstream = values.upstreams.length > 1;

  const healthChecks: HealthCheck | undefined = values.health_check_enabled
    ? {
        enabled: true,
        path: values.health_check_path,
        interval: values.health_check_interval,
        timeout: values.health_check_timeout,
        unhealthy_threshold: 3,
        healthy_threshold: 2,
      }
    : undefined;

  return {
    type: 'reverse_proxy',
    name: values.name,
    hostname: values.hostname,
    description: values.description || undefined,
    ssl_enabled: values.ssl_enabled,
    upstreams: values.upstreams,
    block_exploits: values.block_exploits,
    tls_insecure_skip_verify: values.tls_insecure_skip_verify,
    ...(multiUpstream
      ? { load_balancing: { strategy: values.lb_strategy, health_checks: healthChecks } }
      : {}),
    ...(hasHeaders
      ? {
          custom_headers: {
            ...(Object.keys(request).length ? { request } : {}),
            ...(Object.keys(response).length ? { response } : {}),
          },
        }
      : {}),
  };
}

// ---------- redirect ----------

export const REDIRECT_DEFAULTS: RedirectFormValues = {
  name: '',
  hostname: '',
  description: '',
  ssl_enabled: true,
  target: '',
  status_code: 301,
  preserve_path: true,
  preserve_query: true,
};

export function mapProxyToRedirectDefaults(data: ProxyConfig): RedirectFormValues {
  const r = data.redirect;
  return {
    name: data.name,
    hostname: data.hostname,
    description: data.description ?? '',
    ssl_enabled: data.ssl_enabled ?? true,
    target: r?.target ?? '',
    status_code: r?.status_code ?? 301,
    preserve_path: r?.preserve_path ?? true,
    preserve_query: r?.preserve_query ?? true,
  };
}

export function mapRedirectValuesToRequest(values: RedirectFormValues): CreateRedirectRequest {
  return {
    type: 'redirect',
    name: values.name,
    hostname: values.hostname,
    description: values.description || undefined,
    ssl_enabled: values.ssl_enabled,
    redirect: {
      target: values.target,
      status_code: values.status_code as 301 | 302 | 307 | 308,
      preserve_path: values.preserve_path,
      preserve_query: values.preserve_query,
    },
  };
}

// ---------- static ----------

export const STATIC_DEFAULTS: StaticFormValues = {
  name: '',
  hostname: '',
  description: '',
  ssl_enabled: true,
  root_path: '/var/www/html',
  index_file: 'index.html',
  browse: false,
  template_rendering: false,
  try_files: [],
};

export function mapProxyToStaticDefaults(data: ProxyConfig): StaticFormValues {
  const s = data.static;
  return {
    name: data.name,
    hostname: data.hostname,
    description: data.description ?? '',
    ssl_enabled: data.ssl_enabled ?? true,
    root_path: s?.root_path ?? '/var/www/html',
    index_file: s?.index_file ?? 'index.html',
    browse: s?.browse ?? false,
    template_rendering: s?.template_rendering ?? false,
    try_files: (s?.try_files ?? []).map((value) => ({ value })),
  };
}

export function mapStaticValuesToRequest(values: StaticFormValues): CreateStaticRequest {
  return {
    type: 'static',
    name: values.name,
    hostname: values.hostname,
    description: values.description || undefined,
    ssl_enabled: values.ssl_enabled,
    static: {
      root_path: values.root_path,
      index_file: values.index_file,
      browse: values.browse,
      template_rendering: values.template_rendering,
      try_files: values.try_files.map((f) => f.value.trim()).filter(Boolean),
    },
  };
}
