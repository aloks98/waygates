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
  Tabs,
  TabsList,
  TabsTrigger,
} from '@e412/titanium';
import { useNavigate } from '@tanstack/react-router';
import type { PaginationState } from '@tanstack/react-table';
import { Plus, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { UnifiedProxyDataGrid } from '@/components/proxy';
import { useL4Proxies } from '@/hooks/use-l4-proxies';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies } from '@/hooks/use-proxies';
import {
  toUnifiedL4Proxy,
  toUnifiedProxy,
  type UnifiedProxy,
  type UnifiedProxyFilter,
} from '@/types/unified-proxy';

export function ProxiesListPage() {
  const navigate = useNavigate();
  const { canCreateProxies, canUpdateProxies } = usePermissions();

  // Filter state
  const [activeFilter, setActiveFilter] = useState<UnifiedProxyFilter>('all');
  const prevFilterRef = useRef(activeFilter);

  // Pagination state - client-side pagination for combined data
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
    if (prevFilterRef.current !== activeFilter) {
      prevFilterRef.current = activeFilter;
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    }
  });

  // Fetch HTTP proxies (L7)
  const httpParams = useMemo(() => {
    const params: {
      page: number;
      limit: number;
      search?: string;
    } = {
      // Fetch all for client-side filtering/pagination when combined
      page: 1,
      limit: 1000,
    };
    if (debouncedSearch) {
      params.search = debouncedSearch;
    }
    return params;
  }, [debouncedSearch]);

  const {
    proxies: httpProxies,
    isLoading: isLoadingHttp,
    toggle: toggleHttp,
    remove: removeHttp,
    isToggling: isTogglingHttp,
    isDeleting: isDeletingHttp,
  } = useProxies(httpParams);

  // Fetch L4 proxies
  const l4Params = useMemo(() => {
    const params: {
      page: number;
      limit: number;
      search?: string;
      protocol?: string;
    } = {
      page: 1,
      limit: 1000,
    };
    if (debouncedSearch) {
      params.search = debouncedSearch;
    }
    // Filter by protocol if specific L4 type selected
    if (activeFilter === 'tcp') {
      params.protocol = 'tcp';
    } else if (activeFilter === 'udp') {
      params.protocol = 'udp';
    }
    return params;
  }, [debouncedSearch, activeFilter]);

  const {
    proxies: l4Proxies,
    isLoading: isLoadingL4,
    toggleActive: toggleL4,
    remove: removeL4,
    isToggling: isTogglingL4,
    isDeleting: isDeletingL4,
  } = useL4Proxies(l4Params);

  const isLoading = isLoadingHttp || isLoadingL4;
  const isToggling = isTogglingHttp || isTogglingL4;
  const isDeleting = isDeletingHttp || isDeletingL4;

  // Combine and filter proxies
  const unifiedProxies = useMemo<UnifiedProxy[]>(() => {
    const unified: UnifiedProxy[] = [];

    // Add HTTP proxies based on filter
    if (activeFilter === 'all' || activeFilter === 'http') {
      for (const proxy of httpProxies) {
        unified.push(toUnifiedProxy(proxy));
      }
    }

    // Add L4 proxies based on filter
    if (activeFilter === 'all' || activeFilter === 'tcp' || activeFilter === 'udp') {
      for (const proxy of l4Proxies) {
        // When filter is all, add all L4 proxies
        // When filter is tcp/udp, the API already filters, but double-check
        if (
          activeFilter === 'all' ||
          (activeFilter === 'tcp' && proxy.protocol === 'tcp') ||
          (activeFilter === 'udp' && proxy.protocol === 'udp')
        ) {
          unified.push(toUnifiedL4Proxy(proxy));
        }
      }
    }

    // Sort by updated_at descending (most recent first)
    unified.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());

    return unified;
  }, [httpProxies, l4Proxies, activeFilter]);

  // Client-side pagination
  const paginatedProxies = useMemo(() => {
    const start = pagination.pageIndex * pagination.pageSize;
    const end = start + pagination.pageSize;
    return unifiedProxies.slice(start, end);
  }, [unifiedProxies, pagination]);

  const totalPages = Math.ceil(unifiedProxies.length / pagination.pageSize);

  // Delete state
  const [deletingProxy, setDeletingProxy] = useState<UnifiedProxy | null>(null);

  const handleRowClick = useCallback(
    (proxy: UnifiedProxy) => {
      if (proxy.layer === 'L7') {
        navigate({ to: '/dashboard/proxies/$proxyId', params: { proxyId: String(proxy.id) } });
      } else {
        navigate({
          to: '/dashboard/l4-proxies/$l4ProxyId',
          params: { l4ProxyId: String(proxy.id) },
        });
      }
    },
    [navigate],
  );

  const handleToggleStatus = useCallback(
    async (proxy: UnifiedProxy, enable: boolean) => {
      if (proxy.layer === 'L7') {
        await toggleHttp({ id: proxy.id, enable });
      } else {
        // L4 toggle doesn't take enable param, it just toggles
        await toggleL4(proxy.id);
      }
    },
    [toggleHttp, toggleL4],
  );

  const handleDelete = async () => {
    if (!deletingProxy) return;
    if (deletingProxy.layer === 'L7') {
      await removeHttp(deletingProxy.id);
    } else {
      await removeL4(deletingProxy.id);
    }
    setDeletingProxy(null);
  };

  // Count by type for tab badges
  const counts = useMemo(() => {
    const httpCount = httpProxies.length;
    const tcpCount = l4Proxies.filter((p) => p.protocol === 'tcp').length;
    const udpCount = l4Proxies.filter((p) => p.protocol === 'udp').length;
    return {
      all: httpCount + l4Proxies.length,
      http: httpCount,
      tcp: tcpCount,
      udp: udpCount,
    };
  }, [httpProxies, l4Proxies]);

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
        {/* Filter tabs */}
        <Tabs
          value={activeFilter}
          onValueChange={(value) => setActiveFilter(value as UnifiedProxyFilter)}
        >
          <TabsList>
            <TabsTrigger value="all" className="gap-2">
              All
              <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {counts.all}
              </span>
            </TabsTrigger>
            <TabsTrigger value="http" className="gap-2">
              HTTP
              <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {counts.http}
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
            placeholder="Search proxies..."
            className="pl-9"
          />
        </div>
      </div>

      <UnifiedProxyDataGrid
        data={paginatedProxies}
        isLoading={isLoading}
        canUpdateProxies={canUpdateProxies}
        onRowClick={handleRowClick}
        onToggleStatus={handleToggleStatus}
        isToggling={isToggling}
        pageCount={totalPages}
        pagination={pagination}
        onPaginationChange={setPagination}
        total={unifiedProxies.length}
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
