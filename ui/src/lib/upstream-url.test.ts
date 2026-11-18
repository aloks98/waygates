import { describe, expect, it } from 'vitest';

import { formatUpstream, parseUpstream } from './upstream-url';
describe('upstream URL conversion', () => {
  it('preserves ports and normalizes IPv6 for the backend', () => {
    expect(parseUpstream('https://[2001:db8::1]:8443')).toEqual({
      host: '2001:db8::1',
      port: 8443,
      scheme: 'https',
    });
    expect(formatUpstream({ host: '2001:db8::1', port: 8443, scheme: 'https' })).toBe(
      'https://[2001:db8::1]:8443',
    );
    expect(parseUpstream('https://service')).toEqual({
      host: 'service',
      port: 443,
      scheme: 'https',
    });
  });
  it.each([
    'ftp://service',
    'http://service/path',
    'http://user:secret@service',
    'http://service?key=value',
    'http://service:99999',
    'not a url',
  ])('rejects destinations the upstream model cannot represent: %s', (value) => {
    expect(parseUpstream(value)).toBeNull();
  });
});
