import {
  Badge,
  Button,
  Checkbox,
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
  Skeleton,
} from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type OnChangeFn,
  type PaginationState,
  type RowSelectionState,
  useReactTable,
} from '@tanstack/react-table';
import { ExternalLink, Globe, Network, Plus } from 'lucide-react';
import { useMemo } from 'react';

import type { ProxyConfig } from '@/types/proxy';

import {
  ProxyAclCell,
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
  onDuplicate?: (proxy: ProxyConfig) => void;
  onToggleStatus: (id: number, enable: boolean) => void;
  isToggling: boolean;
  onRowClick?: (proxy: ProxyConfig) => void;
  // Pagination props
  pageCount: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  total: number;
  // Selection props
  rowSelection: RowSelectionState;
  onRowSelectionChange: OnChangeFn<RowSelectionState>;
}

export function ProxyDataGrid({
  data,
  isLoading,
  canUpdateProxies,
  canDeleteProxies,
  onEdit,
  onDelete,
  onDuplicate,
  onToggleStatus,
  isToggling,
  onRowClick,
  pageCount,
  pagination,
  onPaginationChange,
  total,
  rowSelection,
  onRowSelectionChange,
}: ProxyDataGridProps) {
  const columns = useMemo<ColumnDef<ProxyConfig>[]>(
    () => [
      {
        id: 'select',
        header: ({ table }) => (
          // Stop propagation so toggling selection doesn't trigger the row's
          // onRowClick (which navigates to the proxy detail page).
          <span className="flex" onClick={(e) => e.stopPropagation()}>
            <Checkbox
              checked={table.getIsAllPageRowsSelected()}
              indeterminate={table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()}
              onCheckedChange={(v) => table.toggleAllPageRowsSelected(!!v)}
              aria-label="Select all"
            />
          </span>
        ),
        cell: ({ row }) => (
          <span className="flex" onClick={(e) => e.stopPropagation()}>
            <Checkbox
              checked={row.getIsSelected()}
              onCheckedChange={(v) => row.toggleSelected(!!v)}
              aria-label="Select row"
            />
          </span>
        ),
        enableSorting: false,
        size: 40,
        minSize: 40,
        maxSize: 40,
        enableResizing: false,
        meta: { skeleton: <Skeleton className="size-4" /> },
      },
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
          skeleton: <Skeleton className="h-6 w-40 rounded" />,
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
          skeleton: <Skeleton className="h-6 w-28 rounded" />,
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
          skeleton: <Skeleton className="h-6 w-16 rounded" />,
        },
      },
      {
        id: 'acl',
        accessorKey: 'acl_group_count',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Access" />,
        cell: ({ row }) => (
          <ProxyAclCell count={row.original.acl_group_count} names={row.original.acl_group_names} />
        ),
        enableSorting: false,
        minSize: 100,
        maxSize: 160,
        meta: {
          skeleton: <Skeleton className="h-5 w-20" />,
        },
      },
      ...(canUpdateProxies || canDeleteProxies || onDuplicate
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
                  onDuplicate={onDuplicate}
                />
              ),
              enableSorting: false,
              minSize: 80,
              maxSize: 120,
              meta: {
                skeleton: (
                  <div className="flex justify-end">
                    <Skeleton className="size-8" />
                  </div>
                ),
              },
            },
          ]
        : []),
    ],
    [canUpdateProxies, canDeleteProxies, onDuplicate, onEdit, onDelete, onToggleStatus, isToggling],
  );

  const table = useReactTable({
    data,
    columns,
    pageCount,
    state: {
      pagination,
      rowSelection,
    },
    enableRowSelection: true,
    getRowId: (row) => String(row.id),
    onRowSelectionChange,
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
      onRowClick={onRowClick}
      emptyMessage={
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <div className="rounded bg-muted p-4">
            <Globe className="size-8 text-muted-foreground" />
          </div>
          <h3 className="mt-4 text-base font-medium">No proxies yet</h3>
          <p className="mt-1.5 text-sm text-muted-foreground max-w-[260px]">
            Create your first proxy to start routing traffic through Waygates.
          </p>
          <div className="mt-4 flex gap-2">
            <Button size="sm" render={<Link to="/proxies/new" />}>
              <Plus className="size-4" />
              HTTP Proxy
            </Button>
            <Button size="sm" variant="outline" render={<Link to="/proxies/tcp-udp/new" />}>
              <Network className="size-4" />
              TCP/UDP Proxy
            </Button>
          </div>
        </div>
      }
    >
      <DataGridContainer>
        <DataGridTable />
        <div className="border-t px-4 py-2">
          <DataGridPagination
            sizes={[10, 20, 50]}
            className="[&_[data-slot=select-trigger]]:w-fit"
          />
        </div>
      </DataGridContainer>
    </DataGrid>
  );
}
