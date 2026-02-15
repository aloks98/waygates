import {
  Badge,
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
  Skeleton,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/titanium';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type PaginationState,
  useReactTable,
} from '@tanstack/react-table';
import { ExternalLink, Layers, Server } from 'lucide-react';
import { useMemo } from 'react';
import type { UnifiedProxy } from '@/types/unified-proxy';
import { L4ProtocolBadge, ProxyStatusBadge, ProxyTypeBadge } from './cells';

interface UnifiedProxyDataGridProps {
  data: UnifiedProxy[];
  isLoading: boolean;
  canUpdateProxies: boolean;
  onRowClick: (proxy: UnifiedProxy) => void;
  onToggleStatus: (proxy: UnifiedProxy, enable: boolean) => void;
  isToggling: boolean;
  pageCount: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  total: number;
}

function formatUpstream(upstream: { scheme: string; host: string; port: number }): string {
  return `${upstream.scheme}://${upstream.host}:${upstream.port}`;
}

function L4TargetCell({ proxy }: { proxy: UnifiedProxy }) {
  if (proxy.layer !== 'L4') return null;

  const routes = proxy.original.routes ?? [];
  const upstreamCount = routes.reduce((acc, r) => acc + (r.upstreams?.length ?? 0), 0);

  if (upstreamCount === 0) {
    return <span className="text-muted-foreground">No upstreams</span>;
  }

  const firstUpstream = routes[0]?.upstreams?.[0];
  if (!firstUpstream) {
    return <span className="text-muted-foreground">-</span>;
  }

  const targetDisplay = `${firstUpstream.host}:${firstUpstream.port}`;

  if (upstreamCount === 1) {
    return (
      <Badge variant="secondary" className="font-mono text-xs">
        {targetDisplay}
      </Badge>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="font-mono text-xs shrink-0">
            {targetDisplay}
          </Badge>
          <Badge variant="outline" className="text-xs shrink-0 gap-1">
            <Server className="size-3" />+{upstreamCount - 1}
          </Badge>
        </div>
      </TooltipTrigger>
      <TooltipContent side="bottom" align="start">
        <div className="space-y-1">
          {routes.map((route) =>
            route.upstreams?.map((u) => (
              <div key={`${route.id}-${u.host}:${u.port}`} className="font-mono text-xs">
                {u.host}:{u.port}
              </div>
            )),
          )}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

function HttpTargetCell({ proxy }: { proxy: UnifiedProxy }) {
  if (proxy.layer !== 'L7') return null;

  const original = proxy.original;

  if (original.type === 'reverse_proxy' && original.upstreams?.length) {
    const count = original.upstreams.length;

    if (count === 1) {
      return (
        <Badge variant="secondary" className="font-mono text-xs">
          {formatUpstream(original.upstreams[0])}
        </Badge>
      );
    }

    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="font-mono text-xs shrink-0">
              {formatUpstream(original.upstreams[0])}
            </Badge>
            <Badge variant="outline" className="text-xs shrink-0 gap-1">
              <Server className="size-3" />+{count - 1}
            </Badge>
          </div>
        </TooltipTrigger>
        <TooltipContent side="bottom" align="start">
          <div className="space-y-1">
            {original.upstreams.map((u) => (
              <div key={`${u.host}:${u.port}`} className="font-mono text-xs">
                {formatUpstream(u)}
              </div>
            ))}
          </div>
        </TooltipContent>
      </Tooltip>
    );
  }

  if (original.type === 'redirect' && original.redirect) {
    return (
      <Badge variant="secondary" className="font-mono text-xs max-w-[250px] truncate">
        {original.redirect.target}
      </Badge>
    );
  }

  if (original.type === 'static' && original.static) {
    return (
      <Badge variant="secondary" className="font-mono text-xs">
        {original.static.root_path}
      </Badge>
    );
  }

  return <span className="text-muted-foreground">-</span>;
}

function UnifiedTargetCell({ proxy }: { proxy: UnifiedProxy }) {
  if (proxy.layer === 'L4') {
    return <L4TargetCell proxy={proxy} />;
  }
  return <HttpTargetCell proxy={proxy} />;
}

export function UnifiedProxyDataGrid({
  data,
  isLoading,
  canUpdateProxies,
  onRowClick,
  onToggleStatus,
  isToggling,
  pageCount,
  pagination,
  onPaginationChange,
  total,
}: UnifiedProxyDataGridProps) {
  const columns = useMemo<ColumnDef<UnifiedProxy>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Name" />,
        cell: ({ row }) => <span className="font-medium break-words">{row.getValue('name')}</span>,
        minSize: 120,
        maxSize: 200,
        meta: {
          skeleton: <Skeleton className="h-5 w-32" />,
        },
      },
      {
        id: 'layer',
        header: 'Layer',
        cell: ({ row }) => {
          const proxy = row.original;
          return (
            <Badge variant={proxy.layer === 'L7' ? 'outline' : 'secondary'} className="gap-1">
              <Layers className="size-3" />
              {proxy.layer}
            </Badge>
          );
        },
        enableSorting: false,
        minSize: 70,
        maxSize: 90,
        meta: {
          skeleton: <Skeleton className="h-6 w-14 rounded-full" />,
        },
      },
      {
        id: 'type',
        header: 'Type',
        cell: ({ row }) => {
          const proxy = row.original;
          if (proxy.layer === 'L7') {
            return <ProxyTypeBadge type={proxy.type} />;
          }
          return <L4ProtocolBadge protocol={proxy.protocol} />;
        },
        enableSorting: false,
        minSize: 100,
        maxSize: 150,
        meta: {
          skeleton: <Skeleton className="h-6 w-28 rounded-full" />,
        },
      },
      {
        id: 'endpoint',
        header: 'Endpoint',
        cell: ({ row }) => {
          const proxy = row.original;
          if (proxy.layer === 'L7') {
            const hostname = proxy.hostname;
            const url = `https://${hostname}`;
            return (
              <a
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 group"
                onClick={(e) => e.stopPropagation()}
              >
                <Badge
                  variant="secondary"
                  className="font-mono text-xs group-hover:bg-primary group-hover:text-primary-foreground transition-colors"
                >
                  {hostname}
                  <ExternalLink className="ml-1 size-3 opacity-50 group-hover:opacity-100" />
                </Badge>
              </a>
            );
          }
          // L4 proxy - show listen port
          return (
            <Badge variant="secondary" className="font-mono text-xs">
              :{proxy.listen_port}
            </Badge>
          );
        },
        minSize: 120,
        maxSize: 220,
        meta: {
          skeleton: <Skeleton className="h-6 w-32 rounded-full" />,
        },
      },
      {
        id: 'target',
        header: 'Target',
        cell: ({ row }) => <UnifiedTargetCell proxy={row.original} />,
        enableSorting: false,
        minSize: 180,
        maxSize: 300,
        meta: {
          skeleton: <Skeleton className="h-5 w-48" />,
        },
      },
      {
        accessorKey: 'is_active',
        header: 'Status',
        cell: ({ row }) => {
          const proxy = row.original;
          return (
            <ProxyStatusBadge
              isActive={proxy.is_active}
              canToggle={canUpdateProxies}
              onToggle={() => onToggleStatus(proxy, !proxy.is_active)}
              isToggling={isToggling}
            />
          );
        },
        enableSorting: false,
        minSize: 80,
        maxSize: 120,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded-full" />,
        },
      },
    ],
    [canUpdateProxies, onToggleStatus, isToggling],
  );

  const table = useReactTable({
    data,
    columns,
    pageCount,
    state: {
      pagination,
    },
    onPaginationChange: (updater) => {
      const newPagination = typeof updater === 'function' ? updater(pagination) : updater;
      onPaginationChange(newPagination);
    },
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    manualPagination: true,
  });

  return (
    <DataGrid
      table={table}
      recordCount={total}
      isLoading={isLoading}
      loadingMode="skeleton"
      emptyMessage="No proxies found. Create your first proxy to get started."
      onRowClick={onRowClick}
    >
      <DataGridContainer>
        <DataGridTable />
        <div className="border-t px-4 py-2">
          <DataGridPagination sizes={[10, 20, 50]} />
        </div>
      </DataGridContainer>
    </DataGrid>
  );
}
