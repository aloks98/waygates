import { describe, expect, it } from 'vitest';

import { isAccessLogLine, parseCaddyLogLine } from './caddy-logs';

describe('parseCaddyLogLine', () => {
  it('parses a valid access log JSON line (request details nested under `request`)', () => {
    const raw = JSON.stringify({
      ts: 1700000000.123,
      level: 'info',
      logger: 'http.log.access.srv0',
      msg: 'handled request',
      request: {
        remote_ip: '172.18.0.1',
        method: 'GET',
        host: 'example.com',
        uri: '/api/health',
      },
      status: 200,
      duration: 0.002,
    });

    const result = parseCaddyLogLine(raw);

    expect(result.raw).toBe(raw);
    expect(result.ts).toBeCloseTo(1700000000.123);
    expect(result.level).toBe('info');
    expect(result.logger).toBe('http.log.access.srv0');
    expect(result.msg).toBe('handled request');
    expect(result.status).toBe(200);
    expect(result.method).toBe('GET');
    expect(result.host).toBe('example.com');
    expect(result.uri).toBe('/api/health');
    expect(result.remoteIp).toBe('172.18.0.1');
    expect(result.duration).toBeCloseTo(0.002);
  });

  it('parses a valid runtime log JSON line', () => {
    const raw = JSON.stringify({
      ts: 1700000001.0,
      level: 'warn',
      logger: 'tls',
      msg: 'certificate expiring soon',
    });

    const result = parseCaddyLogLine(raw);

    expect(result.raw).toBe(raw);
    expect(result.ts).toBe(1700000001.0);
    expect(result.level).toBe('warn');
    expect(result.logger).toBe('tls');
    expect(result.msg).toBe('certificate expiring soon');
    expect(result.status).toBeUndefined();
    expect(result.method).toBeUndefined();
  });

  it('keeps the error and identifier of an ACME failure', () => {
    // `msg` is identical for every ACME failure; only `error` says what broke.
    const raw = JSON.stringify({
      ts: 1783664455.66108,
      level: 'error',
      logger: 'tls.obtain',
      msg: 'could not get certificate from issuer',
      identifier: 'profilarr.e412.in',
      issuer: 'acme-v02.api.letsencrypt.org-directory',
      error:
        'solving challenges: checking DNS propagation of "_acme-challenge.profilarr.e412.in.": lookup sanchez. on 127.0.0.11:53: no such host',
    });

    const result = parseCaddyLogLine(raw);

    expect(result.level).toBe('error');
    expect(result.logger).toBe('tls.obtain');
    expect(result.msg).toBe('could not get certificate from issuer');
    expect(result.identifier).toBe('profilarr.e412.in');
    expect(result.error).toContain('no such host');
  });

  it('leaves error and identifier undefined when absent', () => {
    const raw = JSON.stringify({ ts: 1700000001.0, level: 'info', msg: 'serving initial config' });

    const result = parseCaddyLogLine(raw);

    expect(result.error).toBeUndefined();
    expect(result.identifier).toBeUndefined();
  });

  // http.log.error nests `request` and carries a status, exactly like an access
  // entry. It must not be mistaken for one, or its msg/error vanish.
  it('parses an upstream failure without losing its message', () => {
    const raw = JSON.stringify({
      ts: 1783624023.3274474,
      level: 'error',
      logger: 'http.log.error',
      msg: 'dial tcp 192.168.150.50:8989: connect: connection refused',
      request: { remote_ip: '192.168.2.121', method: 'GET', host: 'sonarr.e412.in', uri: '/' },
      duration: 0.0022,
      status: 502,
    });

    const result = parseCaddyLogLine(raw);

    expect(isAccessLogLine(result)).toBe(false);
    expect(result.msg).toContain('connection refused');
    expect(result.status).toBe(502);
    expect(result.host).toBe('sonarr.e412.in');
  });

  it('parses a reverse_proxy warning including its upstream', () => {
    const raw = JSON.stringify({
      ts: 1783624032.210055,
      level: 'warn',
      logger: 'http.handlers.reverse_proxy',
      msg: 'aborting with incomplete response',
      upstream: '192.168.150.50:80',
      duration: 0.284640285,
      error: 'writing: H3_REQUEST_CANCELLED',
      request: { method: 'GET', host: 'nas.e412.in', uri: '/plugins/x.js' },
    });

    const result = parseCaddyLogLine(raw);

    expect(isAccessLogLine(result)).toBe(false);
    expect(result.upstream).toBe('192.168.150.50:80');
    expect(result.error).toBe('writing: H3_REQUEST_CANCELLED');
    expect(result.level).toBe('warn');
  });

  // admin.api logs request details at the top level rather than nesting them.
  it('reads top-level request details when there is no `request` object', () => {
    const raw = JSON.stringify({
      ts: 1783628783.2461596,
      level: 'info',
      logger: 'admin.api',
      msg: 'received request',
      method: 'POST',
      host: 'localhost:2019',
      uri: '/load',
      remote_ip: '127.0.0.1',
    });

    const result = parseCaddyLogLine(raw);

    expect(isAccessLogLine(result)).toBe(false);
    expect(result.method).toBe('POST');
    expect(result.uri).toBe('/load');
    expect(result.remoteIp).toBe('127.0.0.1');
  });

  it('prefers the nested request object over top-level keys', () => {
    const raw = JSON.stringify({
      logger: 'http.log.error',
      method: 'TOP',
      request: { method: 'NESTED', host: 'a.example', uri: '/x' },
    });

    expect(parseCaddyLogLine(raw).method).toBe('NESTED');
  });

  describe('isAccessLogLine', () => {
    it('accepts the access logger, with or without a server suffix', () => {
      expect(isAccessLogLine({ raw: '', logger: 'http.log.access' })).toBe(true);
      expect(isAccessLogLine({ raw: '', logger: 'http.log.access.srv0' })).toBe(true);
    });

    it('accepts a renamed access logger via its fixed msg', () => {
      expect(
        isAccessLogLine({ raw: '', logger: 'custom', msg: 'handled request', status: 200 }),
      ).toBe(true);
    });

    it('rejects runtime loggers that happen to carry a status', () => {
      expect(
        isAccessLogLine({ raw: '', logger: 'http.log.error', status: 502, method: 'GET' }),
      ).toBe(false);
      expect(
        isAccessLogLine({ raw: '', logger: 'http.handlers.reverse_proxy', method: 'GET' }),
      ).toBe(false);
    });
  });

  it('falls back to { raw } for non-JSON input', () => {
    const raw = 'this is not json';
    const result = parseCaddyLogLine(raw);
    expect(result).toEqual({ raw });
  });

  it('falls back to { raw } for malformed JSON', () => {
    const raw = '{ "ts": 123, "level":';
    const result = parseCaddyLogLine(raw);
    expect(result).toEqual({ raw });
  });

  it('ignores fields with wrong types', () => {
    const raw = JSON.stringify({
      ts: 'not-a-number',
      status: 'not-a-number',
      msg: 42,
      error: { nested: 'object' },
      identifier: 7,
    });

    const result = parseCaddyLogLine(raw);
    expect(result.raw).toBe(raw);
    expect(result.ts).toBeUndefined();
    expect(result.status).toBeUndefined();
    expect(result.msg).toBeUndefined();
    expect(result.error).toBeUndefined();
    expect(result.identifier).toBeUndefined();
  });
});
