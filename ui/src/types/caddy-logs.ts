export type CaddyLogSource = 'runtime' | 'access';

export interface CaddyLogLine {
  id?: number;
  raw: string;
  ts?: number;
  level?: string;
  logger?: string;
  msg?: string;
  // access log fields (method/host/uri/remoteIp come from the nested `request` object)
  status?: number;
  method?: string;
  host?: string;
  uri?: string;
  remoteIp?: string;
  duration?: number;
}

export function parseCaddyLogLine(raw: string): CaddyLogLine {
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const line: CaddyLogLine = { raw };
    if (typeof parsed['ts'] === 'number') line.ts = parsed['ts'];
    if (typeof parsed['level'] === 'string') line.level = parsed['level'];
    if (typeof parsed['logger'] === 'string') line.logger = parsed['logger'];
    if (typeof parsed['msg'] === 'string') line.msg = parsed['msg'];
    if (typeof parsed['status'] === 'number') line.status = parsed['status'];
    if (typeof parsed['duration'] === 'number') line.duration = parsed['duration'];
    // Caddy access logs nest the request details under a `request` object;
    // only `status`/`duration` are top-level. `msg` is always "handled request".
    const req = parsed['request'];
    if (req !== null && typeof req === 'object') {
      const r = req as Record<string, unknown>;
      if (typeof r['method'] === 'string') line.method = r['method'];
      if (typeof r['host'] === 'string') line.host = r['host'];
      if (typeof r['uri'] === 'string') line.uri = r['uri'];
      if (typeof r['remote_ip'] === 'string') line.remoteIp = r['remote_ip'];
    }
    return line;
  } catch {
    return { raw };
  }
}
