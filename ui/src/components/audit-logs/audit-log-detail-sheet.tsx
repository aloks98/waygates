import {
  Badge,
  ScrollArea,
  Separator,
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  Skeleton,
} from '@e412/titanium';
import { format } from 'date-fns';
import { AlertCircle, ArrowRight, Clock, Globe, Monitor, Server, User } from 'lucide-react';
import type { ReactNode } from 'react';
import { useAuditLogById } from '@/hooks';
import type { AuditLog } from '@/types/audit';
import { ActionBadge, StatusBadge } from './cells';

interface AuditLogDetailSheetProps {
  logId: number | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface DetailRowProps {
  icon: ReactNode;
  label: string;
  children: ReactNode;
}

function DetailRow({ icon, label, children }: DetailRowProps) {
  return (
    <div className="flex items-start gap-3 py-3">
      <div className="text-muted-foreground mt-0.5">{icon}</div>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-muted-foreground mb-1">{label}</p>
        <div className="text-sm font-medium">{children}</div>
      </div>
    </div>
  );
}

interface ChangeItemProps {
  field: string;
  oldValue: unknown;
  newValue: unknown;
  context?: Record<string, unknown>;
  contextLabel?: string;
}

function isComplexValue(value: unknown): boolean {
  return typeof value === 'object' && value !== null;
}

function formatSimpleValue(value: unknown): string {
  if (value === null || value === undefined) return '-';
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
  return String(value);
}

function formatContextValue(value: unknown): string {
  if (Array.isArray(value)) {
    return value.join(', ');
  }
  if (typeof value === 'object' && value !== null) {
    return JSON.stringify(value);
  }
  return String(value);
}

function SimpleChangeItem({ field, oldValue, newValue, context, contextLabel }: ChangeItemProps) {
  return (
    <div className="rounded-md border bg-muted/30 p-3">
      <p className="text-sm font-medium text-foreground mb-2 capitalize">
        {field.replace(/_/g, ' ')}
      </p>
      <div className="flex items-center gap-2 text-sm">
        <span className="text-muted-foreground line-through">{formatSimpleValue(oldValue)}</span>
        <ArrowRight className="size-3 text-muted-foreground flex-shrink-0" />
        <span className="text-foreground font-medium">{formatSimpleValue(newValue)}</span>
      </div>
      {context && Object.keys(context).length > 0 && (
        <div className="mt-3 pt-3 border-t border-border/50">
          <p className="text-xs text-muted-foreground mb-2">{contextLabel || 'Configuration'}:</p>
          <div className="space-y-1">
            {Object.entries(context).map(([key, value]) => (
              <div key={key} className="flex items-start gap-2 text-xs">
                <span className="text-muted-foreground capitalize">{key.replace(/_/g, ' ')}:</span>
                <span className="text-foreground font-medium">{formatContextValue(value)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function ComplexChangeItem({ field, oldValue, newValue }: ChangeItemProps) {
  return (
    <div className="rounded-md border bg-muted/30 p-3">
      <p className="text-sm font-medium text-foreground mb-3 capitalize">
        {field.replace(/_/g, ' ')}
      </p>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <p className="text-xs text-muted-foreground mb-1">Before</p>
          <pre className="text-xs bg-background/50 p-2 rounded overflow-auto max-h-32">
            {oldValue ? JSON.stringify(oldValue, null, 2) : '-'}
          </pre>
        </div>
        <div>
          <p className="text-xs text-muted-foreground mb-1">After</p>
          <pre className="text-xs bg-background/50 p-2 rounded overflow-auto max-h-32">
            {newValue ? JSON.stringify(newValue, null, 2) : '-'}
          </pre>
        </div>
      </div>
    </div>
  );
}

function ChangeItem({ field, oldValue, newValue, context, contextLabel }: ChangeItemProps) {
  if (isComplexValue(oldValue) || isComplexValue(newValue)) {
    return <ComplexChangeItem field={field} oldValue={oldValue} newValue={newValue} />;
  }
  return (
    <SimpleChangeItem
      field={field}
      oldValue={oldValue}
      newValue={newValue}
      context={context}
      contextLabel={contextLabel}
    />
  );
}

interface ChangeEntry {
  old: unknown;
  new: unknown;
  with_config?: Record<string, unknown>;
  was_config?: Record<string, unknown>;
}

interface ChangesDisplayProps {
  changes: Record<string, ChangeEntry>;
}

function ChangesDisplay({ changes }: ChangesDisplayProps) {
  const entries = Object.entries(changes);

  if (entries.length === 0) {
    return <p className="text-sm text-muted-foreground">No changes recorded</p>;
  }

  // Extract context from change entry
  const getContext = (
    change: ChangeEntry,
  ): { context?: Record<string, unknown>; label?: string } => {
    if (change.with_config && Object.keys(change.with_config).length > 0) {
      return { context: change.with_config, label: 'Enabled configuration' };
    }
    if (change.was_config && Object.keys(change.was_config).length > 0) {
      return { context: change.was_config, label: 'Previous configuration' };
    }
    return {};
  };

  // Separate simple and complex changes
  const simpleChanges = entries.filter(
    ([, { old: o, new: n }]) => !isComplexValue(o) && !isComplexValue(n),
  );
  const complexChanges = entries.filter(
    ([, { old: o, new: n }]) => isComplexValue(o) || isComplexValue(n),
  );

  return (
    <div className="space-y-3">
      {simpleChanges.length > 0 && (
        <div className="grid gap-2 sm:grid-cols-2">
          {simpleChanges.map(([field, change]) => {
            const { context, label } = getContext(change);
            return (
              <ChangeItem
                key={field}
                field={field}
                oldValue={change.old}
                newValue={change.new}
                context={context}
                contextLabel={label}
              />
            );
          })}
        </div>
      )}
      {complexChanges.length > 0 && (
        <div className="space-y-2">
          {complexChanges.map(([field, change]) => {
            const { context, label } = getContext(change);
            return (
              <ChangeItem
                key={field}
                field={field}
                oldValue={change.old}
                newValue={change.new}
                context={context}
                contextLabel={label}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}

interface DetailsDisplayProps {
  details: Record<string, unknown>;
  action: string;
}

function DetailsDisplay({ details, action }: DetailsDisplayProps) {
  // Handle changes specially for update actions
  if (action.includes('update') && details.changes) {
    const changes = details.changes as Record<string, { old: unknown; new: unknown }>;
    const otherDetails = { ...details };
    delete otherDetails.changes;

    return (
      <div className="space-y-4">
        <div>
          <h4 className="text-sm font-medium mb-3">Changes Made</h4>
          <ChangesDisplay changes={changes} />
        </div>
        {Object.keys(otherDetails).length > 0 && (
          <div>
            <h4 className="text-sm font-medium mb-3">Additional Details</h4>
            <pre className="rounded-md bg-muted p-3 text-xs overflow-auto max-h-48">
              {JSON.stringify(otherDetails, null, 2)}
            </pre>
          </div>
        )}
      </div>
    );
  }

  // For non-update actions, show all details as formatted JSON
  return (
    <pre className="rounded-md bg-muted p-3 text-xs overflow-auto max-h-64">
      {JSON.stringify(details, null, 2)}
    </pre>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6 p-1">
      <div className="flex items-center gap-3">
        <Skeleton className="h-6 w-20 rounded-full" />
        <Skeleton className="h-6 w-16 rounded-full" />
      </div>
      <div className="space-y-4">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-start gap-3 py-3">
            <Skeleton className="size-5 rounded" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-5 w-40" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function ErrorState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <AlertCircle className="size-12 text-destructive/50" />
      <h3 className="mt-4 text-lg font-medium">Failed to load audit log</h3>
      <p className="mt-2 text-sm text-muted-foreground max-w-sm">{message}</p>
    </div>
  );
}

function AuditLogContent({ log }: { log: AuditLog }) {
  const formattedDate = format(new Date(log.created_at), 'EEEE, MMMM d, yyyy');
  const formattedTime = format(new Date(log.created_at), 'h:mm:ss a zzz');

  return (
    <div className="space-y-1">
      {/* Header with badges */}
      <div className="flex flex-wrap items-center gap-2 pb-4">
        <ActionBadge action={log.action} />
        <StatusBadge status={log.status} />
        {log.resource_type && (
          <Badge variant="outline" className="text-xs capitalize">
            {log.resource_type}
          </Badge>
        )}
      </div>

      <Separator />

      {/* Main details */}
      <div className="divide-y">
        <DetailRow icon={<Clock className="size-4" />} label="Timestamp">
          <div>
            <p>{formattedDate}</p>
            <p className="text-muted-foreground font-normal">{formattedTime}</p>
          </div>
        </DetailRow>

        {(log.resource_type || log.resource_name || log.resource_id) && (
          <DetailRow icon={<Server className="size-4" />} label="Resource">
            <div className="space-y-1">
              {log.resource_name && <p>{log.resource_name}</p>}
              <div className="flex flex-wrap gap-2 text-muted-foreground font-normal">
                {log.resource_type && <span className="capitalize">{log.resource_type}</span>}
                {log.resource_id && <span>ID: {log.resource_id}</span>}
              </div>
            </div>
          </DetailRow>
        )}

        <DetailRow icon={<User className="size-4" />} label="User">
          {log.user ? (
            <div>
              <p>{log.user.username}</p>
              <p className="text-muted-foreground font-normal">{log.user.email}</p>
            </div>
          ) : log.user_id ? (
            <p className="text-muted-foreground">User #{log.user_id}</p>
          ) : (
            <p className="text-muted-foreground italic">System</p>
          )}
        </DetailRow>

        {log.ip_address && (
          <DetailRow icon={<Globe className="size-4" />} label="IP Address">
            <code className="text-xs bg-muted px-2 py-1 rounded">{log.ip_address}</code>
          </DetailRow>
        )}

        {log.user_agent && (
          <DetailRow icon={<Monitor className="size-4" />} label="User Agent">
            <p className="text-xs text-muted-foreground font-normal break-all">{log.user_agent}</p>
          </DetailRow>
        )}
      </div>

      {/* Error message for failed actions */}
      {log.status === 'failure' && log.error_message && (
        <>
          <Separator className="my-4" />
          <div className="rounded-md bg-destructive/10 border border-destructive/20 p-4">
            <div className="flex items-start gap-3">
              <AlertCircle className="size-5 text-destructive mt-0.5" />
              <div>
                <p className="text-sm font-medium text-destructive">Error Message</p>
                <p className="text-sm text-destructive/80 mt-1">{log.error_message}</p>
              </div>
            </div>
          </div>
        </>
      )}

      {/* Details section */}
      {log.details && Object.keys(log.details).length > 0 && (
        <>
          <Separator className="my-4" />
          <div>
            <h3 className="text-sm font-medium mb-4">Details</h3>
            <DetailsDisplay details={log.details} action={log.action} />
          </div>
        </>
      )}
    </div>
  );
}

export function AuditLogDetailSheet({ logId, open, onOpenChange }: AuditLogDetailSheetProps) {
  const { log, isLoading, isError, error } = useAuditLogById(logId ?? 0);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle>Audit Log Details</SheetTitle>
        </SheetHeader>
        <ScrollArea className="h-[calc(100vh-8rem)] pr-4">
          {isLoading ? (
            <LoadingSkeleton />
          ) : isError ? (
            <ErrorState message={error?.message ?? 'An unknown error occurred'} />
          ) : log ? (
            <AuditLogContent log={log} />
          ) : null}
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
