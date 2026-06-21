import { Plus } from 'lucide-react';
import { describe, expect, it } from 'vitest';

import { getActionMeta, toneTextClass } from './activity-actions';

describe('getActionMeta', () => {
  it('maps a known action to label + tone + icon', () => {
    const m = getActionMeta('proxy.create');
    expect(m.label).toBe('created a proxy');
    expect(m.tone).toBe('create');
    expect(m.icon).toBe(Plus);
  });

  it('classifies delete and failed-auth tones', () => {
    expect(getActionMeta('proxy.delete').tone).toBe('delete');
    expect(getActionMeta('auth.login_failed').tone).toBe('auth');
  });

  it('falls back safely for an unknown action', () => {
    const m = getActionMeta('something.weird');
    expect(m.tone).toBe('neutral');
    expect(typeof m.label).toBe('string');
    expect(m.icon).toBeTruthy();
  });
});

describe('toneTextClass', () => {
  it('returns a class string per tone and a default', () => {
    expect(toneTextClass('delete')).toContain('text-');
    expect(toneTextClass('neutral')).toContain('text-');
  });
});
