import { Badge, Tooltip, TooltipContent, TooltipTrigger } from '@e412/rnui-react';
import { Shield, ShieldOff } from 'lucide-react';

export function ProxyAclCell({ count, names }: { count?: number; names?: string[] }) {
  const n = count ?? 0;
  if (n === 0) {
    return (
      <span className="inline-flex items-center gap-1.5 text-muted-foreground">
        <ShieldOff className="size-3.5" />
        <span className="text-sm">Unprotected</span>
      </span>
    );
  }
  const list = names ?? [];
  return (
    <Tooltip>
      <TooltipTrigger render={<Badge variant="secondary" className="gap-1" />}>
        <Shield className="size-3.5" />
        {n}
      </TooltipTrigger>
      <TooltipContent className="max-w-xs whitespace-normal break-words">
        {list.length ? list.join(', ') : `${n} ACL group(s)`}
      </TooltipContent>
    </Tooltip>
  );
}
