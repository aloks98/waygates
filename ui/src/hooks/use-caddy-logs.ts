import { useCallback, useEffect, useRef, useState } from 'react';

import { handleTokenRefresh } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { type CaddyLogLine, type CaddyLogSource, parseCaddyLogLine } from '@/types/caddy-logs';

const BUFFER_CAP = 3000;
let lineIdCounter = 0;
export class LogStreamError extends Error {
  constructor(public status: number) {
    super(
      status === 401
        ? 'Session expired. Sign in again.'
        : status === 403
          ? 'You do not have access to these logs.'
          : `Connection interrupted (${status}).`,
    );
  }
}

export async function streamCaddyLogs(
  source: CaddyLogSource,
  onLine: (raw: string) => void,
  signal: AbortSignal,
  options: { backfill?: boolean; onConnected?: () => void; onActivity?: () => void } = {},
): Promise<void> {
  const connect = () => {
    const { accessToken } = useAuthStore.getState();
    return fetch(
      `/api/caddy-logs/stream?source=${source}${options.backfill === false ? '&backfill=0' : ''}`,
      { headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {}, signal },
    );
  };
  let res = await connect();
  if (res.status === 401 && !signal.aborted) {
    const tokens = await handleTokenRefresh();
    if (tokens && !signal.aborted) res = await connect();
  }
  if (!res.ok || !res.body) throw new LogStreamError(res.status);
  options.onConnected?.();
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done || signal.aborted) break;
      buffer = (buffer + decoder.decode(value, { stream: true })).replace(/\r\n/g, '\n');
      const frames = buffer.split('\n\n');
      buffer = frames.pop() ?? '';
      for (const frame of frames) {
        options.onActivity?.();
        const data = frame
          .split('\n')
          .filter((l) => l.startsWith('data:'))
          .map((l) => l.slice(5).replace(/^ /, ''))
          .join('\n');
        if (data) onLine(data);
      }
    }
  } finally {
    reader.releaseLock?.();
  }
}

export function useCaddyLogs(source: CaddyLogSource) {
  const [lines, setLines] = useState<CaddyLogLine[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [paused, setPaused] = useState(false);
  const [restart, setRestart] = useState(0);
  const history = useRef({ source, received: false });
  const abortRef = useRef<AbortController | null>(null);
  useEffect(() => {
    history.current = { source, received: false };
    setLines([]);
    setError(null);
  }, [source]);
  useEffect(() => {
    if (paused) return;
    let disposed = false;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let attempts = 0;
    const start = async () => {
      clearTimeout(retryTimer);
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      const stale = () => disposed || controller.signal.aborted || abortRef.current !== controller;
      setIsStreaming(false);
      let connectedAt = 0;
      try {
        await streamCaddyLogs(
          source,
          (raw) => {
            if (stale()) return;
            history.current.received = true;
            const line = { ...parseCaddyLogLine(raw), id: ++lineIdCounter };
            setLines((prev) => [...prev, line].slice(-BUFFER_CAP));
          },
          controller.signal,
          {
            backfill: !history.current.received,
            onConnected: () => {
              if (!stale()) {
                connectedAt = Date.now();
                setIsStreaming(true);
                setError(null);
              }
            },
            onActivity: () => {
              if (Date.now() - connectedAt > 5000) attempts = 0;
            },
          },
        );
        if (stale()) return;
        throw new Error('Connection closed. Reconnecting…');
      } catch (cause) {
        if (stale()) return;
        setIsStreaming(false);
        const failure = cause instanceof Error ? cause : new Error(String(cause));
        setError(failure);
        if (failure instanceof LogStreamError && [401, 403].includes(failure.status)) return;
        retryTimer = setTimeout(start, Math.min(1000 * 2 ** attempts++, 30_000));
      }
    };
    void start();
    const online = () => {
      void start();
    };
    window.addEventListener('online', online);
    return () => {
      disposed = true;
      clearTimeout(retryTimer);
      abortRef.current?.abort();
      window.removeEventListener('online', online);
    };
  }, [source, paused, restart]);
  const pause = useCallback(() => {
    abortRef.current?.abort();
    setPaused(true);
    setIsStreaming(false);
    setError(null);
  }, []);
  const resume = useCallback(() => {
    setPaused(false);
    setRestart((v) => v + 1);
  }, []);
  const clear = useCallback(() => setLines([]), []);
  return { lines, isStreaming, error, paused, pause, resume, clear };
}
