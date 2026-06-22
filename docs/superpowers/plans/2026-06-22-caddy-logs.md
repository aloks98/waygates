# View Caddy Logs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface Caddy's runtime/error log and HTTP access log in the Waygates UI as a live SSE stream — by enabling structured file logging in the generated Caddy JSON config, tailing those files in the Go backend, and streaming them to a new dashboard page.

**Architecture:** Populate the (currently-unused) `config.Logging` in the Caddy config builder so Caddy writes rolled JSON logs to `$CADDY_LOG_PATH` (default `/var/log/caddy`). A poll-based Go tailer reads those files (handling rotation); a snapshot endpoint and an SSE endpoint (`/api/caddy-logs*`, RBAC `caddy_logs:read`) expose them. The UI streams via `fetch` + `ReadableStream` (so the Bearer token is sent) into a live viewer.

**Tech Stack:** Go 1.25 (chi, zap, testify), React 19 + `@e412/rnui-react`, TanStack Router (code-based), ky.

## Global Constraints

- **The config-gen change is the highest-risk item** — it mutates the live Caddy config every proxy depends on. The generated config MUST still pass `caddy validate` and reload via the admin API. Extend the builder's existing tests; never break existing generation.
- **RBAC key `caddy_logs:read`** (parallels `audit_logs:read`).
- **SSE auth:** the frontend MUST use `fetch` + `ReadableStream` (NOT `EventSource`, which can't send headers), reading `useAuthStore.getState().accessToken` and sending `Authorization: Bearer …`. Never put the token in the query string.
- **Defaults:** runtime log → `$CADDY_LOG_PATH/runtime.log`, access log → `$CADDY_LOG_PATH/access.log`; JSON encoder; roll **100 MiB equivalent? NO — 10 MiB** (`roll_size_mb: 10`), `roll_keep: 5`, `roll_keep_for: 604800` (7 days). Backfill **last 500 lines**; UI buffer cap **~3000 lines**.
- HTTP access logs only (server `srv0`); no L4 access logs (L4 issues appear in the runtime log).
- **Per-task gates (repo root):** backend tasks → `make format-backend && make lint-backend && make backend-test`; frontend tasks → `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` (the existing **57 UI tests** must stay green). Backend tests use testify.
- **Commit trailer:** `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Stage only the files each task lists; never `git add -A`.

## Reference Files / facts

- Structs: `backend/internal/caddy/config/types.go` — `CaddyConfig{Admin,Logging,Storage,Apps}`, `LoggingConfig{Logs map[string]*LogConfig}`, `LogConfig{Writer,Encoder,Level}`, `LogWriter{Output,Filename}`, `LogEncoder{Format}`, `HTTPServer{Listen,Routes,…}`.
- Builder: `backend/internal/caddy/config/builder.go` — `Build()` assembles `CaddyConfig` (Admin localhost:2019, Storage /data, Apps); the single HTTP server is `NewHTTPServer(":443", ":80")` registered under `DefaultServerName` (`"srv0"`). `NewBuilder(opts...)`; settings (incl. `StoragePath`) arrive via `SetSettings(*Settings)`.
- Config/env: `backend/internal/config/config.go` — `CADDY_STORAGE_PATH` pattern (struct field `StoragePath`, `viper.GetString` in `Load()`, `viper.SetDefault` in `setDefaults()`), threaded → `SyncService` → `Builder.SetSettings`. `CADDY_BASE_PATH` is read inline in `routes.go`.
- Read-endpoint pattern: `handlers/audit_handler.go` (struct + `NewAuditHandler(service, logger)` + `List(w,r)`), `routes/routes.go` (`r.Route("/api/audit-logs", …)` with `chimw.RequirePermission(authAdapter, "audit_logs:read", mwConfig)`; handler/service instantiated + injected), `backend/rbac.yaml` (Audit permission group).
- No SSE precedent exists — use `http.NewResponseController(w).Flush()` (Go 1.25).
- UI api/auth: `ui/src/lib/api.ts` (`ky.create({ prefixUrl: '/api', hooks.beforeRequest → request.headers.set('Authorization', \`Bearer ${accessToken}\`) })`); token from `useAuthStore.getState().accessToken`.
- UI route/nav: `ui/src/lib/router.tsx` (leaf `createRoute({ getParentRoute: () => dashboardRoute, path, component: lazyRouteComponent(() => import('…'), 'Name') })` + `addChildren`); `ui/src/components/layout/sidebar.tsx` `navItems` array.
- Infra: `Dockerfile` (`RUN mkdir -p /etc/caddy/backup /data /config` line ~164; `VOLUME [...]` line ~179; runs as root, alpine), `docker/entrypoint.sh` (Caddy started ~line 59; `mkdir -p /etc/caddy/backup` ~line 8), `docker-compose.yml` (service `volumes:` + named `volumes:` block).

---

## Task 1: Caddy logging config (structs + builder + CADDY_LOG_PATH)

Make the generated Caddy config write runtime + access logs to files. **Backend.**

**Files:**
- Modify: `backend/internal/caddy/config/types.go` (add struct fields)
- Modify: `backend/internal/caddy/config/builder.go` (emit logging + server logs)
- Modify: `backend/internal/config/config.go` (`CADDY_LOG_PATH`)
- Modify: wherever `Settings` is defined + threaded (so the builder gets the log path) — `backend/internal/caddy/config/*` (the `Settings` struct + `SetSettings`) and the sync service that calls `SetSettings`
- Test: `backend/internal/caddy/config/builder_test.go` (extend)

- [ ] **Step 1: Add the missing struct fields** in `types.go`:
```go
type LogWriter struct {
	Output      string `json:"output,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Roll        *bool  `json:"roll,omitempty"`
	RollSizeMB  int    `json:"roll_size_mb,omitempty"`
	RollKeep    int    `json:"roll_keep,omitempty"`
	RollKeepFor int    `json:"roll_keep_for,omitempty"` // seconds
}

type LogConfig struct {
	Writer  *LogWriter  `json:"writer,omitempty"`
	Encoder *LogEncoder `json:"encoder,omitempty"`
	Level   string      `json:"level,omitempty"`
	Include []string    `json:"include,omitempty"`
}

// HTTPServerLogs enables access logging for an HTTP server.
type HTTPServerLogs struct {
	DefaultLoggerName string `json:"default_logger_name,omitempty"`
}
```
and add to `HTTPServer`: `Logs *HTTPServerLogs \`json:"logs,omitempty"\``.

- [ ] **Step 2: Add `CADDY_LOG_PATH`** in `config.go` mirroring `CADDY_STORAGE_PATH`: a `LogPath string` field on the Caddy config struct, `LogPath: viper.GetString("CADDY_LOG_PATH")` in `Load()`, and `viper.SetDefault("CADDY_LOG_PATH", "/var/log/caddy")` in `setDefaults()`. Thread it into the builder `Settings` the same way `StoragePath` is threaded (add `LogPath` to the `Settings` struct and set it where the sync service builds `Settings` from config). If `LogPath` is empty, the builder defaults to `/var/log/caddy`.

- [ ] **Step 3: Emit the logging block in `Build()`.** After assembling `config` and (if present) the HTTP app/server, set the runtime + access logs. Use the builder's resolved log path (`logPath`, default `/var/log/caddy`); roll helper `roll := true`:
```go
logPath := b.settings.LogPath
if logPath == "" {
	logPath = "/var/log/caddy"
}
roll := true
config.Logging = &LoggingConfig{
	Logs: map[string]*LogConfig{
		"default": {
			Writer:  &LogWriter{Output: "file", Filename: filepath.Join(logPath, "runtime.log"), Roll: &roll, RollSizeMB: 10, RollKeep: 5, RollKeepFor: 604800},
			Encoder: &LogEncoder{Format: "json"},
			Level:   "INFO",
		},
	},
}
```
When the HTTP server is created (the `if len(routes) > 0` block), also: set `server.Logs = &HTTPServerLogs{DefaultLoggerName: "access"}` and add the access log entry:
```go
config.Logging.Logs["access"] = &LogConfig{
	Include: []string{"http.log.access." + DefaultServerName}, // "http.log.access.srv0"
	Writer:  &LogWriter{Output: "file", Filename: filepath.Join(logPath, "access.log"), Roll: &roll, RollSizeMB: 10, RollKeep: 5, RollKeepFor: 604800},
	Encoder: &LogEncoder{Format: "json"},
}
```
(Import `path/filepath` if not already.)

- [ ] **Step 4: Extend `builder_test.go`** — add a test asserting: `Build()` returns a config whose `Logging.Logs["default"]` writes to `<logpath>/runtime.log` with roll on; when proxies/routes exist, `Logging.Logs["access"]` exists with `Include == ["http.log.access.srv0"]` and the server has `Logs.DefaultLoggerName == "access"`; and a config built with no proxies still has the default (runtime) log. Marshal the config to JSON and assert it is valid JSON (and, if the existing tests already shell out to `caddy validate`, that path stays green).

- [ ] **Step 5: Gate.** `make format-backend && make lint-backend && make backend-test` → all pass.

- [ ] **Step 6: Commit**
```bash
git add backend/internal/caddy/config/types.go backend/internal/caddy/config/builder.go \
  backend/internal/config/config.go backend/internal/caddy/config/builder_test.go
# plus the Settings/sync files touched in Step 2
git commit -m "feat(caddy): emit runtime + access file logging in generated config

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Log directory + Docker plumbing

Ensure `/var/log/caddy` exists, is writable by Caddy + readable by the backend, and persists. **Infra.**

**Files:** `Dockerfile`, `docker/entrypoint.sh`, `docker-compose.yml`.

- [ ] **Step 1: Dockerfile** — add `/var/log/caddy` to the runtime `mkdir -p` line (`RUN mkdir -p /etc/caddy/backup /data /config /var/log/caddy`) and to the `VOLUME` list (`VOLUME ["/data", "/config", "/etc/caddy", "/var/log/caddy"]`).
- [ ] **Step 2: entrypoint.sh** — add `mkdir -p /var/log/caddy` near the existing `mkdir -p /etc/caddy/backup` (before Caddy starts), so the dir exists even on a fresh volume.
- [ ] **Step 3: docker-compose.yml** — add `- caddy-logs:/var/log/caddy` to the waygates service `volumes:` and `caddy-logs:` to the named `volumes:` block.
- [ ] **Step 4: Verify** — no app build needed; sanity-check the files (`grep -n "var/log/caddy" Dockerfile docker/entrypoint.sh docker-compose.yml` shows all three). (Container runs as root, so Caddy can write and the backend can read; note that in the report.)
- [ ] **Step 5: Commit**
```bash
git add Dockerfile docker/entrypoint.sh docker-compose.yml
git commit -m "build: add /var/log/caddy log volume for Caddy logs

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Log tailer + snapshot endpoint + RBAC

The read path: a rotation-aware tailer, a `GET /api/caddy-logs` snapshot, RBAC, wiring. **Backend.**

**Files:**
- Create: `backend/internal/caddy/logtail/tailer.go` + `tailer_test.go`
- Create: `backend/internal/service/caddy_logs_service.go`
- Create: `backend/internal/api/handlers/caddy_logs_handler.go`
- Modify: `backend/internal/api/routes/routes.go`, `backend/rbac.yaml`

**Interfaces:**
- Produces: `logtail.LastLines(path string, n int) ([][]byte, error)`; `logtail.NewTailer(path string) *Tailer` with `Stream(ctx, backfill int, out chan<- []byte) error`; `service.CaddyLogsService` with `Snapshot(source string, limit int) ([]json.RawMessage, error)` and `FilePath(source string) (string, error)` (maps `runtime|access` → file under the configured log path); handler `NewCaddyLogsHandler(svc, logger)` with `List(w,r)` (Task 3) and `Stream(w,r)` (Task 4).

- [ ] **Step 1: Tailer — `LastLines` + `Tailer.Stream`.** `backend/internal/caddy/logtail/tailer.go`:
```go
package logtail

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"time"
)

// LastLines returns up to n trailing lines of the file (each without the newline).
func LastLines(path string, n int) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no log yet → empty, not an error
		}
		return nil, err
	}
	defer f.Close()
	var lines [][]byte
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := append([]byte(nil), sc.Bytes()...)
		lines = append(lines, line)
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines, sc.Err()
}

type Tailer struct {
	path     string
	interval time.Duration
}

func NewTailer(path string) *Tailer { return &Tailer{path: path, interval: 500 * time.Millisecond} }

// Stream backfills the last `backfill` lines then streams appended lines until ctx is done.
// Handles rotation: if the file shrinks or its identity changes, it reopens from the start.
func (t *Tailer) Stream(ctx context.Context, backfill int, out chan<- []byte) error {
	if backfill > 0 {
		lines, err := LastLines(t.path, backfill)
		if err != nil {
			return err
		}
		for _, l := range lines {
			select {
			case out <- l:
			case <-ctx.Done():
				return nil
			}
		}
	}
	var offset int64
	if fi, err := os.Stat(t.path); err == nil {
		offset = fi.Size()
	}
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	var carry []byte
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			fi, err := os.Stat(t.path)
			if err != nil {
				continue // file may be mid-rotation
			}
			if fi.Size() < offset {
				offset = 0 // rotated/truncated → reread from start
				carry = nil
			}
			if fi.Size() == offset {
				continue
			}
			f, err := os.Open(t.path)
			if err != nil {
				continue
			}
			if _, err := f.Seek(offset, 0); err != nil {
				f.Close()
				continue
			}
			buf := make([]byte, fi.Size()-offset)
			nRead, _ := f.Read(buf)
			f.Close()
			offset += int64(nRead)
			data := append(carry, buf[:nRead]...)
			for {
				i := bytes.IndexByte(data, '\n')
				if i < 0 {
					break
				}
				line := append([]byte(nil), data[:i]...)
				data = data[i+1:]
				select {
				case out <- line:
				case <-ctx.Done():
					return nil
				}
			}
			carry = append([]byte(nil), data...)
		}
	}
}
```

- [ ] **Step 2: Tailer tests** — `tailer_test.go`: (a) `LastLines` on a temp file returns the last n lines and returns `(nil,nil)` for a missing file; (b) `Stream` backfills then picks up appended lines (write to the temp file after starting `Stream` in a goroutine, assert the new line arrives on the channel within a timeout); (c) rotation — truncate/replace the file, append, assert the new line still arrives. Use `t.TempDir()` + short timeouts.

- [ ] **Step 3: Service** — `caddy_logs_service.go`: holds the configured log path; `FilePath(source)` returns `<logPath>/runtime.log` for `"runtime"`, `<logPath>/access.log` for `"access"`, error for anything else; `Snapshot(source, limit)` = `logtail.LastLines(path, limit)` mapped to `[]json.RawMessage` (each line is already JSON). Construct it with the log path from config (mirror how other services receive config). Expose a `Tailer(source)` helper returning `logtail.NewTailer(path)` for Task 4.

- [ ] **Step 4: Handler `List` (snapshot)** — `caddy_logs_handler.go`: struct + `NewCaddyLogsHandler(svc, logger)` (mirror `AuditHandler`); `List(w,r)` reads `source` (default `runtime`; validate `runtime|access`) and `limit` (default 200, cap 1000), calls `svc.Snapshot`, returns via the standard `utils.Success(w, data, msg)` envelope. Invalid source → 400 via the standard error helper.

- [ ] **Step 5: RBAC + route wiring.** Add to `rbac.yaml`:
```yaml
  - name: "Caddy Logs"
    permissions:
      - key: "caddy_logs:read"
        name: "View Caddy Logs"
        description: "View Caddy runtime and access logs"
```
In `routes.go`: instantiate the service + handler (mirror audit wiring, passing the configured log path), and register:
```go
r.Route("/api/caddy-logs", func(r chi.Router) {
	r.With(chimw.RequirePermission(authAdapter, "caddy_logs:read", mwConfig)).Get("/", caddyLogsHandler.List)
	// stream route added in Task 4
})
```

- [ ] **Step 6: Gate.** `make format-backend && make lint-backend && make backend-test` → all pass.

- [ ] **Step 7: Commit** (stage the new logtail/service/handler files + routes.go + rbac.yaml) — message `feat(api): Caddy logs tailer + snapshot endpoint (caddy_logs:read)`.

---

## Task 4: SSE stream endpoint

`GET /api/caddy-logs/stream` — backfill + live tail. **Backend.**

**Files:** Modify `backend/internal/api/handlers/caddy_logs_handler.go`, `backend/internal/api/routes/routes.go`.

- [ ] **Step 1: `Stream` handler** (SSE; first SSE in the codebase — use `http.NewResponseController`):
```go
func (h *CaddyLogsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "runtime"
	}
	tailer, err := h.svc.Tailer(source)
	if err != nil {
		utils.BadRequest(w, "invalid log source") // use the project's standard error helper
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	rc := http.NewResponseController(w)
	lines := make(chan []byte, 256)
	go func() { _ = tailer.Stream(r.Context(), 500, lines) }()
	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", line); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
```
(The tailer goroutine stops when `r.Context()` is cancelled on client disconnect — no leak. Adjust `utils.BadRequest`/error helper to the project's actual name.)

- [ ] **Step 2: Route** — add inside the `/api/caddy-logs` group: `r.With(chimw.RequirePermission(authAdapter, "caddy_logs:read", mwConfig)).Get("/stream", caddyLogsHandler.Stream)`.

- [ ] **Step 3: Gate.** `make format-backend && make lint-backend && make backend-test` → all pass (build compiles the SSE handler; no unit test required for the streaming loop, but keep `make backend-test` green).

- [ ] **Step 4: Commit** — `feat(api): SSE stream endpoint for Caddy logs`.

---

## Task 5: Frontend data layer (types + SSE client hook)

**Files:**
- Create: `ui/src/types/caddy-logs.ts`, `ui/src/hooks/use-caddy-logs.ts`
- Test: `ui/src/hooks/<sse-frame-parser>.test.ts` (only if a pure parser helper is extracted)

**Interfaces:**
- Produces: `CaddyLogSource = 'runtime' | 'access'`; a parsed `CaddyLogLine` type; `useCaddyLogs(source)` returning `{ lines, isStreaming, error, pause, resume, clear }`.

- [ ] **Step 1: Types** — `types/caddy-logs.ts`: `export type CaddyLogSource = 'runtime' | 'access';` and a `CaddyLogLine` shape (`{ raw: string; ts?: number; level?: string; logger?: string; msg?: string; /* access: */ status?: number; method?: string; host?: string; uri?: string; duration?: number }`) plus a `parseCaddyLogLine(raw: string): CaddyLogLine` that JSON-parses and maps known fields (best-effort; on parse failure keep `{ raw }`).

- [ ] **Step 2: SSE client + hook** — `hooks/use-caddy-logs.ts`. The streaming reader (fetch + ReadableStream so the Bearer header is sent):
```ts
import { useAuthStore } from '@/stores/auth';

export async function streamCaddyLogs(
  source: string,
  onLine: (raw: string) => void,
  signal: AbortSignal,
) {
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
```
`useCaddyLogs(source)`: keeps a bounded buffer (cap ~3000, drop oldest) of `parseCaddyLogLine` results in state; starts `streamCaddyLogs` in an effect with an `AbortController`; `pause()` aborts, `resume()` restarts, `clear()` empties the buffer; re-streams when `source` changes; surfaces `error` on failure. (Optionally seed via the snapshot `GET /api/caddy-logs?source=` before/after connect — the backend already backfills 500, so the snapshot is optional; if used, fetch via the existing `api` ky client.)

- [ ] **Step 3 (optional): test** the pure `parseCaddyLogLine` + the SSE frame split if extracted into a pure function — a small vitest asserting a JSON access line parses to the right fields and a non-JSON line falls back to `{ raw }`.

- [ ] **Step 4: Gate.** `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` → green (57+ tests).

- [ ] **Step 5: Commit** — `feat(ui): Caddy logs SSE client + use-caddy-logs hook`.

---

## Task 6: Frontend page + nav

**Files:**
- Create: `ui/src/routes/_dashboard/caddy-logs.tsx`, `ui/src/components/caddy-logs/*` (viewer + row + toolbar)
- Modify: `ui/src/lib/router.tsx`, `ui/src/components/layout/sidebar.tsx`

**Interfaces:** Produces `CaddyLogsPage` (named export) consumed by `router.tsx`.

- [ ] **Step 1: Page + viewer.** `caddy-logs.tsx` exports `CaddyLogsPage`: `space-y-6` container + `<h1>Caddy Logs</h1>`; rnui `Tabs` for **Runtime | Access** (source state); for the active source, `useCaddyLogs(source)` drives a monospace live viewer. Viewer: a scroll container that auto-scrolls to bottom unless the user scrolled up (track via `onScroll`); toolbar with **Pause/Resume**, **Clear**, a **search** input (client-side substring over `raw`), and a level filter (runtime) / status filter (access). Rows render structured: runtime → `timestamp · level · message`; access → `status · method · host · uri · duration`; unparsed → raw text. Show a connection/error banner with a Retry (calls `resume()`), and an empty state ("No log lines yet"). Keep components small (`caddy-logs/log-viewer.tsx`, `log-row.tsx`, `log-toolbar.tsx`).

- [ ] **Step 2: Router** — in `router.tsx` add:
```tsx
const caddyLogsRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/caddy-logs',
  component: lazyRouteComponent(() => import('@/routes/_dashboard/caddy-logs'), 'CaddyLogsPage'),
});
```
and add `caddyLogsRoute` to `dashboardRoute.addChildren([...])`.

- [ ] **Step 3: Sidebar** — add to `navItems`: `{ label: 'Caddy Logs', path: '/dashboard/caddy-logs', icon: <ScrollText className="size-4" /> }` (import `ScrollText` from lucide-react; pick an icon not already used).

- [ ] **Step 4: Gate.** `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` → green. Reason through: the page streams the active source, tab switch re-streams, pause/clear/search work, the Bearer header is sent (fetch-stream).

- [ ] **Step 5: Commit** — `feat(ui): Caddy logs viewer page + nav`.

---

## Self-Review

**Spec coverage:** logging config block (runtime + access, rolled JSON) + `CADDY_LOG_PATH` (Task 1); log volume/Docker (Task 2); tailer + snapshot + RBAC `caddy_logs:read` (Task 3); SSE stream (Task 4); fetch-stream SSE client (Task 5, satisfies the EventSource-auth constraint); page + Runtime/Access tabs + live viewer + filters + sidebar (Task 6). Backfill 500 / buffer 3000 / roll 10MiB×5×7d all specified. HTTP-only access logs; rotation handled in the tailer; disconnect cleanup via `r.Context()`. No spec requirement unaddressed.

**Placeholder scan:** the novel/risky code is complete (struct fields, the `Build()` logging block, the full tailer, the SSE handler, the SSE client); pattern-following boilerplate (handler/service/route wiring) references the exact audit-pattern shapes with the concrete names. The one explicit "adjust to the project's actual name" note is the error-helper (`utils.BadRequest`) — the implementer confirms the real helper name from `utils`/the audit handler. No TBD/"handle X".

**Type/name consistency:** `caddy_logs:read` used in rbac.yaml + both routes; `source` values `runtime|access` consistent across service `FilePath`, handler, hook, page; `CaddyLogsPage` export matches the router import string; `logtail.LastLines`/`NewTailer`/`Tailer.Stream` signatures consistent across tasks 3–4; `LogWriter`/`LogConfig`/`HTTPServerLogs` field names match the json shape Caddy expects (`roll_size_mb`, `roll_keep`, `roll_keep_for`, `include`, `default_logger_name`).
