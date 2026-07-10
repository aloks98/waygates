import { Badge } from '@e412/rnui-react';

export function ProxyGroupCell({ groupName }: { groupName?: string | null }) {
  if (!groupName) {
    return <span className="text-muted-foreground">—</span>;
  }
  return <Badge variant="secondary">{groupName}</Badge>;
}
