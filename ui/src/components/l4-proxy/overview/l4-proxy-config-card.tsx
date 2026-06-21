import { Badge, Card, CardContent, CardHeader, CardTitle } from '@e412/rnui-react';

import { DetailRow } from '@/components/ui/detail-row';
import { L4_MATCHER_CONFIG, type L4LoadBalancingPolicy, type L4Proxy } from '@/types/l4-proxy';

const LB_POLICY_LABELS: Record<L4LoadBalancingPolicy, string> = {
  round_robin: 'Round Robin',
  least_conn: 'Least Connections',
  random: 'Random',
  first: 'First Available',
  ip_hash: 'Sticky (IP Hash)',
};

export function L4ProxyConfigCard({ proxy }: { proxy: L4Proxy }) {
  const routes = proxy.routes ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
      </CardHeader>
      <CardContent className="divide-y">
        <DetailRow label="Listen port">
          <span className="font-mono text-xs">:{proxy.listen_port}</span>
        </DetailRow>
        <DetailRow label="Protocol">
          <Badge variant="outline">{proxy.protocol.toUpperCase()}</Badge>
        </DetailRow>
        {routes.length > 0 && (
          <DetailRow label="Routes">
            <div className="flex flex-col gap-3">
              {routes.map((route) => {
                const matcherConfig = L4_MATCHER_CONFIG[route.matcher_type];
                const tlsMode = route.tls_terminate
                  ? 'Terminate'
                  : route.tls_passthrough
                    ? 'Passthrough'
                    : null;

                return (
                  <div key={route.id} className="flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <Badge variant="secondary" className="text-xs">
                        {matcherConfig?.shortLabel ?? route.matcher_type}
                      </Badge>
                      {tlsMode && (
                        <Badge variant="outline" className="text-xs">
                          TLS {tlsMode}
                        </Badge>
                      )}
                    </div>
                    {route.upstreams.length > 0 && (
                      <div className="flex flex-col gap-0.5 pl-1">
                        {route.upstreams.map((u) => (
                          <span
                            key={`${u.host}:${u.port}`}
                            className="font-mono text-xs text-muted-foreground"
                          >
                            {u.host}:{u.port}
                            {u.weight !== undefined && u.weight !== 1 && (
                              <span className="ml-1 text-muted-foreground/60">
                                (weight {u.weight})
                              </span>
                            )}
                          </span>
                        ))}
                      </div>
                    )}
                    <div className="flex flex-wrap gap-x-4 gap-y-0.5 pl-1 text-xs text-muted-foreground">
                      <span>
                        Load balancing:{' '}
                        {LB_POLICY_LABELS[route.load_balancing_policy] ??
                          route.load_balancing_policy}
                      </span>
                      <span>Priority: {route.priority}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </DetailRow>
        )}
        {routes.length === 0 && (
          <DetailRow label="Routes">
            <span className="text-muted-foreground">No routes configured</span>
          </DetailRow>
        )}
      </CardContent>
    </Card>
  );
}
