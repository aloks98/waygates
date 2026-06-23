# B3 — Traffic Metrics — Design

**Date:** 2026-06-22
**Status:** Approved — ready for implementation plan
**Context:** Last of the backend-pipeline backlog items. Replaces the dashboard's placeholder traffic charts with real **aggregate** traffic metrics from Caddy's built-in Prometheus metrics, AND lets the admin expose those metrics to their own external monitoring (Prometheus/Grafana) via an opt-in, protected endpoint. Branch `feat/traffic-metrics` off master (post-#36; admin routes are top-level). Aggregate-only (no per-host) per the standing scope decision.

## Goal

1. **Internal dashboard:** scrape Caddy's Prometheus metrics on an interval, store a recent time-series in Postgres, and chart aggregate **requests** (by status class), **bandwidth** (in/out), and **latency** (p50/p95) on the dashboard.
2. **External exposure:** an opt-in, admin-configured, protected `/metrics` route in the generated Caddy config so the user's own Prometheus can scrape the same metrics — without exposing Caddy's admin API.

## Context (current state)

From exploration:
- Caddy collects per-request metrics (`caddy_http_requests_total{code,method,server,handler}`, `caddy_http_request_duration_seconds` histogram, `caddy_http_request_size_bytes`/`caddy_http_response_size_bytes` histograms, `caddy_http_requests_in_flight`) **only when server metrics are enabled** — JSON `"metrics": {}` on the HTTP server. They're served on the **admin API** (`localhost:2019/metrics`) and can also be served via a **`metrics` handler** on a normal route.
- `HTTPServer` (`caddy/config/types.go`) has no `Metrics` field today; enabling = add `Metrics *HTTPServerMetrics` + `server.Metrics = &HTTPServerMetrics{}` in `builder.go` (exact mirror of the recently-added `Logs *HTTPServerLogs`).
- The admin API client lives in `caddy/reloader.go` (`DefaultAdminAPIURL = "http://localhost:2019"`, a `*http.Client`); a scraper reuses this to `GET /metrics`.
- The periodic-loop pattern is `SyncService` (`ticker`/`stopChan`/`wg`, `Start(interval)`/`Stop()`, started in `routes.go`, stopped in `main.go`).
- **No `prometheus/*` lib** in go.mod; parsing the exposition format (incl. histogram buckets) → add **`github.com/prometheus/common/expfmt`** (TextParser). Cumulative counters/histograms → charting "over time" needs deltas between samples.
- DB pattern (audit logs): `backend/migrations/{seq}_*.up.sql`/`.down.sql`, a GORM model in `models/`, a repo (interface + impl + `NewXRepository(db)`), wired in `routes.go`.
- Settings pattern: `settingsRepo` with typed accessors (e.g. `GetNotFoundSettings`) is the precedent for the metrics-publish setting.
- **UI charts:** `@e412/rnui-react` exports an ECharts-based family — `AreaChart`, `LineChart`, `BarChart`, `PieChart` (the M1 fleet donut uses `PieChart`). No new chart dependency needed. Dashboard home = `routes/_dashboard/index.tsx`; data via `api` ky + React-Query `use-*` hooks (with `refetchInterval`).
- RBAC groups in `rbac.yaml` (e.g. `caddy_logs:read`, `caddy_config:read`); role templates (admin via `*`; operator/viewer explicit).

## Scope

**In:**
- **Shared:** enable `"metrics": {}` on the generated HTTP server.
- **Part 1 (internal):** a scraper service (Postgres-backed time-series), a range-aware read endpoint (`metrics:read`), and dashboard charts (requests/bandwidth/latency).
- **Part 2 (external):** a DB-backed, UI-configured, opt-in **protected** `metrics` route in the generated config.

**Out of scope:**
- Per-host / per-proxy metrics (aggregate only).
- Long-term/high-resolution history beyond the ~7-day retention; downsampling/rollup tiers; alerting/thresholds.
- Exposing the Caddy admin API.

## Shared foundation — enable server metrics

`caddy/config/types.go`: add `type HTTPServerMetrics struct{}` and `Metrics *HTTPServerMetrics \`json:"metrics,omitempty"\`` on `HTTPServer`. `caddy/config/builder.go` (where `server.Logs = &HTTPServerLogs{}` is set): also `server.Metrics = &HTTPServerMetrics{}`. Result: `"metrics": {}` on the server → `caddy_http_*` collected. Additive; must not break generation/validate/reload (extend the builder test).

## Part 1 — Internal traffic dashboard

### Scraper service
A new service mirroring `SyncService`'s lifecycle (`Start(interval)`/`Stop()`, ticker + `stopChan` + `wg`; started in `routes.go`, stopped in `main.go`). Every **30s**: `GET http://localhost:2019/metrics` (reuse the reloader's admin URL + an `http.Client`), parse with `expfmt.TextParser`, aggregate the labeled series into totals, and write **one cumulative sample row**. On the same loop, delete rows older than **7 days** (retention).

### Storage (Postgres)
A new table `traffic_samples` (migration + GORM model + repo, audit-log pattern), one row per scrape, storing **cumulative** values at that instant:
- `id`, `collected_at` (indexed)
- requests by status class: `req_2xx`, `req_3xx`, `req_4xx`, `req_5xx`, `req_other` (cumulative counts, summed across code/method/server/handler)
- `bytes_in`, `bytes_out` (cumulative, from the size histograms' `_sum`)
- `duration_buckets` JSONB (`{le: cumulative_count}`), `duration_sum`, `duration_count`
- `in_flight` (gauge, point-in-time)

Storing cumulative (not deltas) keeps it robust to missed scrapes; the read layer computes deltas.

### Read endpoint
`GET /api/metrics/traffic?range=1h|24h|7d` (RBAC **`metrics:read`**). Loads samples in the range, computes **per-step deltas** between consecutive samples (a Caddy restart resets counters → negative delta → treat as a reset, contribute 0), **downsamples** to a sane point count per range (1h→~30s native, 24h→5-min buckets, 7d→hourly buckets), and computes **p50/p95** via `histogram_quantile` over the **bucket deltas** across each step. Returns tidy series: `[{ t, req2xx, req3xx, req4xx, req5xx, bytesIn, bytesOut, p50, p95, inFlight }]` (rates per second or per-bucket totals — chosen consistently and documented in the response).

### Dashboard charts
On the dashboard home, three rnui ECharts cards (no new dep): **Requests** (stacked `AreaChart` by status class), **Bandwidth** (`AreaChart`/`LineChart`, in/out), **Latency** (`LineChart`, p50/p95) + a **range selector** (1h/24h/7d). `useTrafficMetrics(range)` (React Query, `refetchInterval` ~30s). Replace the M1 traffic placeholders in `components/dashboard/` and slot into the home grid.

## Part 2 — External metrics exposure (opt-in, protected, UI-configured)

### Setting (DB-backed)
A metrics-publish setting (migration + model + repo, or a typed accessor on the settings layer): `enabled` (default false), `host` (required when enabled), `path` (default `/metrics`), basic-auth `username` + `password_hash` (bcrypt — input plaintext in the UI, hashed server-side like ACL basic-auth users), optional `allowed_cidrs` (IP allowlist). Read/update endpoints under the existing settings RBAC (`settings:read` / `settings:write`).

### Settings UI
A new section in the Settings area (RHF form, per the project form convention): toggle enable, host, path, basic-auth user + password, optional CIDR list. Surfaces the scrape URL hint (`https://<host><path>`).

### Config generation
When the setting is `enabled`, the config builder adds a dedicated route to the HTTP server: a **host matcher** (the configured host) [+ path matcher for `path`] → optional **`remote_ip` matcher** (the allowlisted CIDRs) → a **`basic_auth` handler** (the configured user + bcrypt hash) → Caddy's **`metrics` handler** (serves the Prometheus exposition). Auto-HTTPS issues a cert for the host. The route ordering MUST ensure unauthenticated/secret-less requests are rejected (basic_auth before the metrics handler; the route only matches the configured host/path). The admin API is never exposed.

## Decisions (approved)

- Source: Caddy Prometheus metrics (admin `/metrics` internally; `metrics` handler externally). Aggregate-only.
- Storage: **Postgres** time-series; scrape **30s**; retention **7 days**; ranges **1h / 24h / 7d**.
- Metrics charted: **requests (by status class) + bandwidth + latency p50/p95**.
- Parser: **`github.com/prometheus/common/expfmt`** (new dep; justified by the histogram).
- Charts: rnui ECharts (`AreaChart`/`LineChart`) — no new FE dep.
- External exposure: **opt-in, off by default, protected** (basic-auth required; IP allowlist optional); **UI-configured** (DB-backed, runtime). Never exposes the admin API.

## Architecture & files

- **Backend:** `caddy/config/types.go` + `builder.go` (enable metrics + the external metrics route); new `service/metrics_scraper_service.go` (+ an expfmt parse helper, maybe `caddy/metrics/`); `migrations/{seq}_create_traffic_samples` + `models/traffic_sample.go` + `repository/traffic_sample_repository.go`; `migrations/{seq}_create_metrics_publish_settings` (or extend settings) + model + repo/accessor; `handlers/metrics_handler.go` (traffic read) + the metrics-publish read/update handler; `routes.go` (start scraper, wire handlers/routes); `rbac.yaml` (`metrics:read`); `main.go` (stop scraper). `go.mod`/`go.sum` (+ expfmt). Tests alongside.
- **Frontend:** `types/metrics.ts`, `hooks/use-traffic-metrics.ts`, `components/dashboard/*` traffic chart cards + range selector, the home page wiring; a Settings section for the metrics-publish form + its hook.

## Decomposition (for the plan)

Subagent-driven, ordered so each is gate-able:
1. **Enable server metrics** (config-builder + test) — shared foundation.
2. **Storage + scraper** — migration/model/repo for `traffic_samples`, the expfmt parse helper, the scraper ticker service (+ retention) + lifecycle wiring; Go tests (parser, aggregation, delta/reset).
3. **Read endpoint** — range aggregation + downsampling + `histogram_quantile` + `metrics:read` RBAC + route; Go tests (delta math, quantiles, reset handling).
4. **Dashboard charts** — `useTrafficMetrics` + the three rnui charts + range selector on the home page (replace placeholders).
5. **External exposure (backend)** — metrics-publish setting (migration/model/repo) + read/update endpoint + config-gen protected route (host/path + optional remote_ip + basic_auth + metrics handler); builder tests (route present only when enabled; auth ordering correct).
6. **External exposure (frontend)** — Settings section RHF form for the metrics-publish config + hook.

## Testing

- **Backend (testify):** the expfmt parse + label aggregation (fixture exposition → expected totals); delta computation incl. counter-reset clamping; `histogram_quantile` over bucket deltas (fixture buckets → expected p50/p95); the builder emits `"metrics": {}` and (when enabled) a correctly-ordered protected metrics route, and still validates. Pure logic — no Docker/caddy needed. The scraper loop + DB are integration-level (repo tested with the testcontainers pattern where the project already uses it; otherwise unit-test the pure parse/delta/quantile helpers).
- **Frontend:** gate (`pnpm build` + `check` + `test`, existing tests green); unit-test pure helpers (e.g. series shaping) if extracted; charts/forms verified at the gate level.

## Risks & notes

- **Latency quantiles are the highest-risk piece** — storing cumulative buckets (JSONB), delta-ing them across each step, and running `histogram_quantile`. Get the bucket math + interpolation right; cover with fixture tests.
- **Counter resets** (Caddy reload/restart zeroes counters) — a negative delta must be treated as a reset (contribute 0), or charts spike negative.
- **Live `caddy validate` can't run in this env** (no caddy binary/Docker) — verify the metrics enablement + the external route JSON against Caddy v2.11's schema by review; the prod reloader validates before applying (safety net). Recommend a container smoke test (metrics scraped; external endpoint serves + rejects unauth) at deploy.
- **External route security:** off by default; basic-auth required when enabled (bcrypt-hashed, never stored/returned in plaintext — the read endpoint must NOT return the password hash); IP allowlist optional; matcher/handler ordering must reject unauthenticated scrapes; the admin API stays localhost-only.
- **Scope creep guard:** aggregate-only; no alerting; no per-host; 7-day retention only.
