import {
  Badge,
  Button,
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
import { format, formatDistanceToNow } from 'date-fns';
import { Eye } from 'lucide-react';
import { useCallback, useMemo, useState } from 'react';
import type { AuditLog } from '@/types/audit';
import { AuditLogDetailSheet } from './audit-log-detail-sheet';
import { ActionBadge, StatusBadge } from './cells';

interface AuditDataGridProps {
  data: AuditLog[];
  isLoading: boolean;
  pageCount: number;
  pagination: PaginationState;
  onPaginationChange: (pagination: PaginationState) => void;
  total: number;
}

export function AuditDataGrid({
  data,
  isLoading,
  pageCount,
  pagination,
  onPaginationChange,
  total,
}: AuditDataGridProps) {
  const [selectedLogId, setSelectedLogId] = useState<number | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  const handleViewLog = useCallback((log: AuditLog) => {
    setSelectedLogId(log.id);
    setSheetOpen(true);
  }, []);

  const columns = useMemo<ColumnDef<AuditLog>[]>(
    () => [
      {
        accessorKey: 'created_at',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Time" />,
        cell: ({ row }) => {
          const date = new Date(row.getValue('created_at') as string);
          return (
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="text-muted-foreground text-sm cursor-default">
                  {formatDistanceToNow(date, { addSuffix: true })}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <p>{format(date, 'dd/MM/yyyy HH:mm:ss xxx')}</p>
              </TooltipContent>
            </Tooltip>
          );
        },
        meta: {
          skeleton: <Skeleton className="h-5 w-24" />,
        },
      },
      {
        accessorKey: 'action',
        header: 'Action',
        cell: ({ row }) => <ActionBadge action={row.getValue('action')} />,
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-6 w-24 rounded-full" />,
        },
      },
      {
        id: 'resource',
        header: 'Resource',
        cell: ({ row }) => {
          const log = row.original;
          if (!log.resource_type && !log.resource_name) {
            return <span className="text-muted-foreground">-</span>;
          }
          return (
            <div className="flex items-center gap-2">
              {log.resource_type && (
                <Badge variant="outline" className="text-xs capitalize">
                  {log.resource_type}
                </Badge>
              )}
              {log.resource_name && <span className="font-medium">{log.resource_name}</span>}
            </div>
          );
        },
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-5 w-32" />,
        },
      },
      {
        id: 'user',
        header: 'User',
        cell: ({ row }) => {
          const log = row.original;
          if (log.user) {
            return <span className="font-medium">{log.user.username}</span>;
          }
          if (log.user_id) {
            return <span className="text-muted-foreground">User #{log.user_id}</span>;
          }
          return <span className="text-muted-foreground italic">System</span>;
        },
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-5 w-20" />,
        },
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) => <StatusBadge status={row.getValue('status')} />,
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-6 w-20 rounded-full" />,
        },
      },
      {
        accessorKey: 'ip_address',
        header: 'IP',
        cell: ({ row }) => {
          const ip = row.getValue('ip_address') as string | null;
          return ip ? (
            <code className="text-xs text-muted-foreground">{ip}</code>
          ) : (
            <span className="text-muted-foreground">-</span>
          );
        },
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="h-5 w-28" />,
        },
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <div className="flex justify-end">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="size-8 p-0"
                  onClick={() => handleViewLog(row.original)}
                >
                  <Eye className="size-4" />
                  <span className="sr-only">View details</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>View details</p>
              </TooltipContent>
            </Tooltip>
          </div>
        ),
        enableSorting: false,
        meta: {
          skeleton: <Skeleton className="size-8 rounded" />,
        },
      },
    ],
    [handleViewLog],
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
    <>
      <DataGrid
        table={table}
        recordCount={total}
        isLoading={isLoading}
        loadingMode="skeleton"
        emptyMessage="No audit logs found."
      >
        <DataGridContainer>
          <DataGridTable />
          <div className="border-t px-4 py-2">
            <DataGridPagination sizes={[10, 20, 50, 100]} />
          </div>
        </DataGridContainer>
      </DataGrid>

      <AuditLogDetailSheet logId={selectedLogId} open={sheetOpen} onOpenChange={setSheetOpen} />
    </>
  );
}
