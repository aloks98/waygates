export type TrafficRange = '1h' | '24h' | '7d';

export interface TrafficPoint {
  t: string;
  req_2xx: number;
  req_3xx: number;
  req_4xx: number;
  req_5xx: number;
  req_other: number;
  bytes_in: number;
  bytes_out: number;
  p50_ms: number;
  p95_ms: number;
  in_flight: number;
}

export interface TrafficSeries {
  range: TrafficRange;
  step_seconds: number;
  points: TrafficPoint[];
}
