import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router';
import { RootLayout } from '@/routes/__root';
import { DashboardIndex } from '@/routes/_dashboard';
import { ACLGroupDetailPage } from '@/routes/_dashboard/acl/$groupId';
import { ACLGroupsPage } from '@/routes/_dashboard/acl/index';
import { AuditLogsPage } from '@/routes/_dashboard/audit-logs';
import { L4ProxyDetailPage } from '@/routes/_dashboard/l4-proxies/$l4ProxyId';
import { L4ProxiesListPage } from '@/routes/_dashboard/l4-proxies/index';
import { L4ProxyCreatePage } from '@/routes/_dashboard/l4-proxies/new';
import { ProxyDetailPage } from '@/routes/_dashboard/proxies/$proxyId';
import { ProxiesListPage } from '@/routes/_dashboard/proxies/index';
import { ProxyCreatePage } from '@/routes/_dashboard/proxies/new';
import { DashboardLayout } from '@/routes/_dashboard/route';
import { SettingsPage } from '@/routes/_dashboard/settings';
import { ACLForbiddenPage } from '@/routes/auth/acl-forbidden';
import { ACLLoginPage } from '@/routes/auth/acl-login';
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

// ACL Login route - public route for proxy authentication
// This is separate from admin login and is shown when accessing protected proxies
const aclLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/login',
  component: ACLLoginPage,
});

// ACL Forbidden route - shown when user is authenticated but not authorized
const aclForbiddenRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/forbidden',
  component: ACLForbiddenPage,
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
  component: ProxiesListPage,
});

const proxyCreateRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/new',
  component: ProxyCreatePage,
});

const proxyDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/$proxyId',
  component: ProxyDetailPage,
});

// L4 Proxies routes
const l4ProxiesRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/l4-proxies',
  component: L4ProxiesListPage,
});

const l4ProxyCreateRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/l4-proxies/new',
  component: L4ProxyCreatePage,
});

const l4ProxyDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/l4-proxies/$l4ProxyId',
  component: L4ProxyDetailPage,
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

const aclRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/acl',
  component: ACLGroupsPage,
});

const aclGroupDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/acl/$groupId',
  component: ACLGroupDetailPage,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  signupRoute,
  aclLoginRoute,
  aclForbiddenRoute,
  dashboardRoute.addChildren([
    dashboardIndexRoute,
    proxiesRoute,
    proxyCreateRoute,
    proxyDetailRoute,
    l4ProxiesRoute,
    l4ProxyCreateRoute,
    l4ProxyDetailRoute,
    settingsRoute,
    auditLogsRoute,
    aclRoute,
    aclGroupDetailRoute,
  ]),
]);

export const router = createRouter({ routeTree });
