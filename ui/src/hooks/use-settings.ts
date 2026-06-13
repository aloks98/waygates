import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';

import { api } from '../lib/api';
import type { ApiResponse } from '../types/api';

const QUERY_KEY = ['settings', '404'] as const;

export interface NotFoundSettings {
  mode: 'default' | 'redirect';
  redirect_url: string;
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

export function useNotFoundSettings() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: QUERY_KEY,
    queryFn: async () => {
      const response = await api.get('settings/404').json<ApiResponse<NotFoundSettings>>();
      return response.data;
    },
  });

  const mutation = useMutation({
    mutationFn: async (data: NotFoundSettings) => {
      return await api.put('settings/404', { json: data }).json<ApiResponse<NotFoundSettings>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success('404 settings updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update settings', { description: message });
    },
  });

  return {
    settings: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    update: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}
