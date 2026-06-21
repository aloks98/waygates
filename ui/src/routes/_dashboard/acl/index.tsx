import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
  Input,
  Skeleton,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/rnui-react';
import { useNavigate } from '@tanstack/react-router';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type PaginationState,
  useReactTable,
} from '@tanstack/react-table';
import { format, formatDistanceToNow } from 'date-fns';
import { Plus, Search, Shield, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { getModeLabel, groupAuthMethods } from '@/components/acl/access-labels';
import { ACLGroupFormModal } from '@/components/acl/acl-group-form-modal';
import { useACLGroups, useDeleteACLGroup } from '@/hooks';
import type { ACLGroup } from '@/types/acl';

export function ACLGroupsPage() {
  const navigate = useNavigate();
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  });
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [deletingGroup, setDeletingGroup] = useState<ACLGroup | null>(null);

  // Debounce search input
  useEffect(() => {
    if (searchDebounceRef.current) {
      clearTimeout(searchDebounceRef.current);
    }
    searchDebounceRef.current = setTimeout(() => {
      setDebouncedSearch(search);
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    }, 300);

    return () => {
      if (searchDebounceRef.current) {
        clearTimeout(searchDebounceRef.current);
      }
    };
  }, [search]);

  const { groups, total, totalPages, isLoading } = useACLGroups({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    search: debouncedSearch || undefined,
  });

  const { deleteGroup, isDeleting } = useDeleteACLGroup();

  const handleRowClick = useCallback(
    (group: ACLGroup) => {
      navigate({ to: '/dashboard/access/$groupId', params: { groupId: String(group.id) } });
    },
    [navigate],
  );

  const handleDelete = async () => {
    if (!deletingGroup) return;
    await deleteGroup(deletingGroup.id);
    setDeletingGroup(null);
  };

  const columns = useMemo<ColumnDef<ACLGroup>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Shield className="size-4 text-muted-foreground" />
            <span className="font-medium">{row.getValue('name')}</span>
          </div>
        ),
        minSize: 140,
        maxSize: 240,
        meta: {
          skeleton: <Skeleton className="h-5 w-32" />,
        },
      },
      {
        accessorKey: 'combination_mode',
        header: 'Mode',
        cell: ({ row }) => {
          return (
            <Badge variant="secondary">
              {getModeLabel(row.getValue('combination_mode') as string)}
            </Badge>
          );
        },
        enableSorting: false,
        minSize: 100,
        maxSize: 140,
        meta: {
          skeleton: <Skeleton className="h-6 w-24 rounded" />,
        },
      },
      {
        id: 'ip_rules',
        header: 'IP Rules',
        cell: ({ row }) => {
          const count = row.original.ip_rules?.length ?? 0;
          return <span className="text-muted-foreground">{count > 0 ? count : '-'}</span>;
        },
        enableSorting: false,
        minSize: 60,
        maxSize: 100,
        meta: {
          skeleton: <Skeleton className="h-5 w-8" />,
        },
      },
      {
        id: 'basic_auth',
        header: 'Methods',
        cell: ({ row }) => {
          const methods = groupAuthMethods(row.original);
          if (methods.length === 0) {
            return <span className="text-sm text-muted-foreground">None</span>;
          }
          return (
            <div className="flex flex-wrap gap-1">
              {methods.map((m) => (
                <Badge key={m.key} variant="secondary" className="text-xs">
                  {m.label}
                </Badge>
              ))}
            </div>
          );
        },
        enableSorting: false,
        minSize: 120,
        maxSize: 220,
        meta: {
          skeleton: <Skeleton className="h-5 w-24" />,
        },
      },
      {
        accessorKey: 'created_at',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Created" />,
        cell: ({ row }) => {
          const date = new Date(row.getValue('created_at') as string);
          return (
            <Tooltip>
              <TooltipTrigger
                render={<span className="text-muted-foreground text-sm cursor-default" />}
              >
                {formatDistanceToNow(date, { addSuffix: true })}
              </TooltipTrigger>
              <TooltipContent>
                <p>{format(date, 'dd/MM/yyyy HH:mm:ss')}</p>
              </TooltipContent>
            </Tooltip>
          );
        },
        minSize: 100,
        maxSize: 160,
        meta: {
          skeleton: <Skeleton className="h-5 w-24" />,
        },
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <div className="flex justify-end gap-1">
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    size="sm"
                    className="size-8 p-0 text-destructive hover:text-destructive"
                    onClick={(e) => {
                      e.stopPropagation();
                      setDeletingGroup(row.original);
                    }}
                  />
                }
              >
                <Trash2 className="size-4" />
                <span className="sr-only">Delete</span>
              </TooltipTrigger>
              <TooltipContent>Delete group</TooltipContent>
            </Tooltip>
          </div>
        ),
        enableSorting: false,
        minSize: 60,
        maxSize: 80,
        meta: {
          skeleton: (
            <div className="flex justify-end gap-1">
              <Skeleton className="size-8" />
            </div>
          ),
        },
      },
    ],
    [],
  );

  const table = useReactTable({
    data: groups,
    columns,
    pageCount: totalPages,
    state: {
      pagination,
    },
    onPaginationChange: (updater) => {
      const newPagination = typeof updater === 'function' ? updater(pagination) : updater;
      setPagination(newPagination);
    },
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    manualPagination: true,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Access Groups</h1>
        <Button onClick={() => setCreateModalOpen(true)}>
          <Plus className="size-4" />
          New Group
        </Button>
      </div>

      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by name..."
          aria-label="Search access groups"
          className="pl-9"
        />
      </div>

      <DataGrid
        table={table}
        recordCount={total}
        isLoading={isLoading}
        loadingMode="skeleton"
        emptyMessage={
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <div className="rounded bg-muted p-4">
              <Shield className="size-8 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-base font-medium">No access groups yet</h3>
            <p className="mt-1.5 text-sm text-muted-foreground max-w-[280px]">
              Create a group to protect your proxies with IP rules, passwords, or OAuth.
            </p>
            <Button size="sm" className="mt-4" onClick={() => setCreateModalOpen(true)}>
              <Plus className="size-4" />
              New Group
            </Button>
          </div>
        }
        onRowClick={handleRowClick}
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

      <ACLGroupFormModal open={createModalOpen} onOpenChange={setCreateModalOpen} mode="create" />

      <AlertDialog open={!!deletingGroup} onOpenChange={(open) => !open && setDeletingGroup(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete access group</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deletingGroup?.name}</strong>? This will
              remove all associated IP rules, authentication settings, and provider configurations.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
