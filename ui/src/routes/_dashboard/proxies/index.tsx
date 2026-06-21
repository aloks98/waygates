import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  type Filter,
  type FilterFieldsConfig,
  Filters,
  Input,
} from '@e412/rnui-react';
import { useNavigate } from '@tanstack/react-router';
import type { PaginationState, RowSelectionState } from '@tanstack/react-table';
import { Download, Plus, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';

import { ProxiesTabs } from '@/components/layout/proxies-tabs';
import { ProxyDataGrid } from '@/components/proxy';
import { ProxyBulkBar } from '@/components/proxy/proxy-bulk-bar';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies } from '@/hooks/use-proxies';
import { api } from '@/lib/api';
import { downloadJson, type ProxyExport } from '@/lib/proxy-export';
import type { ApiResponse } from '@/types/api';
import type { ProxyConfig } from '@/types/proxy';

const typeOptions = [
  { value: 'reverse_proxy', label: 'Reverse Proxy' },
  { value: 'redirect', label: 'Redirect' },
  { value: 'static', label: 'Static File Server' },
];

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
];

const sslOptions = [
  { value: 'true', label: 'Enabled' },
  { value: 'false', label: 'Disabled' },
];

const filterFields: FilterFieldsConfig<string> = [
  {
    key: 'type',
    label: 'Type',
    type: 'select',
    options: typeOptions,
    operators: [
      { value: 'is', label: 'is' },
      { value: 'is_not', label: 'is not' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: statusOptions,
    operators: [{ value: 'is', label: 'is' }],
  },
  {
    key: 'ssl_enabled',
    label: 'SSL',
    type: 'select',
    options: sslOptions,
    operators: [{ value: 'is', label: 'is' }],
  },
];

export function ProxiesListPage() {
  const navigate = useNavigate();
  const { canCreateProxies, canUpdateProxies, canDeleteProxies } = usePermissions();

  // Pagination state
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  });

  // Search state
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Filter state
  const [filters, setFilters] = useState<Filter<string>[]>([]);

  const handleFiltersChange = useCallback((newFilters: Filter<string>[]) => {
    setFilters(newFilters);
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
  }, []);

  // Row selection state — owned by the page, cleared on page/search/filter change
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  useEffect(() => {
    setRowSelection({});
  }, [pagination, debouncedSearch, filters]);

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

  // Build API params from search + filters
  const params = useMemo(() => {
    const p: {
      page: number;
      limit: number;
      search?: string;
      type?: string;
      status?: string;
      ssl_enabled?: string;
    } = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    };
    if (debouncedSearch) {
      p.search = debouncedSearch;
    }
    for (const filter of filters) {
      if (filter.values.length === 0) continue;
      const value = filter.values[0];
      switch (filter.field) {
        case 'type':
          p.type = value;
          break;
        case 'status':
          p.status = value;
          break;
        case 'ssl_enabled':
          p.ssl_enabled = value;
          break;
      }
    }
    return p;
  }, [pagination.pageIndex, pagination.pageSize, debouncedSearch, filters]);

  const {
    proxies,
    total,
    totalPages,
    isLoading,
    toggle,
    remove,
    isToggling,
    isDeleting,
    bulkSetActive,
    bulkRemove,
    isBulkRunning,
  } = useProxies(params);

  // Derived selection
  const selectedIds = useMemo(() => Object.keys(rowSelection).map(Number), [rowSelection]);

  // Delete state
  const [deletingProxy, setDeletingProxy] = useState<ProxyConfig | null>(null);
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);

  // Export state
  const [isExporting, setIsExporting] = useState(false);

  const handleEdit = useCallback(
    (proxy: ProxyConfig) => {
      navigate({ to: '/dashboard/proxies/$proxyId', params: { proxyId: String(proxy.id) } });
    },
    [navigate],
  );

  const handleDuplicate = useCallback(
    (proxy: ProxyConfig) => {
      navigate({
        to: '/dashboard/proxies/new',
        search: { type: proxy.type, duplicate: proxy.id },
      });
    },
    [navigate],
  );

  const handleToggleStatus = useCallback(
    async (id: number, enable: boolean) => {
      await toggle({ id, enable });
    },
    [toggle],
  );

  const handleDelete = async () => {
    if (!deletingProxy) return;
    await remove(deletingProxy.id);
    setDeletingProxy(null);
  };

  const handleBulkEnable = useCallback(async () => {
    const s = await bulkSetActive(selectedIds, true);
    if (s.failed > 0) {
      toast.error(`Enabled ${s.succeeded}${s.failed ? `, ${s.failed} failed` : ''}`);
    } else {
      toast.success(`Enabled ${s.succeeded}${s.failed ? `, ${s.failed} failed` : ''}`);
    }
    setRowSelection({});
  }, [bulkSetActive, selectedIds, setRowSelection]);

  const handleBulkDisable = useCallback(async () => {
    const s = await bulkSetActive(selectedIds, false);
    if (s.failed > 0) {
      toast.error(`Disabled ${s.succeeded}${s.failed ? `, ${s.failed} failed` : ''}`);
    } else {
      toast.success(`Disabled ${s.succeeded}${s.failed ? `, ${s.failed} failed` : ''}`);
    }
    setRowSelection({});
  }, [bulkSetActive, selectedIds, setRowSelection]);

  const handleBulkDelete = useCallback(async () => {
    setBulkDeleteOpen(false);
    const s = await bulkRemove(selectedIds);
    if (s.failed > 0) {
      toast.error(`Deleted ${s.succeeded}, ${s.failed} failed`);
    } else {
      toast.success(`Deleted ${s.succeeded} ${s.succeeded === 1 ? 'proxy' : 'proxies'}`);
    }
    setRowSelection({});
  }, [bulkRemove, selectedIds, setRowSelection]);

  const handleExport = useCallback(async () => {
    setIsExporting(true);
    try {
      const searchParams: Record<string, string> = {};
      if (selectedIds.length > 0) {
        searchParams.ids = selectedIds.join(',');
      } else {
        if (debouncedSearch) searchParams.search = debouncedSearch;
        for (const f of filters) {
          if (f.values.length === 0) continue;
          const v = f.values[0];
          if (f.field === 'type') searchParams.type = v;
          else if (f.field === 'status') searchParams.status = v;
          else if (f.field === 'ssl_enabled') searchParams.ssl_enabled = v;
        }
      }
      const response = await api
        .get('proxies/export', { searchParams })
        .json<ApiResponse<ProxyExport[]>>();
      const data = response.data ?? [];
      downloadJson(`waygates-proxies-${data.length}.json`, data);
      toast.success(`Exported ${data.length} ${data.length === 1 ? 'proxy' : 'proxies'}`);
    } catch {
      toast.error('Export failed');
    } finally {
      setIsExporting(false);
    }
  }, [selectedIds, debouncedSearch, filters]);

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="mb-4">
        <ProxiesTabs active="http" />
      </div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Proxies</h1>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleExport} disabled={isExporting}>
            <Download className="size-4" />
            {isExporting ? 'Exporting...' : 'Export'}
          </Button>
          {canCreateProxies && (
            <Button onClick={() => navigate({ to: '/dashboard/proxies/new' })}>
              <Plus className="size-4" />
              Add Proxy
            </Button>
          )}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name or hostname..."
            aria-label="Search proxies"
            className="pl-9"
          />
        </div>

        <Filters
          filters={filters}
          fields={filterFields}
          onChange={handleFiltersChange}
          addButtonText="Add Filter"
          variant="outline"
          size="md"
        />
      </div>

      <ProxyBulkBar
        count={selectedIds.length}
        onEnable={handleBulkEnable}
        onDisable={handleBulkDisable}
        onDelete={() => setBulkDeleteOpen(true)}
        onClear={() => setRowSelection({})}
        running={isBulkRunning}
      />

      <ProxyDataGrid
        data={proxies}
        isLoading={isLoading}
        canUpdateProxies={canUpdateProxies}
        canDeleteProxies={canDeleteProxies}
        onEdit={handleEdit}
        onDelete={setDeletingProxy}
        onDuplicate={canCreateProxies ? handleDuplicate : undefined}
        onToggleStatus={handleToggleStatus}
        isToggling={isToggling}
        pageCount={totalPages}
        pagination={pagination}
        onPaginationChange={setPagination}
        total={total}
        rowSelection={rowSelection}
        onRowSelectionChange={setRowSelection}
      />

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
            <AlertDialogTitle>Delete Proxy</AlertDialogTitle>
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
