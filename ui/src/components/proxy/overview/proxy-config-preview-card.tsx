import { Copy, Download, ExternalLink, FileCode2 } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter } from '@/components/ui/sheet';
import { useProxyConfigPreview } from '@/hooks/use-config-preview';
import { usePermissions } from '@/hooks/use-permissions';

export function ProxyConfigPreviewCard({ proxyId }: { proxyId: number }) {
  const { canReadCaddyConfig } = usePermissions();
  const [open, setOpen] = useState(false);
  const { data, isLoading, isError, refetch } = useProxyConfigPreview(
    open && canReadCaddyConfig ? proxyId : 0,
  );
  if (!canReadCaddyConfig) return null;
  const code = data ? JSON.stringify(data, null, 2) : '';
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileCode2 className="size-4 text-muted-foreground" />
            Generated configuration
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Button variant="outline" onClick={() => setOpen(true)}>
            <ExternalLink data-icon="inline-start" />
            View generated config
          </Button>
        </CardContent>
      </Card>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent
          className="overflow-hidden data-[side=right]:w-full data-[side=right]:sm:max-w-4xl"
          aria-describedby={undefined}
        >
          <SheetHeader className="shrink-0 pr-12">
            <SheetTitle>Generated configuration</SheetTitle>
          </SheetHeader>
          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-auto px-6 py-2">
            {isLoading ? (
              <p className="text-sm text-muted-foreground" role="status">
                Loading configuration…
              </p>
            ) : isError ? (
              <div className="flex flex-col items-start gap-3">
                <p className="text-sm text-destructive">Couldn’t load config.</p>
                <Button variant="outline" onClick={() => refetch()}>
                  Retry
                </Button>
              </div>
            ) : (
              <pre
                role="region"
                aria-label="Generated JSON configuration"
                tabIndex={0}
                className="min-h-0 min-w-0 flex-1 overflow-auto overscroll-contain rounded-md bg-muted p-4 font-mono text-xs leading-6"
              >
                {code}
              </pre>
            )}
          </div>
          <SheetFooter className="shrink-0 flex-row justify-end border-t">
            <Button
              variant="outline"
              disabled={!code}
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(code);
                  toast.success('Config copied');
                } catch {
                  toast.error('Could not copy config');
                }
              }}
            >
              <Copy data-icon="inline-start" />
              Copy
            </Button>
            <Button
              disabled={!code}
              onClick={() => {
                const url = URL.createObjectURL(new Blob([code], { type: 'application/json' }));
                const a = document.createElement('a');
                a.href = url;
                a.download = 'caddy-config.json';
                a.click();
                setTimeout(() => URL.revokeObjectURL(url), 1000);
              }}
            >
              <Download data-icon="inline-start" />
              Download
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  );
}
