import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';

import { api } from '../lib/api';
import type { ApiResponse } from '../types/api';
import type { MetricsPublishSettings, UpdateMetricsPublishRequest } from '../types/metrics-publish';

export interface SsoConfig {
  enabled: boolean;
  issuer: string;
  client_id: string;
  has_client_secret: boolean;
  auto_provision: boolean;
  default_role: string;
  button_label: string;
  base_url: string;
  redirect_uri: string;
}

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

const METRICS_PUBLISH_QUERY_KEY = ['settings', 'metrics-publish'] as const;

export function useMetricsPublishSettings() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: METRICS_PUBLISH_QUERY_KEY,
    queryFn: async () => {
      const response = await api
        .get('settings/metrics-publish')
        .json<ApiResponse<MetricsPublishSettings>>();
      return response.data;
    },
  });

  const mutation = useMutation({
    mutationFn: async (data: UpdateMetricsPublishRequest) => {
      return await api
        .put('settings/metrics-publish', { json: data })
        .json<ApiResponse<MetricsPublishSettings>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: METRICS_PUBLISH_QUERY_KEY });
      toast.success('Metrics publish settings updated successfully');
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

const SSO_QUERY_KEY = ['settings', 'sso'] as const;

export function useSsoSettings() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: SSO_QUERY_KEY,
    queryFn: async () => {
      const response = await api.get('auth/sso/config').json<ApiResponse<SsoConfig>>();
      return response.data;
    },
  });

  const mutation = useMutation({
    mutationFn: async (
      data: Omit<SsoConfig, 'has_client_secret' | 'redirect_uri'> & { client_secret: string },
    ) => {
      return await api.put('auth/sso/config', { json: data }).json<ApiResponse<SsoConfig>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SSO_QUERY_KEY });
      toast.success('SSO settings updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update SSO settings', { description: message });
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
