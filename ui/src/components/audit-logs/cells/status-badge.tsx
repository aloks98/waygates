import { Badge } from '@e412/titanium';
import { CheckCircle2, XCircle } from 'lucide-react';
import type { AuditStatus } from '@/types/audit';

interface StatusBadgeProps {
  status: AuditStatus;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const isSuccess = status === 'success';

  return (
    <Badge variant={isSuccess ? 'default' : 'destructive'} className="gap-1">
      {isSuccess ? <CheckCircle2 className="size-3" /> : <XCircle className="size-3" />}
      {isSuccess ? 'Success' : 'Failed'}
    </Badge>
  );
}
