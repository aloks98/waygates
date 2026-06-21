import { describe, expect, it } from 'vitest';

import { parseImportJson } from './proxy-import';

describe('parseImportJson', () => {
  it('accepts a non-empty array', () => {
    const r = parseImportJson('[{"type":"redirect","name":"x","hostname":"x.test"}]');
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.items).toHaveLength(1);
  });
  it('rejects non-JSON', () => {
    const r = parseImportJson('not json');
    expect(r).toEqual({ ok: false, error: 'Not valid JSON.' });
  });
  it('rejects a non-array (object)', () => {
    const r = parseImportJson('{"type":"redirect"}');
    expect(r).toEqual({ ok: false, error: 'Expected a JSON array of proxies.' });
  });
  it('rejects an empty array', () => {
    const r = parseImportJson('[]');
    expect(r).toEqual({ ok: false, error: 'The file contains no proxies.' });
  });
});
