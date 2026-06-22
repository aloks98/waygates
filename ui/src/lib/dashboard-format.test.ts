import { expect, test } from 'vitest';

import {
  buildCompositionData,
  formatUptime,
  getActionColor,
  getActivityLink,
} from './dashboard-format';

test('buildCompositionData maps types/protocols and drops zeros', () => {
  expect(
    buildCompositionData(
      { total: 9, active: 7, inactive: 2, by_type: { reverse_proxy: 6, redirect: 0, static: 1 } },
      {
        total_proxies: 3,
        active_proxies: 3,
        tcp_proxies: 2,
        udp_proxies: 1,
        total_routes: 4,
        total_upstreams: 5,
      },
    ),
  ).toEqual([
    { name: 'Reverse', value: 6 },
    { name: 'Static', value: 1 },
    { name: 'TCP', value: 2 },
    { name: 'UDP', value: 1 },
  ]);
});

test('buildCompositionData with no data is empty', () => {
  expect(buildCompositionData(undefined, undefined)).toEqual([]);
});

test('formatUptime parses a Go duration', () => {
  expect(formatUptime('49h5m10.048s')).toMatch(/2 days/);
});

test('formatUptime falls back for sub-minute uptimes', () => {
  expect(formatUptime('45.2s')).toBe('less than a minute');
});

test('getActionColor flags destructive and success', () => {
  expect(getActionColor('proxy.delete')).toContain('destructive');
  expect(getActionColor('proxy.create')).toContain('green');
});

test('getActivityLink builds resource links and skips deletes', () => {
  expect(
    getActivityLink({ action: 'proxy.update', resource_type: 'proxy', resource_id: 5 } as any),
  ).toBe('/proxies/5');
  expect(
    getActivityLink({ action: 'acl_group.update', resource_type: 'acl', resource_id: 2 } as any),
  ).toBe('/access/2');
  expect(
    getActivityLink({ action: 'proxy.delete', resource_type: 'proxy', resource_id: 5 } as any),
  ).toBeNull();
});
