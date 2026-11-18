import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { ArrowUpRight, Plus } from 'lucide-react';
import { useState } from 'react';
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from 'recharts';

import { ProxyStatusBadge } from '@/components/proxy/cells/proxy-status-badge';
import { getProxyTarget } from '@/components/proxy/cells/proxy-target-cell';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';
import { Empty, EmptyContent, EmptyHeader, EmptyTitle } from '@/components/ui/empty';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { usePermissions } from '@/hooks/use-permissions';
import { useTrafficMetrics } from '@/hooks/use-traffic-metrics';
import { api } from '@/lib/api';
import type { ApiResponse, PaginatedResponse, ProxyStats } from '@/types/api';
import type { L4ProxyStats } from '@/types/l4-proxy';
import type { TrafficRange } from '@/types/metrics';
import type { ProxyConfig } from '@/types/proxy';

const chartConfig = { requests: { label: 'Requests', color: 'var(--chart-1)' } };
const count = new Intl.NumberFormat(undefined, { notation: 'compact', maximumFractionDigits: 1 });

export function DashboardIndex() {
  const permissions = usePermissions();
  const [range, setRange] = useState<TrafficRange>('24h');
  const traffic = useTrafficMetrics(range, permissions.canReadMetrics);
  const proxies = useQuery({
    queryKey: ['proxies', 'overview'],
    queryFn: async () =>
      (
        await api
          .get('proxies', { searchParams: { limit: 5, sort: 'updated_at', order: 'desc' } })
          .json<ApiResponse<PaginatedResponse<ProxyConfig>>>()
      ).data,
    enabled: permissions.canReadProxies,
  });
  const stats = useQuery({
    queryKey: ['proxies', 'stats'],
    queryFn: async () => (await api.get('proxies/stats').json<ApiResponse<ProxyStats>>()).data,
    enabled: permissions.canReadProxies,
  });
  const l4 = useQuery({
    queryKey: ['l4-proxies', 'stats'],
    queryFn: async () => (await api.get('l4-proxies/stats').json<ApiResponse<L4ProxyStats>>()).data,
    enabled: permissions.hasPermission('l4proxies:read'),
  });
  const points = traffic.series?.points ?? [];
  const data = points.map((p) => ({
    ...p,
    requests: p.req_2xx + p.req_3xx + p.req_4xx + p.req_5xx + p.req_other,
  }));
  const requests = data.reduce((sum, p) => sum + p.requests, 0);
  const errors = data.reduce((sum, p) => sum + p.req_5xx, 0);
  const last = data.filter((p) => p.requests > 0).at(-1);
  const hasMetrics = permissions.canReadMetrics && !!traffic.series;
  const total = (stats.data?.total ?? 0) + (l4.data?.total_proxies ?? 0);
  const cards = [
    {
      label: 'Proxies',
      value: permissions.canReadProxies && stats.data ? String(total) : '—',
      loading: stats.isLoading || l4.isLoading,
    },
    {
      label: 'Requests',
      value: hasMetrics ? count.format(requests) : '—',
      loading: traffic.isLoading,
    },
    {
      label: 'Server errors',
      value: hasMetrics ? (requests ? `${((errors / requests) * 100).toFixed(2)}%` : '0%') : '—',
      loading: traffic.isLoading,
    },
    {
      label: 'Latest p95 response',
      value: last ? `${Math.round(last.p95_ms)} ms` : '—',
      loading: traffic.isLoading,
    },
  ];
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-[28px] font-semibold">Overview</h1>
        {permissions.canCreateProxies && (
          <Button render={<Link to="/proxies/new" />}>
            <Plus data-icon="inline-start" />
            Create proxy
          </Button>
        )}
      </div>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => (
          <Card key={card.label}>
            <CardContent className="flex flex-col gap-3">
              <span className="text-sm text-muted-foreground">{card.label}</span>
              {card.loading ? (
                <Skeleton className="h-9 w-24" />
              ) : (
                <span className="text-[30px] font-semibold tabular-nums tracking-tight">
                  {card.value}
                </span>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
      {permissions.canReadMetrics && (
        <Card>
          <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
            <CardTitle>Request volume</CardTitle>
            <Tabs value={range} onValueChange={(v) => setRange(v as TrafficRange)}>
              <TabsList>
                <TabsTrigger value="1h">1 hour</TabsTrigger>
                <TabsTrigger value="24h">24 hours</TabsTrigger>
                <TabsTrigger value="7d">7 days</TabsTrigger>
              </TabsList>
            </Tabs>
          </CardHeader>
          <CardContent>
            {traffic.isLoading ? (
              <Skeleton className="h-64 w-full" />
            ) : traffic.isError ? (
              <div className="flex h-64 flex-col items-center justify-center gap-3 text-sm text-muted-foreground">
                <p>Traffic data is unavailable.</p>
                <Button variant="outline" onClick={() => traffic.refetch()}>
                  Retry
                </Button>
              </div>
            ) : data.length === 0 ? (
              <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">
                No traffic recorded in this period.
              </div>
            ) : (
              <ChartContainer
                config={chartConfig}
                className="h-64 w-full"
                aria-label="Requests over time"
              >
                <AreaChart accessibilityLayer data={data} margin={{ left: 0, right: 12, top: 12 }}>
                  <defs>
                    <linearGradient id="request-fill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--chart-1)" stopOpacity={0.2} />
                      <stop offset="100%" stopColor="var(--chart-1)" stopOpacity={0.01} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid vertical={false} strokeDasharray="3 3" />
                  <XAxis
                    dataKey="t"
                    tick={{ fill: 'var(--muted-foreground)' }}
                    tickLine={false}
                    axisLine={false}
                    minTickGap={40}
                    tickFormatter={(v) =>
                      new Date(v).toLocaleString(
                        undefined,
                        range === '7d'
                          ? { month: 'short', day: 'numeric' }
                          : { hour: '2-digit', minute: '2-digit' },
                      )
                    }
                  />
                  <YAxis
                    tick={{ fill: 'var(--muted-foreground)' }}
                    tickLine={false}
                    axisLine={false}
                    width={40}
                    allowDecimals={false}
                    tickFormatter={(v) => count.format(v)}
                  />
                  <ChartTooltip
                    content={
                      <ChartTooltipContent labelFormatter={(v) => new Date(v).toLocaleString()} />
                    }
                  />
                  <Area
                    type="monotone"
                    dataKey="requests"
                    stroke="var(--chart-1)"
                    strokeWidth={2}
                    fill="url(#request-fill)"
                    isAnimationActive={false}
                  />
                </AreaChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
      )}
      {permissions.canReadProxies && (
        <Card className="gap-0 pb-0">
          <CardHeader className="mb-5 flex flex-row items-center justify-between">
            <CardTitle>Recently updated proxies</CardTitle>
            <Button size="sm" variant="ghost" render={<Link to="/proxies" />}>
              View all
              <ArrowUpRight data-icon="inline-end" />
            </Button>
          </CardHeader>
          {proxies.isError ? (
            <div className="flex items-center justify-center gap-3 p-8 text-sm">
              <span>Couldn’t load proxies.</span>
              <Button variant="outline" onClick={() => proxies.refetch()}>
                Retry
              </Button>
            </div>
          ) : !proxies.isLoading && !proxies.data?.items.length ? (
            <Empty>
              <EmptyHeader>
                <EmptyTitle>No proxies yet</EmptyTitle>
              </EmptyHeader>
              {permissions.canCreateProxies && (
                <EmptyContent>
                  <Button render={<Link to="/proxies/new" />}>Create proxy</Button>
                </EmptyContent>
              )}
            </Empty>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40">
                  <TableHead className="px-6">Proxy</TableHead>
                  <TableHead>Destination</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>
                    <span className="sr-only">Open</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {proxies.isLoading
                  ? [1, 2, 3].map((i) => (
                      <TableRow key={i}>
                        <TableCell colSpan={4} className="p-6">
                          <Skeleton className="h-8 w-full" />
                        </TableCell>
                      </TableRow>
                    ))
                  : proxies.data?.items.map((proxy) => (
                      <TableRow key={proxy.id}>
                        <TableCell className="px-6 py-4">
                          <Link
                            to="/proxies/$proxyId"
                            params={{ proxyId: String(proxy.id) }}
                            className="flex flex-col gap-1 hover:underline"
                          >
                            <span className="font-medium">{proxy.name}</span>
                            <span className="font-mono text-xs text-muted-foreground">
                              {proxy.hostname}
                            </span>
                          </Link>
                        </TableCell>
                        <TableCell
                          className="max-w-64 truncate font-mono text-xs"
                          title={getProxyTarget(proxy)}
                        >
                          {getProxyTarget(proxy)}
                        </TableCell>
                        <TableCell>
                          <ProxyStatusBadge isActive={proxy.is_active} />
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Open ${proxy.name}`}
                            render={
                              <Link to="/proxies/$proxyId" params={{ proxyId: String(proxy.id) }} />
                            }
                          >
                            <ArrowUpRight />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
              </TableBody>
            </Table>
          )}
        </Card>
      )}
    </div>
  );
}
