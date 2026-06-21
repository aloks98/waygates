import {
  Badge,
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
  Skeleton,
} from '@e412/rnui-react';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  type OnChangeFn,
  type PaginationState,
  useReactTable,
} from '@tanstack/react-table';
import { format } from 'date-fns';
import { useMemo } from 'react';

import type { AuditLog } from '@/types/audit';

import { getActionMeta } from './activity-actions';

export interface ActivityTableProps {
  logs: AuditLog[];
  isLoading: boolean;
  total: number;
  pageCount: number;
  pagination: PaginationState;
  onPaginationChange: OnChangeFn<PaginationState>;
  onSelect: (id: number) => void;
}

export function ActivityTable({
  logs,
  isLoading,
  total,
  pageCount,
  pagination,
  onPaginationChange,
  onSelect,
}: ActivityTableProps) {
  const columns = useMemo<ColumnDef<AuditLog>[]>(
    () => [
      {
        accessorKey: 'created_at',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Time" />,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {format(new Date(row.original.created_at), 'PP p')}
          </span>
        ),
        meta: { skeleton: <Skeleton className="h-5 w-32" /> },
      },
      {
        accessorKey: 'action',
        header: 'Action',
        cell: ({ row }) => {
          const meta = getActionMeta(row.original.action);
          const Icon = meta.icon;
          return (
            <span className="flex items-center gap-2">
              <Icon className="size-4 text-muted-foreground" />
              <span className="text-sm">{meta.label}</span>
            </span>
          );
        },
        enableSorting: false,
        meta: { skeleton: <Skeleton className="h-5 w-40" /> },
      },
      {
        accessorKey: 'resource_name',
        header: 'Resource',
        cell: ({ row }) => <span className="text-sm">{row.original.resource_name ?? '—'}</span>,
        enableSorting: false,
        meta: { skeleton: <Skeleton className="h-5 w-24" /> },
      },
      {
        id: 'user',
        header: 'User',
        cell: ({ row }) => (
          <span className="text-sm">{row.original.user?.username ?? 'system'}</span>
        ),
        enableSorting: false,
        meta: { skeleton: <Skeleton className="h-5 w-20" /> },
      },
      {
        accessorKey: 'ip_address',
        header: 'IP',
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.ip_address ?? '—'}
          </span>
        ),
        enableSorting: false,
        meta: { skeleton: <Skeleton className="h-5 w-24" /> },
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) => (
          <Badge variant={row.original.status === 'failure' ? 'destructive' : 'secondary'}>
            {row.original.status}
          </Badge>
        ),
        enableSorting: false,
        meta: { skeleton: <Skeleton className="h-6 w-16 rounded" /> },
      },
    ],
    [],
  );

  const table = useReactTable({
    data: logs,
    columns,
    pageCount,
    state: { pagination },
    onPaginationChange,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    manualPagination: true,
    getRowId: (row) => String(row.id),
  });

  return (
    <DataGrid
      table={table}
      recordCount={total}
      isLoading={isLoading}
      loadingMode="skeleton"
      emptyMessage="No activity found."
      onRowClick={(row: AuditLog) => onSelect(row.id)}
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
