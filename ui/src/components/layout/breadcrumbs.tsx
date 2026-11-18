import { Link, useLocation } from '@tanstack/react-router';
import { Fragment } from 'react';

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';

const LABELS: Record<string, string> = {
  dashboard: 'Overview',
  proxies: 'Proxies',
  'tcp-udp': 'TCP/UDP',
  new: 'Create',
  'caddy-logs': 'Logs',
  'caddy-config': 'Generated config',
  access: 'Access',
  activity: 'Activity',
  settings: 'Settings',
};

function labelFor(segment: string): string {
  if (LABELS[segment]) return LABELS[segment];
  if (/^\d+$/.test(segment)) return 'Details';
  return segment.charAt(0).toUpperCase() + segment.slice(1);
}

export function buildCrumbs(pathname: string): { label: string; href: string }[] {
  const segments = pathname.split('/').filter(Boolean);
  const crumbs: { label: string; href: string }[] = [];
  let href = '';
  for (const segment of segments) {
    href += `/${segment}`;
    crumbs.push({ label: labelFor(segment), href });
  }
  return crumbs;
}

export function Breadcrumbs() {
  const { pathname } = useLocation();
  const crumbs = buildCrumbs(pathname);

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {crumbs.length === 0 && (
          <BreadcrumbItem>
            <BreadcrumbPage>Overview</BreadcrumbPage>
          </BreadcrumbItem>
        )}
        {crumbs.map((crumb, i) => {
          const isLast = i === crumbs.length - 1;
          return (
            <Fragment key={crumb.href}>
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink render={<Link to={crumb.href} />}>{crumb.label}</BreadcrumbLink>
                )}
              </BreadcrumbItem>
              {!isLast && <BreadcrumbSeparator />}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
