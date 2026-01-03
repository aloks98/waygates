import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {api} from '../lib/api';
import type {ApiResponse, PaginatedResponse} from '../types/api';
import type {CreateProxyRequest, Proxy, UpdateProxyRequest} from '../types/proxy';

const QUERY_KEY = ['proxies'] as const;

export function useProxies() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: QUERY_KEY,
    queryFn: async () => {
      const response = await api
        .get('proxies', { searchParams: { limit: '100' } })
        .json<ApiResponse<PaginatedResponse<Proxy>>>();
      return response.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: CreateProxyRequest) => {
      return await api
          .post('proxies', {json: data})
          .json<ApiResponse<Proxy>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: number; data: UpdateProxyRequest }) => {
      return await api
          .put(`proxies/${id}`, {json: data})
          .json<ApiResponse<Proxy>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`proxies/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ id, enable }: { id: number; enable: boolean }) => {
      await api.post(`proxies/${id}/${enable ? 'enable' : 'disable'}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    },
  });

  return {
    // Query
    proxies: query.data?.items ?? [],
    total: query.data?.total ?? 0,
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
