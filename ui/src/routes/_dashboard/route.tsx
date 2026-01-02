import { Outlet } from '@tanstack/react-router';
import { useEffect } from 'react';
import { AppSidebar } from '../../components/sidebar';
import { api } from '../../lib/api';
import { useAuthStore } from '../../stores/auth';
import type { ApiResponse } from '../../types/api';
import type { User } from '../../types/auth';

export function DashboardLayout() {
  const { setUser, user } = useAuthStore();

  useEffect(() => {
    if (!user) {
      api
        .get('auth/me')
        .json<ApiResponse<User>>()
        .then((response) => {
          if (response.success && response.data) {
            setUser(response.data);
          }
        })
        .catch(() => {
          // Handle error silently, auth middleware will handle redirect
        });
    }
  }, [user, setUser]);

  return (
    <div className="flex h-screen">
      <AppSidebar>
        <Outlet />
      </AppSidebar>
    </div>
  );
}
