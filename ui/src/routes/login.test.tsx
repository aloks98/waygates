import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, test, vi } from 'vitest';

const navigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigate,
  useSearch: () => ({}),
  Link: ({ to, children, ...props }: any) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));
const postJson = vi.fn();
vi.mock('@/lib/api', () => ({ publicApi: { post: () => ({ json: postJson }) } }));
vi.mock('../lib/api', () => ({ publicApi: { post: () => ({ json: postJson }) } }));

import { LoginPage } from './login';

test('shows validation errors on empty submit and does not call the API', async () => {
  render(<LoginPage />);
  await userEvent.click(screen.getByRole('button', { name: /sign in/i }));
  expect(await screen.findByText(/username or email is required/i)).toBeInTheDocument();
  expect(screen.getByText(/password is required/i)).toBeInTheDocument();
  expect(postJson).not.toHaveBeenCalled();
});

test('submits valid credentials to the API', async () => {
  postJson.mockResolvedValueOnce({
    success: true,
    data: { access_token: 'a', refresh_token: 'b' },
  });
  render(<LoginPage />);
  await userEvent.type(screen.getByLabelText(/username or email/i), 'admin');
  await userEvent.type(screen.getByLabelText(/password/i), 'secret');
  await userEvent.click(screen.getByRole('button', { name: /sign in/i }));
  await waitFor(() => expect(postJson).toHaveBeenCalled());
});
