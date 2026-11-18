import { Outlet, useLocation } from '@tanstack/react-router';

import { ThemeToggle } from '@/components/layout/theme-toggle';

export function RootLayout() {
  const { pathname } = useLocation();
  const publicPage =
    pathname === '/login' || pathname === '/signup' || pathname.startsWith('/auth/');
  return (
    <div className="min-h-screen bg-background text-foreground">
      {publicPage && (
        <div className="absolute right-5 top-5 z-10">
          <ThemeToggle />
        </div>
      )}
      <Outlet />
    </div>
  );
}
