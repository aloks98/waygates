import { Badge } from '@e412/titanium';
import { Globe, ArrowRight, FolderOpen } from 'lucide-react';
import type { ProxyType } from '@/../types/proxy';

const proxyTypeConfig: Record<ProxyType, { label: string; icon: React.ReactNode }> = {
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

export function getProxyTypeIcon(type: ProxyType): React.ReactNode {
  return proxyTypeConfig[type].icon;
}
