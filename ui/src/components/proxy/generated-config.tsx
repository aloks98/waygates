import { useQuery } from '@tanstack/react-query';
import { Copy, Download } from 'lucide-react';
import { useState, type ReactElement } from 'react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet';
import { usePermissions } from '@/hooks/use-permissions';
import {
  configPreviewErrorMessage,
  loadConfigPreview,
  type ConfigPreviewRequest,
} from '@/lib/config-preview';

import type { ACLAssignment } from './forms/acl-selector';

export function GeneratedConfig({
  request,
  enabled,
  acl = [],
  saved = false,
  trigger,
}: {
  request: ConfigPreviewRequest;
  enabled: boolean;
  acl?: ACLAssignment[];
  saved?: boolean;
  trigger: ReactElement;
}) {
  const [open, setOpen] = useState(false);
  const { canReadCaddyConfig } = usePermissions();
  const query = useQuery({
    queryKey: ['draft-config', request, acl, saved],
    queryFn: ({ signal }) => loadConfigPreview({ request, assignments: acl, saved, signal }),
    enabled: open && enabled && canReadCaddyConfig,
    retry: false,
  });
  if (!canReadCaddyConfig) return null;
  const code = query.data ? JSON.stringify(query.data, null, 2) : '';
  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger render={trigger} />
      <SheetContent
        className="overflow-hidden data-[side=right]:w-full data-[side=right]:sm:max-w-4xl"
        aria-describedby={undefined}
      >
        <SheetHeader className="shrink-0 pr-12">
          <SheetTitle>Generated config</SheetTitle>
        </SheetHeader>
        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-auto px-6 py-2">
          {!enabled ? (
            <p className="text-sm text-muted-foreground">
              Complete the proxy details to generate its config.
            </p>
          ) : query.isLoading ? (
            <p role="status" className="text-sm text-muted-foreground">
              Generating config…
            </p>
          ) : query.isError ? (
            <div className="flex flex-col items-start gap-3">
              <p role="alert" className="text-sm text-destructive">
                {configPreviewErrorMessage(query.error, 'protocol' in request)}
              </p>
              <Button variant="outline" onClick={() => query.refetch()}>
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
            type="button"
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
            type="button"
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
  );
}
