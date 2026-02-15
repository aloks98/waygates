import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';
import { api } from '../lib/api';
import type { ApiResponse, PaginatedResponse } from '../types/api';
import type {
  CreateL4ProxyRequest,
  L4Proxy,
  L4ProxyStats,
  UpdateL4ProxyRequest,
} from '../types/l4-proxy';

const QUERY_KEY = ['l4-proxies'] as const;

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

export function useL4Proxy(id: number) {
  const query = useQuery({
    queryKey: [...QUERY_KEY, id],
    queryFn: async () => {
      const response = await api.get(`l4-proxies/${id}`).json<ApiResponse<L4Proxy>>();
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

export function useL4ProxyStats() {
  const query = useQuery({
    queryKey: [...QUERY_KEY, 'stats'],
    queryFn: async () => {
      const response = await api.get('l4-proxies/stats').json<ApiResponse<L4ProxyStats>>();
      return response.data;
    },
  });

  return {
    stats: query.data ?? null,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
  };
}

interface UseL4ProxiesOptions {
  page?: number;
  limit?: number;
  search?: string;
  protocol?: string;
  is_active?: boolean;
  sort?: string;
  order?: string;
}

export function useL4Proxies(options: UseL4ProxiesOptions = {}) {
  const { page = 1, limit = 20, search, protocol, is_active, sort, order } = options;
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: [...QUERY_KEY, { page, limit, search, protocol, is_active, sort, order }],
    queryFn: async () => {
      const searchParams: Record<string, string> = {
        page: String(page),
        limit: String(limit),
      };
      if (search) {
        searchParams.search = search;
      }
      if (protocol) {
        searchParams.protocol = protocol;
      }
      if (is_active !== undefined) {
        searchParams.is_active = String(is_active);
      }
      if (sort) {
        searchParams.sort = sort;
      }
      if (order) {
        searchParams.order = order;
      }
      const response = await api
        .get('l4-proxies', { searchParams })
        .json<ApiResponse<PaginatedResponse<L4Proxy>>>();
      return response.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: CreateL4ProxyRequest) => {
      return await api.post('l4-proxies', { json: data }).json<ApiResponse<L4Proxy>>();
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success(`L4 Proxy "${response.data?.name}" created successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to create L4 proxy', { description: message });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: number; data: UpdateL4ProxyRequest }) => {
      return await api.put(`l4-proxies/${id}`, { json: data }).json<ApiResponse<L4Proxy>>();
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success(`L4 Proxy "${response.data?.name}" updated successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update L4 proxy', { description: message });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`l4-proxies/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success('L4 Proxy deleted successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to delete L4 proxy', { description: message });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async (id: number) => {
      return await api.patch(`l4-proxies/${id}/toggle`).json<ApiResponse<L4Proxy>>();
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      const status = response.data?.is_active ? 'enabled' : 'disabled';
      toast.success(`L4 Proxy ${status} successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to toggle L4 proxy status', { description: message });
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
    toggleActive: toggleMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
    isToggling: toggleMutation.isPending,
  };
}
