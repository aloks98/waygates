import { Badge, Card, CardContent, CardHeader, CardTitle, Skeleton } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { AlertTriangle, ShieldOff } from 'lucide-react';

import { useProxyACL } from '@/hooks';

export function ProxyAccessCard({ proxyId }: { proxyId: number }) {
  const { assignments, isLoading, isError } = useProxyACL(proxyId);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Access — what protects this</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-12 w-full" />
        ) : isError ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <AlertTriangle className="size-4" />
            Couldn&apos;t load access info.
          </div>
        ) : assignments.length === 0 ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <ShieldOff className="size-4" />
            Unprotected &middot;{' '}
            <Link to="/access" className="text-primary hover:underline">
              Configure access
            </Link>
          </div>
        ) : (
          <ul className="space-y-2">
            {assignments.map((a) => (
              <li key={a.acl_group_id} className="flex items-center justify-between gap-3 text-sm">
                <Link
                  to="/access/$groupId"
                  params={{ groupId: String(a.acl_group_id) }}
                  className="font-medium text-primary hover:underline"
                >
                  {a.acl_group?.name ?? `Group #${a.acl_group_id}`}
                </Link>
                <span className="text-muted-foreground font-mono text-xs">{a.path_pattern}</span>
                <span className="text-muted-foreground text-xs">#{a.priority}</span>
                <Badge variant={a.enabled ? 'default' : 'secondary'}>
                  {a.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
