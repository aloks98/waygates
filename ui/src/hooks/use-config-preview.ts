import { useQuery } from '@tanstack/react-query';

import { api } from '../lib/api';
import type { ApiResponse } from '../types/api';
import type { CaddyConfig } from '../types/caddy-config';

export function useProxyConfigPreview(proxyId: number) {
  return useQuery({
    queryKey: ['proxy-config-preview', proxyId],
    queryFn: async () =>
      (await api.get(`proxies/${proxyId}/config-preview`).json<ApiResponse<CaddyConfig>>()).data,
    enabled: proxyId > 0,
  });
}

export function useCaddyConfig() {
  return useQuery({
    queryKey: ['caddy-config'],
    queryFn: async () => (await api.get('caddy-config').json<ApiResponse<CaddyConfig>>()).data,
  });
}
