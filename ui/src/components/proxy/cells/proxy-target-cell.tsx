import { Server } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { formatUpstream } from '@/lib/upstream-url';
import type { ProxyConfig } from '@/types/proxy';

interface ProxyTargetCellProps {
  proxy: ProxyConfig;
}

export function ProxyTargetCell({ proxy }: ProxyTargetCellProps) {
  if (proxy.type === 'reverse_proxy' && proxy.upstreams?.length) {
    const count = proxy.upstreams.length;

    if (count === 1) {
      return <span className="font-mono text-xs">{formatUpstream(proxy.upstreams[0])}</span>;
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
      <span className="max-w-64 truncate font-mono text-xs" title={proxy.redirect.target}>
        {proxy.redirect.target}
      </span>
    );
  }

  if (proxy.type === 'static' && proxy.static) {
    return <span className="font-mono text-xs">{proxy.static.root_path}</span>;
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
