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
  Tabs,
  TabsList,
  TabsTrigger,
} from '@e412/titanium';
import { useNavigate } from '@tanstack/react-router';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type PaginationState,
  useReactTable,
} from '@tanstack/react-table';
import { Network, Plus, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { L4ProtocolBadge, ProxyStatusBadge } from '@/components/proxy/cells';
import { useL4Proxies, useL4ProxyStats } from '@/hooks/use-l4-proxies';
import { usePermissions } from '@/hooks/use-permissions';
import type { L4Proxy } from '@/types/l4-proxy';

type ProtocolFilter = 'all' | 'tcp' | 'udp';

export function L4ProxiesListPage() {
  const navigate = useNavigate();
  const { canCreateProxies, canUpdateProxies } = usePermissions();

  // Filter state
  const [protocolFilter, setProtocolFilter] = useState<ProtocolFilter>('all');
  const prevFilterRef = useRef(protocolFilter);

  // Pagination state
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  });

  // Search state
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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

  // Reset pagination when filter changes
  useEffect(() => {
    if (prevFilterRef.current !== protocolFilter) {
      prevFilterRef.current = protocolFilter;
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    }
  });

  // Fetch L4 proxies
  const params = useMemo(() => {
    const p: {
      page: number;
      limit: number;
      search?: string;
      protocol?: string;
    } = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    };
    if (debouncedSearch) {
      p.search = debouncedSearch;
    }
    if (protocolFilter !== 'all') {
      p.protocol = protocolFilter;
    }
    return p;
  }, [pagination.pageIndex, pagination.pageSize, debouncedSearch, protocolFilter]);

  const { proxies, total, totalPages, isLoading, toggleActive, remove, isToggling, isDeleting } =
    useL4Proxies(params);

  // Delete state
  const [deletingProxy, setDeletingProxy] = useState<L4Proxy | null>(null);

  const handleRowClick = useCallback(
    (proxy: L4Proxy) => {
      navigate({
        to: '/dashboard/l4-proxies/$l4ProxyId',
        params: { l4ProxyId: String(proxy.id) },
      });
    },
    [navigate],
  );

  const handleToggleStatus = useCallback(
    async (proxy: L4Proxy) => {
      await toggleActive(proxy.id);
    },
    [toggleActive],
  );

  const handleDelete = async () => {
    if (!deletingProxy) return;
    await remove(deletingProxy.id);
    setDeletingProxy(null);
  };

  // Fetch stats for counts
  const { stats } = useL4ProxyStats();
  const counts = useMemo(() => {
    return {
      all: stats?.total_proxies ?? 0,
      tcp: stats?.tcp_proxies ?? 0,
      udp: stats?.udp_proxies ?? 0,
    };
  }, [stats]);

  // DataGrid columns
  const columns = useMemo<ColumnDef<L4Proxy>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Network className="size-4 text-muted-foreground" />
            <span className="font-medium">{row.getValue('name')}</span>
          </div>
        ),
        minSize: 150,
        maxSize: 250,
        meta: {
          skeleton: <Skeleton className="h-5 w-32" />,
        },
      },
      {
        accessorKey: 'protocol',
        header: 'Protocol',
        cell: ({ row }) => <L4ProtocolBadge protocol={row.original.protocol} />,
        enableSorting: false,
        minSize: 80,
        maxSize: 120,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded" />,
        },
      },
      {
        accessorKey: 'listen_port',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Listen Port" />,
        cell: ({ row }) => (
          <Badge variant="secondary" className="font-mono text-xs">
            :{row.getValue('listen_port')}
          </Badge>
        ),
        minSize: 100,
        maxSize: 140,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded" />,
        },
      },
      {
        id: 'routes',
        header: 'Routes',
        cell: ({ row }) => {
          const routeCount = row.original.routes?.length ?? 0;
          return (
            <Badge variant="outline" className="text-xs">
              {routeCount} route{routeCount !== 1 ? 's' : ''}
            </Badge>
          );
        },
        enableSorting: false,
        minSize: 80,
        maxSize: 120,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded" />,
        },
      },
      {
        accessorKey: 'is_active',
        header: 'Status',
        cell: ({ row }) => (
          <ProxyStatusBadge
            isActive={row.original.is_active}
            canToggle={canUpdateProxies}
            onToggle={() => handleToggleStatus(row.original)}
            isToggling={isToggling}
          />
        ),
        enableSorting: false,
        minSize: 80,
        maxSize: 120,
        meta: {
          skeleton: <Skeleton className="h-6 w-16 rounded" />,
        },
      },
    ],
    [canUpdateProxies, handleToggleStatus, isToggling],
  );

  const table = useReactTable({
    data: proxies,
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
        <h1 className="text-2xl font-bold">L4 Proxies</h1>
        {canCreateProxies && (
          <Button onClick={() => navigate({ to: '/dashboard/l4-proxies/new' })}>
            <Plus className="size-4" />
            Add L4 Proxy
          </Button>
        )}
      </div>

      <div className="flex flex-col gap-4">
        {/* Filter tabs */}
        <Tabs
          value={protocolFilter}
          onValueChange={(value) => setProtocolFilter(value as ProtocolFilter)}
        >
          <TabsList>
            <TabsTrigger value="all" className="gap-2">
              All
              <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {counts.all}
              </span>
            </TabsTrigger>
            <TabsTrigger value="tcp" className="gap-2">
              TCP
              <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {counts.tcp}
              </span>
            </TabsTrigger>
            <TabsTrigger value="udp" className="gap-2">
              UDP
              <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {counts.udp}
              </span>
            </TabsTrigger>
          </TabsList>
        </Tabs>

        {/* Search input */}
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search L4 proxies..."
            className="pl-9"
          />
        </div>
      </div>

      <DataGrid
        table={table}
        recordCount={total}
        isLoading={isLoading}
        loadingMode="skeleton"
        emptyMessage="No L4 proxies found. Create your first L4 proxy to get started."
        onRowClick={handleRowClick}
      >
        <DataGridContainer>
          <DataGridTable />
          <div className="border-t px-4 py-2">
            <DataGridPagination sizes={[10, 20, 50]} />
          </div>
        </DataGridContainer>
      </DataGrid>

      <AlertDialog open={!!deletingProxy} onOpenChange={(open) => !open && setDeletingProxy(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete L4 Proxy</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deletingProxy?.name}</strong>? This action
              cannot be undone.
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
