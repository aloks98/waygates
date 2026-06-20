import { Badge, Tooltip, TooltipContent, TooltipTrigger } from '@e412/rnui-react';
import { Server } from 'lucide-react';

import type { ProxyConfig } from '@/types/proxy';

interface ProxyTargetCellProps {
  proxy: ProxyConfig;
}

function formatUpstream(upstream: { scheme: string; host: string; port: number }): string {
  return `${upstream.scheme}://${upstream.host}:${upstream.port}`;
}

export function ProxyTargetCell({ proxy }: ProxyTargetCellProps) {
  if (proxy.type === 'reverse_proxy' && proxy.upstreams?.length) {
    const count = proxy.upstreams.length;

    if (count === 1) {
      return (
        <Badge variant="secondary" className="font-mono text-xs">
          {formatUpstream(proxy.upstreams[0])}
        </Badge>
      );
    }

    return (
      <Tooltip>
        <TooltipTrigger render={<div className="flex items-center gap-2" />}>
          <Badge variant="secondary" className="font-mono text-xs shrink-0">
            {formatUpstream(proxy.upstreams[0])}
          </Badge>
          <Badge variant="outline" className="text-xs shrink-0 gap-1">
            <Server className="size-3" />+{count - 1}
          </Badge>
        </TooltipTrigger>
        <TooltipContent side="bottom" align="start">
          <div className="space-y-1">
            {proxy.upstreams.map((u) => (
              <div key={`${u.host}:${u.port}`} className="font-mono text-xs">
                {formatUpstream(u)}
              </div>
            ))}
          </div>
        </TooltipContent>
      </Tooltip>
    );
  }

  if (proxy.type === 'redirect' && proxy.redirect) {
    return (
      <Badge variant="secondary" className="font-mono text-xs max-w-[250px] truncate">
        {proxy.redirect.target}
      </Badge>
    );
  }

  if (proxy.type === 'static' && proxy.static) {
    return (
      <Badge variant="secondary" className="font-mono text-xs">
        {proxy.static.root_path}
      </Badge>
    );
  }

  return <span className="text-muted-foreground">-</span>;
}

export function getProxyTarget(proxy: ProxyConfig): string {
  if (proxy.type === 'reverse_proxy' && proxy.upstreams?.length) {
    const upstream = proxy.upstreams[0];
    const count = proxy.upstreams.length;
    const target = formatUpstream(upstream);
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
