import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';
import { api } from '../lib/api';
import type { ApiResponse } from '../types/api';
import type {
  AuditConfig,
  AuditLog,
  AuditLogListParams,
  AuditLogListResponse,
  AuditLogStats,
} from '../types/audit';

const QUERY_KEY = ['audit-logs'] as const;
const CONFIG_QUERY_KEY = ['audit-config'] as const;
const STATS_QUERY_KEY = ['audit-stats'] as const;

async function handleApiError(error: unknown): Promise<string> {
  if (error instanceof HTTPError) {
    try {
      const body = (await error.response.json()) as { error?: { message?: string } };
      return body?.error?.message || error.message;
    } catch {
      return error.message;
    }
  }
  return error instanceof Error ? error.message : 'An unknown error occurred';
}

export function useAuditLogs(params: AuditLogListParams = {}) {
  const { page = 1, limit = 20, ...filters } = params;

  const query = useQuery({
    queryKey: [...QUERY_KEY, { page, limit, ...filters }],
    queryFn: async () => {
      const searchParams: Record<string, string> = {
        page: String(page),
        limit: String(limit),
      };

      if (filters.search) searchParams.search = filters.search;
      if (filters.action) searchParams.action = filters.action;
      if (filters.resource_type) searchParams.resource_type = filters.resource_type;
      if (filters.user_id) searchParams.user_id = String(filters.user_id);
      if (filters.status) searchParams.status = filters.status;
      if (filters.date_from) searchParams.date_from = filters.date_from;
      if (filters.date_to) searchParams.date_to = filters.date_to;
      if (filters.sort) searchParams.sort = filters.sort;
      if (filters.order) searchParams.order = filters.order;

      const response = await api
        .get('audit-logs', { searchParams })
        .json<ApiResponse<AuditLogListResponse>>();
      return response.data;
    },
  });

  return {
    logs: query.data?.items ?? [],
    total: query.data?.total ?? 0,
    page: query.data?.page ?? 1,
    limit: query.data?.limit ?? limit,
    totalPages: query.data?.total_pages ?? 1,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useAuditLogById(id: number) {
  const query = useQuery({
    queryKey: [...QUERY_KEY, id],
    queryFn: async () => {
      const response = await api.get(`audit-logs/${id}`).json<ApiResponse<AuditLog>>();
      return response.data;
    },
    enabled: id > 0,
  });

  return {
    log: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
}

export function useAuditStats() {
  const query = useQuery({
    queryKey: STATS_QUERY_KEY,
    queryFn: async () => {
      const response = await api.get('audit-logs/stats').json<ApiResponse<AuditLogStats>>();
      return response.data;
    },
  });

  return {
    stats: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
}

export function useAuditConfig() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: CONFIG_QUERY_KEY,
    queryFn: async () => {
      const response = await api.get('audit-logs/config').json<ApiResponse<AuditConfig>>();
      return response.data;
    },
  });

  const updateMutation = useMutation({
    mutationFn: async (config: AuditConfig) => {
      return await api.put('audit-logs/config', { json: config }).json<ApiResponse<AuditConfig>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONFIG_QUERY_KEY });
      toast.success('Audit configuration updated');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update audit configuration', { description: message });
    },
  });

  return {
    config: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    updateConfig: updateMutation.mutateAsync,
    isUpdating: updateMutation.isPending,
  };
}

export function useExportAuditLogs() {
  const exportMutation = useMutation({
    mutationFn: async (params: AuditLogListParams = {}) => {
      const searchParams: Record<string, string> = {};

      if (params.search) searchParams.search = params.search;
      if (params.action) searchParams.action = params.action;
      if (params.resource_type) searchParams.resource_type = params.resource_type;
      if (params.user_id) searchParams.user_id = String(params.user_id);
      if (params.status) searchParams.status = params.status;
      if (params.date_from) searchParams.date_from = params.date_from;
      if (params.date_to) searchParams.date_to = params.date_to;

      const response = await api.get('audit-logs/export', { searchParams });
      const blob = await response.blob();

      // Create download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `audit_logs_${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    },
    onSuccess: () => {
      toast.success('Audit logs exported successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to export audit logs', { description: message });
    },
  });

  return {
    exportLogs: exportMutation.mutateAsync,
    isExporting: exportMutation.isPending,
  };
}
