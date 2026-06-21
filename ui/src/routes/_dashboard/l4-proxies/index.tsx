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
  Checkbox,
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
  type OnChangeFn,
  type PaginationState,
  type RowSelectionState,
  useReactTable,
} from '@tanstack/react-table';
import { Copy, Download, Network, Plus, Search, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';

import { ProxiesTabs } from '@/components/layout/proxies-tabs';
import { L4ProtocolBadge, ProxyStatusBadge } from '@/components/proxy/cells';
import { ProxyBulkBar } from '@/components/proxy/proxy-bulk-bar';
import { useL4Proxies, useL4ProxyStats } from '@/hooks/use-l4-proxies';
import { usePermissions } from '@/hooks/use-permissions';
import { api } from '@/lib/api';
import type { L4Export } from '@/lib/l4-export';
import { downloadJson } from '@/lib/proxy-export';
import type { ApiResponse } from '@/types/api';
import type { L4Proxy } from '@/types/l4-proxy';

type ProtocolFilter = 'all' | 'tcp' | 'udp';

export function L4ProxiesListPage() {
  const navigate = useNavigate();
  const { canCreateProxies, canUpdateProxies, canDeleteProxies } = usePermissions();

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

  // Row selection state — owned by the page, cleared on page/search/protocol change
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  useEffect(() => {
    setRowSelection({});
  }, [pagination, debouncedSearch, protocolFilter]);

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

  const {
    proxies,
    total,
    totalPages,
    isLoading,
    toggleActive,
    remove,
    isToggling,
    isDeleting,
    bulkSetActive,
    bulkRemove,
    isBulkRunning,
  } = useL4Proxies(params);

  // Derived selection
  const selectedIds = useMemo(() => Object.keys(rowSelection).map(Number), [rowSelection]);

  // Delete state
  const [deletingProxy, setDeletingProxy] = useState<L4Proxy | null>(null);
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);

  // Export state
  const [isExporting, setIsExporting] = useState(false);

  const handleExport = useCallback(async () => {
    setIsExporting(true);
    try {
      const searchParams: Record<string, string> = {};
      if (selectedIds.length > 0) {
        searchParams.ids = selectedIds.join(',');
      } else {
        if (debouncedSearch) searchParams.search = debouncedSearch;
        if (protocolFilter !== 'all') searchParams.protocol = protocolFilter;
      }
      const response = await api
        .get('l4-proxies/export', { searchParams })
        .json<ApiResponse<L4Export[]>>();
      const data = response.data ?? [];
      downloadJson(`waygates-l4-proxies-${data.length}.json`, data);
      toast.success(`Exported ${data.length} ${data.length === 1 ? 'proxy' : 'proxies'}`);
    } catch {
      toast.error('Export failed');
    } finally {
      setIsExporting(false);
    }
  }, [selectedIds, debouncedSearch, protocolFilter]);

  const handleRowClick = useCallback(
    (proxy: L4Proxy) => {
      navigate({
        to: '/dashboard/proxies/tcp-udp/$l4ProxyId',
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

  const handleDuplicate = useCallback(
    (p: L4Proxy) => {
      navigate({ to: '/dashboard/proxies/tcp-udp/new', search: { duplicate: p.id } });
    },
    [navigate],
  );

  const handleBulkEnable = useCallback(async () => {
    const s = await bulkSetActive(selectedIds, true);
    if (s.failed > 0) {
      toast.error(`Enabled ${s.succeeded}, ${s.failed} failed`);
    } else {
      toast.success(`Enabled ${s.succeeded}`);
    }
    setRowSelection({});
  }, [bulkSetActive, selectedIds]);

  const handleBulkDisable = useCallback(async () => {
    const s = await bulkSetActive(selectedIds, false);
    if (s.failed > 0) {
      toast.error(`Disabled ${s.succeeded}, ${s.failed} failed`);
    } else {
      toast.success(`Disabled ${s.succeeded}`);
    }
    setRowSelection({});
  }, [bulkSetActive, selectedIds]);

  const handleBulkDelete = useCallback(async () => {
    setBulkDeleteOpen(false);
    const s = await bulkRemove(selectedIds);
    if (s.failed > 0) {
      toast.error(`Deleted ${s.succeeded}, ${s.failed} failed`);
    } else {
      toast.success(`Deleted ${s.succeeded} ${s.succeeded === 1 ? 'proxy' : 'proxies'}`);
    }
    setRowSelection({});
  }, [bulkRemove, selectedIds]);

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
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            indeterminate={table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()}
            onCheckedChange={(v) => table.toggleAllPageRowsSelected(!!v)}
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(v) => row.toggleSelected(!!v)}
            aria-label="Select row"
          />
        ),
        enableSorting: false,
        meta: { skeleton: <Skeleton className="size-4" /> },
      },
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
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <div className="flex items-center gap-1">
            {canCreateProxies && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDuplicate(row.original);
                      }}
                    />
                  }
                >
                  <Copy className="size-4" />
                  <span className="sr-only">Duplicate proxy</span>
                </TooltipTrigger>
                <TooltipContent>Duplicate</TooltipContent>
              </Tooltip>
            )}
            {canDeleteProxies && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeletingProxy(row.original);
                      }}
                    />
                  }
                >
                  <Trash2 className="size-4 text-destructive" />
                  <span className="sr-only">Delete proxy</span>
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
          skeleton: <Skeleton className="h-8 w-16 rounded" />,
        },
      },
    ],
    [
      canCreateProxies,
      canDeleteProxies,
      canUpdateProxies,
      handleDuplicate,
      handleToggleStatus,
      isToggling,
    ],
  );

  const table = useReactTable({
    data: proxies,
    columns,
    pageCount: totalPages,
    state: {
      pagination,
      rowSelection,
    },
    enableRowSelection: true,
    getRowId: (row) => String(row.id),
    onRowSelectionChange: setRowSelection as OnChangeFn<RowSelectionState>,
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
      <div className="mb-4">
        <ProxiesTabs active="tcp-udp" />
      </div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">TCP/UDP Proxies</h1>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleExport} disabled={isExporting}>
            <Download className="size-4" />
            {isExporting ? 'Exporting...' : 'Export'}
          </Button>
          {canCreateProxies && (
            <Button onClick={() => navigate({ to: '/dashboard/proxies/tcp-udp/new' })}>
              <Plus className="size-4" />
              Add TCP/UDP Proxy
            </Button>
          )}
        </div>
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

      <ProxyBulkBar
        count={selectedIds.length}
        onEnable={handleBulkEnable}
        onDisable={handleBulkDisable}
        onDelete={() => setBulkDeleteOpen(true)}
        onClear={() => setRowSelection({})}
        running={isBulkRunning}
      />

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
            <DataGridPagination
              sizes={[10, 20, 50]}
              className="[&_[data-slot=select-trigger]]:w-fit"
            />
          </div>
        </DataGridContainer>
      </DataGrid>

      <AlertDialog open={bulkDeleteOpen} onOpenChange={setBulkDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete {selectedIds.length} {selectedIds.length === 1 ? 'Proxy' : 'Proxies'}
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{selectedIds.length}</strong>{' '}
              {selectedIds.length === 1 ? 'proxy' : 'proxies'}? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleBulkDelete}
              disabled={isBulkRunning}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isBulkRunning ? 'Deleting...' : `Delete ${selectedIds.length}`}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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
