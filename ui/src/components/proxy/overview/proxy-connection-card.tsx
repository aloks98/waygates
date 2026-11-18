import { ArrowDown, Globe, Server } from 'lucide-react';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { ProxyConfig } from '@/types/proxy';

import { getProxyTarget } from '../cells/proxy-target-cell';
export function ProxyConnectionCard({ proxy }: { proxy: ProxyConfig }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Connection preview</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-col gap-3 rounded-md bg-muted p-4">
          <Globe className="size-4 text-accent-foreground" />
          <span className="break-all font-mono text-sm">{proxy.hostname}</span>
        </div>
        <ArrowDown className="mx-auto size-5 text-muted-foreground" />
        <div className="flex flex-col gap-3 rounded-md bg-muted p-4">
          <Server className="size-4 text-accent-foreground" />
          <span className="break-all font-mono text-sm">{getProxyTarget(proxy)}</span>
        </div>
      </CardContent>
    </Card>
  );
}
