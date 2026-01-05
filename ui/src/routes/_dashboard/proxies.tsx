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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  type Filter,
  type FilterFieldsConfig,
  Filters,
  Input,
} from '@e412/titanium';
import type { PaginationState } from '@tanstack/react-table';
import { ArrowRight, ChevronDown, FolderOpen, Globe, Plus, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getProxyTypeLabel, ProxyDataGrid, ProxyFormModal } from '@/components/proxy';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies } from '@/hooks/use-proxies';
import type { ProxyConfig, ProxyType } from '@/types/proxy';

// Map Titanium filter operators to backend operators
const operatorMap: Record<string, string> = {
  is: '', // defaults to eq
  is_any_of: 'in',
  is_not_any_of: 'not_in',
  is_not: 'not',
};

const typeOptions = [
  { value: 'reverse_proxy', label: 'Reverse Proxy' },
  { value: 'redirect', label: 'Redirect' },
  { value: 'static', label: 'Static File Server' },
];

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
];

const filterFields: FilterFieldsConfig<string> = {
  type: {
    label: 'Type',
    type: 'multiselect',
    options: typeOptions,
  },
  status: {
    label: 'Status',
    type: 'select',
    options: statusOptions,
  },
};

export function ProxiesPage() {
  const { canCreateProxies, canUpdateProxies, canDeleteProxies } = usePermissions();

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  });
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [filters, setFilters] = useState<Filter<string>[]>([]);
  const [debouncedFilters, setDebouncedFilters] = useState<Filter<string>[]>([]);
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const filtersDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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

  // Debounce filters
  useEffect(() => {
    if (filtersDebounceRef.current) {
      clearTimeout(filtersDebounceRef.current);
    }
    filtersDebounceRef.current = setTimeout(() => {
      setDebouncedFilters(filters);
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    }, 500);

    return () => {
      if (filtersDebounceRef.current) {
        clearTimeout(filtersDebounceRef.current);
      }
    };
  }, [filters]);

  // Helper to build filter value with operator prefix
  const buildFilterValue = useCallback((operator: string, values: string[]): string => {
    const backendOp = operatorMap[operator] || '';
    const value = values.join(',');
    return backendOp ? `${backendOp}:${value}` : value;
  }, []);

  // Convert filters to API params using operator:value format
  const apiParams = useMemo(() => {
    const params: {
      page: number;
      limit: number;
      search?: string;
      type?: string;
      status?: string;
    } = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    };

    if (debouncedSearch) {
      params.search = debouncedSearch;
    }

    for (const filter of debouncedFilters) {
      if (filter.values.length === 0) continue;

      switch (filter.field) {
        case 'type':
          params.type = buildFilterValue(filter.operator, filter.values);
          break;
        case 'status':
          params.status = buildFilterValue(filter.operator, filter.values);
          break;
      }
    }

    return params;
  }, [pagination, debouncedSearch, debouncedFilters, buildFilterValue]);

  const {
    proxies,
    total,
    totalPages,
    isLoading,
    create,
    update,
    remove,
    toggle,
    isCreating,
    isUpdating,
    isDeleting,
    isToggling,
  } = useProxies(apiParams);

  const handleFiltersChange = useCallback((newFilters: Filter<string>[]) => {
    setFilters(newFilters);
  }, []);

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createProxyType, setCreateProxyType] = useState<ProxyType>('reverse_proxy');
  const [editingProxy, setEditingProxy] = useState<ProxyConfig | null>(null);
  const [deletingProxy, setDeletingProxy] = useState<ProxyConfig | null>(null);

  const handleCreateProxy = (type: ProxyType) => {
    setCreateProxyType(type);
    setCreateModalOpen(true);
  };

  const handleCreate = async (data: Parameters<typeof create>[0]) => {
    await create(data);
    setCreateModalOpen(false);
  };

  const handleUpdate = async (data: Parameters<typeof create>[0]) => {
    if (!editingProxy) return;
    await update({ id: editingProxy.id, data });
    setEditingProxy(null);
  };

  const handleDelete = async () => {
    if (!deletingProxy) return;
    await remove(deletingProxy.id);
    setDeletingProxy(null);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Proxies</h1>
        {canCreateProxies && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button>
                <Plus className="size-4" />
                Add Proxy
                <ChevronDown className="ml-1 size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => handleCreateProxy('reverse_proxy')}>
                <Globe className="size-4" />
                Reverse Proxy
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleCreateProxy('redirect')}>
                <ArrowRight className="size-4" />
                Redirect
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleCreateProxy('static')}>
                <FolderOpen className="size-4" />
                Static File Server
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search proxies..."
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
        onEdit={setEditingProxy}
        onDelete={setDeletingProxy}
        onToggleStatus={(id, enable) => toggle({ id, enable })}
        isToggling={isToggling}
        pageCount={totalPages}
        pagination={pagination}
        onPaginationChange={setPagination}
        total={total}
      />

      <ProxyFormModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onSubmit={handleCreate}
        proxyType={createProxyType}
        title={`Create ${getProxyTypeLabel(createProxyType)}`}
        loading={isCreating}
      />

      {editingProxy && (
        <ProxyFormModal
          open={!!editingProxy}
          onOpenChange={(open) => !open && setEditingProxy(null)}
          onSubmit={handleUpdate}
          initialData={editingProxy}
          proxyType={editingProxy.type}
          title={`Edit ${getProxyTypeLabel(editingProxy.type)}`}
          loading={isUpdating}
        />
      )}

      <AlertDialog open={!!deletingProxy} onOpenChange={(open) => !open && setDeletingProxy(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Proxy</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete{' '}
              <strong>{deletingProxy?.name || deletingProxy?.hostname}</strong>? This action cannot
              be undone.
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
