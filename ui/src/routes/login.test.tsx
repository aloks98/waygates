import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
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
const getJson = vi.fn().mockResolvedValue({ success: true, data: { open: false } });
vi.mock('@/lib/api', () => ({
  publicApi: { post: () => ({ json: postJson }), get: () => ({ json: getJson }) },
}));
vi.mock('../lib/api', () => ({
  publicApi: { post: () => ({ json: postJson }), get: () => ({ json: getJson }) },
}));

import { LoginPage } from './login';

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

test('shows validation errors on empty submit and does not call the API', async () => {
  render(<LoginPage />, { wrapper });
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
  render(<LoginPage />, { wrapper });
  await userEvent.type(screen.getByLabelText(/username or email/i), 'admin');
  await userEvent.type(screen.getByLabelText(/password/i), 'secret');
  await userEvent.click(screen.getByRole('button', { name: /sign in/i }));
  await waitFor(() => expect(postJson).toHaveBeenCalled());
});
