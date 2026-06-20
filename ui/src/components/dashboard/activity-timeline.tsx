import {
  Button,
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
  Skeleton,
  Timeline,
  TimelineContent,
  TimelineDate,
  TimelineIndicator,
  TimelineItem,
  TimelineSeparator,
  TimelineTitle,
} from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { formatDistanceToNow } from 'date-fns';
import { Activity, ArrowRight, CheckCircle2, XCircle } from 'lucide-react';

import { getActionColor, getActionLabel, getActivityLink } from '@/lib/dashboard-format';
import type { AuditLog } from '@/types/audit';

export function ActivityTimeline({
  activity,
  isLoading,
}: {
  activity: AuditLog[];
  isLoading: boolean;
}) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Recent activity</CardTitle>
        <CardAction>
          <Button variant="ghost" size="sm" render={<Link to="/dashboard/activity" />}>
            View all
            <ArrowRight className="ml-1 size-3.5" />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="pt-0">
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3, 4].map((i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : activity.length === 0 ? (
          <div className="flex flex-col items-center py-6 text-center">
            <Activity className="size-5 text-muted-foreground/40" />
            <p className="mt-2 text-sm text-muted-foreground">No recent activity</p>
            <p className="mt-1 max-w-[220px] text-xs text-muted-foreground/70">
              Creating proxies and changing settings will appear here.
            </p>
          </div>
        ) : (
          <Timeline orientation="vertical">
            {activity.map((log, i) => {
              const actor = log.user?.username ?? 'system';
              const link = getActivityLink(log);
              const failed = log.status === 'failure';
              const title = (
                <span className="text-sm leading-snug">
                  <span className="font-medium">{actor}</span>{' '}
                  <span className={getActionColor(log.action)}>{getActionLabel(log.action)}</span>
                  {log.resource_name && (
                    <span className="text-muted-foreground"> · {log.resource_name}</span>
                  )}
                </span>
              );
              return (
                <TimelineItem key={log.id} step={i + 1}>
                  <TimelineIndicator className="flex size-4 items-center justify-center border-0 bg-background">
                    {failed ? (
                      <XCircle className="size-4 text-destructive" />
                    ) : (
                      <CheckCircle2 className="size-4 text-green-500" />
                    )}
                  </TimelineIndicator>
                  <TimelineSeparator />
                  <TimelineContent>
                    <TimelineTitle>
                      {link ? (
                        <Link to={link} className="hover:underline">
                          {title}
                        </Link>
                      ) : (
                        title
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
        )}
      </CardContent>
    </Card>
  );
}
