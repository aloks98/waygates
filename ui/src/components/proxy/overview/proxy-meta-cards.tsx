import { Card, CardContent, CardHeader, CardTitle } from '@e412/rnui-react';

import { DetailRow } from '@/components/ui/detail-row';
import type { ProxyConfig } from '@/types/proxy';

function yesNo(v: boolean | undefined) {
  return v ? 'Yes' : 'No';
}

export function ProxyHttpsCard({ proxy }: { proxy: ProxyConfig }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>HTTPS / TLS</CardTitle>
      </CardHeader>
      <CardContent className="divide-y">
        <DetailRow label="HTTPS enabled">{yesNo(proxy.ssl_enabled)}</DetailRow>
        <DetailRow label="Force HTTPS">{yesNo(proxy.ssl_forced)}</DetailRow>
      </CardContent>
    </Card>
  );
}

export function ProxyDetailsCard({ proxy }: { proxy: ProxyConfig }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Details</CardTitle>
      </CardHeader>
      <CardContent className="divide-y">
        <DetailRow label="Description">{proxy.description || '—'}</DetailRow>
        <DetailRow label="Created">{new Date(proxy.created_at).toLocaleString()}</DetailRow>
        <DetailRow label="Updated">{new Date(proxy.updated_at).toLocaleString()}</DetailRow>
      </CardContent>
    </Card>
  );
}
