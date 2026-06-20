import { Card, CardContent, CardHeader, CardTitle, PieChart, Skeleton } from '@e412/rnui-react';

import { buildCompositionData } from '@/lib/dashboard-format';
import type { ProxyStats } from '@/types/api';
import type { L4ProxyStats } from '@/types/l4-proxy';

const CHART_COLORS = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
];

export function FleetComposition({
  proxyStats,
  l4ProxyStats,
  isLoading,
}: {
  proxyStats?: ProxyStats;
  l4ProxyStats?: L4ProxyStats;
  isLoading: boolean;
}) {
  const data = buildCompositionData(proxyStats, l4ProxyStats);
  const total = data.reduce((sum, d) => sum + d.value, 0);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Fleet composition</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-48 w-full" />
        ) : total === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">No proxies to summarize.</p>
        ) : (
          <div className="flex flex-col items-center gap-8 sm:flex-row sm:justify-center sm:gap-12">
            <div className="h-52 w-52 shrink-0">
              <PieChart data={data} donut showLegend={false} showLabels={false} height={208} />
            </div>
            <ul className="w-full max-w-[200px] space-y-2">
              {data.map((d, i) => (
                <li key={d.name} className="flex items-center justify-between gap-6 text-sm">
                  <span className="flex items-center gap-2">
                    <span
                      className="inline-block size-2.5 rounded-sm"
                      style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }}
                    />
                    {d.name}
                  </span>
                  <span className="font-medium tabular-nums">{d.value}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
