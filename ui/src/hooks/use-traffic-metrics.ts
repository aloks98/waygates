import { useQuery } from '@tanstack/react-query';

import { api } from '@/lib/api';
import type { ApiResponse } from '@/types/api';
import type { TrafficRange, TrafficSeries } from '@/types/metrics';

export function useTrafficMetrics(range: TrafficRange) {
  const query = useQuery({
    queryKey: ['traffic-metrics', range],
    queryFn: async () => {
      const response = await api
        .get(`metrics/traffic?range=${range}`)
        .json<ApiResponse<TrafficSeries>>();
      return response.data;
    },
    refetchInterval: 30_000,
  });

  return {
    series: query.data,
    isLoading: query.isLoading,
  };
}
