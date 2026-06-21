import { useMutation, useQueryClient } from '@tanstack/react-query';

import { api } from '../lib/api';
import type { ImportReport } from '../lib/proxy-import';
import type { ApiResponse } from '../types/api';

const PROXIES_QUERY_KEY = ['proxies'] as const;

export function useProxyImport() {
  const queryClient = useQueryClient();

  const call = (items: unknown[], dryRun: boolean) =>
    api
      .post('proxies/import', { json: { dry_run: dryRun, proxies: items } })
      .json<ApiResponse<ImportReport>>()
      .then((r) => r.data);

  const previewMutation = useMutation({
    mutationFn: (items: unknown[]) => call(items, true),
  });

  const applyMutation = useMutation({
    mutationFn: (items: unknown[]) => call(items, false),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PROXIES_QUERY_KEY });
    },
  });

  return {
    previewImport: previewMutation.mutateAsync,
    applyImport: applyMutation.mutateAsync,
    isPreviewing: previewMutation.isPending,
    isApplying: applyMutation.isPending,
  };
}
