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

// L4 Proxies routes
const l4ProxiesRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/tcp-udp',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/l4-proxies/index'),
    'L4ProxiesListPage',
  ),
});

const l4ProxyCreateRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/tcp-udp/new',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/l4-proxies/new'),
    'L4ProxyCreatePage',
  ),
});

const l4ProxyDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/proxies/tcp-udp/$l4ProxyId',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/l4-proxies/$l4ProxyId'),
    'L4ProxyDetailPage',
  ),
});

const settingsRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/settings',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/settings/layout'),
    'SettingsLayout',
  ),
});

const settingsIndexRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/dashboard/settings/default-page' });
  },
});

const settingsDefaultPageRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: '/default-page',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/settings/default-page'),
    'SettingsDefaultPage',
  ),
});

const settingsBrandingRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: '/login-branding',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/settings/login-branding'),
    'SettingsLoginBrandingPage',
  ),
});

const settingsAuditRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: '/audit-logs',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/settings/audit-logs'),
    'SettingsAuditLogsPage',
  ),
});

const auditLogsRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/activity',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/activity'), 'ActivityPage'),
});

const aclRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/access',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/acl/index'), 'ACLGroupsPage'),
});

const aclGroupDetailRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/access/$groupId',
  component: lazyRouteComponent(
    () => import('@/routes/_dashboard/acl/$groupId'),
    'ACLGroupDetailPage',
  ),
});

const l4RedirectRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/l4-proxies',
  beforeLoad: () => {
    throw redirect({ to: '/dashboard/proxies/tcp-udp' });
  },
});

const aclRedirectRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/acl',
  beforeLoad: () => {
    throw redirect({ to: '/dashboard/access' });
  },
});

const auditRedirectRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/audit-logs',
  beforeLoad: () => {
    throw redirect({ to: '/dashboard/activity' });
  },
});

const themePreviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/theme-preview',
  component: lazyRouteComponent(() => import('@/routes/theme-preview'), 'ThemePreviewPage'),
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  signupRoute,
  themePreviewRoute,
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
    settingsRoute.addChildren([
      settingsIndexRoute,
      settingsDefaultPageRoute,
      settingsBrandingRoute,
      settingsAuditRoute,
    ]),
    auditLogsRoute,
    aclRoute,
    aclGroupDetailRoute,
    l4RedirectRoute,
    aclRedirectRoute,
    auditRedirectRoute,
  ]),
]);

export const router = createRouter({ routeTree });
