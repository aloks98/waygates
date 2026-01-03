import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  DataGrid,
  DataGridTable,
  Skeleton,
} from '@e412/titanium';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { type ColumnDef, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { Activity, ArrowRight, ExternalLink, FolderOpen, Globe, Pause, Plus } from 'lucide-react';
import { useMemo } from 'react';
import { ProxyStatusBadge, ProxyTargetCell, ProxyTypeBadge } from '@/components/proxy';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import type { ApiResponse, PaginatedResponse, ProxyStats } from '@/types/api';
import type { ProxyConfig } from '@/types/proxy';

export function DashboardIndex() {
  const { user } = useAuthStore();

  const { data: statsData, isLoading: isStatsLoading } = useQuery({
    queryKey: ['proxies', 'stats'],
    queryFn: async () => {
      const response = await api.get('proxies/stats').json<ApiResponse<ProxyStats>>();
      return response.data;
    },
  });

  const { data: recentData, isLoading: isRecentLoading } = useQuery({
    queryKey: ['proxies', 'recent'],
    queryFn: async () => {
      const response = await api
        .get('proxies', { searchParams: { limit: '5', sort: 'created_at', order: 'desc' } })
        .json<ApiResponse<PaginatedResponse<ProxyConfig>>>();
      return response.data;
    },
  });

  const isLoading = isStatsLoading || isRecentLoading;
  const recentProxies = recentData?.items || [];

  const columns = useMemo<ColumnDef<ProxyConfig>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
        meta: { skeleton: <Skeleton className="h-5 w-32" /> },
      },
      {
        accessorKey: 'hostname',
        header: 'Hostname',
        cell: ({ row }) => <span className="text-muted-foreground">{row.original.hostname}</span>,
        meta: { skeleton: <Skeleton className="h-5 w-40" /> },
      },
      {
        id: 'target',
        header: 'Target',
        cell: ({ row }) => <ProxyTargetCell proxy={row.original} />,
        meta: { skeleton: <Skeleton className="h-5 w-48" /> },
      },
      {
        accessorKey: 'type',
        header: 'Type',
        cell: ({ row }) => <ProxyTypeBadge type={row.original.type} />,
        meta: { skeleton: <Skeleton className="h-6 w-24 rounded-full" /> },
      },
      {
        accessorKey: 'is_active',
        header: 'Status',
        cell: ({ row }) => <ProxyStatusBadge isActive={row.original.is_active} />,
        meta: { skeleton: <Skeleton className="h-6 w-16 rounded-full" /> },
      },
    ],
    [],
  );

  const table = useReactTable({
    data: recentProxies,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const stats = [
    { label: 'Total Proxies', value: statsData?.total ?? 0, icon: Globe, color: 'text-primary' },
    { label: 'Active', value: statsData?.active ?? 0, icon: Activity, color: 'text-green-500' },
    {
      label: 'Inactive',
      value: statsData?.inactive ?? 0,
      icon: Pause,
      color: 'text-muted-foreground',
    },
  ];

  const typeStats = [
    {
      label: 'Reverse Proxies',
      value: statsData?.by_type?.reverse_proxy ?? 0,
      icon: Globe,
      color: 'text-blue-500',
    },
    {
      label: 'Redirects',
      value: statsData?.by_type?.redirect ?? 0,
      icon: ArrowRight,
      color: 'text-amber-500',
    },
    {
      label: 'Static Sites',
      value: statsData?.by_type?.static ?? 0,
      icon: FolderOpen,
      color: 'text-purple-500',
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Welcome back, {user?.name || 'User'}</h1>
          <p className="mt-1 text-muted-foreground">
            Here&apos;s an overview of your proxy configuration.
          </p>
        </div>
        <Button asChild>
          <Link to="/dashboard/proxies">
            <ExternalLink className="size-4" />
            Manage Proxies
          </Link>
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {isLoading
          ? [1, 2, 3].map((i) => (
              <Card key={i}>
                <CardContent className="flex items-center gap-4 p-6">
                  <Skeleton className="size-12 rounded-full" />
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-8 w-12" />
                  </div>
                </CardContent>
              </Card>
            ))
          : stats.map((stat) => (
              <Card key={stat.label}>
                <CardContent className="flex items-center gap-4 p-6">
                  <div className={`rounded-full bg-secondary p-3 ${stat.color}`}>
                    <stat.icon className="size-6" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">{stat.label}</p>
                    <p className="text-2xl font-bold">{stat.value}</p>
                  </div>
                </CardContent>
              </Card>
            ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Proxies by Type</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-3">
          {isLoading
            ? [1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="size-8 rounded-md" />
                  <div className="space-y-1">
                    <Skeleton className="h-4 w-24" />
                    <Skeleton className="h-6 w-8" />
                  </div>
                </div>
              ))
            : typeStats.map((stat) => (
                <div key={stat.label} className="flex items-center gap-3">
                  <div className={`rounded-md bg-secondary p-2 ${stat.color}`}>
                    <stat.icon className="size-4" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">{stat.label}</p>
                    <p className="text-lg font-semibold">{stat.value}</p>
                  </div>
                </div>
              ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Recent Proxies</CardTitle>
          {(statsData?.total ?? 0) > 5 && (
            <Button variant="ghost" size="sm" asChild>
              <Link to="/dashboard/proxies">
                View all
                <ArrowRight className="ml-1 size-4" />
              </Link>
            </Button>
          )}
        </CardHeader>
        <CardContent className="p-0">
          {!isLoading && (statsData?.total ?? 0) === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Globe className="size-12 text-muted-foreground/50" />
              <h3 className="mt-4 text-lg font-medium">No proxies configured</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                Get started by creating your first proxy configuration.
              </p>
              <Button className="mt-4" asChild>
                <Link to="/dashboard/proxies">
                  <Plus className="size-4" />
                  Create Proxy
                </Link>
              </Button>
            </div>
          ) : (
            <DataGrid
              table={table}
              recordCount={recentProxies.length}
              isLoading={isLoading}
              loadingMode="skeleton"
              emptyMessage="No proxies configured yet."
            >
              <DataGridTable />
            </DataGrid>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
