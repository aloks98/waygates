import { Copy, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';

import { SettingsNav } from '@/components/settings/settings-nav';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useCaddyConfig } from '@/hooks/use-config-preview';

export function CaddyConfigPage() {
  const { data, isLoading, isError, refetch } = useCaddyConfig();

  const handleCopy = async () => {
    if (data) {
      try {
        await navigator.clipboard.writeText(JSON.stringify(data, null, 2));
        toast.success('Config copied');
      } catch {
        toast.error('Could not copy config');
      }
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-[28px] font-semibold">Settings</h1>
      <div className="flex flex-col gap-6 md:flex-row">
        <SettingsNav />
        <div className="flex min-w-0 flex-1 flex-col gap-6">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-semibold">Generated configuration</h2>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="icon" onClick={() => refetch()} disabled={isLoading}>
                <RefreshCw className="size-4" />
                <span className="sr-only">Refresh config</span>
              </Button>
              <Button variant="outline" size="icon" onClick={handleCopy} disabled={!data}>
                <Copy className="size-4" />
                <span className="sr-only">Copy config</span>
              </Button>
            </div>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Generated configuration</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-5/6" />
                  <Skeleton className="h-4 w-4/6" />
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                </div>
              ) : isError ? (
                <p className="text-sm text-muted-foreground">Couldn&apos;t load Caddy config.</p>
              ) : data ? (
                <pre className="max-h-[65vh] overflow-auto rounded-md bg-muted p-4 font-mono text-xs leading-6">
                  {JSON.stringify(data, null, 2)}
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground">No config available.</p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
