import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useCaddyLogs } from './use-caddy-logs';

// Chrome rejects an in-flight streaming fetch's reader with `TypeError: network
// error` when the request is aborted, rather than with an AbortError. A reader
// that never yields keeps the stream open until the test aborts it.
function abortableStreamResponse(signal: AbortSignal): Response {
  const read = () =>
    new Promise<never>((_resolve, reject) => {
      const fail = () => reject(new TypeError('network error'));
      if (signal.aborted) fail();
      else signal.addEventListener('abort', fail, { once: true });
    });

  return { ok: true, body: { getReader: () => ({ read }) } } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('useCaddyLogs', () => {
  it('does not surface an error when the stream is aborted by pause()', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((_url: string, init: RequestInit) =>
        Promise.resolve(abortableStreamResponse(init.signal as AbortSignal)),
      ),
    );

    const { result } = renderHook(() => useCaddyLogs('runtime'));
    await waitFor(() => expect(result.current.isStreaming).toBe(true));

    act(() => {
      result.current.pause();
    });

    // Let the reader's rejection settle before asserting.
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(result.current.error).toBeNull();
    expect(result.current.isStreaming).toBe(false);
  });

  // Switching tabs aborts the old stream while the new one is starting. The old
  // stream's rejection must not clobber the new stream's state.
  it('does not let the superseded stream clobber the replacement', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((_url: string, init: RequestInit) =>
        Promise.resolve(abortableStreamResponse(init.signal as AbortSignal)),
      ),
    );

    const { result, rerender } = renderHook(({ src }) => useCaddyLogs(src), {
      initialProps: { src: 'runtime' as const },
    });
    await waitFor(() => expect(result.current.isStreaming).toBe(true));

    rerender({ src: 'access' as unknown as 'runtime' });

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(result.current.error).toBeNull();
    expect(result.current.isStreaming).toBe(true);
  });

  it('surfaces a real connection failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve({ ok: false, status: 502, body: null } as unknown as Response)),
    );

    const { result } = renderHook(() => useCaddyLogs('runtime'));

    await waitFor(() => expect(result.current.error).not.toBeNull());
    expect(result.current.error?.message).toContain('502');
    expect(result.current.isStreaming).toBe(false);
  });
});

it('Retry reconnects after failure even when the user never paused', async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce({ ok: false, status: 502, body: null })
    .mockImplementation((_url: string, init: RequestInit) =>
      Promise.resolve(abortableStreamResponse(init.signal as AbortSignal)),
    );
  vi.stubGlobal('fetch', fetcher);
  const { result, unmount } = renderHook(() => useCaddyLogs('runtime'));
  await waitFor(() => expect(result.current.error).not.toBeNull());
  act(() => result.current.resume());
  await waitFor(() => expect(result.current.isStreaming).toBe(true));
  expect(fetcher).toHaveBeenCalledTimes(2);
  expect(result.current.error).toBeNull();
  unmount();
});

it('automatically reconnects and cancels pending retry on pause', async () => {
  vi.useFakeTimers();
  const fetcher = vi.fn().mockResolvedValue({ ok: false, status: 502, body: null });
  vi.stubGlobal('fetch', fetcher);
  const { result, unmount } = renderHook(() => useCaddyLogs('runtime'));
  await act(async () => {
    await Promise.resolve();
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1000);
  });
  expect(fetcher).toHaveBeenCalledTimes(2);
  act(() => result.current.pause());
  await act(async () => {
    await vi.advanceTimersByTimeAsync(30_000);
  });
  expect(fetcher).toHaveBeenCalledTimes(2);
  unmount();
  vi.useRealTimers();
});

it('does not repeatedly retry a forbidden stream', async () => {
  vi.useFakeTimers();
  const fetcher = vi.fn().mockResolvedValue({ ok: false, status: 403, body: null });
  vi.stubGlobal('fetch', fetcher);
  const { result, unmount } = renderHook(() => useCaddyLogs('runtime'));
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60_000);
  });
  expect(fetcher).toHaveBeenCalledTimes(1);
  expect(result.current.error?.message).toContain('access');
  unmount();
  vi.useRealTimers();
});
