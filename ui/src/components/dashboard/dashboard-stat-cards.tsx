import { Skeleton, StatCard } from '@e412/rnui-react';
import { CheckCircle2, Server, Shield } from 'lucide-react';

import { useACLGroups } from '@/hooks';
import type { ProxyStats } from '@/types/api';
import type { L4ProxyStats } from '@/types/l4-proxy';

export function DashboardStatCards({
  proxyStats,
  l4ProxyStats,
  isLoading,
}: {
  proxyStats?: ProxyStats;
  l4ProxyStats?: L4ProxyStats;
  isLoading: boolean;
}) {
  const { total: aclTotal, isLoading: aclLoading } = useACLGroups({ page: 1, limit: 1 });

  if (isLoading || aclLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-3">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-24 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  const totalProxies = (proxyStats?.total ?? 0) + (l4ProxyStats?.total_proxies ?? 0);
  const activeProxies = (proxyStats?.active ?? 0) + (l4ProxyStats?.active_proxies ?? 0);

  return (
    <div className="grid gap-4 sm:grid-cols-3">
      <StatCard
        title="Total proxies"
        value={totalProxies}
        description={`${proxyStats?.total ?? 0} HTTP · ${l4ProxyStats?.total_proxies ?? 0} TCP/UDP`}
        icon={<Server className="size-4" />}
      />
      <StatCard
        title="Active"
        value={`${activeProxies} / ${totalProxies}`}
        description={`${totalProxies - activeProxies} inactive`}
        icon={<CheckCircle2 className="size-4" />}
      />
      <StatCard
        title="Access groups"
        value={aclTotal ?? 0}
        description="Auth & IP rules"
        icon={<Shield className="size-4" />}
      />
    </div>
  );
}
