import {
  Badge,
  Button,
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridTable,
  Skeleton,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/titanium';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { type ColumnDef, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { ExternalLink, Globe, Shield } from 'lucide-react';
import { useCallback, useMemo } from 'react';

import { api } from '@/lib/api';
import type { ProxyACLAssignment } from '@/types/acl';
import type { ApiResponse } from '@/types/api';

interface GroupUsageTabProps {
  groupId: number;
}

export function GroupUsageTab({ groupId }: GroupUsageTabProps) {
  const navigate = useNavigate();

  const { data: assignments, isLoading } = useQuery({
    queryKey: ['acl-groups', groupId, 'usage'],
    queryFn: async () => {
      const response = await api
        .get(`acl/groups/${groupId}/usage`)
        .json<ApiResponse<ProxyACLAssignment[]>>();
      return response.data ?? [];
    },
    staleTime: 30000,
  });

  const handleProxyClick = useCallback(
    (_proxyId: number) => {
      navigate({ to: '/dashboard/proxies' });
    },
    [navigate],
  );

  const columns = useMemo<ColumnDef<ProxyACLAssignment>[]>(
    () => [
      {
        accessorKey: 'proxy.name',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Proxy" />,
        cell: ({ row }) => {
          const proxy = row.original.proxy;
          if (!proxy) return null;
          return (
            <div className="flex items-center gap-2">
              <div className="flex items-center justify-center size-8 rounded bg-muted">
                <Globe className="size-4 text-muted-foreground" />
              </div>
              <span className="font-medium">{proxy.name || proxy.hostname}</span>
            </div>
          );
        },
        minSize: 140,
        maxSize: 240,
        meta: {
          skeleton: (
            <div className="flex items-center gap-2">
              <Skeleton className="size-8 rounded" />
              <Skeleton className="h-5 w-32" />
            </div>
          ),
        },
      },
      {
        accessorKey: 'proxy.hostname',
        header: 'Hostname',
        cell: ({ row }) => {
          const proxy = row.original.proxy;
          if (!proxy) return null;
          return (
            <a
              href={`https://${proxy.hostname}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              onClick={(e) => e.stopPropagation()}
            >
              <code className="bg-muted px-1.5 py-0.5 rounded text-xs">{proxy.hostname}</code>
              <ExternalLink className="size-3" />
            </a>
          );
        },
        enableSorting: false,
        minSize: 140,
        maxSize: 260,
        meta: {
          skeleton: <Skeleton className="h-6 w-40" />,
        },
      },
      {
        accessorKey: 'path_pattern',
        header: 'Path Pattern',
        cell: ({ row }) => (
          <code className="text-xs bg-muted px-2 py-0.5 rounded">
            {row.original.path_pattern || '/*'}
          </code>
        ),
        enableSorting: false,
        minSize: 80,
        maxSize: 140,
        meta: {
          skeleton: <Skeleton className="h-6 w-16" />,
        },
      },
      {
        accessorKey: 'priority',
        header: 'Priority',
        cell: ({ row }) => (
          <Badge variant="outline" className="font-mono">
            {row.original.priority}
          </Badge>
        ),
        enableSorting: false,
        minSize: 60,
        maxSize: 100,
        meta: {
          skeleton: <Skeleton className="h-6 w-12 rounded" />,
        },
      },
      {
        accessorKey: 'enabled',
        header: 'Status',
        cell: ({ row }) =>
          row.original.enabled ? (
            <Badge variant="secondary">Active</Badge>
          ) : (
            <Badge variant="outline">Disabled</Badge>
          ),
        enableSorting: false,
        minSize: 70,
        maxSize: 100,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded" />,
        },
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => {
          const proxy = row.original.proxy;
          if (!proxy) return null;
          return (
            <div className="flex justify-end">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="size-8 p-0"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleProxyClick(proxy.id);
                    }}
                  >
                    <ExternalLink className="size-4" />
                    <span className="sr-only">View proxy</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>View proxy</TooltipContent>
              </Tooltip>
            </div>
          );
        },
        enableSorting: false,
        minSize: 60,
        maxSize: 80,
        meta: {
          skeleton: (
            <div className="flex justify-end">
              <Skeleton className="size-8" />
            </div>
          ),
        },
      },
    ],
    [handleProxyClick],
  );

  const table = useReactTable({
    data: assignments ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  if (!isLoading && (!assignments || assignments.length === 0)) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <Shield className="size-12 mx-auto mb-4 opacity-50" />
        <p>No proxies using this group</p>
        <p className="text-sm mt-1">
          Assign this ACL group to a proxy to protect it with these access controls
        </p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => navigate({ to: '/dashboard/proxies' })}
        >
          <Globe className="size-4" />
          Go to Proxies
        </Button>
      </div>
    );
  }

  return (
    <DataGrid
      table={table}
      recordCount={assignments?.length ?? 0}
      isLoading={isLoading}
      loadingMode="skeleton"
      emptyMessage="No proxies using this group"
    >
      <DataGridContainer>
        <DataGridTable />
      </DataGridContainer>
    </DataGrid>
  );
}
