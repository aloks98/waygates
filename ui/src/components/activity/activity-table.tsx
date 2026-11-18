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

import {
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
} from '@/components/ui/data-grid';
import { Skeleton } from '@/components/ui/skeleton';
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
        enableSorting: false,
        header: ({ column }) => <DataGridColumnHeader column={column} title="When" />,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {format(new Date(row.original.created_at), 'PP p')}
          </span>
        ),
        meta: { skeleton: <Skeleton className="h-5 w-32" /> },
      },
      {
        accessorKey: 'action',
        header: 'Change',
        cell: ({ row }) => {
          const meta = getActionMeta(row.original.action);
          const Icon = meta.icon;
          return (
            <span className="flex items-center gap-2">
              <Icon className="size-4 text-muted-foreground" />
              <span className="text-sm">
                {meta.label}
                {row.original.status === 'failure' && (
                  <span className="ml-2 text-destructive">Failed</span>
                )}
              </span>
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
        header: 'By',
        cell: ({ row }) => (
          <span className="text-sm">{row.original.user?.username ?? 'system'}</span>
        ),
        enableSorting: false,
        meta: { skeleton: <Skeleton className="h-5 w-20" /> },
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
