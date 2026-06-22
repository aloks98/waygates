import { expect, test } from 'vitest';

import { buildCrumbs } from './breadcrumbs';

test('root path → single Dashboard crumb', () => {
  expect(buildCrumbs('/')).toEqual([]);
});

test('nested proxies tcp-udp path', () => {
  expect(buildCrumbs('/proxies/tcp-udp')).toEqual([
    { label: 'Proxies', href: '/proxies' },
    { label: 'TCP/UDP', href: '/proxies/tcp-udp' },
  ]);
});

test('numeric id segment becomes Details', () => {
  expect(buildCrumbs('/access/42')).toEqual([
    { label: 'Access', href: '/access' },
    { label: 'Details', href: '/access/42' },
  ]);
});

test('new segment becomes New', () => {
  expect(buildCrumbs('/proxies/new')).toEqual([
    { label: 'Proxies', href: '/proxies' },
    { label: 'New', href: '/proxies/new' },
  ]);
});
