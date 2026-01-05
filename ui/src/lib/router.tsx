import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router';
import { RootLayout } from '@/routes/__root';
import { DashboardIndex } from '@/routes/_dashboard';
import { AuditLogsPage } from '@/routes/_dashboard/audit-logs';
import { ProxiesPage } from '@/routes/_dashboard/proxies';
import { DashboardLayout } from '@/routes/_dashboard/route';
import { SettingsPage } from '@/routes/_dashboard/settings';
import { LoginPage } from '@/routes/login';
import { SignupPage } from '@/routes/signup';
import { useAuthStore } from '@/stores/auth';

const rootRoute = createRootRoute({
  component: RootLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    const { isAuthenticated } = useAuthStore.getState();
    if (isAuthenticated) {
      throw redirect({ to: '/dashboard' });
    }
    throw redirect({ to: '/login' });
  },
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: () => {
    const { isAuthenticated } = useAuthStore.getState();
    if (isAuthenticated) {
      throw redirect({ to: '/dashboard' });
    }
  },
  component: LoginPage,
});

const signupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/signup',
  beforeLoad: () => {
    const { isAuthenticated } = useAuthStore.getState();
    if (isAuthenticated) {
      throw redirect({ to: '/dashboard' });
    }
  },
  component: SignupPage,
});

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dashboard',
  beforeLoad: () => {
    const { isAuthenticated } = useAuthStore.getState();
    if (!isAuthenticated) {
      throw redirect({ to: '/login' });
    }
  },
  component: DashboardLayout,
});

const dashboardIndexRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/',
  component: DashboardIndex,
});

const proxiesRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies',
  component: ProxiesPage,
});

const settingsRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/settings',
  component: SettingsPage,
});

const auditLogsRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/audit-logs',
  component: AuditLogsPage,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  signupRoute,
  dashboardRoute.addChildren([dashboardIndexRoute, proxiesRoute, settingsRoute, auditLogsRoute]),
]);

export const router = createRouter({ routeTree });
