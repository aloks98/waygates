import { cn } from '@e412/rnui-react';
import { Link, useLocation } from '@tanstack/react-router';
import { Activity, FileQuestion, Palette, ScrollText, Users } from 'lucide-react';
import type { ReactNode } from 'react';

import { usePermissions } from '@/hooks/use-permissions';

interface SettingsNavItem {
  to: string;
  label: string;
  description: string;
  icon: ReactNode;
  requiresPermission?: string;
}

const SETTINGS_NAV_ITEMS: SettingsNavItem[] = [
  {
    to: '/settings/default-page',
    label: 'Default Page',
    description: 'Behavior for unmatched hostnames',
    icon: <FileQuestion className="size-4" />,
    requiresPermission: 'canReadSettings',
  },
  {
    to: '/settings/login-branding',
    label: 'Login Branding',
    description: 'Customize the login page',
    icon: <Palette className="size-4" />,
    requiresPermission: 'canReadSettings',
  },
  {
    to: '/settings/audit-logs',
    label: 'Audit Logs',
    description: 'Configure event logging',
    icon: <ScrollText className="size-4" />,
    requiresPermission: 'canReadAuditLogs',
  },
  {
    to: '/settings/metrics',
    label: 'Metrics',
    description: 'Expose Prometheus metrics externally',
    icon: <Activity className="size-4" />,
    requiresPermission: 'canReadSettings',
  },
  {
    to: '/settings/users',
    label: 'Users',
    description: 'Manage user accounts & access',
    icon: <Users className="size-4" />,
    requiresPermission: 'canManageUsers',
  },
];

export function SettingsNav() {
  const location = useLocation();
  const permissions = usePermissions();

  const visibleItems = SETTINGS_NAV_ITEMS.filter((item) => {
    if (!item.requiresPermission) return true;
    return permissions[item.requiresPermission as keyof typeof permissions] === true;
  });

  return (
    <nav className="flex shrink-0 gap-1 overflow-x-auto md:w-56 md:flex-col md:overflow-visible">
      {visibleItems.map((item) => {
        const isActive =
          location.pathname === item.to || location.pathname.startsWith(`${item.to}/`);
        return (
          <Link
            key={item.to}
            to={item.to}
            className={cn(
              'flex items-start gap-3 whitespace-nowrap rounded-md px-3 py-2 text-sm transition-colors md:whitespace-normal',
              'hover:bg-accent hover:text-accent-foreground',
              isActive ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground',
            )}
          >
            <span className="mt-0.5">{item.icon}</span>
            <span className="flex min-w-0 flex-col">
              <span>{item.label}</span>
              <span className="hidden text-xs text-muted-foreground md:block">
                {item.description}
              </span>
            </span>
          </Link>
        );
      })}
    </nav>
  );
}
