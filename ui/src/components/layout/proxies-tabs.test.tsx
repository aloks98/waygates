import { render, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';

vi.mock('@tanstack/react-router', () => ({
  Link: ({ to, children, ...props }: any) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));

import { ProxiesTabs } from './proxies-tabs';

test('renders both tabs with correct hrefs', () => {
  render(<ProxiesTabs active="http" />);
  expect(screen.getByRole('link', { name: /HTTP/i })).toHaveAttribute('href', '/proxies');
  expect(screen.getByRole('link', { name: /TCP\/UDP/i })).toHaveAttribute(
    'href',
    '/proxies/tcp-udp',
  );
});

test('marks the active tab with aria-current', () => {
  render(<ProxiesTabs active="tcp-udp" />);
  expect(screen.getByRole('link', { name: /TCP\/UDP/i })).toHaveAttribute('aria-current', 'page');
  expect(screen.getByRole('link', { name: /HTTP/i })).not.toHaveAttribute('aria-current');
});
