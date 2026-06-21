import {
  Badge,
  Skeleton,
  Timeline,
  TimelineContent,
  TimelineDate,
  TimelineIndicator,
  TimelineItem,
  TimelineSeparator,
  TimelineTitle,
} from '@e412/rnui-react';
import { formatDistanceToNow } from 'date-fns';
import { Activity } from 'lucide-react';

import type { AuditLog } from '@/types/audit';

import { getActionMeta, toneTextClass } from './activity-actions';
import { groupLogsByDay } from './activity-grouping';

export interface ActivityTimelineProps {
  logs: AuditLog[];
  isLoading: boolean;
  onSelect: (id: number) => void;
}

export function ActivityTimeline({ logs, isLoading, onSelect }: ActivityTimelineProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3, 4, 5].map((i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (logs.length === 0) {
    return (
      <div className="flex flex-col items-center py-16 text-center">
        <Activity className="size-8 text-muted-foreground/40" />
        <p className="mt-3 text-sm font-medium">No activity found</p>
        <p className="mt-1 max-w-[260px] text-xs text-muted-foreground">
          Try adjusting your filters or date range.
        </p>
      </div>
    );
  }

  const groups = groupLogsByDay(logs);

  return (
    <div className="space-y-6">
      {groups.map((group) => (
        <div key={group.key}>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {group.label}
          </h3>
          <Timeline orientation="vertical">
            {group.logs.map((log, i) => {
              const meta = getActionMeta(log.action);
              const Icon = meta.icon;
              const failed = log.status === 'failure';
              const actor = log.user?.username ?? 'system';
              return (
                <TimelineItem key={log.id} step={i + 1}>
                  <TimelineIndicator className="flex size-5 items-center justify-center border-0 bg-background">
                    <Icon
                      className={`size-4 ${failed ? 'text-destructive' : toneTextClass(meta.tone)}`}
                    />
                  </TimelineIndicator>
                  <TimelineSeparator />
                  <TimelineContent>
                    <TimelineTitle>
                      <button
                        type="button"
                        onClick={() => onSelect(log.id)}
                        className="text-left text-sm leading-snug hover:underline"
                      >
                        <span className="font-medium">{actor}</span>{' '}
                        <span className={failed ? 'text-destructive' : toneTextClass(meta.tone)}>
                          {meta.label}
                        </span>
                        {log.resource_name && (
                          <span className="text-muted-foreground"> · {log.resource_name}</span>
                        )}
                      </button>
                      {failed && (
                        <Badge variant="destructive" className="ml-2">
                          failed
                        </Badge>
                      )}
                    </TimelineTitle>
                    <TimelineDate>
                      {formatDistanceToNow(new Date(log.created_at), { addSuffix: true })}
                    </TimelineDate>
                  </TimelineContent>
                </TimelineItem>
              );
            })}
          </Timeline>
        </div>
      ))}
    </div>
  );
}
