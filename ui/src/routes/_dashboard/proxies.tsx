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
import {
  type ACLAssignment,
  getProxyTypeLabel,
  ProxyDataGrid,
  ProxyFormModal,
} from '@/components/proxy';
import { useAssignACL, useProxyACL, useRemoveACL } from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies } from '@/hooks/use-proxies';
import type { CreateProxyRequest, ProxyConfig, ProxyType } from '@/types/proxy';

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
  { value: 'static', label: 'Static' },
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
    type: 'multiselect',
    options: typeOptions,
    operators: [
      { value: 'is_any_of', label: 'is any of', supportsMultiple: true },
      { value: 'is_not_any_of', label: 'is not any of', supportsMultiple: true },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select',
    options: statusOptions,
    operators: [
      { value: 'is', label: 'is' },
      { value: 'is_not', label: 'is not' },
    ],
  },
  {
    key: 'ssl_enabled',
    label: 'SSL',
    type: 'select',
    options: sslOptions,
    operators: [{ value: 'is', label: 'is' }],
  },
  {
    key: 'target',
    label: 'Target',
    type: 'text',
    placeholder: 'Enter target URL...',
    defaultOperator: 'is',
    operators: [{ value: 'is', label: 'contains' }],
  },
];

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

  // Convert filters to API params
  const apiParams = useMemo(() => {
    const params: {
      page: number;
      limit: number;
      search?: string;
      type?: string;
      status?: string;
      ssl_enabled?: string;
      target?: string;
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
        case 'ssl_enabled':
          // SSL is just a simple value, no operator prefix needed
          params.ssl_enabled = filter.values[0];
          break;
        case 'target':
          // Target is a text search, pass directly
          params.target = filter.values[0];
          break;
      }
    }

    return params;
  }, [pagination, debouncedSearch, debouncedFilters, buildFilterValue]);

  const handleFiltersChange = useCallback((newFilters: Filter<string>[]) => {
    setFilters(newFilters);
  }, []);

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

  const { assignACL } = useAssignACL();
  const { removeACL } = useRemoveACL();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createProxyType, setCreateProxyType] = useState<ProxyType>('reverse_proxy');
  const [editingProxy, setEditingProxy] = useState<ProxyConfig | null>(null);
  const [deletingProxy, setDeletingProxy] = useState<ProxyConfig | null>(null);

  // Fetch ACL assignments for the proxy being edited
  const { assignments: editingProxyACL } = useProxyACL(editingProxy?.id ?? 0);

  // Convert API ACL assignments to form ACL assignments
  const editingProxyACLAssignments: ACLAssignment[] = useMemo(() => {
    return editingProxyACL.map((a) => ({
      acl_group_id: a.acl_group_id,
      acl_group_name: a.acl_group?.name,
      path_pattern: a.path_pattern,
      priority: a.priority,
      enabled: a.enabled,
    }));
  }, [editingProxyACL]);

  const handleCreateProxy = (type: ProxyType) => {
    setCreateProxyType(type);
    setCreateModalOpen(true);
  };

  const handleCreate = async (data: CreateProxyRequest, aclAssignments?: ACLAssignment[]) => {
    const response = await create(data);
    const proxyId = response.data?.id;

    // Assign ACLs if provided
    if (proxyId && aclAssignments && aclAssignments.length > 0) {
      for (const assignment of aclAssignments) {
        await assignACL({
          proxyId,
          data: {
            acl_group_id: assignment.acl_group_id,
            path_pattern: assignment.path_pattern,
            priority: assignment.priority,
            enabled: assignment.enabled,
          },
        });
      }
    }

    setCreateModalOpen(false);
  };

  const handleUpdate = async (data: CreateProxyRequest, aclAssignments?: ACLAssignment[]) => {
    if (!editingProxy) return;
    await update({ id: editingProxy.id, data });

    // Sync ACL assignments
    const newAssignments = aclAssignments ?? [];
    const oldGroupIds = editingProxyACL.map((a) => a.acl_group_id);
    const newGroupIds = newAssignments.map((a) => a.acl_group_id);

    // Remove ACL groups that are no longer assigned
    for (const oldAssignment of editingProxyACL) {
      if (!newGroupIds.includes(oldAssignment.acl_group_id)) {
        await removeACL({ proxyId: editingProxy.id, groupId: oldAssignment.acl_group_id });
      }
    }

    // Add new ACL groups
    for (const newAssignment of newAssignments) {
      if (!oldGroupIds.includes(newAssignment.acl_group_id)) {
        await assignACL({
          proxyId: editingProxy.id,
          data: {
            acl_group_id: newAssignment.acl_group_id,
            path_pattern: newAssignment.path_pattern,
            priority: newAssignment.priority,
            enabled: newAssignment.enabled,
          },
        });
      }
    }

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
          key={editingProxy.id}
          open={!!editingProxy}
          onOpenChange={(open) => !open && setEditingProxy(null)}
          onSubmit={handleUpdate}
          initialData={editingProxy}
          initialACLAssignments={editingProxyACLAssignments}
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
