import { describe, expect, it } from 'vitest';

import type { ProxyConfig } from '@/types/proxy';

import {
  mapProxyToRedirectDefaults,
  mapProxyToReverseDefaults,
  mapProxyToStaticDefaults,
  mapRedirectValuesToRequest,
  mapReverseValuesToRequest,
  mapStaticValuesToRequest,
  REVERSE_PROXY_DEFAULTS,
} from './proxy-form-mappers';

const baseProxy = (over: Partial<ProxyConfig>): ProxyConfig => ({
  id: 1,
  type: 'reverse_proxy',
  name: 'svc',
  hostname: 'svc.example.com',
  ssl_enabled: true,
  ssl_forced: false,
  block_exploits: null,
  tls_insecure_skip_verify: null,
  is_active: true,
  created_at: '',
  updated_at: '',
  ...over,
});

describe('reverse proxy mappers', () => {
  it('defaults seed one empty upstream and leave settings at inherit', () => {
    expect(REVERSE_PROXY_DEFAULTS.upstreams).toHaveLength(1);
    // null = inherit (system default when ungrouped) — a brand-new proxy
    // shouldn't hard-code `true` where "inherit" is equally correct and
    // keeps working if the proxy is later added to a group.
    expect(REVERSE_PROXY_DEFAULTS.ssl_enabled).toBeNull();
    expect(REVERSE_PROXY_DEFAULTS.group_id).toBeNull();
    expect(REVERSE_PROXY_DEFAULTS.request_headers).toEqual([]);
  });

  it('maps a proxy with multiple upstreams + headers to defaults', () => {
    const d = mapProxyToReverseDefaults(
      baseProxy({
        upstreams: [
          { host: 'a', port: 80, scheme: 'http' },
          { host: 'b', port: 443, scheme: 'https' },
        ],
        load_balancing: {
          strategy: 'least_conn',
          health_checks: {
            enabled: true,
            path: '/up',
            interval: '10s',
            timeout: '2s',
            unhealthy_threshold: 3,
            healthy_threshold: 2,
          },
        },
        custom_headers: { request: { 'X-A': '1' }, response: { 'X-B': '2' } },
      }),
    );
    expect(d.upstreams).toHaveLength(2);
    expect(d.lb_strategy).toBe('least_conn');
    expect(d.health_check_enabled).toBe(true);
    expect(d.health_check_path).toBe('/up');
    expect(d.request_headers).toEqual([{ name: 'X-A', value: '1' }]);
    expect(d.response_headers).toEqual([{ name: 'X-B', value: '2' }]);
  });

  it('omits load_balancing + custom_headers when single upstream and no headers', () => {
    const req = mapReverseValuesToRequest({
      ...REVERSE_PROXY_DEFAULTS,
      name: 'svc',
      hostname: 'svc.example.com',
      upstreams: [{ host: 'a', port: 80, scheme: 'http' }],
    });
    expect(req.type).toBe('reverse_proxy');
    expect(req.load_balancing).toBeUndefined();
    expect(req.custom_headers).toBeUndefined();
  });

  it('includes load_balancing with health_checks when >1 upstream', () => {
    const req = mapReverseValuesToRequest({
      ...REVERSE_PROXY_DEFAULTS,
      name: 'svc',
      hostname: 'svc.example.com',
      upstreams: [
        { host: 'a', port: 80, scheme: 'http' },
        { host: 'b', port: 81, scheme: 'http' },
      ],
      lb_strategy: 'ip_hash',
      health_check_enabled: true,
      health_check_path: '/h',
      health_check_interval: '15s',
      health_check_timeout: '3s',
    });
    expect(req.load_balancing?.strategy).toBe('ip_hash');
    expect(req.load_balancing?.health_checks?.path).toBe('/h');
  });

  it('drops empty/blank header names', () => {
    const req = mapReverseValuesToRequest({
      ...REVERSE_PROXY_DEFAULTS,
      name: 'svc',
      hostname: 'svc.example.com',
      request_headers: [
        { name: ' ', value: 'x' },
        { name: 'X-Keep', value: 'y' },
      ],
    });
    expect(req.custom_headers?.request).toEqual({ 'X-Keep': 'y' });
  });
});

describe('redirect mappers', () => {
  it('round-trips nested redirect config', () => {
    const d = mapProxyToRedirectDefaults(
      baseProxy({
        type: 'redirect',
        redirect: {
          target: 'https://x',
          status_code: 308,
          preserve_path: true,
          preserve_query: false,
        },
      }),
    );
    expect(d.target).toBe('https://x');
    expect(d.status_code).toBe(308);
    const req = mapRedirectValuesToRequest(d);
    expect(req.type).toBe('redirect');
    expect(req.redirect.status_code).toBe(308);
    expect(req.redirect.preserve_path).toBe(true);
  });
});

describe('static mappers', () => {
  it('wraps/unwraps try_files and drops blanks', () => {
    const d = mapProxyToStaticDefaults(
      baseProxy({
        type: 'static',
        static: {
          root_path: '/srv',
          index_file: 'index.html',
          browse: true,
          template_rendering: false,
          try_files: ['{path}', 'index.html'],
        },
      }),
    );
    expect(d.try_files).toEqual([{ value: '{path}' }, { value: 'index.html' }]);
    const req = mapStaticValuesToRequest({ ...d, try_files: [{ value: '{path}' }, { value: '' }] });
    expect(req.type).toBe('static');
    expect(req.static.try_files).toEqual(['{path}']);
  });
});
