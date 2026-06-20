import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { expect, test, vi } from 'vitest';

import { TagsInput } from './tags-input';

function Harness({ onChange }: { onChange?: (v: string[]) => void }) {
  const [value, setValue] = useState<string[]>([]);
  return (
    <TagsInput
      value={value}
      onValueChange={(v) => {
        setValue(v);
        onChange?.(v);
      }}
      placeholder="Add email..."
      delimiters={['Enter', ',']}
      validation={{ pattern: /^[^@\s]+@[^@\s]+\.[^@\s]+$/ }}
    />
  );
}

test('adds a valid tag on Enter', async () => {
  const onChange = vi.fn();
  render(<Harness onChange={onChange} />);
  const input = screen.getByPlaceholderText('Add email...');
  await userEvent.type(input, 'a@b.com{Enter}');
  expect(onChange).toHaveBeenLastCalledWith(['a@b.com']);
  expect(screen.getByText('a@b.com')).toBeInTheDocument();
});

test('rejects an invalid tag (fails pattern)', async () => {
  const onChange = vi.fn();
  render(<Harness onChange={onChange} />);
  const input = screen.getByPlaceholderText('Add email...');
  await userEvent.type(input, 'not-an-email{Enter}');
  expect(onChange).not.toHaveBeenCalled();
  expect(screen.queryByText('not-an-email')).not.toBeInTheDocument();
});

test('dedupes: no onValueChange when re-adding existing tag', async () => {
  const onChange = vi.fn();
  render(<Harness onChange={onChange} />);
  const input = screen.getByPlaceholderText('Add email...');
  await userEvent.type(input, 'a@b.com{Enter}');
  expect(onChange).toHaveBeenCalledTimes(1);
  // Re-add the same tag — should be silently ignored
  await userEvent.type(input, 'a@b.com{Enter}');
  expect(onChange).toHaveBeenCalledTimes(1);
});

// Chip removal is handled internally by Base UI's Combobox chip remove control.
// The chip renders a remove button that updates `value` via `onValueChange` without
// any custom code in TagsInput. Testing this reliably in jsdom would require
// querying Base UI's internal DOM structure, which is fragile; the behaviour is
// covered by Base UI's own test suite.
