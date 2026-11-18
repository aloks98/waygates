import type { Page } from '@playwright/test';
const date = '2026-11-18T10:00:00Z';
export const user = {
  id: 1,
  name: 'Alok',
  username: 'alok',
  email: 'alok@example.com',
  role: 'admin',
  is_active: true,
  created_at: date,
  permissions: [
    'proxies:read',
    'proxies:create',
    'proxies:update',
    'proxies:delete',
    'l4proxies:read',
    'l4proxies:create',
    'l4proxies:update',
    'l4proxies:delete',
    'acl:read',
    'acl:create',
    'acl:update',
    'acl:delete',
    'settings:read',
    'settings:write',
    'users:read',
    'users:manage',
    'audit_logs:read',
    'audit_logs:update',
    'caddy_logs:read',
    'caddy_config:read',
    'metrics:read',
    'sync:read',
    'sync:trigger',
  ],
};
export const proxy = {
  id: 1,
  type: 'reverse_proxy',
  name: 'Analytics',
  hostname: 'analytics.waygates.io',
  ssl_enabled: true,
  ssl_forced: true,
  is_active: true,
  upstreams: [{ host: 'analytics', port: 3000, scheme: 'http' }],
  block_exploits: true,
  tls_insecure_skip_verify: false,
  created_at: date,
  updated_at: date,
  acl_group_count: 1,
  acl_group_names: ['Team'],
};
const proxies = [
  proxy,
  {
    ...proxy,
    id: 2,
    name: 'Documentation',
    hostname: 'docs.waygates.io',
    type: 'static',
    upstreams: undefined,
    static: { root_path: '/srv/docs', index_file: 'index.html' },
    acl_group_count: 0,
  },
  {
    ...proxy,
    id: 3,
    name: 'Legacy',
    hostname: 'old.waygates.io',
    type: 'redirect',
    redirect: { target: 'https://waygates.io', status_code: 301 },
    acl_group_count: 0,
  },
];
const group = {
  id: 1,
  name: 'Team',
  description: '',
  combination_mode: 'any',
  created_at: date,
  updated_at: date,
  ip_rules: [],
  basic_auth_users: [],
  external_providers: [],
  waygates_auth: {
    enabled: true,
    allowed_users: ['alok@example.com'],
    allowed_roles: ['admin'],
    allowed_email_patterns: [],
    allowed_providers: [],
    session_ttl: 86400,
    require_2fa: false,
  },
  oauth_provider_restrictions: [],
};
const route = {
  id: 1,
  priority: 0,
  matcher_type: 'any',
  tls_terminate: false,
  tls_passthrough: false,
  upstreams: [{ host: 'database', port: 5432 }],
  load_balancing_policy: 'round_robin',
};
const l4 = {
  id: 1,
  name: 'Database',
  listen_port: 5432,
  protocol: 'tcp',
  is_active: true,
  routes: [route],
  created_at: date,
  updated_at: date,
};
const audit = {
  id: 1,
  action: 'proxy.update',
  resource_type: 'proxy',
  resource_id: 1,
  resource_name: 'Analytics',
  user_id: 1,
  user,
  ip_address: '10.0.0.24',
  status: 'success',
  created_at: date,
  details: { changes: { name: { old: 'App', new: 'Analytics' } } },
};
const paged = (items: unknown[]) => ({
  items,
  total: items.length,
  page: 1,
  limit: 20,
  total_pages: 1,
});
export async function mockApp(
  page: Page,
  options: { authenticated?: boolean; theme?: string } = {},
) {
  await page.addInitScript(
    ({ user, authenticated, theme }) => {
      localStorage.setItem('theme', theme);
      if (authenticated)
        localStorage.setItem(
          'auth-storage',
          JSON.stringify({
            state: {
              user,
              accessToken: 'test-token',
              refreshToken: 'test-refresh',
              isAuthenticated: true,
            },
            version: 0,
          }),
        );
    },
    { user, authenticated: options.authenticated !== false, theme: options.theme ?? 'light' },
  );
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname.replace(/\/$/, '');
    let data: unknown = {};
    if (path === '/api/auth/me') data = user;
    else if (path === '/api/auth/registration-status') data = { open: true };
    else if (path === '/api/auth/sso/status') data = { enabled: false };
    else if (path === '/api/auth/sso/config')
      data = {
        enabled: true,
        issuer: 'https://sso.example.com',
        client_id: 'waygates',
        has_client_secret: true,
        auto_provision: true,
        default_role: 'viewer',
        button_label: 'Continue with SSO',
        base_url: 'https://admin.waygates.io',
        redirect_uri: 'https://admin.waygates.io/api/auth/sso/callback',
      };
    else if (path === '/api/auth/oauth/providers')
      data = [{ id: 'google', name: 'Google', available: true, enabled: true }];
    else if (path === '/api/sync/status')
      data = { last_sync_success: true, is_syncing: false, last_sync_time: date };
    else if (path === '/api/status') data = { user_setup_complete: true, caddy_status: 'healthy' };
    else if (path === '/api/health')
      data = {
        status: 'healthy',
        service: 'waygates',
        version: '1.0.0',
        uptime: '3 days',
        components: { database: 'healthy' },
      };
    else if (path === '/api/proxies/stats')
      data = {
        total: 3,
        active: 3,
        inactive: 0,
        by_type: { reverse_proxy: 1, static: 1, redirect: 1 },
      };
    else if (path === '/api/proxies')
      data = req.method() === 'POST' ? { ...req.postDataJSON(), id: 4 } : paged(proxies);
    else if (/^\/api\/proxies\/\d+$/.test(path))
      data = proxies.find((p) => p.id === Number(path.split('/').pop())) ?? proxy;
    else if (/^\/api\/proxies\/\d+\/acl$/.test(path)) data = [];
    else if (path === '/api/l4-proxies/stats')
      data = {
        total_proxies: 1,
        active_proxies: 1,
        total_routes: 1,
        tcp_proxies: 1,
        udp_proxies: 0,
      };
    else if (path === '/api/l4-proxies') data = paged([l4]);
    else if (/^\/api\/l4-proxies\/\d+$/.test(path)) data = l4;
    else if (path === '/api/acl/groups') data = paged([group]);
    else if (/^\/api\/acl\/groups\/\d+$/.test(path)) data = group;
    else if (path.endsWith('/waygates-auth')) data = group.waygates_auth;
    else if (/\/(ip-rules|basic-auth|providers|oauth-restrictions|usage)$/.test(path)) data = [];
    else if (path === '/api/acl/branding')
      data = {
        id: 1,
        title: 'Waygates',
        subtitle: 'Sign in to continue',
        primary_color: '',
        background_color: '',
        updated_at: date,
      };
    else if (path === '/api/acl/options')
      data = {
        requires_auth: true,
        waygates_auth: { enabled: true },
        oauth_providers: [{ id: 'google', name: 'Google', enabled: true }],
      };
    else if (path === '/api/audit-logs') data = paged([audit]);
    else if (path === '/api/audit-logs/event-groups') data = [];
    else if (path === '/api/audit-logs/config')
      data = { enabled: true, retention_days: 30, events: {} };
    else if (/^\/api\/audit-logs\/\d+$/.test(path)) data = audit;
    else if (path === '/api/settings/404') data = { mode: 'default', redirect_url: '' };
    else if (path === '/api/settings/metrics-publish')
      data = {
        enabled: false,
        host: '',
        path: '/metrics',
        basic_auth_user: '',
        has_basic_auth: false,
        allowed_cidrs: [],
      };
    else if (path === '/api/users') data = paged([user]);
    else if (path === '/api/metrics/traffic')
      data = {
        range: url.searchParams.get('range'),
        step_seconds: 3600,
        points: Array.from({ length: 24 }, (_, i) => ({
          t: new Date(Date.UTC(2026, 10, 18, i)).toISOString(),
          req_2xx: 200 + ((i * 157) % 600),
          req_3xx: 10,
          req_4xx: 5,
          req_5xx: 1,
          req_other: 0,
          p95_ms: 42 + i,
          p50_ms: 20,
          bytes_in: 10000,
          bytes_out: 30000,
          in_flight: 1,
        })),
      };
    else if (path === '/api/caddy-logs/stream') {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: 'data: {"level":"info","ts":1794996000,"msg":"server is listening","logger":"http"}\n\n',
      });
      return;
    } else if (path.includes('config-preview') || path.startsWith('/api/caddy-config'))
      data = {
        apps: {
          http: {
            servers: {
              proxy: {
                listen: [':443'],
                routes: [
                  {
                    match: [{ host: [proxy.hostname] }],
                    handle: [{ handler: 'reverse_proxy', upstreams: [{ dial: 'analytics:3000' }] }],
                  },
                ],
              },
            },
          },
        },
      };
    await route.fulfill({ json: { success: true, data } });
  });
}
