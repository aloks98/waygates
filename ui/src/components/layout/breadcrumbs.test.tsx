import { expect, test } from 'vitest';

import { buildCrumbs } from './breadcrumbs';

test('dashboard root → single Dashboard crumb', () => {
  expect(buildCrumbs('/dashboard')).toEqual([{ label: 'Dashboard', href: '/dashboard' }]);
});

test('nested proxies tcp-udp path', () => {
  expect(buildCrumbs('/dashboard/proxies/tcp-udp')).toEqual([
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Proxies', href: '/dashboard/proxies' },
    { label: 'TCP/UDP', href: '/dashboard/proxies/tcp-udp' },
  ]);
});

test('numeric id segment becomes Details', () => {
  expect(buildCrumbs('/dashboard/access/42')).toEqual([
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Access', href: '/dashboard/access' },
    { label: 'Details', href: '/dashboard/access/42' },
  ]);
});

test('new segment becomes New', () => {
  expect(buildCrumbs('/dashboard/proxies/new')).toEqual([
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Proxies', href: '/dashboard/proxies' },
    { label: 'New', href: '/dashboard/proxies/new' },
  ]);
});
