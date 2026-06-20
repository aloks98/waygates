import { cn } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { Globe, Network } from 'lucide-react';

const tabs = [
  { key: 'http', label: 'HTTP', to: '/dashboard/proxies', icon: Globe },
  { key: 'tcp-udp', label: 'TCP/UDP', to: '/dashboard/proxies/tcp-udp', icon: Network },
] as const;

export function ProxiesTabs({ active }: { active: 'http' | 'tcp-udp' }) {
  return (
    <div className="bg-muted/60 inline-flex items-center gap-1 rounded-lg p-1">
      {tabs.map(({ key, label, to, icon: Icon }) => {
        const isActive = key === active;
        return (
          <Link
            key={key}
            to={to}
            aria-current={isActive ? 'page' : undefined}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              isActive
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            <Icon className="size-4" />
            {label}
          </Link>
        );
      })}
    </div>
  );
}
