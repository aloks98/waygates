import { useMemo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  type ColumnDef,
} from '@tanstack/react-table';
import {
  DataGrid,
  DataGridTable,
  DataGridColumnHeader,
  Skeleton,
} from '@e412/titanium';
import type { Proxy } from '@/types/proxy';
import {
  ProxyTypeBadge,
  ProxyStatusBadge,
  ProxyTargetCell,
  ProxySslCell,
  ProxyActionsCell,
} from './cells';

interface ProxyDataGridProps {
  data: Proxy[];
  isLoading: boolean;
  canUpdateProxies: boolean;
  canDeleteProxies: boolean;
  onEdit: (proxy: Proxy) => void;
  onDelete: (proxy: Proxy) => void;
  onToggleStatus: (id: number, enable: boolean) => void;
  isToggling: boolean;
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
}: ProxyDataGridProps) {
  const columns = useMemo<ColumnDef<Proxy>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataGridColumnHeader column={column} title="Name" />
        ),
        cell: ({ row }) => (
          <span className="font-medium">{row.getValue('name')}</span>
        ),
        meta: {
          skeleton: <Skeleton className="h-5 w-32" />,
        },
      },
      {
        accessorKey: 'hostname',
        header: ({ column }) => (
          <DataGridColumnHeader column={column} title="Hostname" />
        ),
        meta: {
          skeleton: <Skeleton className="h-5 w-40" />,
        },
      },
      {
        id: 'target',
        header: 'Target',
        cell: ({ row }) => <ProxyTargetCell proxy={row.original} />,
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-5 w-48" />,
        },
      },
      {
        accessorKey: 'type',
        header: 'Type',
        cell: ({ row }) => <ProxyTypeBadge type={row.original.type} />,
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-6 w-28 rounded-full" />,
        },
      },
      {
        accessorKey: 'ssl_enabled',
        header: 'SSL',
        cell: ({ row }) => <ProxySslCell enabled={row.original.ssl_enabled} />,
        enableSorting: false,
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
            onToggle={() =>
              onToggleStatus(row.original.id, !row.original.is_active)
            }
            isToggling={isToggling}
          />
        ),
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded-full" />,
        },
      },
      ...(canUpdateProxies || canDeleteProxies
        ? [
            {
              id: 'actions',
              header: '',
              cell: ({ row }: { row: { original: Proxy } }) => (
                <ProxyActionsCell
                  proxy={row.original}
                  canEdit={canUpdateProxies}
                  canDelete={canDeleteProxies}
                  onEdit={onEdit}
                  onDelete={onDelete}
                />
              ),
              enableSorting: false,
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
    [canUpdateProxies, canDeleteProxies, onEdit, onDelete, onToggleStatus, isToggling]
  );

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  return (
    <DataGrid
      table={table}
      recordCount={data.length}
      isLoading={isLoading}
      loadingMode="skeleton"
      emptyMessage="No proxies found. Create your first proxy to get started."
    >
      <DataGridTable />
    </DataGrid>
  );
}
