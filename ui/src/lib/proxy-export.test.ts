import { describe, expect, it } from 'vitest';

import { summarizeBulkResults } from './proxy-export';

describe('summarizeBulkResults', () => {
  it('counts fulfilled vs rejected', () => {
    const results: PromiseSettledResult<unknown>[] = [
      { status: 'fulfilled', value: 1 },
      { status: 'rejected', reason: new Error('x') },
      { status: 'fulfilled', value: 2 },
    ];
    expect(summarizeBulkResults(results)).toEqual({ succeeded: 2, failed: 1 });
  });
  it('handles empty', () => {
    expect(summarizeBulkResults([])).toEqual({ succeeded: 0, failed: 0 });
  });
});
