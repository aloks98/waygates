import { useNavigate, useSearch } from '@tanstack/react-router';
import { ArrowRight, FolderOpen, Globe } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';

import { type ACLAssignment, RedirectForm, ReverseProxyForm, StaticForm } from '@/components/proxy';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useAssignACL } from '@/hooks';
import { useProxy, useProxies } from '@/hooks/use-proxies';
import type { CreateProxyRequest, ProxyConfig, ProxyType } from '@/types/proxy';

interface ProxyTypeOption {
  type: ProxyType;
  label: string;
  icon: React.ReactNode;
}

const proxyTypes: ProxyTypeOption[] = [
  {
    type: 'reverse_proxy',
    label: 'Reverse Proxy',
    icon: <Globe className="size-3.5" />,
  },
  {
    type: 'redirect',
    label: 'Redirect',
    icon: <ArrowRight className="size-3.5" />,
  },
  {
    type: 'static',
    label: 'Static File Server',
    icon: <FolderOpen className="size-3.5" />,
  },
];

export function ProxyCreatePage() {
  const navigate = useNavigate();
  const searchParams = useSearch({ strict: false }) as { type?: string; duplicate?: string };

  const validTypes: ProxyType[] = ['reverse_proxy', 'redirect', 'static'];
  const initialType = validTypes.includes(searchParams.type as ProxyType)
    ? (searchParams.type as ProxyType)
    : 'reverse_proxy';

  const [selectedType, setSelectedType] = useState<ProxyType>(initialType);

  // Duplicate support — fetch source proxy when ?duplicate=<id> is present
  const dupId = Number(searchParams.duplicate) || 0;
  const { proxy: source } = useProxy(dupId);

  // Once the source proxy loads, lock the type selector to match its type
  useEffect(() => {
    if (source) {
      setSelectedType(source.type as ProxyType);
    }
  }, [source]);

  // Build the seed: clear hostname, suffix name. null when not duplicating or source not yet loaded.
  const seed = useMemo<ProxyConfig | null>(
    () => (source ? { ...source, hostname: '', name: `${source.name} (copy)` } : null),
    [source],
  );

  const { create: createProxy, isCreating } = useProxies();
  const { assignACL } = useAssignACL();

  const handleSubmit = async (data: CreateProxyRequest, aclAssignments?: ACLAssignment[]) => {
    const response = await createProxy(data);
    const proxyId = response.data?.id;

    if (proxyId && aclAssignments && aclAssignments.length > 0) {
      for (const assignment of aclAssignments) {
        await assignACL({
          proxyId,
          data: {
            acl_group_id: assignment.acl_group_id,
            path_pattern: assignment.path_pattern,
            priority: assignment.priority,
            enabled: assignment.enabled,
          },
        });
      }
    }

    if (proxyId) {
      navigate({ to: '/proxies/$proxyId', params: { proxyId: String(proxyId) } });
    } else {
      navigate({ to: '/proxies' });
    }
  };

  const handleCancel = () => {
    navigate({ to: '/proxies' });
  };

  const typeControl = (
    <Tabs value={selectedType} onValueChange={(v) => setSelectedType(v as ProxyType)}>
      <TabsList className="w-full justify-start">
        {proxyTypes.map((option) => (
          <TabsTrigger key={option.type} value={option.type} disabled={dupId > 0}>
            {option.icon}
            {option.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
  return (
    <div className="flex max-w-[1440px] flex-col gap-6">
      <h1 className="text-[28px] font-semibold">Create proxy</h1>

      {/* While waiting for duplicate source to load, show skeleton */}
      {dupId > 0 && !source ? (
        <div className="space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-2/3" />
        </div>
      ) : (
        <>
          {selectedType === 'reverse_proxy' && (
            <ReverseProxyForm
              mode="create"
              typeControl={typeControl}
              initialData={seed}
              onSubmit={handleSubmit}
              loading={isCreating}
              onCancel={handleCancel}
            />
          )}
          {selectedType === 'redirect' && (
            <RedirectForm
              mode="create"
              typeControl={typeControl}
              initialData={seed}
              onSubmit={handleSubmit}
              loading={isCreating}
              onCancel={handleCancel}
            />
          )}
          {selectedType === 'static' && (
            <StaticForm
              mode="create"
              typeControl={typeControl}
              initialData={seed}
              onSubmit={handleSubmit}
              loading={isCreating}
              onCancel={handleCancel}
            />
          )}
        </>
      )}
    </div>
  );
}
