import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { api, publicApi } from '@/lib/api';
import type {
  ApiResponse,
  AppStatus,
  HealthStatus,
  PaginatedResponse,
  ProxyStats,
  SyncStatus,
} from '@/types/api';
import type { AuditLog } from '@/types/audit';
import type { L4ProxyStats } from '@/types/l4-proxy';
import type { ProxyConfig } from '@/types/proxy';

const SYNC_STATUS_KEY = ['sync-status'] as const;
const HEALTH_KEY = ['health'] as const;
const APP_STATUS_KEY = ['app-status'] as const;

export function useSyncStatus() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: SYNC_STATUS_KEY,
    queryFn: async () => {
      const response = await api.get('sync/status').json<ApiResponse<SyncStatus>>();
      return response.data;
    },
    refetchInterval: 15_000, // Poll every 15s
  });

  const triggerMutation = useMutation({
    mutationFn: async () => {
      return await api.post('sync/trigger').json<ApiResponse<null>>();
    },
    onSuccess: () => {
      toast.success('Sync triggered');
      queryClient.invalidateQueries({ queryKey: SYNC_STATUS_KEY });
    },
    onError: () => {
      toast.error('Failed to trigger sync');
    },
  });

  return {
    syncStatus: query.data,
    isLoading: query.isLoading,
    triggerSync: triggerMutation.mutateAsync,
    isSyncing: triggerMutation.isPending || (query.data?.is_syncing ?? false),
  };
}

export function useHealthStatus() {
  const query = useQuery({
    queryKey: HEALTH_KEY,
    queryFn: async () => {
      const response = await publicApi.get('health').json<ApiResponse<HealthStatus>>();
      return response.data;
    },
    refetchInterval: 30_000,
  });

  return {
    health: query.data,
    isLoading: query.isLoading,
  };
}

export function useAppStatus() {
  const query = useQuery({
    queryKey: APP_STATUS_KEY,
    queryFn: async () => {
      const response = await publicApi.get('status').json<ApiResponse<AppStatus>>();
      return response.data;
    },
    refetchInterval: 30_000,
  });

  return {
    appStatus: query.data,
    isLoading: query.isLoading,
  };
}

export function useDashboardData() {
  const proxyStats = useQuery({
    queryKey: ['proxies', 'stats'],
    queryFn: async () => {
      const response = await api.get('proxies/stats').json<ApiResponse<ProxyStats>>();
      return response.data;
    },
  });

  const l4ProxyStats = useQuery({
    queryKey: ['l4-proxies', 'stats'],
    queryFn: async () => {
      const response = await api.get('l4-proxies/stats').json<ApiResponse<L4ProxyStats>>();
      return response.data;
    },
  });

  const recentProxies = useQuery({
    queryKey: ['proxies', 'recent'],
    queryFn: async () => {
      const response = await api
        .get('proxies', { searchParams: { limit: '5', sort: 'created_at', order: 'desc' } })
        .json<ApiResponse<PaginatedResponse<ProxyConfig>>>();
      return response.data?.items ?? [];
    },
  });

  const recentActivity = useQuery({
    queryKey: ['audit-logs', 'dashboard-recent'],
    queryFn: async () => {
      const response = await api
        .get('audit-logs', { searchParams: { limit: '8', sort: 'created_at', order: 'desc' } })
        .json<ApiResponse<{ items: AuditLog[] }>>();
      return response.data?.items ?? [];
    },
  });

  return {
    proxyStats: proxyStats.data,
    l4ProxyStats: l4ProxyStats.data,
    recentProxies: recentProxies.data ?? [],
    recentActivity: recentActivity.data ?? [],
    isLoading:
      proxyStats.isLoading ||
      l4ProxyStats.isLoading ||
      recentProxies.isLoading ||
      recentActivity.isLoading,
  };
}
