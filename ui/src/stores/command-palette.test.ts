import { expect, test } from 'vitest';

import { useCommandPalette } from './command-palette';

test('toggle flips open and setOpen sets it', () => {
  useCommandPalette.getState().setOpen(false);
  expect(useCommandPalette.getState().open).toBe(false);
  useCommandPalette.getState().toggle();
  expect(useCommandPalette.getState().open).toBe(true);
  useCommandPalette.getState().setOpen(false);
  expect(useCommandPalette.getState().open).toBe(false);
});
