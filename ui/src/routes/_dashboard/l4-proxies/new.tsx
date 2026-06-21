import { Button, Skeleton } from '@e412/rnui-react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { ArrowLeft } from 'lucide-react';
import { useMemo } from 'react';

import { L4ProxyForm } from '@/components/l4-proxy';
import { useL4Proxies, useL4Proxy } from '@/hooks/use-l4-proxies';
import type { CreateL4ProxyRequest, L4Proxy } from '@/types/l4-proxy';

export function L4ProxyCreatePage() {
  const navigate = useNavigate();
  const searchParams = useSearch({ strict: false }) as { duplicate?: string };

  // Duplicate support — fetch source proxy when ?duplicate=<id> is present
  const dupId = Number(searchParams.duplicate) || 0;
  const { proxy: source } = useL4Proxy(dupId);

  // Build the seed: clear listen_port, suffix name. null when not duplicating or source not yet loaded.
  const seed = useMemo<L4Proxy | null>(
    () =>
      source
        ? {
            ...source,
            listen_port: undefined as unknown as number,
            name: `${source.name} (copy)`,
          }
        : null,
    [source],
  );

  const { create: createL4Proxy, isCreating } = useL4Proxies();

  const handleSubmit = async (data: CreateL4ProxyRequest) => {
    const response = await createL4Proxy(data);
    const l4ProxyId = response.data?.id;

    if (l4ProxyId) {
      navigate({
        to: '/dashboard/proxies/tcp-udp/$l4ProxyId',
        params: { l4ProxyId: String(l4ProxyId) },
      });
    } else {
      navigate({ to: '/dashboard/proxies/tcp-udp' });
    }
  };

  const handleCancel = () => {
    navigate({ to: '/dashboard/proxies/tcp-udp' });
  };

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => navigate({ to: '/dashboard/proxies/tcp-udp' })}
        >
          <ArrowLeft className="size-4" />
          <span className="sr-only">Back</span>
        </Button>
        <h1 className="text-2xl font-bold">Create L4 Proxy</h1>
      </div>

      {/* While waiting for duplicate source to load, show skeleton */}
      {dupId > 0 && !source ? (
        <div className="space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-2/3" />
        </div>
      ) : (
        <L4ProxyForm
          mode="create"
          initialData={seed}
          onSubmit={handleSubmit}
          loading={isCreating}
          onCancel={handleCancel}
        />
      )}
    </div>
  );
}
