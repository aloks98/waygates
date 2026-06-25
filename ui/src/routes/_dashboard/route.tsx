import { useQuery } from '@tanstack/react-query';
import { Outlet, useLocation } from '@tanstack/react-router';
import { useEffect } from 'react';

import { AppSidebar } from '@/components/layout';
import { ChangePasswordDialog } from '@/components/layout/change-password-dialog';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import type { ApiResponse } from '@/types/api';
import type { User } from '@/types/auth';

export function DashboardLayout() {
  const { setUser, user } = useAuthStore();
  const location = useLocation();

  const { data, isLoading } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => api.get('auth/me').json<ApiResponse<User>>(),
    // Always fetch on mount so must_change_password is current (even after reload)
    staleTime: 5 * 60 * 1000,
  });

  // Sync fetched user to auth store
  useEffect(() => {
    if (data?.success && data.data) {
      setUser(data.data);
    }
  }, [data, setUser]);

  // Show loading state while fetching user data
  if (!user && isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  // Determine must-change state from the freshly fetched /me response so it
  // survives page reloads (not just the login moment).
  const mustChangePassword =
    data?.data?.must_change_password ?? user?.must_change_password ?? false;

  return (
    <div className="flex h-screen">
      <AppSidebar>
        <div key={location.pathname} className="page-enter">
          <Outlet />
        </div>
      </AppSidebar>

      {/* Forced password change gate — non-dismissable until the server clears the flag */}
      <ChangePasswordDialog
        open={mustChangePassword}
        onOpenChange={() => {
          /* no-op: forced mode ignores close requests */
        }}
        forced
      />
    </div>
  );
}
