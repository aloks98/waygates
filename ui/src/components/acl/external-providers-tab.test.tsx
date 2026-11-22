import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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
