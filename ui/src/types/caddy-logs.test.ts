import { describe, expect, it } from 'vitest';

import { parseCaddyLogLine } from './caddy-logs';

describe('parseCaddyLogLine', () => {
  it('parses a valid access log JSON line', () => {
    const raw = JSON.stringify({
      ts: 1700000000.123,
      level: 'info',
      logger: 'http.log.access',
      msg: 'handled request',
      status: 200,
      method: 'GET',
      host: 'example.com',
      uri: '/api/health',
      duration: 0.002,
    });

    const result = parseCaddyLogLine(raw);

    expect(result.raw).toBe(raw);
    expect(result.ts).toBeCloseTo(1700000000.123);
    expect(result.level).toBe('info');
    expect(result.logger).toBe('http.log.access');
    expect(result.msg).toBe('handled request');
    expect(result.status).toBe(200);
    expect(result.method).toBe('GET');
    expect(result.host).toBe('example.com');
    expect(result.uri).toBe('/api/health');
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
    });

    const result = parseCaddyLogLine(raw);
    expect(result.raw).toBe(raw);
    expect(result.ts).toBeUndefined();
    expect(result.status).toBeUndefined();
    expect(result.msg).toBeUndefined();
  });
});
