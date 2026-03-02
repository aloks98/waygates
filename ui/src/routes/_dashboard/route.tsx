import { useQuery } from '@tanstack/react-query';
import { Outlet } from '@tanstack/react-router';
import { useEffect } from 'react';

import { AppSidebar } from '@/components/layout';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import type { ApiResponse } from '@/types/api';
import type { User } from '@/types/auth';

export function DashboardLayout() {
  const { setUser, user } = useAuthStore();

  const { data, isLoading } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => api.get('auth/me').json<ApiResponse<User>>(),
    enabled: !user, // Only fetch if user is not already in store
    staleTime: 5 * 60 * 1000, // Consider data fresh for 5 minutes
  });

  // Sync fetched user to auth store
  useEffect(() => {
    if (data?.success && data.data && !user) {
      setUser(data.data);
    }
  }, [data, setUser, user]);

  // Show loading state while fetching user data
  if (!user && isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  return (
    <div className="flex h-screen">
      <AppSidebar>
        <Outlet />
      </AppSidebar>
    </div>
  );
}
