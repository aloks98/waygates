import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, test, vi } from 'vitest';

const { useNotFoundSettings, update } = vi.hoisted(() => ({
  useNotFoundSettings: vi.fn(),
  update: vi.fn(),
}));

vi.mock('@/hooks/use-settings', () => ({ useNotFoundSettings }));
vi.mock('@/hooks/use-permissions', () => ({
  usePermissions: () => ({ canWriteSettings: true }),
}));

import { CatchallSettings } from './catchall-settings';

beforeEach(() => {
  update.mockReset().mockResolvedValue(undefined);
  useNotFoundSettings.mockReset().mockReturnValue({
    settings: { mode: 'redirect', redirect_url: 'https://example.com/' },
    isLoading: false,
    isUpdating: false,
    update,
  });
});

test('saves the generic response option and clears the previous redirect', async () => {
  const user = userEvent.setup();
  render(<CatchallSettings />);

  await user.click(screen.getByRole('radio', { name: 'Generic 404 Response' }));
  expect(screen.queryByPlaceholderText('https://example.com')).not.toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: 'Save Changes' }));

  await waitFor(() => expect(update).toHaveBeenCalledWith({ mode: 'generic', redirect_url: '' }));
});

test('shows a saved generic response selection', () => {
  useNotFoundSettings.mockReturnValue({
    settings: { mode: 'generic', redirect_url: '' },
    isLoading: false,
    isUpdating: false,
    update,
  });
  render(<CatchallSettings />);

  expect(screen.getByRole('radio', { name: 'Generic 404 Response' })).toBeChecked();
  expect(screen.getByRole('radio', { name: 'Branded 404 Page' })).not.toBeChecked();
});
