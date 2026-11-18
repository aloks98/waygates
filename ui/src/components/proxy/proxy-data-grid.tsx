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

import { buttonVariants } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
} from '@/components/ui/data-grid';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import { Skeleton } from '@/components/ui/skeleton';
import type { ProxyConfig } from '@/types/proxy';

import { ProxyActionsCell, ProxyStatusBadge, ProxyTargetCell } from './cells';

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
        header: ({ column }) => <DataGridColumnHeader column={column} title="Proxy" />,
        cell: ({ row }) => (
          <div className="flex min-w-44 flex-col gap-1">
            <Link
              to="/proxies/$proxyId"
              params={{ proxyId: String(row.original.id) }}
              className="font-medium hover:underline"
            >
              {row.original.name}
            </Link>
            <a
              href={`${row.original.ssl_enabled ? 'https' : 'http'}://${row.original.hostname}`}
              target="_blank"
              rel="noopener noreferrer"
              className="flex w-fit items-center gap-1.5 font-mono text-xs text-muted-foreground hover:text-accent-foreground"
            >
              {row.original.hostname}
              <ExternalLink className="size-3" />
            </a>
          </div>
        ),
      },
      {
        id: 'target',
        header: 'Destination',
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex flex-col gap-1">
            <ProxyTargetCell proxy={row.original} />
            <span className="text-xs text-muted-foreground">
              {
                { reverse_proxy: 'Reverse proxy', redirect: 'Redirect', static: 'Static files' }[
                  row.original.type
                ]
              }
            </span>
          </div>
        ),
      },
      {
        id: 'security',
        header: 'Security & access',
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex flex-col gap-1">
            <span className="text-sm">{row.original.ssl_enabled ? 'HTTPS' : 'HTTP'}</span>
            <span className="text-xs text-muted-foreground">
              {row.original.acl_group_count
                ? row.original.acl_group_names?.join(', ') ||
                  `${row.original.acl_group_count} access groups`
                : 'Public'}
            </span>
          </div>
        ),
      },
      {
        accessorKey: 'is_active',
        header: 'Status',
        enableSorting: false,
        cell: ({ row }) => (
          <ProxyStatusBadge
            isActive={row.original.is_active}
            canToggle={canUpdateProxies}
            onToggle={() => onToggleStatus(row.original.id, !row.original.is_active)}
            isToggling={isToggling}
          />
        ),
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
        <Empty className="py-12">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Globe />
            </EmptyMedia>
            <EmptyTitle>No proxies yet</EmptyTitle>
            <EmptyDescription>
              Create your first proxy to start routing traffic through Waygates.
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <div className="flex flex-wrap justify-center gap-2">
              <Link to="/proxies/new" className={buttonVariants({ size: 'sm' })}>
                <Plus data-icon="inline-start" />
                HTTP Proxy
              </Link>
              <Link
                to="/proxies/tcp-udp/new"
                className={buttonVariants({ size: 'sm', variant: 'outline' })}
              >
                <Network data-icon="inline-start" />
                TCP/UDP Proxy
              </Link>
            </div>
          </EmptyContent>
        </Empty>
      }
    >
      <DataGridContainer className="flex min-h-0 flex-1 flex-col">
        <DataGridTable scrollable />
        <div className="shrink-0 border-t px-4 py-2">
          <DataGridPagination
            sizes={[10, 20, 50]}
            className="[&_[data-slot=select-trigger]]:w-fit"
          />
        </div>
      </DataGridContainer>
    </DataGrid>
  );
}
