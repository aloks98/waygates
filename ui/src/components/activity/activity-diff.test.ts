import { describe, expect, it } from 'vitest';

import { extractFieldChanges, formatDiffValue } from './activity-diff';

describe('extractFieldChanges', () => {
  it('reads the per-field {old,new} map from details.changes', () => {
    // Every update event (proxy/acl/settings) nests changes the same way:
    // { ...metadata, changes: { field: {old,new} } }.
    const out = extractFieldChanges({
      hostname: 'b.com',
      type: 'reverse_proxy',
      changes: {
        hostname: { old: 'a.com', new: 'b.com' },
        is_active: { old: true, new: false },
      },
    });
    expect(out).toEqual([
      { field: 'hostname', oldValue: 'a.com', newValue: 'b.com' },
      { field: 'is_active', oldValue: true, newValue: false },
    ]);
  });

  it('ignores entries in changes that are not {old,new} objects', () => {
    expect(extractFieldChanges({ changes: { ok: { old: 1, new: 2 }, note: 'hi' } })).toEqual([
      { field: 'ok', oldValue: 1, newValue: 2 },
    ]);
  });

  it('returns [] when there is no changes map', () => {
    expect(extractFieldChanges({ hostname: 'b.com', type: 'reverse_proxy' })).toEqual([]);
    expect(extractFieldChanges({ changes: 'nope' })).toEqual([]);
    expect(extractFieldChanges({})).toEqual([]);
    expect(extractFieldChanges(null)).toEqual([]);
    expect(extractFieldChanges(undefined)).toEqual([]);
  });
});

describe('formatDiffValue', () => {
  it('renders primitives and objects compactly', () => {
    expect(formatDiffValue('x')).toBe('x');
    expect(formatDiffValue(true)).toBe('true');
    expect(formatDiffValue(null)).toBe('—');
    expect(formatDiffValue(undefined)).toBe('—');
    expect(formatDiffValue({ a: 1 })).toBe('{"a":1}');
  });
});
