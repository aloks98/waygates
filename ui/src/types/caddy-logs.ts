export type CaddyLogSource = 'runtime' | 'access';

export interface CaddyLogLine {
  id?: number;
  raw: string;
  ts?: number;
  level?: string;
  logger?: string;
  msg?: string;
  // access log fields
  status?: number;
  method?: string;
  host?: string;
  uri?: string;
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
    if (typeof parsed['method'] === 'string') line.method = parsed['method'];
    if (typeof parsed['host'] === 'string') line.host = parsed['host'];
    if (typeof parsed['uri'] === 'string') line.uri = parsed['uri'];
    if (typeof parsed['duration'] === 'number') line.duration = parsed['duration'];
    return line;
  } catch {
    return { raw };
  }
}
