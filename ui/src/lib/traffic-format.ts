import type { TrafficPoint, TrafficRange } from '@/types/metrics';

export function pointToCategory(t: string, range: TrafficRange): string {
  const date = new Date(t);
  if (range === '7d') {
    // Short date + hour: "Jun 22 14:00"
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
  }
  // 1h / 24h: "HH:MM" in local time
  return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
}

export function pointsToCategories(points: TrafficPoint[], range: TrafficRange): string[] {
  return points.map((p) => pointToCategory(p.t, range));
}

export function seriesFor<K extends keyof TrafficPoint>(points: TrafficPoint[], key: K): number[] {
  return points.map((p) => p[key] as number);
}

const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

/**
 * Formats a byte count into a human-readable string with a unit (e.g.
 * 123456789 -> "117.7 MB"). Uses 1024-based steps. Bytes show no decimals;
 * larger units show one. Used for the bandwidth chart axis and tooltip.
 */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const exp = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), BYTE_UNITS.length - 1);
  const value = bytes / 1024 ** exp;
  return `${exp === 0 ? value : value.toFixed(1)} ${BYTE_UNITS[exp]}`;
}

/**
 * Formats a millisecond duration with a "ms" unit (e.g. 12.34 -> "12.3 ms",
 * 123.4 -> "123 ms"). One decimal under 100ms, whole numbers above. Used for
 * the latency chart axis and tooltip.
 */
export function formatMs(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '0 ms';
  const rounded = ms >= 100 ? Math.round(ms) : Math.round(ms * 10) / 10;
  return `${rounded} ms`;
}
