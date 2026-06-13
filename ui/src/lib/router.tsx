import {
  createRootRoute,
  createRoute,
  createRouter,
  lazyRouteComponent,
  redirect,
} from '@tanstack/react-router';

import { RootLayout } from '@/routes/__root';
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
  component: lazyRouteComponent(() => import('@/routes/auth/acl-forbidden'), 'ACLForbiddenPage'),
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
  component: lazyRouteComponent(() => import('@/routes/_dashboard/route'), 'DashboardLayout'),
});

const dashboardIndexRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/',
  component: lazyRouteComponent(() => import('@/routes/_dashboard'), 'DashboardIndex'),
});

const proxiesRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/proxies/index'),
    'ProxiesListPage',
  ),
});

const proxyCreateRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/new',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/proxies/new'), 'ProxyCreatePage'),
});

const proxyDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/$proxyId',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/proxies/$proxyId'),
    'ProxyDetailPage',
  ),
});

const settingsRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/settings',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/settings'), 'SettingsPage'),
});

const auditLogsRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/audit-logs',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/audit-logs'), 'AuditLogsPage'),
});

const aclRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/acl',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/acl/index'), 'ACLGroupsPage'),
});

const aclGroupDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/acl/$groupId',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/acl/$groupId'),
    'ACLGroupDetailPage',
  ),
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
    settingsRoute,
    auditLogsRoute,
    aclRoute,
    aclGroupDetailRoute,
  ]),
]);

export const router = createRouter({ routeTree });
