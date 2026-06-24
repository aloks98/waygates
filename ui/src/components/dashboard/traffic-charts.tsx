import {
  AreaChart,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  LineChart,
  Skeleton,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@e412/rnui-react';
import { useState } from 'react';

import { useChartColors } from '@/hooks/use-chart-colors';
import { useTrafficMetrics } from '@/hooks/use-traffic-metrics';
import { pointsToCategories, seriesFor } from '@/lib/traffic-format';
import type { TrafficRange } from '@/types/metrics';

const CHART_HEIGHT = 220;

function EmptyState() {
  return <p className="py-10 text-center text-sm text-muted-foreground">No traffic data yet.</p>;
}

export function TrafficCharts() {
  const [range, setRange] = useState<TrafficRange>('1h');
  const { series, isLoading } = useTrafficMetrics(range);
  const colors = useChartColors();

  const points = series?.points ?? [];
  const categories = pointsToCategories(points, range);
  const isEmpty = !isLoading && points.length === 0;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Traffic</h2>
        <Tabs value={range} onValueChange={(v) => setRange(v as TrafficRange)}>
          <TabsList variant="line">
            <TabsTrigger value="1h">1h</TabsTrigger>
            <TabsTrigger value="24h">24h</TabsTrigger>
            <TabsTrigger value="7d">7d</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Requests over time */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Requests</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-[220px] w-full" />
            ) : isEmpty ? (
              <EmptyState />
            ) : (
              <AreaChart
                categories={categories}
                series={[
                  { name: '2xx', data: seriesFor(points, 'req_2xx'), color: colors['--chart-2'] },
                  { name: '3xx', data: seriesFor(points, 'req_3xx'), color: colors['--chart-3'] },
                  { name: '4xx', data: seriesFor(points, 'req_4xx'), color: '#f59e0b' },
                  {
                    name: '5xx',
                    data: seriesFor(points, 'req_5xx'),
                    color: colors['--destructive'],
                  },
                  {
                    name: 'other',
                    data: seriesFor(points, 'req_other'),
                    color: colors['--chart-5'],
                  },
                ]}
                stacked
                smooth
                showLegend
                height={CHART_HEIGHT}
              />
            )}
          </CardContent>
        </Card>

        {/* Bandwidth */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Bandwidth</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-[220px] w-full" />
            ) : isEmpty ? (
              <EmptyState />
            ) : (
              <AreaChart
                categories={categories}
                series={[
                  { name: 'In', data: seriesFor(points, 'bytes_in'), color: colors['--chart-1'] },
                  { name: 'Out', data: seriesFor(points, 'bytes_out'), color: colors['--chart-4'] },
                ]}
                smooth
                showLegend
                height={CHART_HEIGHT}
              />
            )}
          </CardContent>
        </Card>

        {/* Latency — full width on lg */}
        <Card className="lg:col-span-2">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Latency (ms)</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-[220px] w-full" />
            ) : isEmpty ? (
              <EmptyState />
            ) : (
              <LineChart
                categories={categories}
                series={[
                  { name: 'p50', data: seriesFor(points, 'p50_ms'), smooth: true },
                  { name: 'p95', data: seriesFor(points, 'p95_ms'), smooth: true },
                ]}
                showLegend
                height={CHART_HEIGHT}
              />
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
