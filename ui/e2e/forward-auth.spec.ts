import { expect, test } from '@playwright/test';

import { mockApp } from './fixtures';

for (const viewport of [
  { width: 1280, height: 720 },
  { width: 640, height: 360 },
  { width: 320, height: 480 },
]) {
  test(`forward-auth form stays usable at ${viewport.width}x${viewport.height}`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await mockApp(page);
    await page.goto('/access/1');
    await page.getByRole('tab', { name: 'Forward Auth', exact: true }).click();
    await page.getByRole('button', { name: 'Add Provider', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Add Forward Auth Provider' });
    const submit = dialog.getByRole('button', { name: 'Add Provider', exact: true });
    const cancel = dialog.getByRole('button', { name: 'Cancel', exact: true });
    const title = dialog.getByRole('heading', { name: 'Add Forward Auth Provider' });

    await expect(dialog.getByText('Use your Authelia forward-auth endpoint.')).toBeVisible();
    await expect(dialog).toBeInViewport({ ratio: 1 });
    await dialog.getByRole('combobox', { name: 'Provider Type' }).click();
    await page.getByRole('option', { name: 'Authentik', exact: true }).click();
    await expect(dialog.getByLabel('Outpost URL (optional)')).toBeVisible();

    // Validation messages and many selected headers can make the form taller than the viewport.
    await submit.click();
    await expect(dialog.getByText('Name is required', { exact: true })).toBeVisible();
    const headers = dialog.getByLabel('Headers to Copy (optional)');
    await headers.fill(Array.from({ length: 12 }, (_, i) => `X-App-Identity-${i}`).join(','));
    await headers.press('Enter');
    await expect(dialog.locator('[data-slot="combobox-chip"]')).toHaveCount(12);
    await expect(dialog).toBeInViewport({ ratio: 1 });
    await expect(title).toBeInViewport({ ratio: 1 });
    await expect(submit).toBeInViewport({ ratio: 1 });
    await expect(cancel).toBeInViewport({ ratio: 1 });
    expect(await dialog.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);

    const titleBefore = await title.boundingBox();
    const submitBefore = await submit.boundingBox();
    const body = dialog.locator('form > div').first();
    await body.evaluate((el) => {
      el.scrollTop = 0;
    });
    await body.evaluate((el) => {
      el.scrollTop = el.scrollHeight;
    });
    expect(await body.evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
    expect(await title.boundingBox()).toEqual(titleBefore);
    expect(await submit.boundingBox()).toEqual(submitBefore);

    await cancel.click();
    await expect(dialog).not.toBeVisible();
  });
}
