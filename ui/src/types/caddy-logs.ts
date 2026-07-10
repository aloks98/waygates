export type CaddyLogSource = 'runtime' | 'access';

export interface CaddyLogLine {
  id?: number;
  raw: string;
  ts?: number;
  level?: string;
  logger?: string;
  msg?: string;
  // `msg` is a fixed string per call site, so the cause of a runtime failure
  // lives in `error`. ACME entries name the affected host in `identifier`,
  // and proxy failures name the backend in `upstream`.
  error?: string;
  identifier?: string;
  upstream?: string;
  // Request details. Access logs and most http.* runtime loggers nest these
  // under `request`; admin.api puts them at the top level.
  status?: number;
  method?: string;
  host?: string;
  uri?: string;
  remoteIp?: string;
  duration?: number;
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined;
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined;
}

export function parseCaddyLogLine(raw: string): CaddyLogLine {
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const line: CaddyLogLine = { raw };
    line.ts = asNumber(parsed['ts']);
    line.level = asString(parsed['level']);
    line.logger = asString(parsed['logger']);
    line.msg = asString(parsed['msg']);
    line.error = asString(parsed['error']);
    line.identifier = asString(parsed['identifier']);
    line.upstream = asString(parsed['upstream']);
    line.status = asNumber(parsed['status']);
    line.duration = asNumber(parsed['duration']);

    // Prefer the nested `request` object, falling back to top-level keys so
    // admin.api entries (which do not nest) keep their request details.
    const req = parsed['request'];
    const r = (req !== null && typeof req === 'object' ? req : parsed) as Record<string, unknown>;
    line.method = asString(r['method']);
    line.host = asString(r['host']);
    line.uri = asString(r['uri']);
    line.remoteIp = asString(r['remote_ip']);

    return line;
  } catch {
    return { raw };
  }
}

// Caddy names its access logger `http.log.access[.<server>]`. Classifying by
// logger rather than by the presence of a status code matters: http.log.error
// and http.handlers.reverse_proxy also carry a request and a status, and
// rendering those as access lines hides their level, message, and error. The
// `handled request` fallback covers a renamed access logger.
export function isAccessLogLine(line: CaddyLogLine): boolean {
  if (line.logger?.startsWith('http.log.access') === true) return true;
  return line.msg === 'handled request' && line.status != null;
}
