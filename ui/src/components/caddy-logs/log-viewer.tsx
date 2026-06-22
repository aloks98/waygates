import { Alert, AlertDescription, Button } from '@e412/rnui-react';
import { AlertCircle } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

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
  error: Error | null;
  onPause: () => void;
  onResume: () => void;
  onClear: () => void;
}

export function LogViewer({
  source,
  lines,
  isStreaming,
  error,
  onPause,
  onResume,
  onClear,
}: LogViewerProps) {
  const [search, setSearch] = useState('');
  const [levelFilter, setLevelFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
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
    if (isAtBottomRef.current && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth', block: 'end' });
    }
  }, [lines]);

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
        isStreaming={isStreaming}
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

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="size-4" />
          <AlertDescription className="flex items-center justify-between gap-2">
            <span>Stream error: {error.message}</span>
            <Button size="sm" variant="outline" onClick={onResume}>
              Retry
            </Button>
          </AlertDescription>
        </Alert>
      )}

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="relative h-[600px] overflow-y-auto rounded border bg-muted/30 p-3 font-mono text-xs"
      >
        {lines.length === 0 ? (
          <p className="text-muted-foreground">No log lines yet.</p>
        ) : filteredLines.length === 0 ? (
          <p className="text-muted-foreground">No lines match the current filter.</p>
        ) : (
          filteredLines.map((line) => <LogRow key={line.id} line={line} />)
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
