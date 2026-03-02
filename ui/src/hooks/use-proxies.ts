import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';

import { api } from '../lib/api';
import type { ApiResponse, PaginatedResponse } from '../types/api';
import type { CreateProxyRequest, ProxyConfig, UpdateProxyRequest } from '../types/proxy';

const QUERY_KEY = ['proxies'] as const;

export function useProxy(id: number) {
  const query = useQuery({
    queryKey: [...QUERY_KEY, id],
    queryFn: async () => {
      const response = await api.get(`proxies/${id}`).json<ApiResponse<ProxyConfig>>();
      return response.data;
    },
    enabled: id > 0,
  });

  return {
    proxy: query.data ?? null,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
}

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

interface UseProxiesOptions {
  page?: number;
  limit?: number;
  search?: string;
  type?: string;
  status?: string;
  ssl_enabled?: string;
  target?: string;
}

export function useProxies(options: UseProxiesOptions = {}) {
  const { page = 1, limit = 20, search, type, status, ssl_enabled, target } = options;
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: [...QUERY_KEY, { page, limit, search, type, status, ssl_enabled, target }],
    queryFn: async () => {
      const searchParams: Record<string, string> = {
        page: String(page),
        limit: String(limit),
      };
      if (search) {
        searchParams.search = search;
      }
      if (type) {
        searchParams.type = type;
      }
      if (status) {
        searchParams.status = status;
      }
      if (ssl_enabled) {
        searchParams.ssl_enabled = ssl_enabled;
      }
      if (target) {
        searchParams.target = target;
      }
      const response = await api
        .get('proxies', { searchParams })
        .json<ApiResponse<PaginatedResponse<ProxyConfig>>>();
      return response.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: CreateProxyRequest) => {
      return await api.post('proxies', { json: data }).json<ApiResponse<ProxyConfig>>();
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success(`Proxy "${response.data?.name}" created successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to create proxy', { description: message });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: number; data: UpdateProxyRequest }) => {
      return await api.put(`proxies/${id}`, { json: data }).json<ApiResponse<ProxyConfig>>();
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success(`Proxy "${response.data?.name}" updated successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update proxy', { description: message });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`proxies/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success('Proxy deleted successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to delete proxy', { description: message });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ id, enable }: { id: number; enable: boolean }) => {
      await api.post(`proxies/${id}/${enable ? 'enable' : 'disable'}`);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success(`Proxy ${variables.enable ? 'enabled' : 'disabled'} successfully`);
    },
    onError: async (error, variables) => {
      const message = await handleApiError(error);
      toast.error(`Failed to ${variables.enable ? 'enable' : 'disable'} proxy`, {
        description: message,
      });
    },
  });

  return {
    // Query
    proxies: query.data?.items ?? [],
    total: query.data?.total ?? 0,
    page: query.data?.page ?? 1,
    limit: query.data?.limit ?? limit,
    totalPages: query.data?.total_pages ?? 1,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,

    // Mutations
    create: createMutation.mutateAsync,
    update: updateMutation.mutateAsync,
    remove: deleteMutation.mutateAsync,
    toggle: toggleMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
    isToggling: toggleMutation.isPending,
  };
}
