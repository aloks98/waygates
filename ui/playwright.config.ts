import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  workers: 3,
  timeout: 30_000,
  use: {
    baseURL: 'http://127.0.0.1:8009',
    viewport: { width: 1440, height: 1040 },
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'node node_modules/vite/bin/vite.js --host 127.0.0.1 --port 8009',
    url: 'http://127.0.0.1:8009',
    reuseExistingServer: true,
  },
});
