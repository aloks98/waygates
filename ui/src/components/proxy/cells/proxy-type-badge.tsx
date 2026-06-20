import { Badge } from '@e412/rnui-react';
import { ArrowRight, FolderOpen, Globe, Network, Radio } from 'lucide-react';
import type { ReactNode } from 'react';

import type { L4Protocol } from '@/types/l4-proxy';
import type { ProxyType } from '@/types/proxy';

// HTTP (L7) proxy type configuration
const proxyTypeConfig: Record<ProxyType, { label: string; icon: ReactNode }> = {
  reverse_proxy: {
    label: 'Reverse Proxy',
    icon: <Globe className="size-4" />,
  },
  redirect: {
    label: 'Redirect',
    icon: <ArrowRight className="size-4" />,
  },
  static: {
    label: 'Static File Server',
    icon: <FolderOpen className="size-4" />,
  },
};

// L4 proxy protocol configuration
const l4ProtocolConfig: Record<L4Protocol, { label: string; icon: ReactNode }> = {
  tcp: {
    label: 'TCP',
    icon: <Network className="size-4" />,
  },
  udp: {
    label: 'UDP',
    icon: <Radio className="size-4" />,
  },
};

interface ProxyTypeBadgeProps {
  type: ProxyType;
}

export function ProxyTypeBadge({ type }: ProxyTypeBadgeProps) {
  const config = proxyTypeConfig[type];

  return (
    <Badge variant="secondary" className="gap-1">
      {config.icon}
      {config.label}
    </Badge>
  );
}

export function getProxyTypeLabel(type: ProxyType): string {
  return proxyTypeConfig[type].label;
}

export function getProxyTypeIcon(type: ProxyType): ReactNode {
  return proxyTypeConfig[type].icon;
}

// L4 protocol badge and helpers
interface L4ProtocolBadgeProps {
  protocol: L4Protocol;
}

export function L4ProtocolBadge({ protocol }: L4ProtocolBadgeProps) {
  const config = l4ProtocolConfig[protocol];

  return (
    <Badge variant="secondary" className="gap-1">
      {config.icon}
      {config.label}
    </Badge>
  );
}

export function getL4ProtocolLabel(protocol: L4Protocol): string {
  return l4ProtocolConfig[protocol].label;
}

export function getL4ProtocolIcon(protocol: L4Protocol): ReactNode {
  return l4ProtocolConfig[protocol].icon;
}
