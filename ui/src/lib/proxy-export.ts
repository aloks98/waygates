import type { ProxyConfig } from '@/types/proxy';

export interface ProxyExport {
  type: ProxyConfig['type'];
  name: string;
  hostname: string;
  description?: string;
  ssl_enabled: boolean;
  is_active?: boolean;
  upstreams?: ProxyConfig['upstreams'];
  load_balancing?: ProxyConfig['load_balancing'];
  block_exploits?: boolean;
  tls_insecure_skip_verify?: boolean;
  custom_headers?: ProxyConfig['custom_headers'];
  redirect?: ProxyConfig['redirect'];
  static?: ProxyConfig['static'];
}

export function downloadJson(filename: string, data: unknown): void {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function summarizeBulkResults(results: PromiseSettledResult<unknown>[]): {
  succeeded: number;
  failed: number;
} {
  let succeeded = 0;
  let failed = 0;
  for (const r of results) {
    if (r.status === 'fulfilled') succeeded++;
    else failed++;
  }
  return { succeeded, failed };
}
