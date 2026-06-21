import { describe, expect, it } from 'vitest';

import type { AuditLog } from '@/types/audit';

import { groupLogsByDay } from './activity-grouping';

const log = (id: number, iso: string): AuditLog =>
  ({ id, action: 'proxy.create', status: 'success', created_at: iso }) as AuditLog;

describe('groupLogsByDay', () => {
  const now = new Date('2026-06-21T12:00:00Z');

  it('labels today / yesterday / older and groups in order', () => {
    const groups = groupLogsByDay(
      [
        log(1, '2026-06-21T09:00:00Z'),
        log(2, '2026-06-21T08:00:00Z'),
        log(3, '2026-06-20T10:00:00Z'),
        log(4, '2026-06-18T10:00:00Z'),
      ],
      now,
    );
    expect(groups.map((g) => g.label)).toEqual(['Today', 'Yesterday', 'Jun 18, 2026']);
    expect(groups[0].logs.map((l) => l.id)).toEqual([1, 2]);
    expect(groups[1].logs.map((l) => l.id)).toEqual([3]);
  });

  it('returns [] for no logs', () => {
    expect(groupLogsByDay([], now)).toEqual([]);
  });
});
