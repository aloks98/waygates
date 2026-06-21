import { describe, expect, it } from 'vitest';

import type { L4Proxy } from '@/types/l4-proxy';

import {
  createDefaultRoute,
  L4_PROXY_DEFAULTS,
  mapL4FormValuesToRequest,
  mapL4ProxyToDefaults,
} from './l4-proxy-form-mappers';

describe('createDefaultRoute / defaults', () => {
  it('seeds one route with one empty upstream', () => {
    expect(L4_PROXY_DEFAULTS.routes).toHaveLength(1);
    expect(L4_PROXY_DEFAULTS.routes?.[0].upstreams).toHaveLength(1);
    expect(L4_PROXY_DEFAULTS.protocol).toBe('tcp');
    expect(createDefaultRoute().matcher_type).toBe('any');
  });
});

describe('mapL4ProxyToDefaults', () => {
  it('wraps sni_hostnames / allowed_ip_ranges as {value}[]', () => {
    const proxy = {
      id: 1,
      name: 'p',
      listen_port: 5432,
      protocol: 'tcp',
      is_active: true,
      created_at: '',
      updated_at: '',
      routes: [
        {
          priority: 0,
          matcher_type: 'tls',
          sni_hostnames: ['a.com', 'b.com'],
          upstreams: [{ host: 'h', port: 5432 }],
          load_balancing_policy: 'round_robin',
          tls_terminate: true,
          tls_passthrough: false,
          created_at: '',
          updated_at: '',
        },
      ],
    } as unknown as L4Proxy;
    const d = mapL4ProxyToDefaults(proxy);
    expect(d.routes?.[0].sni_hostnames).toEqual([{ value: 'a.com' }, { value: 'b.com' }]);
    expect(d.routes?.[0].allowed_ip_ranges).toEqual([]);
    expect(d.routes?.[0].tls_terminate).toBe(true);
  });
  it('falls back to a default route when none', () => {
    const proxy = {
      id: 1,
      name: 'p',
      listen_port: 80,
      protocol: 'udp',
      is_active: true,
      created_at: '',
      updated_at: '',
    } as unknown as L4Proxy;
    expect(mapL4ProxyToDefaults(proxy).routes).toHaveLength(1);
  });
});

describe('mapL4FormValuesToRequest', () => {
  const base = () => ({
    ...L4_PROXY_DEFAULTS,
    name: 'p',
    routes: [createDefaultRoute()],
  });

  it('omits conditional matcher fields that do not match the matcher_type', () => {
    const v = base();
    v.routes[0].matcher_type = 'any';
    v.routes[0].sni_hostnames = [{ value: 'x.com' }];
    v.routes[0].regex_pattern = 'foo';
    const req = mapL4FormValuesToRequest(v);
    expect(req.routes?.[0].sni_hostnames).toBeUndefined();
    expect(req.routes?.[0].allowed_ip_ranges).toBeUndefined();
    expect(req.routes?.[0].regex_pattern).toBeUndefined();
  });

  it('unwraps + trims + drops blank sni for a tls matcher', () => {
    const v = base();
    v.routes[0].matcher_type = 'tls';
    v.routes[0].sni_hostnames = [{ value: 'a.com' }, { value: '  ' }];
    expect(mapL4FormValuesToRequest(v).routes?.[0].sni_hostnames).toEqual(['a.com']);
  });

  it('omits weight when not set', () => {
    const v = base();
    v.routes[0].upstreams = [{ host: 'h', port: 80 }];
    expect(mapL4FormValuesToRequest(v).routes?.[0].upstreams[0]).not.toHaveProperty('weight');
  });

  it('drops empty description', () => {
    const v = base();
    v.description = '';
    expect(mapL4FormValuesToRequest(v).description).toBeUndefined();
  });
});
