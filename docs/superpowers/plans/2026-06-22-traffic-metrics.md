# B3 — Traffic Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** (1) Scrape Caddy's Prometheus metrics on an interval, store a 7-day Postgres time-series, and chart aggregate requests/bandwidth/latency on the dashboard; (2) an opt-in, admin-configured, protected external `/metrics` route in the generated Caddy config.

**Architecture:** Enable `"metrics": {}` on the generated HTTP server (shared). A scraper service (ticker like `SyncService`) reads `localhost:2019/metrics` (expfmt), stores cumulative samples; a range-aware read endpoint computes deltas + histogram quantiles; rnui ECharts render them. The external endpoint is a protected route (host/path + optional remote_ip + basic-auth → Caddy `metrics` handler), driven by a key-value setting.

**Tech Stack:** Go 1.25 (chi, zap, GORM, testify, +`github.com/prometheus/common/expfmt`), React 19 + `@e412/rnui-react` (ECharts charts), TanStack Router/Query, ky.

## Global Constraints

- **Aggregate-only** (no per-host). **Generated config** changes are additive + must keep `caddy validate`/reload working (no caddy binary/Docker locally → verify JSON against Caddy v2.11 by review; prod reloader validates before apply).
- Scrape **30s**; retention **7 days**; ranges **1h / 24h / 7d**; parser **expfmt**.
- External endpoint: **opt-in, off by default, protected** (basic-auth required when enabled; IP allowlist optional); **never** expose the admin API; the read endpoint/UI must NOT return the basic-auth password hash.
- **Per-task gates:** backend → `gofmt -l <files>` + `go build ./...` + `go test ./... -short` (golangci-lint runs in CI; proactively: handle every returned error incl. `Close()` via `_ =`, no `appendAssign`, US spelling). Frontend → `pnpm --dir ui build && pnpm --dir ui check && pnpm --dir ui test` (existing tests green).
- Commit trailer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; stage only each task's files; never `git add -A`.
- **Form convention:** the Part-2 settings form binds via RHF `Form`/`FormField` (project convention).

## Reference facts (verbatim from exploration)

- **Config model** (`caddy/config/types.go`): `HTTPServer{ Listen, Routes []*HTTPRoute, …, Logs *HTTPServerLogs }` (add `Metrics`). `HTTPRoute{ Match []MatcherSet, Handle []HTTPHandler, Terminal bool }`; `MatcherSet = map[string]interface{}`, `HTTPHandler = map[string]interface{}`. Matchers: `NewHostMatcher(hosts...)`, `NewHostPathMatcher(host, paths...)`, `NewRemoteIPMatcher(ranges...)` / `AddRemoteIPToMatcher(m, ranges...)` (combine host+path+remote_ip in ONE MatcherSet = AND). Basic-auth: `NewAuthenticationHandler(accounts []*BasicAuthAccount, realm)` (handler name `"authentication"`, provider `http_basic`), `NewBasicAuthAccount(user, bcryptHash)`, emitted via `ToHTTPHandler(authHandler)`. Metrics handler has no typed struct → raw `HTTPHandler{"handler": "metrics"}`.
- **builder.go** `Build()` (`~214-229`): the `if len(routes) > 0 {` block sets `server.Logs = &HTTPServerLogs{}` — add `server.Metrics` here; the catch-all route is appended to `routes` (~207-211) before `server.AddRoutes(routes...)` — prepend the external metrics route before the catch-all.
- **Scraper lifecycle** (`service/sync_service.go`): `ticker *time.Ticker`, `stopChan chan struct{}`, `wg sync.WaitGroup`; `Start(interval)` (ticker goroutine, select on `ticker.C`/`stopChan`), `Stop()` (close stopChan, ticker.Stop, wg.Wait). Started in `routes.go` (`syncService.Start(60*time.Second)`); stopped in `cmd/server/main.go` (`syncService.Stop()` in the signal handler).
- **Admin client** (`caddy/reloader.go`): `DefaultAdminAPIURL = "http://localhost:2019"`; scraper makes its own `&http.Client{Timeout: 10*time.Second}` and `GET {admin}/metrics`.
- **DB** (audit-log pattern): next migration is **`000012`** (`backend/migrations/000012_*.up.sql`/`.down.sql`). Model: `models/audit_log.go` (`gorm:"primaryKey;autoIncrement"`, `TableName()`). **JSON columns use the custom `JSONField` (`gorm:"type:text"` + marshal/unmarshal), NOT native jsonb.** Repo: `NewXRepository(db *gorm.DB)`, wired in `routes.go` (`auditLogRepo := repository.NewAuditLogRepository(db)`).
- **Settings (key-value)**: `SettingsRepositoryInterface` has `Get/GetValue/Set/GetAll` + typed `GetNotFoundSettings`/`SetNotFoundSettings` (composed from `Setting` key-value rows — **no per-setting table/migration**). Route `r.Route("/api/settings", …) .Get("/404", …) .Put("/404", …)` gated `settings:read`/`settings:write`. Frontend `use-settings.ts` (`useNotFoundSettings`: query + mutation) + `components/settings/catchall-settings.tsx` (RHF form).
- **Charts** (`@e412/rnui-react`): `AreaChart{ categories: string[], series: AreaChartSeries[]{name,data:number[]}, stacked?, smooth?, showLegend?, height }`; `LineChart{ categories?, series?, data?, smooth?, showLegend? }`. Parallel arrays. (Fleet donut uses `PieChart data={[{name,value}]}`.)
- **Dashboard home** (`ui/src/routes/_dashboard/index.tsx`): `space-y-6` → `SystemStatusBar`, `DashboardStatCards`, `grid lg:grid-cols-3`(FleetComposition col-span-2 + DashboardQuickActions), `ActivityTimeline`. **No traffic placeholders exist** → the charts are net-new cards (insert a `grid gap-6 lg:grid-cols-2` row after `DashboardStatCards`). `api` = ky `prefixUrl:'/api'`; `refetchInterval: 30_000` precedent in `use-dashboard.ts`.
- **RBAC** (`rbac.yaml`): `caddy_logs:read`/`caddy_config:read` group shape; `admin` = `["*"]`; `operator`/`viewer` explicit lists. New `metrics:read` → add to `viewer` + `operator` (dashboard data is read for all roles).

---

## Task 1: Enable server metrics (shared foundation)

**Files:** `backend/internal/caddy/config/types.go`, `builder.go`, `builder_test.go`.

- [ ] **Step 1:** In `types.go` add `type HTTPServerMetrics struct{}` and on `HTTPServer`: `Metrics *HTTPServerMetrics \`json:"metrics,omitempty"\``.
- [ ] **Step 2:** In `builder.go`, in the `if len(routes) > 0 {` block right after `server.Logs = &HTTPServerLogs{}`, add `server.Metrics = &HTTPServerMetrics{}`.
- [ ] **Step 3:** Extend `builder_test.go`: assert the built config's server has `Metrics != nil` (→ `"metrics": {}` in JSON) and the config still marshals to valid JSON.
- [ ] **Step 4: Gate** (`gofmt`/`go build`/`go test -short`). **Commit:** `feat(caddy): enable server metrics in generated config`.

---

## Task 2: Storage + scraper service

**Files:** `backend/migrations/000012_create_traffic_samples.{up,down}.sql`; `backend/internal/models/traffic_sample.go`; `backend/internal/repository/traffic_sample_repository.go` (+ interface entry); a parse helper (e.g. `backend/internal/caddy/metrics/parse.go`); `backend/internal/service/metrics_scraper_service.go`; `backend/internal/api/routes/routes.go` (instantiate repo + start scraper); `backend/cmd/server/main.go` (stop scraper); `go.mod`/`go.sum` (+ expfmt). Tests: parse + scraper-aggregation/delta unit tests.

- [ ] **Step 1: Migration `000012`.** `traffic_samples` table: `id SERIAL PRIMARY KEY`, `collected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`, `req_2xx/req_3xx/req_4xx/req_5xx/req_other BIGINT NOT NULL DEFAULT 0`, `bytes_in BIGINT NOT NULL DEFAULT 0`, `bytes_out BIGINT NOT NULL DEFAULT 0`, `duration_buckets TEXT NOT NULL DEFAULT '{}'` (JSON map `le→cumulative count`), `duration_sum DOUBLE PRECISION NOT NULL DEFAULT 0`, `duration_count BIGINT NOT NULL DEFAULT 0`, `in_flight BIGINT NOT NULL DEFAULT 0`. `CREATE INDEX idx_traffic_samples_collected_at ON traffic_samples(collected_at);`. down: drop index + table.
- [ ] **Step 2: Model** `models/traffic_sample.go` — `TrafficSample` struct (fields above; `DurationBuckets` uses the existing `JSONField`/text pattern as a `map[string]int64`), `TableName() string { return "traffic_samples" }`.
- [ ] **Step 3: Repo** `traffic_sample_repository.go` — `NewTrafficSampleRepository(db)`; `Create(*TrafficSample) error`; `ListSince(t time.Time) ([]TrafficSample, error)` (ordered by `collected_at ASC`); `DeleteOlderThan(t time.Time) (int64, error)`. Add the interface to `repository/interfaces.go`.
- [ ] **Step 4: expfmt dependency + parse helper.** `go get github.com/prometheus/common/expfmt` (+ `go mod tidy`). In `caddy/metrics/parse.go`: `ParseSnapshot(r io.Reader) (*Snapshot, error)` using `expfmt.TextParser.TextToMetricFamilies` — aggregate `caddy_http_requests_total` by `code` first digit into req_2xx..req_other (sum across all label combos); `caddy_http_request_size_bytes_sum`/`caddy_http_response_size_bytes_sum` → bytes_in/out; `caddy_http_request_duration_seconds_bucket` summed per `le` → bucket map, `_sum`/`_count` → duration_sum/count; `caddy_http_requests_in_flight` (gauge) summed → in_flight. Return a `Snapshot` mirroring the `TrafficSample` numeric fields. Unit-test with a fixture exposition string → expected aggregates (include multiple code/method/handler series to prove summation).
- [ ] **Step 5: Scraper service** `metrics_scraper_service.go` — mirror `SyncService`'s `ticker`/`stopChan`/`wg` + `Start(interval)`/`Stop()`. Each tick: `GET http://localhost:2019/metrics` (own `http.Client{Timeout:10s}`), `ParseSnapshot`, build a `TrafficSample{collected_at: now-passed-in?}` — note: time is needed; the scraper records `time.Now()` at scrape (Date.now is fine in prod code; not a workflow constraint) — `repo.Create`, then `repo.DeleteOlderThan(now - 7*24h)`. Log + continue on scrape/parse errors (Caddy may be briefly down). Constructor takes the repo + logger + admin URL (default `DefaultAdminAPIURL`).
- [ ] **Step 6: Wire lifecycle.** `routes.go`: `trafficSampleRepo := repository.NewTrafficSampleRepository(db)`; construct the scraper; `metricsScraper.Start(30 * time.Second)` (after `syncService.Start(...)`). `cmd/server/main.go`: `metricsScraper.Stop()` (after `syncService.Stop()` in the signal handler). (Pass the scraper out of `routes.go` the same way `syncService` is, so main can stop it — mirror how `syncService` is returned/held.)
- [ ] **Step 7: Gate** + **Commit:** `feat(metrics): scrape + store Caddy traffic samples (30s, 7d retention)`.

---

## Task 3: Traffic read endpoint

**Files:** `backend/internal/service/traffic_metrics_service.go` (or fold into the scraper service), `backend/internal/api/handlers/metrics_handler.go`, `routes.go`, `rbac.yaml`. Tests: delta/downsample/quantile/reset.

- [ ] **Step 1: Service** — `GetTraffic(range string) (*TrafficSeries, error)`: validate range ∈ {`1h`,`24h`,`7d`} (else error); load samples via `ListSince(now - rangeDur)`; compute **per-consecutive-sample deltas** for the counters (req_*, bytes_*, duration buckets/sum/count) — **if any delta < 0 (Caddy counter reset), treat that step's deltas as 0**; **downsample** into N buckets (1h→step ~30s native, 24h→5min, 7d→hourly) by summing counter-deltas within each bucket and taking the last `in_flight`; compute **p50/p95 (ms)** per output bucket from the summed bucket-deltas via `histogram_quantile` (linear interpolation within the matched `le` bucket, standard Prometheus algorithm). Return `TrafficSeries{ Range, StepSeconds, Points []TrafficPoint }` where `TrafficPoint{ T time.Time, Req2xx/3xx/4xx/5xx/Other int64, BytesIn/Out int64, P50Ms/P95Ms float64, InFlight int64 }`.
- [ ] **Step 2: Tests** — fixture sample rows → assert: delta math; a reset (later sample lower) contributes 0 not negative; downsample bucket counts; `histogram_quantile` on known bucket deltas → expected p50/p95.
- [ ] **Step 3: Handler + RBAC + route** — `metrics_handler.go` `GetTraffic(w,r)`: read `range` (default `1h`), call the service, `utils.Success`; invalid range → `utils.BadRequest`. `rbac.yaml`: add a "Traffic Metrics" group with `metrics:read`, and add `metrics:read` to the **viewer** and **operator** role templates. `routes.go`: `r.Route("/api/metrics", …) .With(RequirePermission(authAdapter,"metrics:read",mwConfig)).Get("/traffic", metricsHandler.GetTraffic)`.
- [ ] **Step 4: Gate** + **Commit:** `feat(api): traffic metrics read endpoint (metrics:read)`.

---

## Task 4: Dashboard traffic charts

**Files:** `ui/src/types/metrics.ts`, `ui/src/hooks/use-traffic-metrics.ts`, `ui/src/components/dashboard/traffic-charts.tsx` (+ maybe split per chart), `ui/src/routes/_dashboard/index.tsx`.

- [ ] **Step 1: Types + hook** — `types/metrics.ts`: `TrafficRange = '1h'|'24h'|'7d'`, `TrafficSeries`/`TrafficPoint` matching the API. `use-traffic-metrics.ts`: `useTrafficMetrics(range)` → `api.get(\`metrics/traffic?range=${range}\`).json<ApiResponse<TrafficSeries>>()` returning `.data`, `refetchInterval: 30_000`.
- [ ] **Step 2: Charts** — a `TrafficCharts` component with a range selector (1h/24h/7d via rnui Tabs/Select) and three rnui chart `Card`s: **Requests** (`AreaChart stacked` — `categories` = formatted point times; `series` = 2xx/3xx/4xx/5xx/other), **Bandwidth** (`AreaChart` — in/out), **Latency** (`LineChart` — p50/p95 ms). Map `points[]` → `categories` + parallel `series.data` arrays. Loading `Skeleton`; empty → "No traffic data yet."
- [ ] **Step 3: Mount** — in `routes/_dashboard/index.tsx`, insert `<TrafficCharts />` (its own `grid gap-6 lg:grid-cols-2`/stacked) after `<DashboardStatCards>` and before the fleet/actions grid.
- [ ] **Step 4: Gate** + **Commit:** `feat(ui): dashboard traffic charts (requests/bandwidth/latency)`.

---

## Task 5: External metrics exposure — backend

**Files:** `backend/internal/models/setting.go` (add `MetricsPublishSettings` struct + keys), `backend/internal/repository/settings_repository.go` (+ interface) accessors, `backend/internal/api/handlers/settings_handler.go` (+ routes), `backend/internal/caddy/config/builder.go` (the protected metrics route), `backend/internal/service/sync_service.go` (load the setting into the build — it already gathers settings), `routes.go`. Tests: builder (route present only when enabled, correct matcher/handler ordering, no password leak path).

- [ ] **Step 1: Setting (key-value, no migration).** In `setting.go` add keys `metrics.publish_enabled`, `metrics.publish_host`, `metrics.publish_path`, `metrics.basic_auth_user`, `metrics.basic_auth_hash`, `metrics.allowed_cidrs` (CSV), and a `MetricsPublishSettings` struct (`Enabled bool`, `Host string`, `Path string`, `BasicAuthUser string`, `BasicAuthHash string`, `AllowedCIDRs []string`). In `settings_repository.go` add `GetMetricsPublishSettings()` (compose from `GetValue`, default path `/metrics`, `enabled=false`) and `SetMetricsPublishSettings(*MetricsPublishSettings)` (mirror `Set*NotFound*`), + interface entries.
- [ ] **Step 2: Handler + routes.** In `settings_handler.go`: `GetMetricsPublish` (returns the settings **with the hash redacted** — never return `basic_auth_hash`; return a `has_basic_auth bool` instead) and `UpdateMetricsPublish` (validate: if `enabled`, `host` required + a basic-auth user+password required; **bcrypt the submitted password** with the same util ACL basic-auth uses, store the hash; if password omitted on update, keep the existing hash; validate each CIDR parses). `routes.go`: `.Get("/metrics-publish", …)` (`settings:read`) + `.Put("/metrics-publish", …)` (`settings:write`) in the `/api/settings` group. Fire an audit event on update (like `UpdateNotFound`).
- [ ] **Step 3: Config-gen the protected route.** In `builder.go`, the builder needs the `MetricsPublishSettings` (have the sync service `SetMetricsPublishSettings(...)` on the builder during gather, mirroring `SetNotFoundSettings`; add a builder field + setter). When `enabled && host != ""`: prepend a route to `routes` (before catch-all):
  ```
  match := NewHostMatcher(host)            // one MatcherSet (AND)
  AddPathToMatcher(match, path)            // path (default /metrics) — use the existing path-add helper
  if len(cidrs) > 0 { AddRemoteIPToMatcher(match, cidrs...) }
  route := &HTTPRoute{
    Match:  []MatcherSet{match},
    Handle: []HTTPHandler{
      ToHTTPHandler(NewAuthenticationHandler([]*BasicAuthAccount{NewBasicAuthAccount(user, hash)}, "Metrics")),
      HTTPHandler{"handler": "metrics"},
    },
    Terminal: true,
  }
  ```
  (Verify the exact path-matcher helper name; if only `NewHostPathMatcher` exists, use it + add remote_ip via `AddRemoteIPToMatcher`.) The route only matches the configured host/path; basic-auth precedes the metrics handler so unauth'd scrapes are rejected. This requires `"metrics": {}` on the server (Task 1) to be present — it is.
- [ ] **Step 4: Tests** — builder: with the setting disabled, NO metrics route is emitted; enabled, the route exists with host/path[/remote_ip] match + an `authentication` handler (http_basic, the bcrypt hash) BEFORE the `{"handler":"metrics"}` handler; the generated config still marshals valid JSON. Settings repo round-trip; the GET handler never returns the hash.
- [ ] **Step 5: Gate** + **Commit:** `feat(metrics): opt-in protected external Caddy metrics route + setting`.

---

## Task 6: External metrics exposure — frontend

**Files:** `ui/src/hooks/use-settings.ts` (add `useMetricsPublishSettings`), `ui/src/types/*` (the settings type), a Settings section component `ui/src/components/settings/metrics-publish-settings.tsx`, and wire it into the Settings area (a new section route under `/settings/*` + the `SettingsNav`).

- [ ] **Step 1: Hook + type** — `useMetricsPublishSettings()` (query `settings/metrics-publish` + mutation `put`), mirroring `useNotFoundSettings`. Type: `{ enabled, host, path, basic_auth_user, has_basic_auth, allowed_cidrs }` (no hash).
- [ ] **Step 2: Form** — `metrics-publish-settings.tsx`: RHF `Form`/`FormField` (project convention) — Switch `enabled`; `host`; `path` (default `/metrics`); basic-auth `user` + `password` (password optional on edit if `has_basic_auth`, shown as "leave blank to keep"); `allowed_cidrs` (TagsInput or comma field). Zod: when `enabled`, host required + (user required, and password required unless `has_basic_auth`); validate CIDR format. On submit `update(...)`. Show the scrape URL hint `https://{host}{path}`.
- [ ] **Step 3: Wire into Settings** — add a new section to the settings IA: a `SettingsNav` entry "Metrics" → a new child route `/settings/metrics` rendering the form (mirror how `default-page`/`login-branding`/`audit-logs` sections are registered in `router.tsx` + `SettingsNav`).
- [ ] **Step 4: Gate** + **Commit:** `feat(ui): metrics-publish settings section`.

---

## Decomposition note
6 tasks across two semi-independent parts (1-4 = internal dashboard; 1+5-6 = external exposure; Task 1 is shared). This is one plan; at execution it MAY be split into two PRs (internal dashboard; external exposure) if preferred — the task boundaries already support that.

## Self-Review

**Spec coverage:** shared metrics enablement (T1); scraper+storage 30s/7d (T2); range endpoint w/ deltas+downsample+quantiles+`metrics:read` (T3); dashboard charts requests/bandwidth/latency + range selector (T4); external opt-in protected route + setting, off-by-default, basic-auth+optional CIDR, no admin exposure, hash never returned (T5); settings UI (T6). Aggregate-only; expfmt; rnui charts (no new FE dep). All addressed.

**Placeholder scan:** novel/risky code (HTTPServerMetrics, the external route construction with matchers+auth+metrics handler, the scraper, the migration/model, the histogram-quantile algorithm) is concrete or precisely specified; "verify the exact path-matcher helper / admin /metrics path" are explicit verification points, not vague TODOs.

**Type/name consistency:** `metrics:read` in rbac + the read route; `MetricsPublishSettings`/keys consistent across model↔repo↔handler↔hook; `traffic_samples`/`TrafficSample`/`TrafficSeries`/`TrafficPoint` consistent service↔handler↔FE; `server.Metrics` (`HTTPServerMetrics`) mirrors `server.Logs`; the external route reuses real helpers (`NewHostMatcher`/`AddRemoteIPToMatcher`/`NewAuthenticationHandler`/`ToHTTPHandler`/`{"handler":"metrics"}`); scraper lifecycle mirrors `SyncService.Start/Stop` (routes.go start, main.go stop).
