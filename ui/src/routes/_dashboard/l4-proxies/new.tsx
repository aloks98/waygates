import { Button } from '@e412/rnui-react';
import { useNavigate } from '@tanstack/react-router';
import { ArrowLeft } from 'lucide-react';

import { L4ProxyForm } from '@/components/l4-proxy';
import { useL4Proxies } from '@/hooks/use-l4-proxies';
import type { CreateL4ProxyRequest } from '@/types/l4-proxy';

export function L4ProxyCreatePage() {
  const navigate = useNavigate();
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

      {/* L4 Proxy Form */}
      <L4ProxyForm
        mode="create"
        onSubmit={handleSubmit}
        loading={isCreating}
        onCancel={handleCancel}
      />
    </div>
  );
}
