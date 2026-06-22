import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  JsonViewer,
  Skeleton,
} from '@e412/rnui-react';
import { Copy, RefreshCw } from 'lucide-react';

import { useCaddyConfig } from '@/hooks/use-config-preview';

export function CaddyConfigPage() {
  const { data, isLoading, isError, refetch } = useCaddyConfig();

  const handleCopy = () => {
    if (data) {
      navigator.clipboard.writeText(JSON.stringify(data, null, 2));
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Caddy Config</h1>
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
            <JsonViewer data={data} showLineNumbers defaultExpanded={1} collapseOn="click" />
          ) : (
            <p className="text-sm text-muted-foreground">No config available.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
