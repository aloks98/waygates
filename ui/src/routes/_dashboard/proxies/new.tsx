import { Button } from '@e412/rnui-react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { ArrowLeft, ArrowRight, Check, FolderOpen, Globe } from 'lucide-react';
import { useState } from 'react';

import { type ACLAssignment, RedirectForm, ReverseProxyForm, StaticForm } from '@/components/proxy';
import { useAssignACL } from '@/hooks';
import { useProxies } from '@/hooks/use-proxies';
import type { CreateProxyRequest, ProxyType } from '@/types/proxy';

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
  const searchParams = useSearch({ strict: false }) as { type?: string };

  const validTypes: ProxyType[] = ['reverse_proxy', 'redirect', 'static'];
  const initialType = validTypes.includes(searchParams.type as ProxyType)
    ? (searchParams.type as ProxyType)
    : 'reverse_proxy';

  const [selectedType, setSelectedType] = useState<ProxyType>(initialType);

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
      navigate({ to: '/dashboard/proxies/$proxyId', params: { proxyId: String(proxyId) } });
    } else {
      navigate({ to: '/dashboard/proxies' });
    }
  };

  const handleCancel = () => {
    navigate({ to: '/dashboard/proxies' });
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
        <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/dashboard/proxies' })}>
          <ArrowLeft className="size-4" />
          <span className="sr-only">Back</span>
        </Button>
        <h1 className="text-2xl font-bold">Create Proxy</h1>
      </div>

      {/* Type switcher pills */}
      <div className="flex flex-wrap items-center gap-2">{proxyTypes.map(renderTypePill)}</div>

      {/* Forms */}
      {selectedType === 'reverse_proxy' && (
        <ReverseProxyForm onSubmit={handleSubmit} loading={isCreating} onCancel={handleCancel} />
      )}
      {selectedType === 'redirect' && (
        <RedirectForm onSubmit={handleSubmit} loading={isCreating} onCancel={handleCancel} />
      )}
      {selectedType === 'static' && (
        <StaticForm onSubmit={handleSubmit} loading={isCreating} onCancel={handleCancel} />
      )}
    </div>
  );
}
