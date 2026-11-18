import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { Button } from '@/components/ui/button';
import { LogStreamError } from '@/hooks/use-caddy-logs';
import type { CaddyLogLine, CaddyLogSource } from '@/types/caddy-logs';

import { LogRow } from './log-row';
import { LogToolbar } from './log-toolbar';

function matchesLevelFilter(line: CaddyLogLine, filter: string): boolean {
  if (filter === 'all') return true;
  return (line.level ?? 'info').toLowerCase() === filter;
}

function matchesStatusFilter(line: CaddyLogLine, filter: string): boolean {
  if (filter === 'all') return true;
  const status = line.status;
  if (status == null) return true;
  switch (filter) {
    case '2xx':
      return status >= 200 && status < 300;
    case '3xx':
      return status >= 300 && status < 400;
    case '4xx':
      return status >= 400 && status < 500;
    case '5xx':
      return status >= 500;
    default:
      return true;
  }
}

interface LogViewerProps {
  source: CaddyLogSource;
  lines: CaddyLogLine[];
  isStreaming: boolean;
  paused: boolean;
  error: Error | null;
  onPause: () => void;
  onResume: () => void;
  onClear: () => void;
}

export function LogViewer({
  source,
  lines,
  isStreaming,
  paused,
  error,
  onPause,
  onResume,
  onClear,
}: LogViewerProps) {
  const [search, setSearch] = useState('');
  const [levelFilter, setLevelFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const scrollRef = useRef<HTMLDivElement>(null);

  // Track whether the user has scrolled away from the bottom
  const isAtBottomRef = useRef(true);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
    isAtBottomRef.current = distanceFromBottom < 50;
  }, []);

  // Auto-scroll to bottom when new lines arrive, unless user scrolled up
  useEffect(() => {
    if (isAtBottomRef.current && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [lines]);

  useEffect(() => {
    setLevelFilter('all');
    setStatusFilter('all');
  }, [source]);

  const filteredLines = useMemo(() => {
    return lines.filter((line) => {
      if (search && !line.raw.toLowerCase().includes(search.toLowerCase())) return false;
      if (!matchesLevelFilter(line, levelFilter)) return false;
      if (!matchesStatusFilter(line, statusFilter)) return false;
      return true;
    });
  }, [lines, search, levelFilter, statusFilter]);

  return (
    <div className="flex flex-col gap-3">
      <LogToolbar
        source={source}
        paused={paused}
        search={search}
        onSearchChange={setSearch}
        levelFilter={levelFilter}
        onLevelFilterChange={setLevelFilter}
        statusFilter={statusFilter}
        onStatusFilterChange={setStatusFilter}
        onPause={onPause}
        onResume={onResume}
        onClear={onClear}
      />

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="relative h-[calc(100dvh-280px)] min-h-72 overflow-y-auto break-all rounded-lg border bg-card p-4 font-mono text-xs"
      >
        {lines.length === 0 ? (
          <p className="text-muted-foreground">No log lines yet.</p>
        ) : filteredLines.length === 0 ? (
          <p className="text-muted-foreground">No lines match the current filter.</p>
        ) : (
          filteredLines.map((line) => <LogRow key={line.id} line={line} />)
        )}
      </div>
      <div
        className="flex min-h-8 flex-wrap items-center gap-2 text-xs text-muted-foreground"
        role="status"
      >
        <span
          className={`size-1.5 rounded-full ${isStreaming ? 'bg-success' : error ? 'bg-warning' : 'bg-muted-foreground'}`}
        />
        <span>
          {isStreaming
            ? 'Live'
            : error instanceof LogStreamError && [401, 403].includes(error.status)
              ? error.message
              : error
                ? 'Reconnecting…'
                : paused
                  ? 'Paused'
                  : 'Connecting…'}
        </span>
        <span className="ml-auto">{filteredLines.length.toLocaleString()} lines</span>
        {error && !(error instanceof LogStreamError && [401, 403].includes(error.status)) && (
          <Button variant="ghost" size="xs" onClick={onResume}>
            Retry now
          </Button>
        )}
        {error instanceof LogStreamError && error.status === 401 && (
          <Button variant="ghost" size="xs" render={<a href="/login" />}>
            Sign in
          </Button>
        )}
      </div>
    </div>
  );
}
