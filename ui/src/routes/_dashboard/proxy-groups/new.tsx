import { Button } from '@e412/rnui-react';
import { useNavigate } from '@tanstack/react-router';
import { ArrowLeft } from 'lucide-react';

import { ProxyGroupForm } from '@/components/proxy-group';
import { useProxyGroups } from '@/hooks/use-proxy-groups';
import type { CreateProxyGroupRequest } from '@/types/proxy-group';

export function ProxyGroupCreatePage() {
  const navigate = useNavigate();
  const { create } = useProxyGroups();

  const handleSubmit = async (data: CreateProxyGroupRequest) => {
    const group = await create.mutateAsync(data).catch(() => null);
    if (group) {
      navigate({ to: '/proxy-groups/$groupId/edit', params: { groupId: String(group.id) } });
    }
  };

  const handleCancel = () => {
    navigate({ to: '/proxy-groups' });
  };

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/proxy-groups' })}>
          <ArrowLeft className="size-4" />
          <span className="sr-only">Back</span>
        </Button>
        <h1 className="text-2xl font-bold">Create Proxy Group</h1>
      </div>

      <ProxyGroupForm
        mode="create"
        onSubmit={handleSubmit}
        loading={create.isPending}
        onCancel={handleCancel}
      />
    </div>
  );
}
