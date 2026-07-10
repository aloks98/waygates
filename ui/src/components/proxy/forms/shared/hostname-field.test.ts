import { describe, expect, it } from 'vitest';

import { deriveLabel } from './hostname-field';

describe('deriveLabel', () => {
  it('strips the base domain suffix on an exact match', () => {
    expect(deriveLabel('abc.group.acme.in', 'group.acme.in')).toBe('abc');
  });

  it('returns an empty string when the hostname does not sit under the base domain', () => {
    expect(deriveLabel('abc.example.com', 'group.acme.in')).toBe('');
  });

  it('returns an empty string when the hostname equals the base domain itself', () => {
    // No separating label — "group.acme.in" is not "<label>.group.acme.in".
    // The field must surface this as a validation error rather than submit
    // an empty label.
    expect(deriveLabel('group.acme.in', 'group.acme.in')).toBe('');
  });
});
