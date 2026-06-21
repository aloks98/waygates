import { format, isSameDay, subDays } from 'date-fns';

import type { AuditLog } from '@/types/audit';

export interface DayGroup {
  key: string;
  label: string;
  logs: AuditLog[];
}

function dayLabel(d: Date, now: Date): string {
  if (isSameDay(d, now)) return 'Today';
  if (isSameDay(d, subDays(now, 1))) return 'Yesterday';
  return format(d, 'MMM d, yyyy');
}

export function groupLogsByDay(logs: AuditLog[], now: Date = new Date()): DayGroup[] {
  const groups: DayGroup[] = [];
  let current: DayGroup | null = null;
  for (const log of logs) {
    const d = new Date(log.created_at);
    const key = format(d, 'yyyy-MM-dd');
    if (!current || current.key !== key) {
      current = { key, label: dayLabel(d, now), logs: [] };
      groups.push(current);
    }
    current.logs.push(log);
  }
  return groups;
}
