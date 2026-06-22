import type { CaddyLogLine } from '@/types/caddy-logs';

interface LogRowProps {
  line: CaddyLogLine;
}

function getLevelClass(level?: string): string {
  switch (level?.toLowerCase()) {
    case 'error':
      return 'text-destructive';
    case 'warn':
    case 'warning':
      return 'text-yellow-500';
    case 'debug':
      return 'text-muted-foreground';
    default:
      return 'text-foreground';
  }
}

function getStatusClass(status?: number): string {
  if (!status) return 'text-foreground';
  if (status >= 500) return 'text-destructive';
  if (status >= 400) return 'text-yellow-500';
  if (status >= 300) return 'text-blue-500';
  return 'text-green-600';
}

function formatTimestamp(ts?: number): string {
  if (!ts) return '';
  return new Date(ts * 1000).toISOString().replace('T', ' ').replace('Z', '');
}

function formatDuration(duration?: number): string {
  if (duration == null) return '';
  if (duration < 0.001) return `${(duration * 1_000_000).toFixed(0)}µs`;
  if (duration < 1) return `${(duration * 1000).toFixed(1)}ms`;
  return `${duration.toFixed(3)}s`;
}

export function LogRow({ line }: LogRowProps) {
  // Access log: Caddy access entries carry level=info + msg="handled request",
  // so detect them by the access logger name (or request-derived fields) FIRST —
  // otherwise they'd fall into the runtime branch and only show "handled request".
  const isAccess =
    line.logger?.startsWith('http.log.access') === true ||
    line.status != null ||
    line.method != null;

  if (isAccess) {
    const ts = formatTimestamp(line.ts);
    return (
      <div className="py-0.5">
        {ts && <span className="text-muted-foreground">{ts} </span>}
        {line.status != null && (
          <span className={`font-semibold ${getStatusClass(line.status)}`}>{line.status} </span>
        )}
        {line.method && <span className="font-semibold">{line.method} </span>}
        {line.host && <span className="text-muted-foreground">{line.host}</span>}
        {line.uri && <span>{line.uri} </span>}
        {line.remoteIp && <span className="text-muted-foreground">{line.remoteIp} </span>}
        {line.duration != null && (
          <span className="text-muted-foreground">{formatDuration(line.duration)}</span>
        )}
      </div>
    );
  }

  // Runtime log: has level or msg
  if (line.level != null || line.msg != null) {
    const ts = formatTimestamp(line.ts);
    const level = (line.level ?? 'info').toUpperCase();
    return (
      <div className={`py-0.5 ${getLevelClass(line.level)}`}>
        {ts && <span className="text-muted-foreground">{ts} </span>}
        <span className="font-semibold">[{level}]</span>
        {line.logger && <span className="text-muted-foreground"> {line.logger}</span>}
        {line.msg && <span> {line.msg}</span>}
      </div>
    );
  }

  // Unparsed: raw text
  return <div className="py-0.5 text-muted-foreground">{line.raw}</div>;
}
