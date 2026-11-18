import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { DetailRow } from '@/components/ui/detail-row';
import type { ProxyConfig } from '@/types/proxy';

import { getProxyTarget } from '../cells/proxy-target-cell';
export function ProxyConfigCard({ proxy }: { proxy: ProxyConfig }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
      </CardHeader>
      <CardContent className="divide-y">
        <dl>
          <DetailRow label="Type">
            {
              { reverse_proxy: 'Reverse proxy', redirect: 'Redirect', static: 'Static files' }[
                proxy.type
              ]
            }
          </DetailRow>
          <DetailRow label="Domain">
            <span className="font-mono text-xs">{proxy.hostname}</span>
          </DetailRow>
          <DetailRow label="Destination">
            <span className="font-mono text-xs">{getProxyTarget(proxy)}</span>
          </DetailRow>
          <DetailRow label="HTTPS">{proxy.ssl_enabled ? 'Enabled' : 'Disabled'}</DetailRow>
          <DetailRow label="Access">
            {proxy.acl_group_names?.join(', ') ||
              (proxy.acl_group_count ? `${proxy.acl_group_count} groups` : 'Public access')}
          </DetailRow>
          {proxy.type === 'redirect' && (
            <>
              <DetailRow label="Status code">{proxy.redirect?.status_code}</DetailRow>
              <DetailRow label="Preserve path">
                {proxy.redirect?.preserve_path ? 'Enabled' : 'Disabled'}
              </DetailRow>
              <DetailRow label="Preserve query">
                {proxy.redirect?.preserve_query ? 'Enabled' : 'Disabled'}
              </DetailRow>
            </>
          )}
          {proxy.type === 'static' && (
            <>
              <DetailRow label="Index file">
                <span className="font-mono text-xs">{proxy.static?.index_file}</span>
              </DetailRow>
              <DetailRow label="Directory browsing">
                {proxy.static?.browse ? 'Enabled' : 'Disabled'}
              </DetailRow>
            </>
          )}
          {proxy.description && <DetailRow label="Description">{proxy.description}</DetailRow>}
        </dl>
      </CardContent>
    </Card>
  );
}
