import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  JsonViewer,
  Skeleton,
} from '@e412/rnui-react';
import { Copy } from 'lucide-react';

import { useProxyConfigPreview } from '@/hooks/use-config-preview';

export function ProxyConfigPreviewCard({ proxyId }: { proxyId: number }) {
  const { data, isLoading, isError } = useProxyConfigPreview(proxyId);

  const handleCopy = () => {
    if (data) {
      navigator.clipboard.writeText(JSON.stringify(data, null, 2));
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle>Generated Caddy config</CardTitle>
        <Button variant="ghost" size="icon" onClick={handleCopy} disabled={!data}>
          <Copy className="size-4" />
          <span className="sr-only">Copy config</span>
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-4/6" />
          </div>
        ) : isError ? (
          <p className="text-sm text-muted-foreground">Couldn&apos;t load config preview.</p>
        ) : data ? (
          <JsonViewer data={data} showLineNumbers defaultExpanded={1} collapseOn="click" />
        ) : (
          <p className="text-sm text-muted-foreground">No config available.</p>
        )}
      </CardContent>
    </Card>
  );
}
