import { Button, Skeleton } from '@e412/rnui-react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { ArrowLeft, ArrowRight, Check, FolderOpen, Globe } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';

import { type ACLAssignment, RedirectForm, ReverseProxyForm, StaticForm } from '@/components/proxy';
import { useAssignACL } from '@/hooks';
import { useProxy, useProxies } from '@/hooks/use-proxies';
import type { CreateProxyRequest, ProxyConfig, ProxyType } from '@/types/proxy';

interface ProxyTypeOption {
  type: ProxyType;
  label: string;
  icon: JSX.Element;
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

  const renderTypePill = (option: ProxyTypeOption) => (
    <button
      key={option.type}
      type="button"
      onClick={() => setSelectedType(option.type)}
      className={`inline-flex items-center gap-2 px-3 py-1.5 rounded text-sm font-medium transition-colors border ${
        selectedType === option.type
          ? 'bg-primary text-primary-foreground border-primary'
          : 'bg-muted/50 text-muted-foreground border-transparent hover:bg-muted hover:text-foreground'
      }`}
    >
      {selectedType === option.type ? <Check className="size-3.5" /> : option.icon}
      {option.label}
    </button>
  );

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/proxies' })}>
          <ArrowLeft className="size-4" />
          <span className="sr-only">Back</span>
        </Button>
        <h1 className="text-2xl font-bold">Create Proxy</h1>
      </div>

      {/* Type switcher pills */}
      <div className="flex flex-wrap items-center gap-2">{proxyTypes.map(renderTypePill)}</div>

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
              initialData={seed}
              onSubmit={handleSubmit}
              loading={isCreating}
              onCancel={handleCancel}
            />
          )}
          {selectedType === 'redirect' && (
            <RedirectForm
              mode="create"
              initialData={seed}
              onSubmit={handleSubmit}
              loading={isCreating}
              onCancel={handleCancel}
            />
          )}
          {selectedType === 'static' && (
            <StaticForm
              mode="create"
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
