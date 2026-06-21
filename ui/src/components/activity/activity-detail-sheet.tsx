import {
  Badge,
  JsonViewer,
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  Skeleton,
} from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { format } from 'date-fns';

import { useAuditLogById } from '@/hooks';
import { getActivityLink } from '@/lib/dashboard-format';

import { getActionMeta } from './activity-actions';
import { extractFieldChanges, formatDiffValue } from './activity-diff';

export interface ActivityDetailSheetProps {
  logId: number | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function MetaRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[120px_1fr] gap-2 py-1.5 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="break-words">{value}</span>
    </div>
  );
}

export function ActivityDetailSheet({ logId, open, onOpenChange }: ActivityDetailSheetProps) {
  const { log, isLoading } = useAuditLogById(logId ?? 0);
  const meta = log ? getActionMeta(log.action) : null;
  const changes = log ? extractFieldChanges(log.details) : [];
  const link = log ? getActivityLink(log) : null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-2xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {meta?.label ?? 'Event'}
            {log && (
              <Badge variant={log.status === 'failure' ? 'destructive' : 'secondary'}>
                {log.status}
              </Badge>
            )}
          </SheetTitle>
          <SheetDescription>{log ? log.action : 'Loading…'}</SheetDescription>
        </SheetHeader>

        {isLoading || !log ? (
          <div className="space-y-3 p-4">
            <Skeleton className="h-5 w-2/3" />
            <Skeleton className="h-24 w-full" />
          </div>
        ) : (
          <div className="space-y-6 p-4">
            <div className="border-y divide-y">
              <MetaRow label="User" value={log.user?.username ?? 'system'} />
              <MetaRow label="Time" value={format(new Date(log.created_at), 'PPpp')} />
              {log.resource_name && (
                <MetaRow
                  label="Resource"
                  value={
                    link ? (
                      <Link to={link} className="text-primary hover:underline">
                        {log.resource_name}
                      </Link>
                    ) : (
                      log.resource_name
                    )
                  }
                />
              )}
              {log.ip_address && <MetaRow label="IP address" value={log.ip_address} />}
              {log.user_agent && <MetaRow label="User agent" value={log.user_agent} />}
            </div>

            {changes.length > 0 ? (
              <div>
                <h3 className="mb-2 text-sm font-semibold">Changes</h3>
                <div className="space-y-2">
                  {changes.map((c) => (
                    <div key={c.field} className="rounded-md border p-2 text-sm">
                      <div className="font-medium">{c.field}</div>
                      <div className="mt-1 flex flex-wrap items-center gap-2">
                        <span className="text-muted-foreground line-through break-all">
                          {formatDiffValue(c.oldValue)}
                        </span>
                        <span className="text-muted-foreground">→</span>
                        <span className="text-green-600 dark:text-green-500 break-all">
                          {formatDiffValue(c.newValue)}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : log.details && Object.keys(log.details).length > 0 ? (
              <div>
                <h3 className="mb-2 text-sm font-semibold">Details</h3>
                <JsonViewer data={log.details} />
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No additional details.</p>
            )}

            {log.status === 'failure' && log.error_message && (
              <div className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
                {log.error_message}
              </div>
            )}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
