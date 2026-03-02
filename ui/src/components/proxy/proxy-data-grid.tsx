import {
  Badge,
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
  Skeleton,
} from '@e412/titanium';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type PaginationState,
  useReactTable,
} from '@tanstack/react-table';
import { ExternalLink } from 'lucide-react';
import { useMemo } from 'react';

import type { ProxyConfig } from '@/types/proxy';

import {
  ProxyActionsCell,
  ProxySslCell,
  ProxyStatusBadge,
  ProxyTargetCell,
  ProxyTypeBadge,
} from './cells';

interface ProxyDataGridProps {
  data: ProxyConfig[];
  isLoading: boolean;
  canUpdateProxies: boolean;
  canDeleteProxies: boolean;
  onEdit: (proxy: ProxyConfig) => void;
  onDelete: (proxy: ProxyConfig) => void;
  onToggleStatus: (id: number, enable: boolean) => void;
  isToggling: boolean;
  // Pagination props
  pageCount: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  total: number;
}

export function ProxyDataGrid({
  data,
  isLoading,
  canUpdateProxies,
  canDeleteProxies,
  onEdit,
  onDelete,
  onToggleStatus,
  isToggling,
  pageCount,
  pagination,
  onPaginationChange,
  total,
}: ProxyDataGridProps) {
  const columns = useMemo<ColumnDef<ProxyConfig>[]>(
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
        accessorKey: 'hostname',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Hostname" />,
        cell: ({ row }) => {
          const hostname = row.getValue('hostname') as string;
          const url = `https://${hostname}`;
          return (
            <a
              href={url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 group"
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
        },
        minSize: 150,
        maxSize: 260,
        meta: {
          skeleton: <Skeleton className="h-6 w-40 rounded-full" />,
        },
      },
      {
        id: 'target',
        header: 'Target',
        cell: ({ row }) => <ProxyTargetCell proxy={row.original} />,
        enableSorting: false,
        minSize: 200,
        maxSize: 320,
        meta: {
          skeleton: <Skeleton className="h-5 w-48" />,
        },
      },
      {
        accessorKey: 'type',
        header: 'Type',
        cell: ({ row }) => <ProxyTypeBadge type={row.original.type} />,
        enableSorting: false,
        minSize: 100,
        maxSize: 150,
        meta: {
          skeleton: <Skeleton className="h-6 w-28 rounded-full" />,
        },
      },
      {
        accessorKey: 'ssl_enabled',
        header: 'SSL',
        cell: ({ row }) => <ProxySslCell enabled={row.original.ssl_enabled} />,
        enableSorting: false,
        minSize: 60,
        maxSize: 80,
        meta: {
          skeleton: <Skeleton className="h-5 w-16" />,
        },
      },
      {
        accessorKey: 'is_active',
        header: 'Status',
        cell: ({ row }) => (
          <ProxyStatusBadge
            isActive={row.original.is_active}
            canToggle={canUpdateProxies}
            onToggle={() => onToggleStatus(row.original.id, !row.original.is_active)}
            isToggling={isToggling}
          />
        ),
        enableSorting: false,
        minSize: 80,
        maxSize: 100,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded-full" />,
        },
      },
      ...(canUpdateProxies || canDeleteProxies
        ? [
            {
              id: 'actions',
              header: '',
              cell: ({ row }: { row: { original: ProxyConfig } }) => (
                <ProxyActionsCell
                  proxy={row.original}
                  canEdit={canUpdateProxies}
                  canDelete={canDeleteProxies}
                  onEdit={onEdit}
                  onDelete={onDelete}
                />
              ),
              enableSorting: false,
              minSize: 80,
              maxSize: 100,
              meta: {
                skeleton: (
                  <div className="flex justify-end gap-2">
                    <Skeleton className="size-8" />
                    <Skeleton className="size-8" />
                  </div>
                ),
              },
            },
          ]
        : []),
    ],
    [canUpdateProxies, canDeleteProxies, onEdit, onDelete, onToggleStatus, isToggling],
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
