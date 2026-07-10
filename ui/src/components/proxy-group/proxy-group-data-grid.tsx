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
} from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table';
import { Layers, Pencil, Plus, Trash2 } from 'lucide-react';
import { useMemo } from 'react';

import type { ProxyGroup } from '@/types/proxy-group';

interface ProxyGroupDataGridProps {
  data: ProxyGroup[];
  isLoading: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  onEdit: (group: ProxyGroup) => void;
  onDelete: (group: ProxyGroup) => void;
  onRowClick?: (group: ProxyGroup) => void;
}

export function ProxyGroupDataGrid({
  data,
  isLoading,
  canUpdate,
  canDelete,
  onEdit,
  onDelete,
  onRowClick,
}: ProxyGroupDataGridProps) {
  const columns = useMemo<ColumnDef<ProxyGroup>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Name" />,
        cell: ({ row }) => <span className="font-medium">{row.getValue('name')}</span>,
        minSize: 140,
        maxSize: 240,
        meta: {
          skeleton: <Skeleton className="h-5 w-32" />,
        },
      },
      {
        accessorKey: 'base_domain',
        header: 'Base Domain',
        cell: ({ row }) => {
          const baseDomain = row.original.base_domain;
          return baseDomain ? (
            <Badge variant="secondary" className="font-mono text-xs">
              {baseDomain}
            </Badge>
          ) : (
            <span className="text-sm text-muted-foreground">Not set</span>
          );
        },
        enableSorting: false,
        minSize: 150,
        maxSize: 240,
        meta: {
          skeleton: <Skeleton className="h-6 w-32 rounded" />,
        },
      },
      {
        accessorKey: 'member_count',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Members" />,
        cell: ({ row }) => (
          <Badge variant="outline" className="text-xs">
            {row.original.member_count} {row.original.member_count === 1 ? 'proxy' : 'proxies'}
          </Badge>
        ),
        minSize: 100,
        maxSize: 140,
        meta: {
          skeleton: <Skeleton className="h-6 w-20 rounded" />,
        },
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <div className="flex items-center justify-end gap-1">
            {canUpdate && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        onEdit(row.original);
                      }}
                    />
                  }
                >
                  <Pencil className="size-4" />
                  <span className="sr-only">Edit proxy group</span>
                </TooltipTrigger>
                <TooltipContent>Edit</TooltipContent>
              </Tooltip>
            )}
            {canDelete && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={(e) => {
                        e.stopPropagation();
                        onDelete(row.original);
                      }}
                    />
                  }
                >
                  <Trash2 className="size-4" />
                  <span className="sr-only">Delete proxy group</span>
                </TooltipTrigger>
                <TooltipContent>Delete</TooltipContent>
              </Tooltip>
            )}
          </div>
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
    ],
    [canUpdate, canDelete, onEdit, onDelete],
  );

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  });

  return (
    <DataGrid
      table={table}
      recordCount={data.length}
      isLoading={isLoading}
      loadingMode="skeleton"
      onRowClick={onRowClick}
      emptyMessage={
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <div className="rounded bg-muted p-4">
            <Layers className="size-8 text-muted-foreground" />
          </div>
          <h3 className="mt-4 text-base font-medium">No proxy groups yet</h3>
          <p className="mt-1.5 text-sm text-muted-foreground max-w-[280px]">
            Create a group to share a base domain and settings across a set of proxies.
          </p>
          <Button size="sm" className="mt-4" render={<Link to="/proxy-groups/new" />}>
            <Plus className="size-4" />
            New Proxy Group
          </Button>
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
