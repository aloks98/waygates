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
import type { PaginationState } from '@tanstack/react-table';
import { Plus, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { ProxiesTabs } from '@/components/layout/proxies-tabs';
import { ProxyDataGrid } from '@/components/proxy';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies } from '@/hooks/use-proxies';
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

  const { proxies, total, totalPages, isLoading, toggle, remove, isToggling, isDeleting } =
    useProxies(params);

  // Delete state
  const [deletingProxy, setDeletingProxy] = useState<ProxyConfig | null>(null);

  const handleEdit = useCallback(
    (proxy: ProxyConfig) => {
      navigate({ to: '/dashboard/proxies/$proxyId', params: { proxyId: String(proxy.id) } });
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

  return (
    <div className="space-y-6">
      <div className="mb-4">
        <ProxiesTabs active="http" />
      </div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Proxies</h1>
        {canCreateProxies && (
          <Button onClick={() => navigate({ to: '/dashboard/proxies/new' })}>
            <Plus className="size-4" />
            Add Proxy
          </Button>
        )}
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

      <ProxyDataGrid
        data={proxies}
        isLoading={isLoading}
        canUpdateProxies={canUpdateProxies}
        canDeleteProxies={canDeleteProxies}
        onEdit={handleEdit}
        onDelete={setDeletingProxy}
        onToggleStatus={handleToggleStatus}
        isToggling={isToggling}
        pageCount={totalPages}
        pagination={pagination}
        onPaginationChange={setPagination}
        total={total}
      />

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
