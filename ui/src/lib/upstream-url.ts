import type { Upstream } from '@/types/proxy';

export function formatUpstream(upstream?: Upstream): string {
  if (!upstream?.host) return '';
  const host =
    upstream.host.includes(':') && !upstream.host.startsWith('[')
      ? `[${upstream.host}]`
      : upstream.host;
  return `${upstream.scheme}://${host}:${upstream.port}`;
}

export function parseUpstream(value: string): Upstream | null {
  try {
    const url = new URL(value);
    if (
      !['http:', 'https:'].includes(url.protocol) ||
      url.username ||
      url.password ||
      url.search ||
      url.hash ||
      (url.pathname !== '/' && url.pathname !== '')
    )
      return null;
    return {
      host: url.hostname.replace(/^\[|\]$/g, ''),
      port: Number(url.port || (url.protocol === 'https:' ? 443 : 80)),
      scheme: url.protocol === 'https:' ? 'https' : 'http',
    };
  } catch {
    return null;
  }
}
