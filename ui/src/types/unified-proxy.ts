import type { L4Protocol, L4Proxy } from './l4-proxy';
import type { ProxyConfig, ProxyType } from './proxy';

/**
 * Layer type to distinguish between HTTP (L7) and TCP/UDP (L4) proxies
 */
export type ProxyLayer = 'L7' | 'L4';

/**
 * Unified proxy type that combines HTTP and L4 proxies into a single discriminated union.
 * The `layer` field distinguishes between the two types.
 */
export type UnifiedProxy =
  | {
      layer: 'L7';
      id: number;
      name: string;
      description?: string;
      type: ProxyType;
      hostname: string;
      is_active: boolean;
      ssl_enabled: boolean;
      created_at: string;
      updated_at: string;
      original: ProxyConfig;
    }
  | {
      layer: 'L4';
      id: number;
      name: string;
      description?: string;
      protocol: L4Protocol;
      listen_port: number;
      is_active: boolean;
      created_at: string;
      updated_at: string;
      original: L4Proxy;
    };

/**
 * Filter options for the unified proxy list
 */
export type UnifiedProxyFilter = 'all' | 'http' | 'tcp' | 'udp';

/**
 * Converts an HTTP proxy to a unified proxy
 */
export function toUnifiedProxy(proxy: ProxyConfig): UnifiedProxy {
  return {
    layer: 'L7',
    id: proxy.id,
    name: proxy.name,
    description: proxy.description,
    type: proxy.type,
    hostname: proxy.hostname,
    is_active: proxy.is_active,
    ssl_enabled: proxy.ssl_enabled,
    created_at: proxy.created_at,
    updated_at: proxy.updated_at,
    original: proxy,
  };
}

/**
 * Converts an L4 proxy to a unified proxy
 */
export function toUnifiedL4Proxy(proxy: L4Proxy): UnifiedProxy {
  return {
    layer: 'L4',
    id: proxy.id,
    name: proxy.name,
    description: proxy.description,
    protocol: proxy.protocol,
    listen_port: proxy.listen_port,
    is_active: proxy.is_active,
    created_at: proxy.created_at,
    updated_at: proxy.updated_at,
    original: proxy,
  };
}

/**
 * Generates a unique key for a unified proxy (to handle potential ID collisions between types)
 */
export function getUnifiedProxyKey(proxy: UnifiedProxy): string {
  return `${proxy.layer}-${proxy.id}`;
}
