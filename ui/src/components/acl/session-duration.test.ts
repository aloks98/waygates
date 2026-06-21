import { describe, expect, it } from 'vitest';

import { durationToSeconds, secondsToDuration } from './session-duration';

describe('secondsToDuration', () => {
  it('picks the largest whole unit', () => {
    expect(secondsToDuration(3600)).toEqual({ value: 1, unit: 'hours' });
    expect(secondsToDuration(86400)).toEqual({ value: 1, unit: 'days' });
    expect(secondsToDuration(1800)).toEqual({ value: 30, unit: 'minutes' });
    expect(secondsToDuration(90000)).toEqual({ value: 25, unit: 'hours' }); // not a whole day
  });
});

describe('durationToSeconds', () => {
  it('converts and clamps to [60, 604800]', () => {
    expect(durationToSeconds(1, 'hours')).toBe(3600);
    expect(durationToSeconds(2, 'days')).toBe(172800);
    expect(durationToSeconds(0, 'minutes')).toBe(60); // clamp low
    expect(durationToSeconds(100, 'days')).toBe(604800); // clamp high (7 days)
  });

  it('round-trips', () => {
    const s = durationToSeconds(3, 'hours');
    expect(secondsToDuration(s)).toEqual({ value: 3, unit: 'hours' });
  });
});
