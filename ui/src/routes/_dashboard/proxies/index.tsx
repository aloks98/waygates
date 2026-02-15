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
  Input,
} from '@e412/titanium';
import { useNavigate } from '@tanstack/react-router';
import type { PaginationState } from '@tanstack/react-table';
import { Plus, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ProxyDataGrid } from '@/components/proxy';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies } from '@/hooks/use-proxies';
import type { ProxyConfig } from '@/types/proxy';

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

  // Fetch HTTP proxies (L7)
  const params = useMemo(() => {
    const p: {
      page: number;
      limit: number;
      search?: string;
    } = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    };
    if (debouncedSearch) {
      p.search = debouncedSearch;
    }
    return p;
  }, [pagination.pageIndex, pagination.pageSize, debouncedSearch]);

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
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Proxies</h1>
        {canCreateProxies && (
          <Button onClick={() => navigate({ to: '/dashboard/proxies/new' })}>
            <Plus className="size-4" />
            Add Proxy
          </Button>
        )}
      </div>

      <div className="flex flex-col gap-4">
        {/* Search input */}
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search proxies..."
            className="pl-9"
          />
        </div>
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
