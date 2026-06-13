import { Badge } from '@e412/titanium';
import { ArrowRight, FolderOpen, Globe } from 'lucide-react';
import type { ReactNode } from 'react';

import type { ProxyType } from '@/types/proxy';

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
