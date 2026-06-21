export type DurationUnit = 'minutes' | 'hours' | 'days';

export const SESSION_TTL_MIN = 60;
export const SESSION_TTL_MAX = 604800; // 7 days

const UNIT_SECONDS: Record<DurationUnit, number> = {
  minutes: 60,
  hours: 3600,
  days: 86400,
};

export function secondsToDuration(seconds: number): { value: number; unit: DurationUnit } {
  if (seconds > 0 && seconds % UNIT_SECONDS.days === 0) {
    return { value: seconds / UNIT_SECONDS.days, unit: 'days' };
  }
  if (seconds > 0 && seconds % UNIT_SECONDS.hours === 0) {
    return { value: seconds / UNIT_SECONDS.hours, unit: 'hours' };
  }
  return { value: Math.max(1, Math.round(seconds / UNIT_SECONDS.minutes)), unit: 'minutes' };
}

export function durationToSeconds(value: number, unit: DurationUnit): number {
  const raw = Math.round(value) * UNIT_SECONDS[unit];
  return Math.min(SESSION_TTL_MAX, Math.max(SESSION_TTL_MIN, raw));
}
