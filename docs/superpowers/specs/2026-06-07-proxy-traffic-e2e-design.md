# Proxy Traffic E2E Test Suite — Design

**Date:** 2026-06-07
**Status:** Approved (design)
**Author:** Alok Kumar Sahoo (with Claude)

## Problem

Waygates manages both L7 (HTTP `reverse_proxy` / `redirect` / `static`) and L4
(TCP/UDP, 9 matcher types) proxies by generating Caddy JSON config and applying
it through the Caddy admin API.

Current tests prove two things:

- **Config generation** is correct — unit tests for the Caddy JSON builders
  (`internal/caddy/config/...`), including `layer4_test.go` (67 tests covering
  every matcher, handler, LB policy, and proxy-protocol).
- **API + persistence** work — the `tests/integration` suite drives the REST API
  against a real container and verifies CRUD/validation behavior for both L7 and
  L4 proxies.

Neither layer proves the thing that matters in production: that Caddy, once it
loads the generated config, **actually routes real traffic**. No test opens a
TCP/UDP connection to an L4 listen port, and no test sends an HTTP request
through Caddy (`:80`) to a proxied hostname. The L7 integration tests
(`MakeAuthenticatedRequest`) only ever hit the `:8080` management API.

This is the gap: **we have never verified that a single byte of proxied traffic
flows end-to-end.**

## Goal

An automated, repeatable end-to-end suite that creates proxies through the API,
triggers a sync, and then drives **real traffic** through Caddy to backend
services — covering the core of both the L7 and L4 feature sets.

### Non-goals

- Exhaustive coverage of every L4 matcher. `rdp` and `socks5` are excluded
  (heavy/brittle fixtures, marginal value).
- HTTPS/ACME certificate issuance. The test environment runs with
  `CADDY_DISABLE_AUTO_HTTPS=true`; L7 traffic is exercised over `:80`.
- Performance/load testing. This validates correctness, not throughput.

## Runtime path under test

```
POST /api/l4-proxies | /api/proxies   → row(s) in Postgres
POST /api/sync/trigger                → SyncService.FullSync()
   → builds Caddy JSON (http app + layer4 app)
   → POST http://localhost:2019/load  (Caddy admin API)
   → Caddy listens on :80 and the L4 listen ports, proxies to upstreams
client traffic → Caddy → backend fixture
```

Sync also runs automatically every 60s; tests trigger it explicitly to avoid the
wait.

## Architecture (Approach A: one shared environment)

A single multi-container environment is built **once per run**; every scenario
runs as a `t.Run` subtest against it. (Per-scenario fresh environments were
rejected — ~13× full-stack startup is too slow. A separate `tests/traffic`
package with `TestMain` was rejected — it would force extracting the existing
helpers into a shared exported package for little gain.)

Components, all on one Docker network and addressable by network alias:

| Alias | Image | Role |
|-------|-------|------|
| (app) | `waygates-test:latest` | app under test (Go backend + Caddy w/ layer4) |
| `postgres` | `postgres:16-alpine` | waygates application database |
| `echo1`, `echo2` | `mendhak/http-https-echo` | HTTP+HTTPS echo; reflect method/headers/body as JSON, distinct `hostname` |
| `tcpecho` | `alpine/socat` | raw TCP echo (`socat TCP-LISTEN:N,fork,reuseaddr EXEC:/bin/cat`) |
| `pgtarget` | `postgres:16-alpine` | backend for the L4 `postgres` matcher |

The `mendhak/http-https-echo` fixture is reused widely: L7 reverse_proxy target,
round-robin LB (distinguished by `hostname`), `custom_headers` reflection,
`block_exploits` target, ACL target, the L4 `http` matcher target, and TLS SNI
passthrough (its HTTPS port).

### Port exposure constraint

testcontainers requires a container's exposed ports to be declared at startup,
before any proxy exists. The waygates container therefore pre-exposes:

- `:80` — L7 HTTP traffic.
- A fixed pool `9001–9010` — L4 listen ports.

Each L4 scenario claims a dedicated port from the pool as its `listen_port`, and
the test connects via `MappedPort(<poolPort>)`. (The existing `:8080/:443`
exposures remain.)

### Readiness (condition-based, no fixed sleeps)

After creating a scenario's proxies and calling `POST /api/sync/trigger`, the
test polls until ready (≤30s, then fail):

- **L7:** retry `GET :80` with the `Host` header until the expected status code.
- **L4:** retry `net.DialTimeout` (and, for protocol matchers, the protocol-level
  probe) until it succeeds.

### Cleanup

`t.Cleanup` terminates all fixture containers and removes the network. Existing
`ContainerTestEnv` cleanup handles the app + app-DB containers.

## Test matrix (13 scenarios)

### L7 — real HTTP through Caddy `:80`, `Host:` = proxy hostname

1. **reverse_proxy** → `echo1`: assert `200` and JSON body `hostname == echo1`.
2. **round-robin LB** → `[echo1, echo2]`: issue N requests; assert both backends
   are observed (distinct `hostname`).
3. **redirect**: assert `3xx` with the correct `Location` (client configured not
   to follow redirects).
4. **static**: assert the served body matches expected content.
5. **ACL/auth-protected**: unauthenticated request → blocked (`401` or `3xx` to
   login); with valid credentials → `200` reaching `echo1`.
6. **block_exploits=true**: a known-malicious request → `403`; a benign request →
   `200`.
7. **custom_headers**: assert the proxied response carries the injected
   `X-Test` response header.

### L4 — raw connections through Caddy listen ports

8. **`any` (TCP)** → `tcpecho`: write bytes, assert the same bytes are echoed.
9. **`http` matcher** → `echo1`: HTTP `GET` through the L4 port → `200` from
   `echo1`.
10. **`postgres` matcher** → `pgtarget`: `SELECT 1` through the proxy succeeds.
11. **`tls` SNI passthrough**: `tls.Dial` with `ServerName=a.sni` reaches
    `echo1`, `ServerName=b.sni` reaches `echo2` (asserted via the echoed
    `hostname`).
12. **`remote_ip` allow/deny**: a proxy with `allowed_ip_ranges=["0.0.0.0/0"]`
    connects; a proxy with `["203.0.113.0/24"]` (TEST-NET, excludes the client)
    is refused.
13. **L4 round-robin LB** → `[echo1, echo2]`: N requests through an `http`-matcher
    L4 proxy hit both backends.

## Scenario-specific mechanics

- **remote_ip (#12):** correctness is asserted via configuration, not by spoofing
  a source IP. `0.0.0.0/0` must connect; a TEST-NET range that cannot contain the
  client must be refused. This proves the matcher's allow/deny logic
  deterministically.
- **TLS SNI passthrough (#11):** two `mendhak` echoes serve HTTPS on `:8443` with
  their self-signed certs. The L4 proxy has one `tls` route per SNI hostname.
  `tls.Dial` uses `ServerName` + `InsecureSkipVerify` (self-signed). Routing is
  asserted via the echoed `hostname` in the JSON body, not the certificate CN.
- **ACL/auth (#5):** create an ACL group with a basic-auth credential and assign
  it to the proxy. The exact Caddy mechanism (`basicauth` vs forward-auth to the
  waygates verify endpoint) is confirmed during implementation by reading
  `internal/caddy/config/acl_builder.go` and the ACL handlers; the behavioral
  assertion (blocked when unauthenticated, allowed with credentials) is stable
  regardless.
- **custom_headers (#7):** the suite asserts an injected **response** header,
  which is directly observable on the client response. (`custom_headers` can set
  both request and response headers via `Headers.Request` / `Headers.Response`.)

## Files

- `backend/tests/integration/traffic_test.go` — `//go:build traffic`. Top-level
  `TestTraffic_L7` and `TestTraffic_L4`, each with the subtests above.
- `backend/tests/integration/traffic_harness.go` — `//go:build traffic`.
  Multi-backend environment setup (network + fixtures + waygates with the L4 port
  pool), readiness pollers, and protocol helpers (`tcpEcho`, `pgSelect1`,
  `tlsDialSNI`, `httpGetWithHost`). Reuses the existing `ContainerTestEnv`
  login/auth helpers and `MakeAuthenticatedRequest`.
- `Makefile` — new `test-traffic` target: build `waygates-test:latest`, then
  `go test -tags traffic -run Traffic ./tests/integration/ -count=1 -v`. The
  build tag keeps these tests out of `make backend-test` / `go test ./...`.

## Dependencies

- L4 `postgres` scenario (#10) uses a Go Postgres client. Reuse the driver
  already present via `gorm.io/driver/postgres` (pgx); no new top-level
  dependency anticipated. Confirm during implementation.
- All fixture images are public and pulled by testcontainers at run time.

## Risks & mitigations

- **Caddy reload timing:** mitigated by explicit `POST /api/sync/trigger` +
  condition-based readiness polling.
- **Fixture image availability/network:** images are public; a failed pull fails
  the run loudly (acceptable for an opt-in suite). Documented as a prerequisite.
- **L4 port pool exhaustion:** the pool (`9001–9010`) exceeds the number of L4
  scenarios; scenarios claim distinct ports.
- **ACL auth mechanism uncertainty:** resolved during implementation; the spec's
  assertion is behavior-level and mechanism-agnostic.

## Success criteria

- `make test-traffic` builds the image and runs the suite green.
- All 13 scenarios drive real traffic through Caddy and assert backend delivery
  or correct rejection.
- `make backend-test` is unaffected (suite excluded by build tag).
