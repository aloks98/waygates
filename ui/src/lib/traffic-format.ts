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
