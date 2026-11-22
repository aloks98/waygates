import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, test, vi } from 'vitest';

const { updateProvider, provider } = vi.hoisted(() => ({
  updateProvider: vi.fn(),
  provider: {
    id: 7,
    acl_group_id: 1,
    name: 'NexusID',
    provider_type: 'authentik',
    verify_url: 'https://id.example/outpost.goauthentik.io/auth/caddy',
    outpost_url: 'https://outpost.example:9443',
    auth_redirect_url: 'https://obsolete.example/login',
    headers_to_copy: ['X-Authentik-Username'],
    created_at: '2026-11-22T00:00:00Z',
  },
}));

vi.mock('@/hooks', () => ({
  useExternalProviders: () => ({ providers: [provider], isLoading: false }),
  useAddExternalProvider: () => ({ addProvider: vi.fn(), isAdding: false }),
  useUpdateExternalProvider: () => ({ updateProvider, isUpdating: false }),
  useDeleteExternalProvider: () => ({ deleteProvider: vi.fn(), isDeleting: false }),
}));
vi.mock('@/hooks/use-permissions', () => ({
  usePermissions: () => ({ canUpdateAccess: true, canDeleteAccess: true }),
}));

import { ExternalProvidersTab } from './external-providers-tab';

beforeEach(() => updateProvider.mockReset().mockResolvedValue(undefined));

test('shows guidance and header suggestions for the selected provider without replacing entered URLs', async () => {
  const user = userEvent.setup();
  render(<ExternalProvidersTab groupId={1} />);
  await user.click(screen.getByRole('button', { name: 'Add Provider' }));
  const dialog = within(await screen.findByRole('dialog'));
  const verify = dialog.getByLabelText('Verify URL');
  const headers = dialog.getByLabelText('Headers to Copy (optional)');
  const providerType = dialog.getByRole('combobox', { name: 'Provider Type' });

  expect(verify).toHaveAttribute('placeholder', 'https://auth.example.com/api/authz/forward-auth');
  expect(dialog.getByText('Use your Authelia forward-auth endpoint.')).toBeVisible();
  expect(dialog.queryByLabelText('Outpost URL (optional)')).not.toBeInTheDocument();
  await user.click(headers);
  expect(await screen.findByRole('option', { name: 'Remote-User' })).toBeVisible();
  expect(screen.queryByRole('option', { name: /Authentik/ })).not.toBeInTheDocument();
  await user.keyboard('{Escape}');
  fireEvent.change(verify, { target: { value: 'https://my-auth.example/verify?app=demo' } });

  await user.click(providerType);
  await user.click(await screen.findByRole('option', { name: 'Authentik' }));
  expect(verify).toHaveAttribute(
    'placeholder',
    'https://auth.example.com/outpost.goauthentik.io/auth/caddy',
  );
  expect(dialog.getByText('Use your Authentik outpost’s Caddy endpoint.')).toBeVisible();
  expect(dialog.getByLabelText('Outpost URL (optional)')).toBeVisible();
  await user.click(headers);
  expect(await screen.findByRole('option', { name: 'X-Authentik-Username' })).toBeVisible();
  expect(screen.queryByRole('option', { name: 'Remote-User' })).not.toBeInTheDocument();
  await user.keyboard('{Escape}');

  await user.click(providerType);
  await user.click(await screen.findByRole('option', { name: 'Custom Provider' }));
  expect(dialog.getByText('Enter the full verification endpoint URL.')).toBeVisible();
  expect(verify).toHaveAttribute('placeholder', 'https://auth.example.com/verify');
  expect(dialog.queryByLabelText('Outpost URL (optional)')).not.toBeInTheDocument();
  expect(verify).toHaveValue('https://my-auth.example/verify?app=demo');
});

test('edits the Authentik outpost destination and omits the deprecated redirect setting', async () => {
  const user = userEvent.setup();
  render(<ExternalProvidersTab groupId={1} />);
  await user.click(screen.getByRole('button', { name: 'Edit' }));
  const outpost = await screen.findByLabelText('Outpost URL (optional)');
  expect(outpost).toHaveValue('https://outpost.example:9443');
  expect(screen.queryByLabelText(/Auth Redirect URL/)).not.toBeInTheDocument();
  fireEvent.change(outpost, { target: { value: 'http://outpost:9000' } });
  await user.click(screen.getByRole('button', { name: 'Save Changes' }));
  await waitFor(() => expect(updateProvider).toHaveBeenCalledOnce());
  expect(updateProvider.mock.calls[0][0]).toEqual({
    id: 7,
    groupId: 1,
    data: {
      provider_type: 'authentik',
      name: 'NexusID',
      verify_url: provider.verify_url,
      outpost_url: 'http://outpost:9000',
      headers_to_copy: ['X-Authentik-Username'],
    },
  });
});

test('clears the outpost override to use the verification origin', async () => {
  const user = userEvent.setup();
  render(<ExternalProvidersTab groupId={1} />);
  await user.click(screen.getByRole('button', { name: 'Edit' }));
  fireEvent.change(await screen.findByLabelText('Outpost URL (optional)'), {
    target: { value: '' },
  });
  await user.click(screen.getByRole('button', { name: 'Save Changes' }));
  await waitFor(() => expect(updateProvider).toHaveBeenCalledOnce());
  expect(updateProvider.mock.calls[0][0].data.outpost_url).toBeUndefined();
});
