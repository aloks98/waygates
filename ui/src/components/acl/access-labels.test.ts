import { describe, expect, it } from 'vitest';

import type { ACLGroup } from '@/types/acl';

import { getModeLabel, getRuleTypeLabel, groupAuthMethods } from './access-labels';

describe('getRuleTypeLabel', () => {
  it('de-jargons rule types', () => {
    expect(getRuleTypeLabel('allow')).toBe('Allow');
    expect(getRuleTypeLabel('deny')).toBe('Block');
    expect(getRuleTypeLabel('bypass')).toBe('Trusted — skip auth');
    expect(getRuleTypeLabel('weird')).toBe('weird');
  });
});

describe('getModeLabel', () => {
  it('de-jargons combination modes', () => {
    expect(getModeLabel('any')).toBe('Any match');
    expect(getModeLabel('all')).toBe('All required');
    expect(getModeLabel('ip_bypass')).toBe('Trusted IPs first');
    expect(getModeLabel('weird')).toBe('weird');
  });
});

describe('groupAuthMethods', () => {
  it('derives configured methods from the group relations', () => {
    const group = {
      id: 1,
      name: 'g',
      combination_mode: 'any',
      created_at: '',
      updated_at: '',
      ip_rules: [{ id: 1 }],
      basic_auth_users: [{ id: 1 }],
      external_providers: [{ id: 1 }],
      waygates_auth: { enabled: true, allowed_providers: ['google'] },
    } as unknown as ACLGroup;
    expect(groupAuthMethods(group).map((m) => m.key)).toEqual([
      'ip',
      'basic',
      'oauth',
      'account',
      'forward',
    ]);
  });

  it('returns only configured methods (empty group → none)', () => {
    const group = { id: 1, name: 'g', combination_mode: 'any' } as unknown as ACLGroup;
    expect(groupAuthMethods(group)).toEqual([]);
  });

  it('account requires waygates_auth.enabled, oauth requires allowed_providers', () => {
    const group = {
      id: 1,
      name: 'g',
      combination_mode: 'any',
      waygates_auth: { enabled: false, allowed_providers: [] },
    } as unknown as ACLGroup;
    expect(groupAuthMethods(group)).toEqual([]);
  });
});
