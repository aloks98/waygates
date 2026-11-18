import { test, expect } from '@playwright/test';

import { mockApp } from './fixtures';

const routes = [
  '/',
  '/proxies',
  '/proxies/new',
  '/proxies/1',
  '/proxies/1/edit',
  '/proxies/2/edit',
  '/proxies/3/edit',
  '/proxies/tcp-udp',
  '/proxies/tcp-udp/new',
  '/proxies/tcp-udp/1',
  '/proxies/tcp-udp/1/edit',
  '/activity',
  '/caddy-logs?source=access',
  '/caddy-logs?source=runtime',
  '/access',
  '/access/1',
  '/settings/default-page',
  '/settings/login-branding',
  '/settings/audit-logs',
  '/settings/metrics',
  '/settings/users',
  '/settings/sso',
  '/settings/system',
  '/caddy-config',
];
for (const theme of ['light', 'dark'])
  for (const path of routes)
    test(`${theme} dashboard ${path}`, async ({ page }) => {
      await mockApp(page, { theme });
      const errors: string[] = [];
      page.on('pageerror', (e) => errors.push(e.message));
      await page.goto(path);
      await expect(page.locator('main')).toBeVisible();
      await expect(page.locator('main h1')).toBeVisible();
      await page.waitForTimeout(120);
      expect(errors, `${path} browser errors`).toEqual([]);
      if (
        [
          '/',
          '/proxies',
          '/proxies/new',
          '/settings/sso',
          '/activity',
          '/access/1',
          '/proxies/tcp-udp/new',
          '/caddy-logs?source=runtime',
        ].includes(path)
      )
        await page.screenshot({
          path: `/tmp/waygates-${theme}-${path.replaceAll('/', '-') || 'overview'}.png`,
          fullPage: true,
        });
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
        `${path} horizontal overflow`,
      ).toBe(true);
      await page.setViewportSize({ width: 390, height: 844 });
      await expect
        .poll(
          () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
          { message: `${path} mobile overflow` },
        )
        .toBe(true);
    });

test('proxy advanced settings apply and cancel without submitting', async ({ page }) => {
  await mockApp(page);
  const writes: string[] = [];
  page.on('request', (req) => {
    if (req.method() === 'POST' && req.url().endsWith('/api/proxies'))
      writes.push(req.postData() ?? '');
  });
  await page.goto('/proxies/new');
  await page.getByRole('textbox', { name: 'Name', exact: true }).fill('Analytics');
  await page.getByRole('textbox', { name: 'Domain', exact: true }).fill('analytics.waygates.io');
  await page.getByRole('textbox', { name: 'Destination URL' }).fill('http://analytics:3000');
  await page.getByRole('button', { name: 'Configure advanced settings' }).click();
  let sheet = page.getByRole('dialog', { name: 'Advanced settings' });
  await sheet.getByRole('textbox', { name: 'Description', exact: true }).fill('Discard this');
  await sheet.getByRole('button', { name: 'Cancel', exact: true }).click();
  await page.getByRole('button', { name: 'Configure advanced settings' }).click();
  sheet = page.getByRole('dialog', { name: 'Advanced settings' });
  await expect(sheet.getByRole('textbox', { name: 'Description', exact: true })).toHaveValue('');
  await sheet.getByRole('textbox', { name: 'Description', exact: true }).fill('Team analytics');
  await sheet.getByRole('button', { name: 'Apply', exact: true }).click();
  expect(writes).toEqual([]);
  await page.getByRole('button', { name: 'Create proxy', exact: true }).click();
  await expect.poll(() => writes.length).toBe(1);
  expect(JSON.parse(writes[0])).toMatchObject({
    description: 'Team analytics',
    upstreams: [{ host: 'analytics', port: 3000, scheme: 'http' }],
  });
});

for (const theme of ['light', 'dark'])
  test(`auth screens and mobile layout in ${theme}`, async ({ page }) => {
    await mockApp(page, { authenticated: false, theme });
    const errors: string[] = [];
    page.on('pageerror', (e) => errors.push(e.message));
    for (const path of [
      '/login',
      '/signup',
      '/auth/login?host=analytics.waygates.io',
      '/auth/forbidden?url=https%3A%2F%2Fanalytics.waygates.io',
    ]) {
      await page.goto(path);
      await expect(page.locator('body')).not.toContainText('Something went wrong');
      await expect(page.locator('[data-slot="sidebar"]')).toHaveCount(0);
      await expect(page.locator('h1,h2,h3').first()).toBeVisible();
      expect(errors).toEqual([]);
      await page.screenshot({
        path: `/tmp/waygates-${theme}-auth-${path.split('?')[0].replaceAll('/', '-')}.png`,
        fullPage: true,
      });
      await page.setViewportSize({ width: 390, height: 844 });
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      ).toBe(true);
      await page.setViewportSize({ width: 1440, height: 1040 });
    }
  });

test('destination URLs and generated config use the current draft without saving', async ({
  page,
}) => {
  await mockApp(page);
  let preview: unknown;
  const saves: string[] = [];
  page.on('request', (req) => {
    if (req.url().endsWith('/api/caddy-config/preview')) preview = req.postDataJSON();
    if (req.method() === 'POST' && req.url().endsWith('/api/proxies'))
      saves.push(req.postData() ?? '');
  });
  await page.goto('/proxies/new');
  await page.getByRole('textbox', { name: 'Name', exact: true }).fill('Analytics');
  await page.getByRole('textbox', { name: 'Domain', exact: true }).fill('analytics.waygates.io');
  await page
    .getByRole('textbox', { name: 'Destination URL', exact: true })
    .pressSequentially('http://analytics:3000');
  await page.getByRole('button', { name: 'Add destination' }).click();
  await page.getByRole('textbox', { name: 'Destination URL 2' }).fill('https://[2001:db8::1]:8443');
  await page.getByRole('button', { name: 'View generated config' }).click();
  const sheet = page.getByRole('dialog', { name: 'Generated config' });
  await expect(sheet.locator('pre')).toContainText('reverse_proxy');
  expect(preview).toMatchObject({
    proxy: {
      ssl_forced: true,
      upstreams: [
        { host: 'analytics', port: 3000 },
        { host: '2001:db8::1', port: 8443 },
      ],
    },
  });
  expect(saves).toEqual([]);
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: 'Remove destination 2' }).click();
  await page.getByRole('button', { name: 'Create proxy', exact: true }).click();
  await expect.poll(() => saves.length).toBe(1);
  expect(JSON.parse(saves[0]).upstreams).toHaveLength(1);
});

for (const protocol of ['tcp', 'udp']) {
  for (const status of [404, 405]) {
    test(`saved ${protocol} config works when draft endpoint returns ${status}`, async ({
      page,
    }) => {
      await mockApp(page);
      const savedProxy = {
        id: 1,
        name: 'Game server',
        protocol,
        listen_port: 27015,
        is_active: true,
        routes: [
          {
            id: 1,
            priority: 0,
            matcher_type: 'any',
            upstreams: [{ host: 'game', port: 27015 }],
            tls_terminate: false,
            tls_passthrough: false,
            load_balancing_policy: 'round_robin',
          },
        ],
      };
      const serverName = `Game server_${protocol}_27015`;
      const server = {
        listen: [`${protocol}/:27015`],
        routes: [{ handle: [{ handler: 'proxy', upstreams: [{ dial: ['game:27015'] }] }] }],
      };
      let configReads = 0;
      const saves: string[] = [];
      page.on('request', (request) => {
        if (
          ['POST', 'PUT', 'DELETE'].includes(request.method()) &&
          new URL(request.url()).pathname.startsWith('/api/l4-proxies')
        ) {
          saves.push(request.url());
        }
      });
      await page.route('**/api/l4-proxies/1', (route) =>
        route.fulfill({ json: { success: true, data: savedProxy } }),
      );
      await page.route('**/api/caddy-config/preview', (route) => {
        expect(route.request().postDataJSON()).toEqual({ l4_proxy: savedProxy });
        return route.fulfill({ status, json: { success: false } });
      });
      await page.route('**/api/caddy-config', (route) => {
        configReads++;
        return route.fulfill({
          json: {
            success: true,
            data: {
              apps: {
                http: { servers: { unrelated: {} } },
                layer4: { servers: { [serverName]: server, unrelated: { listen: ['tcp/:1234'] } } },
              },
            },
          },
        });
      });
      await page.goto('/proxies/tcp-udp/1');
      await page.getByRole('button', { name: 'View generated config' }).click();
      const sheet = page.getByRole('dialog', { name: 'Generated config' });
      await expect(sheet.locator('pre')).toContainText('game:27015');
      expect(JSON.parse(await sheet.locator('pre').innerText())).toEqual({
        apps: { layer4: { servers: { [serverName]: server } } },
      });
      await expect(sheet.getByRole('button', { name: 'Copy', exact: true })).toBeEnabled();
      const downloadEvent = page.waitForEvent('download');
      await sheet.getByRole('button', { name: 'Download', exact: true }).click();
      expect((await downloadEvent).suggestedFilename()).toBe('caddy-config.json');
      expect(configReads).toBe(1);
      expect(saves).toEqual([]);
    });
  }
}

test('unsaved TCP config never falls back to saved configuration', async ({ page }) => {
  await mockApp(page);
  let configReads = 0;
  await page.route('**/api/caddy-config/preview', (route) =>
    route.fulfill({ status: 404, json: { success: false } }),
  );
  await page.route('**/api/caddy-config', (route) => {
    configReads++;
    return route.fulfill({ json: { success: true, data: {} } });
  });
  await page.goto('/proxies/tcp-udp/1/edit');
  await page.getByRole('button', { name: 'View generated config' }).click();
  const sheet = page.getByRole('dialog', { name: 'Generated config' });
  await expect(sheet.getByRole('alert')).toContainText('Update the Waygates backend');
  await expect(sheet.getByRole('button', { name: 'Copy', exact: true })).toBeDisabled();
  expect(configReads).toBe(0);
});

for (const [status, message] of [
  [400, 'listener and routing settings'],
  [403, 'permission'],
] as const) {
  test(`TCP config reports ${status} without falling back`, async ({ page }) => {
    await mockApp(page);
    let configReads = 0;
    await page.route('**/api/caddy-config/preview', (route) =>
      route.fulfill({ status, json: { success: false } }),
    );
    await page.route('**/api/caddy-config', (route) => {
      configReads++;
      return route.fulfill({ json: { success: true, data: {} } });
    });
    await page.goto('/proxies/tcp-udp/1');
    await page.getByRole('button', { name: 'View generated config' }).click();
    await expect(page.getByRole('dialog').getByRole('alert')).toContainText(message);
    expect(configReads).toBe(0);
  });
}

test('editing a path rule persists its path and priority', async ({ page }) => {
  await mockApp(page);
  await page.route('**/api/proxies/1/acl', (route) =>
    route.fulfill({
      json: {
        success: true,
        data: [
          {
            id: 11,
            proxy_id: 1,
            acl_group_id: 1,
            path_pattern: '/*',
            priority: 100,
            enabled: true,
          },
        ],
      },
    }),
  );
  await page.goto('/proxies/1/edit');
  await page.getByRole('button', { name: 'Path rules' }).click();
  const sheet = page.getByRole('dialog', { name: 'Access rules' });
  await sheet.getByRole('textbox', { name: 'Path', exact: true }).fill('/admin/*');
  await sheet.getByRole('spinbutton', { name: 'Priority' }).fill('200');
  await sheet.getByRole('button', { name: 'Apply', exact: true }).click();
  const request = page.waitForRequest(
    (req) => req.method() === 'PUT' && req.url().endsWith('/api/proxies/1/acl/11'),
  );
  await page.getByRole('button', { name: 'Save changes', exact: true }).click();
  expect((await request).postDataJSON()).toEqual({
    path_pattern: '/admin/*',
    priority: 200,
    enabled: true,
  });
});

test('access tabs and email pills remain editable', async ({ page }) => {
  await mockApp(page);
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await page.goto('/access/1');
  for (const name of [
    'IP Rules',
    'Basic Auth',
    'OAuth / SSO',
    'Forward Auth',
    'Usage',
    'Waygates Account',
  ]) {
    await page.getByRole('tab', { name, exact: true }).click();
    await expect(page.getByRole('tabpanel')).toBeVisible();
    expect(errors).toEqual([]);
  }
  const emails = page.getByRole('combobox', { name: 'Allowed Email Patterns' });
  await emails.fill('alice@example.com');
  await emails.press('Enter');
  await expect(
    page.locator('[data-slot="combobox-chip"]').filter({ hasText: 'alice@example.com' }),
  ).toBeVisible();
  const request = page.waitForRequest(
    (req) => req.method() === 'PUT' && req.url().endsWith('/waygates-auth'),
  );
  await page.getByRole('button', { name: /Save/ }).click();
  expect((await request).postDataJSON().allowed_email_patterns).toContain('alice@example.com');
});

test('TCP and UDP creation needs no advanced steps', async ({ page }) => {
  await mockApp(page);
  await page.goto('/proxies/tcp-udp/new');
  await page.getByRole('textbox', { name: 'Name', exact: true }).fill('Database');
  await page.getByRole('spinbutton', { name: 'Listen port', exact: true }).fill('5432');
  await page.getByRole('textbox', { name: 'Destination host', exact: true }).fill('postgres');
  await page.getByRole('spinbutton', { name: 'Destination port', exact: true }).fill('5432');
  const request = page.waitForRequest(
    (req) => req.method() === 'POST' && req.url().endsWith('/api/l4-proxies'),
  );
  await page.getByRole('button', { name: 'Create proxy', exact: true }).click();
  expect((await request).postDataJSON()).toMatchObject({
    name: 'Database',
    protocol: 'tcp',
    listen_port: 5432,
    routes: [{ upstreams: [{ host: 'postgres', port: 5432 }] }],
  });
});

test('mobile navigation closes after changing pages', async ({ page }) => {
  await mockApp(page);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/');
  await page.getByRole('button', { name: 'Toggle Sidebar' }).click();
  const sidebar = page.getByRole('dialog');
  await expect(sidebar).toBeVisible();
  await sidebar.getByRole('link', { name: 'Access', exact: true }).click();
  await expect(page).toHaveURL(/\/access$/);
  await expect(sidebar).not.toBeVisible();
});

test('proxy filters and row actions work independently of row navigation', async ({ page }) => {
  await mockApp(page);
  await page.goto('/proxies');
  await page.getByRole('button', { name: 'Add filter', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Type', exact: true }).click();
  const request = page.waitForRequest(
    (req) =>
      new URL(req.url()).pathname === '/api/proxies' &&
      new URL(req.url()).searchParams.get('type') === 'reverse_proxy',
  );
  await page.getByRole('menuitemradio', { name: 'Reverse proxy', exact: true }).click();
  await request;
  await page.getByRole('button', { name: 'Open actions menu' }).first().click();
  await expect(page).toHaveURL(/\/proxies$/);
  await page.getByRole('menuitem', { name: 'Edit', exact: true }).click();
  await expect(page).toHaveURL(/\/proxies\/1\/edit$/);
});

test('routing panel applies SNI pills without submitting the proxy', async ({ page }) => {
  await mockApp(page);
  await page.goto('/proxies/tcp-udp/1/edit');
  await page.getByRole('button', { name: 'Configure routing & TLS', exact: true }).click();
  const sheet = page.getByRole('dialog', { name: 'Routing & TLS' });
  await sheet.getByRole('combobox', { name: 'Connection matcher' }).click();
  await page.getByRole('option', { name: 'TLS/SNI', exact: true }).click();
  await sheet.getByRole('combobox', { name: 'SNI hostnames' }).fill('db.example.com');
  await sheet.getByRole('combobox', { name: 'SNI hostnames' }).press('Enter');
  await sheet.getByRole('button', { name: 'Apply', exact: true }).click();
  await expect(sheet).not.toBeVisible();
  const request = page.waitForRequest(
    (req) => req.method() === 'PUT' && req.url().endsWith('/api/l4-proxies/1'),
  );
  await page.getByRole('button', { name: 'Save changes', exact: true }).click();
  expect((await request).postDataJSON().routes[0]).toMatchObject({
    matcher_type: 'tls',
    sni_hostnames: ['db.example.com'],
    upstreams: [{ host: 'database', port: 5432 }],
  });
});
