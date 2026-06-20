import { Badge } from '@e412/rnui-react';

import type { AuditStatus } from '@/types/audit';

interface StatusBadgeProps {
  status: AuditStatus;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const isSuccess = status === 'success';

  return (
    <Badge variant={isSuccess ? 'success' : 'destructive'}>
      {isSuccess ? 'Success' : 'Failed'}
    </Badge>
  );
}
