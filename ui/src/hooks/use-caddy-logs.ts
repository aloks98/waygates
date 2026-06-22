import { useCallback, useEffect, useRef, useState } from 'react';

import { useAuthStore } from '@/stores/auth';
import { type CaddyLogLine, type CaddyLogSource, parseCaddyLogLine } from '@/types/caddy-logs';

const BUFFER_CAP = 3000;

// Monotonic counter for stable React keys across buffer rotations and filter changes.
let lineIdCounter = 0;

export async function streamCaddyLogs(
  source: CaddyLogSource,
  onLine: (raw: string) => void,
  signal: AbortSignal,
): Promise<void> {
  const { accessToken } = useAuthStore.getState();
  const res = await fetch(`/api/caddy-logs/stream?source=${source}`, {
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
    signal,
  });
  if (!res.ok || !res.body) throw new Error(`stream failed: ${res.status}`);
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const frames = buffer.split('\n\n');
    buffer = frames.pop() ?? '';
    for (const frame of frames) {
      const dataLine = frame.split('\n').find((l) => l.startsWith('data: '));
      if (dataLine) onLine(dataLine.slice(6));
    }
  }
}

interface UseCaddyLogsResult {
  lines: CaddyLogLine[];
  isStreaming: boolean;
  error: Error | null;
  pause: () => void;
  resume: () => void;
  clear: () => void;
}

export function useCaddyLogs(source: CaddyLogSource): UseCaddyLogsResult {
  const [lines, setLines] = useState<CaddyLogLine[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [paused, setPaused] = useState(false);

  // Track the current AbortController so pause/resume can cancel/restart.
  const abortRef = useRef<AbortController | null>(null);

  const startStream = useCallback((src: CaddyLogSource) => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setIsStreaming(true);
    setError(null);

    streamCaddyLogs(
      src,
      (raw) => {
        const line: CaddyLogLine = { ...parseCaddyLogLine(raw), id: ++lineIdCounter };
        setLines((prev) => {
          const next = [...prev, line];
          return next.length > BUFFER_CAP ? next.slice(next.length - BUFFER_CAP) : next;
        });
      },
      controller.signal,
    )
      .then(() => {
        setIsStreaming(false);
      })
      .catch((err: unknown) => {
        setIsStreaming(false);
        if (err instanceof Error && err.name === 'AbortError') return;
        setError(err instanceof Error ? err : new Error(String(err)));
      });
  }, []);

  // Reset the buffer when the source changes so each tab shows only its own
  // log (otherwise the previous source's lines persist and the new source's
  // backfill just appends, making the tabs look identical). Pause/resume keeps
  // the buffer since `source` is unchanged.
  useEffect(() => {
    setLines([]);
  }, [source]);

  // Start/restart stream when source changes or paused state transitions to false.
  useEffect(() => {
    if (paused) return;
    startStream(source);
    return () => {
      abortRef.current?.abort();
    };
  }, [source, paused, startStream]);

  const pause = useCallback(() => {
    abortRef.current?.abort();
    setPaused(true);
    setIsStreaming(false);
  }, []);

  const resume = useCallback(() => {
    setPaused(false);
    // The effect re-runs because paused changed, which calls startStream.
  }, []);

  const clear = useCallback(() => {
    setLines([]);
  }, []);

  return { lines, isStreaming, error, pause, resume, clear };
}
