import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';

test('test harness renders DOM', () => {
  render(<button>hello</button>);
  expect(screen.getByText('hello')).toBeInTheDocument();
});
