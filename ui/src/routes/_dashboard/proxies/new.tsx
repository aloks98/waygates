import { Button, Separator } from '@e412/titanium';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { ArrowLeft, ArrowRight, Check, FolderOpen, Globe, Network } from 'lucide-react';
import { useState } from 'react';
import { L4ProxyForm } from '@/components/l4-proxy';
import { type ACLAssignment, RedirectForm, ReverseProxyForm, StaticForm } from '@/components/proxy';
import { useAssignACL } from '@/hooks';
import { useL4Proxies } from '@/hooks/use-l4-proxies';
import { useProxies } from '@/hooks/use-proxies';
import type { CreateL4ProxyRequest, L4Protocol, L4Proxy } from '@/types/l4-proxy';
import type { CreateProxyRequest, ProxyType } from '@/types/proxy';

// Combined type for all proxy types including L4
type CombinedProxyType = ProxyType | 'tcp' | 'udp';

interface ProxyTypeOption {
  type: CombinedProxyType;
  label: string;
  icon: JSX.Element;
  layer: 'L7' | 'L4';
}

const proxyTypes: ProxyTypeOption[] = [
  // L7 (HTTP) proxies
  {
    type: 'reverse_proxy',
    label: 'Reverse Proxy',
    icon: <Globe className="size-3.5" />,
    layer: 'L7',
  },
  {
    type: 'redirect',
    label: 'Redirect',
    icon: <ArrowRight className="size-3.5" />,
    layer: 'L7',
  },
  {
    type: 'static',
    label: 'Static File Server',
    icon: <FolderOpen className="size-3.5" />,
    layer: 'L7',
  },
  // L4 (TCP/UDP) proxies
  {
    type: 'tcp',
    label: 'TCP Proxy',
    icon: <Network className="size-3.5" />,
    layer: 'L4',
  },
  {
    type: 'udp',
    label: 'UDP Proxy',
    icon: <Network className="size-3.5" />,
    layer: 'L4',
  },
];

const l7Types = proxyTypes.filter((t) => t.layer === 'L7');
const l4Types = proxyTypes.filter((t) => t.layer === 'L4');

function isL4Type(type: CombinedProxyType): type is 'tcp' | 'udp' {
  return type === 'tcp' || type === 'udp';
}

export function ProxyCreatePage() {
  const navigate = useNavigate();
  const searchParams = useSearch({ strict: false }) as { type?: string };

  // Support both L7 and L4 types from URL param
  const validTypes: CombinedProxyType[] = ['reverse_proxy', 'redirect', 'static', 'tcp', 'udp'];
  const initialType = validTypes.includes(searchParams.type as CombinedProxyType)
    ? (searchParams.type as CombinedProxyType)
    : 'reverse_proxy';

  const [selectedType, setSelectedType] = useState<CombinedProxyType>(initialType);

  const { create: createProxy, isCreating: isCreatingProxy } = useProxies();
  const { create: createL4Proxy, isCreating: isCreatingL4Proxy } = useL4Proxies();
  const { assignACL } = useAssignACL();

  const handleL7Submit = async (data: CreateProxyRequest, aclAssignments?: ACLAssignment[]) => {
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

  const handleL4Submit = async (data: CreateL4ProxyRequest) => {
    const response = await createL4Proxy(data);
    const l4ProxyId = response.data?.id;

    if (l4ProxyId) {
      navigate({
        to: '/dashboard/l4-proxies/$l4ProxyId',
        params: { l4ProxyId: String(l4ProxyId) },
      });
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
      className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium transition-colors border ${
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
      <div className="space-y-3">
        {/* L7 HTTP Proxies */}
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-10">
            L7
          </span>
          {l7Types.map(renderTypePill)}
        </div>

        <Separator />

        {/* L4 TCP/UDP Proxies */}
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-10">
            L4
          </span>
          {l4Types.map(renderTypePill)}
        </div>
      </div>

      {/* L7 Forms */}
      {selectedType === 'reverse_proxy' && (
        <ReverseProxyForm
          onSubmit={handleL7Submit}
          loading={isCreatingProxy}
          onCancel={handleCancel}
        />
      )}
      {selectedType === 'redirect' && (
        <RedirectForm onSubmit={handleL7Submit} loading={isCreatingProxy} onCancel={handleCancel} />
      )}
      {selectedType === 'static' && (
        <StaticForm onSubmit={handleL7Submit} loading={isCreatingProxy} onCancel={handleCancel} />
      )}

      {/* L4 Forms */}
      {isL4Type(selectedType) && (
        <L4ProxyForm
          initialData={{ protocol: selectedType as L4Protocol } as Partial<L4Proxy> as L4Proxy}
          onSubmit={handleL4Submit}
          loading={isCreatingL4Proxy}
          onCancel={handleCancel}
        />
      )}
    </div>
  );
}
