import { expect, test } from 'vitest';

import type { TrafficPoint } from '@/types/metrics';

import { formatBytes, formatMs, pointsToCategories, seriesFor } from './traffic-format';

const makePoint = (t: string, overrides: Partial<TrafficPoint> = {}): TrafficPoint => ({
  t,
  req_2xx: 0,
  req_3xx: 0,
  req_4xx: 0,
  req_5xx: 0,
  req_other: 0,
  bytes_in: 0,
  bytes_out: 0,
  p50_ms: 0,
  p95_ms: 0,
  in_flight: 0,
  ...overrides,
});

test('pointsToCategories returns HH:MM for 1h range', () => {
  const points = [makePoint('2026-06-23T10:30:00Z'), makePoint('2026-06-23T10:31:00Z')];
  const cats = pointsToCategories(points, '1h');
  expect(cats).toHaveLength(2);
  // Each label should match HH:MM format
  expect(cats[0]).toMatch(/^\d{2}:\d{2}$/);
  expect(cats[1]).toMatch(/^\d{2}:\d{2}$/);
});

test('pointsToCategories returns date+hour for 7d range', () => {
  const points = [makePoint('2026-06-23T10:30:00Z')];
  const cats = pointsToCategories(points, '7d');
  expect(cats).toHaveLength(1);
  // Should contain month abbreviation and numbers (e.g. "Jun 23 12:30")
  expect(cats[0]).toMatch(/[A-Z][a-z]{2}/);
});

test('pointsToCategories returns empty array for empty points', () => {
  expect(pointsToCategories([], '1h')).toEqual([]);
});

test('seriesFor extracts a numeric field from all points', () => {
  const points = [
    makePoint('2026-06-23T10:00:00Z', { req_2xx: 10 }),
    makePoint('2026-06-23T10:01:00Z', { req_2xx: 20 }),
  ];
  expect(seriesFor(points, 'req_2xx')).toEqual([10, 20]);
});

test('seriesFor handles empty points', () => {
  expect(seriesFor([], 'p50_ms')).toEqual([]);
});

test('formatBytes renders human-readable units', () => {
  expect(formatBytes(0)).toBe('0 B');
  expect(formatBytes(512)).toBe('512 B');
  expect(formatBytes(1024)).toBe('1.0 KB');
  expect(formatBytes(123456789)).toBe('117.7 MB');
  expect(formatBytes(5 * 1024 ** 3)).toBe('5.0 GB');
});

test('formatBytes guards against invalid input', () => {
  expect(formatBytes(-1)).toBe('0 B');
  expect(formatBytes(Number.NaN)).toBe('0 B');
});

test('formatMs renders durations with a unit', () => {
  expect(formatMs(0)).toBe('0 ms');
  expect(formatMs(5)).toBe('5 ms');
  expect(formatMs(12.34)).toBe('12.3 ms');
  expect(formatMs(123.4)).toBe('123 ms');
});

test('formatMs guards against invalid input', () => {
  expect(formatMs(-1)).toBe('0 ms');
  expect(formatMs(Number.NaN)).toBe('0 ms');
});
