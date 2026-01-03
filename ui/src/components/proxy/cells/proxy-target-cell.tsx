import type { ProxyConfig } from '@/types/proxy';

interface ProxyTargetCellProps {
  proxy: ProxyConfig;
}

export function ProxyTargetCell({ proxy }: ProxyTargetCellProps) {
  const target = getProxyTarget(proxy);

  return <span className="text-muted-foreground max-w-[200px] truncate block">{target}</span>;
}

export function getProxyTarget(proxy: ProxyConfig): string {
  if (proxy.type === 'reverse_proxy' && proxy.upstreams?.length) {
    const upstream = proxy.upstreams[0];
    const count = proxy.upstreams.length;
    const target = `${upstream.scheme}://${upstream.host}:${upstream.port}`;
    return count > 1 ? `${target} (+${count - 1} more)` : target;
  }

  if (proxy.type === 'redirect' && proxy.redirect) {
    return proxy.redirect.target;
  }

  if (proxy.type === 'static' && proxy.static) {
    return proxy.static.root_path;
  }

  return '-';
}
