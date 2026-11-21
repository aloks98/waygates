import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, test, vi } from 'vitest';

const { post, postJson } = vi.hoisted(() => ({ post: vi.fn(), postJson: vi.fn() }));

vi.mock('@/lib/api', () => ({ publicApi: { post } }));

import { ACLLoginForm } from './acl-login-form';

beforeEach(() => {
  post.mockReset().mockReturnValue({ json: postJson });
  postJson.mockReset().mockResolvedValue({ success: false, message: 'Invalid credentials' });
});

test('clears the required-password error while typing a password', async () => {
  const user = userEvent.setup();
  render(<ACLLoginForm />);
  await user.type(screen.getByLabelText('Username or Email'), 'alice');
  const password = screen.getByLabelText('Password');
  await user.click(password);
  await user.tab();
  expect(await screen.findByText('Password is required')).toBeInTheDocument();

  await user.type(password, 'example-password');

  await waitFor(() => expect(screen.queryByText('Password is required')).not.toBeInTheDocument());
  expect(password).toHaveAttribute('aria-invalid', 'false');
});

test('submits entered credentials after correcting an empty password', async () => {
  const user = userEvent.setup();
  render(<ACLLoginForm redirectUrl="https://protected.example.com/" />);
  await user.type(screen.getByLabelText('Username or Email'), 'alice');
  await user.click(screen.getByRole('button', { name: 'Sign in' }));
  expect(await screen.findByText('Password is required')).toBeInTheDocument();
  expect(post).not.toHaveBeenCalled();

  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'example-password' } });
  await user.click(screen.getByRole('button', { name: 'Sign in' }));

  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('auth/acl/login', {
      json: {
        email: 'alice',
        password: 'example-password',
        redirect: 'https://protected.example.com/',
      },
    }),
  );
});
