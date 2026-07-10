import { Card, CardContent, CardHeader, CardTitle } from '@e412/rnui-react';

import { DetailRow } from '@/components/ui/detail-row';
import type { ProxyConfig } from '@/types/proxy';

function yesNo(v: boolean | undefined) {
  return v ? 'Yes' : 'No';
}

export function ProxyConfigCard({ proxy }: { proxy: ProxyConfig }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
      </CardHeader>
      <CardContent className="divide-y">
        {proxy.type === 'reverse_proxy' && (
          <>
            <DetailRow label="Upstreams">
              <div className="flex flex-col gap-0.5">
                {proxy.upstreams?.map((u) => (
                  <span key={`${u.scheme}://${u.host}:${u.port}`} className="font-mono text-xs">
                    {u.scheme}://{u.host}:{u.port}
                  </span>
                ))}
              </div>
            </DetailRow>
            <DetailRow label="Load balancing">{proxy.load_balancing?.strategy ?? '—'}</DetailRow>
            {/* raw block_exploits/tls_insecure_skip_verify are nullable (inherit) —
                effective is the already-resolved value actually served. */}
            <DetailRow label="Block exploits">{yesNo(proxy.effective?.block_exploits)}</DetailRow>
            <DetailRow label="Skip TLS verify">
              {yesNo(proxy.effective?.tls_insecure_skip_verify)}
            </DetailRow>
            {proxy.custom_headers?.request &&
              Object.keys(proxy.custom_headers.request).length > 0 && (
                <DetailRow label="Request headers">
                  <div className="flex flex-col gap-0.5">
                    {Object.entries(proxy.custom_headers.request).map(([key, value]) => (
                      <span key={key} className="font-mono text-xs">
                        {key}: {value}
                      </span>
                    ))}
                  </div>
                </DetailRow>
              )}
            {proxy.custom_headers?.response &&
              Object.keys(proxy.custom_headers.response).length > 0 && (
                <DetailRow label="Response headers">
                  <div className="flex flex-col gap-0.5">
                    {Object.entries(proxy.custom_headers.response).map(([key, value]) => (
                      <span key={key} className="font-mono text-xs">
                        {key}: {value}
                      </span>
                    ))}
                  </div>
                </DetailRow>
              )}
          </>
        )}
        {proxy.type === 'redirect' && (
          <>
            <DetailRow label="Target">{proxy.redirect?.target}</DetailRow>
            <DetailRow label="Status code">{proxy.redirect?.status_code}</DetailRow>
            <DetailRow label="Preserve path">{yesNo(proxy.redirect?.preserve_path)}</DetailRow>
            <DetailRow label="Preserve query">{yesNo(proxy.redirect?.preserve_query)}</DetailRow>
          </>
        )}
        {proxy.type === 'static' && (
          <>
            <DetailRow label="Root path">{proxy.static?.root_path}</DetailRow>
            <DetailRow label="Index file">{proxy.static?.index_file}</DetailRow>
            <DetailRow label="Directory browse">{yesNo(proxy.static?.browse)}</DetailRow>
            <DetailRow label="Template rendering">
              {yesNo(proxy.static?.template_rendering)}
            </DetailRow>
            {proxy.static?.try_files && proxy.static.try_files.length > 0 && (
              <DetailRow label="Try files">{proxy.static.try_files.join(' ')}</DetailRow>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
