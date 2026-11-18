import { GeneratedConfig } from '@/components/proxy/generated-config';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { DetailRow } from '@/components/ui/detail-row';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { L4_MATCHER_CONFIG, type L4Proxy, type L4Upstream } from '@/types/l4-proxy';

const endpoint = (upstream: L4Upstream) =>
  `${upstream.host.includes(':') && !upstream.host.startsWith('[') ? `[${upstream.host}]` : upstream.host}:${upstream.port}`;
export function L4ProxyConfigCard({ proxy }: { proxy: L4Proxy }) {
  const routes = proxy.routes ?? [];
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <dl className="divide-y">
            <DetailRow label="Protocol">{proxy.protocol.toUpperCase()}</DetailRow>
            <DetailRow label="Listen port">
              <span className="font-mono">:{proxy.listen_port}</span>
            </DetailRow>
            <DetailRow label="Destination">
              <span className="font-mono">
                {routes[0]?.upstreams[0] ? endpoint(routes[0].upstreams[0]) : '—'}
              </span>
            </DetailRow>
            <DetailRow label="Routes">{routes.length}</DetailRow>
            {proxy.description && <DetailRow label="Description">{proxy.description}</DetailRow>}
          </dl>
          <div>
            <GeneratedConfig
              request={{ ...proxy, routes }}
              enabled
              saved
              trigger={<Button variant="outline">View generated config</Button>}
            />
          </div>
        </CardContent>
      </Card>
      <Card className="gap-0 overflow-hidden pb-0">
        <CardHeader className="mb-5">
          <CardTitle>Routes</CardTitle>
        </CardHeader>
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/40">
              <TableHead className="px-6">Priority</TableHead>
              <TableHead>Matcher</TableHead>
              <TableHead>Destinations</TableHead>
              <TableHead>TLS</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {routes.map((route) => (
              <TableRow key={route.id}>
                <TableCell className="px-6 py-4">{route.priority}</TableCell>
                <TableCell>
                  <div className="flex flex-col gap-1">
                    <span>
                      {L4_MATCHER_CONFIG[route.matcher_type]?.shortLabel ?? route.matcher_type}
                    </span>
                    {route.sni_hostnames?.length ||
                    route.allowed_ip_ranges?.length ||
                    route.regex_pattern ? (
                      <span className="font-mono text-xs text-muted-foreground">
                        {route.sni_hostnames?.join(', ') ||
                          route.allowed_ip_ranges?.join(', ') ||
                          route.regex_pattern}
                      </span>
                    ) : null}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex flex-col gap-1">
                    {route.upstreams.map((upstream) => (
                      <span className="font-mono text-sm" key={endpoint(upstream)}>
                        {endpoint(upstream)}
                        {upstream.weight && upstream.weight !== 1
                          ? ` · weight ${upstream.weight}`
                          : ''}
                      </span>
                    ))}
                  </div>
                </TableCell>
                <TableCell>
                  {route.tls_terminate
                    ? 'Terminate'
                    : route.tls_passthrough
                      ? 'Passthrough'
                      : 'Off'}
                </TableCell>
              </TableRow>
            ))}
            {!routes.length && (
              <TableRow>
                <TableCell colSpan={4} className="h-24 text-center text-muted-foreground">
                  No routes configured.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </Card>
    </>
  );
}
