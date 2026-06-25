import { useRouter } from '@tanstack/react-router';
import { useCallback } from 'react';

import { api } from '@/lib/api';
import { queryClient } from '@/lib/query-client';
import { useAuthStore } from '@/stores/auth';

export function useLogout(): () => Promise<void> {
  const router = useRouter();
  const logout = useAuthStore((s) => s.logout);
  return useCallback(async () => {
    try {
      await api.post('auth/logout');
    } catch {
      // Ignore errors; log out locally anyway.
    } finally {
      queryClient.clear();
    }
    logout();
    router.navigate({ to: '/login' });
  }, [logout, router]);
}
