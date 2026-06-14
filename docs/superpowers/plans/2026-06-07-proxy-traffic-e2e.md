# Proxy Traffic E2E Suite — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an opt-in, build-tagged integration suite that drives real traffic through Caddy for both L7 and L4 proxies, proving end-to-end routing (not just config generation and the API).

**Architecture:** One shared multi-container environment (waygates app + Postgres + fixture backends on a Docker network), built once, with every scenario run as a `t.Run` subtest. Each subtest creates its proxy(s) via the REST API, calls `POST /api/sync/trigger`, polls until Caddy is serving (condition-based, no fixed sleeps), then sends real traffic and asserts backend delivery or correct rejection.

**Tech Stack:** Go, `testcontainers-go`, Caddy (with `caddy-l4`), Postgres, fixtures `mendhak/http-https-echo` (×2), `alpine/socat`, `postgres:16-alpine`; `github.com/jackc/pgx/v5` for the L4 Postgres probe.

**Reference spec:** `docs/superpowers/specs/2026-06-07-proxy-traffic-e2e-design.md`

---

## Background the implementer needs

The suite lives in the existing package `backend/tests/integration` (package `integration`) but in **build-tagged files** (`//go:build traffic`) so it is excluded from `make backend-test` / `go test ./...`. It reuses these existing helpers from `api_test.go` (same package, no import needed):

- `type ContainerTestEnv struct { Network; PostgresContainer; WaygatesContainer; BaseURL string; AccessToken string; RefreshToken string; ctx context.Context }`
- `(env *ContainerTestEnv) RegisterAndLogin(t)` — registers `testuser` and populates `env.AccessToken`.
- `(env *ContainerTestEnv) MakeAuthenticatedRequest(t, method, path string, body interface{}) *http.Response` — JSON body, Bearer auth, hits `env.BaseURL` (the `:8080` API).
- `(env *ContainerTestEnv) ReadJSONResponse(t, resp, &v)` — decodes a response body into `v`.
- `findTestJSONConfig(t) string` — returns the host path of the test Caddy JSON config (auto-HTTPS disabled).

**Prerequisite:** the `waygates-test:latest` image must exist. The `make test-traffic` target (Task 1) builds it first. Docker must be running.

**Runtime contract (verified):**
- L7 proxies are served by Caddy on `:80` (auto-HTTPS disabled in the test env). To hit hostname `H`, send `GET http://<host>:<mapped80>/...` with `req.Host = H`.
- After creating proxies, `POST /api/sync/trigger` runs `SyncService.FullSync()`, which rebuilds the full Caddy JSON (http + layer4 apps) and applies it via the Caddy admin API. Config is cumulative across calls — creating unique proxies per subtest is safe.

**API request bodies (verified against existing tests):**
- L7 reverse_proxy: `{"type":"reverse_proxy","name":..,"hostname":..,"upstreams":[{"host":..,"port":..,"scheme":"http"}],"load_balancing":{"strategy":"round_robin"}}`
- L7 redirect: `{"type":"redirect","name":..,"hostname":..,"redirect":{"target":..,"status_code":301,"preserve_path":true,"preserve_query":true}}`
- L7 static: `{"type":"static","name":..,"hostname":..,"static":{"root_path":..,"index_file":"index.html"}}`
- L7 security: add `"block_exploits":true` and/or `"custom_headers":{...}` to a reverse_proxy body (custom_headers shape confirmed in Task 6).
- L4: `{"name":..,"listen_port":<int>,"protocol":"tcp","routes":[{"matcher_type":..,"load_balancing_policy":"round_robin","upstreams":[{"host":..,"port":..}], ...}]}`
- ACL: `POST /api/acl/groups {"name":..,"description":..}` → `id`; `POST /api/acl/groups/{id}/basic-auth {"username":..,"password":..}`; assign `POST /api/proxies/{proxyID}/acl {"acl_group_id":<id>,"path_pattern":"/*","priority":10}`.

**Fixture details:**
- `mendhak/http-https-echo:31` listens on `8080` (HTTP) and `8443` (HTTPS, built-in self-signed cert). It returns a JSON body that includes the container hostname. Set `ContainerRequest.Hostname` to `echo1`/`echo2` to make backends distinguishable.
- `alpine/socat` entrypoint is `socat`; pass args via `Cmd`. TCP echo: `Cmd: []string{"TCP-LISTEN:7000,fork,reuseaddr","EXEC:cat"}`.
- `postgres:16-alpine` with `POSTGRES_USER/PASSWORD/DB = waygates`.

---

## File structure

- **Create** `backend/tests/integration/traffic_harness.go` (`//go:build traffic`) — `TrafficEnv`, `SetupTrafficEnvironment`, and all helpers (sync trigger, L7 client/poller, L4 addr/poller, protocol probes, echo-hostname decode).
- **Create** `backend/tests/integration/traffic_l7_test.go` (`//go:build traffic`) — `TestTraffic_L7` with one subtest per L7 scenario.
- **Create** `backend/tests/integration/traffic_l4_test.go` (`//go:build traffic`) — `TestTraffic_L4` with one subtest per L4 scenario.
- **Modify** `Makefile` — add `test-traffic` target.
- **Modify** `go.mod` / `go.sum` — promote `github.com/jackc/pgx/v5` to a direct dependency.
- **Modify** `docs/DEPLOYMENT.md` (or `backend/README.md`) — document `make test-traffic` and its prerequisites.

---

## Task 1: Harness + smoke test + Makefile target

**Files:**
- Create: `backend/tests/integration/traffic_harness.go`
- Create: `backend/tests/integration/traffic_l7_test.go` (smoke test placeholder for now)
- Modify: `Makefile`

- [ ] **Step 1: Add the `test-traffic` Makefile target**

Add near the other test targets in `Makefile`:

```makefile
.PHONY: test-traffic
test-traffic: ## Build the image and run the proxy traffic E2E suite (Docker required)
	@echo "Building waygates-test:latest..."
	@docker build -t waygates-test:latest .
	@echo "Running proxy traffic E2E suite..."
	@cd backend && go test -tags traffic -run 'TestTraffic' ./tests/integration/ -count=1 -v
```

- [ ] **Step 2: Write the harness file**

Create `backend/tests/integration/traffic_harness.go`:

```go
//go:build traffic

package integration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// l4PortPool is the fixed set of L4 listen ports pre-exposed on the waygates
// container. testcontainers requires ports to be declared at startup, before any
// proxy exists, so each L4 subtest claims a distinct port from this pool.
var l4PortPool = []int{9001, 9002, 9003, 9004, 9005, 9006, 9007, 9008, 9009, 9010}

// TrafficEnv extends ContainerTestEnv with backend fixtures and mapped-port access.
type TrafficEnv struct {
	*ContainerTestEnv
	fixtures []testcontainers.Container
	host     string // docker host as seen from the test process
	http80   string // mapped :80 port
}

// SetupTrafficEnvironment builds the full multi-container environment once.
func SetupTrafficEnvironment(t *testing.T) *TrafficEnv {
	t.Helper()
	ctx := context.Background()
	base := &ContainerTestEnv{ctx: ctx}

	net1, err := network.New(ctx, network.WithDriver("bridge"))
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	base.Network = net1
	netName := net1.Name

	env := &TrafficEnv{ContainerTestEnv: base}

	// --- App database ---
	base.PostgresContainer = mustStart(t, ctx, testcontainers.ContainerRequest{
		Image:          "postgres:16-alpine",
		ExposedPorts:   []string{"5432/tcp"},
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {"postgres"}},
		Env:            map[string]string{"POSTGRES_USER": "waygates", "POSTGRES_PASSWORD": "waygates", "POSTGRES_DB": "waygates"},
		WaitingFor:     wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	})

	// --- Fixtures ---
	echo1 := mustStart(t, ctx, echoRequest(netName, "echo1"))
	echo2 := mustStart(t, ctx, echoRequest(netName, "echo2"))
	tcpecho := mustStart(t, ctx, testcontainers.ContainerRequest{
		Image:          "alpine/socat",
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {"tcpecho"}},
		Cmd:            []string{"TCP-LISTEN:7000,fork,reuseaddr", "EXEC:cat"},
		WaitingFor:     wait.ForListeningPort("7000/tcp").WithStartupTimeout(30 * time.Second),
	})
	pgtarget := mustStart(t, ctx, testcontainers.ContainerRequest{
		Image:          "postgres:16-alpine",
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {"pgtarget"}},
		Env:            map[string]string{"POSTGRES_USER": "waygates", "POSTGRES_PASSWORD": "waygates", "POSTGRES_DB": "waygates"},
		WaitingFor:     wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	})
	env.fixtures = []testcontainers.Container{echo1, echo2, tcpecho, pgtarget}

	// --- App under test ---
	exposed := []string{"8080/tcp", "80/tcp", "443/tcp"}
	for _, p := range l4PortPool {
		exposed = append(exposed, fmt.Sprintf("%d/tcp", p))
	}
	base.WaygatesContainer = mustStart(t, ctx, testcontainers.ContainerRequest{
		Image:        "waygates-test:latest",
		ExposedPorts: exposed,
		Networks:     []string{netName},
		Files: []testcontainers.ContainerFile{{
			HostFilePath:      findTestJSONConfig(t),
			ContainerFilePath: "/etc/caddy/caddy.json",
			FileMode:          0o644,
		}},
		Env: map[string]string{
			"DB_HOST": "postgres", "DB_PORT": "5432", "DB_USER": "waygates",
			"DB_PASSWORD": "waygates", "DB_NAME": "waygates",
			"JWT_SECRET": "test-secret-key-that-is-at-least-32-characters-long",
			"JWT_ACCESS_EXPIRY": "15m", "JWT_REFRESH_EXPIRY": "24h",
			"BCRYPT_COST": "4", "CORS_ORIGINS": "*",
			"RBAC_PATH": "/app/backend/rbac.yaml", "CADDY_EMAIL": "test@example.com",
			"CADDY_DISABLE_AUTO_HTTPS": "true",
			"CLOUDFLARE_EMAIL": "test@example.com", "CLOUDFLARE_API_TOKEN": "dummy-token-for-testing",
			"LOG_LEVEL": "debug", "LOG_FORMAT": "console", "UI_ENABLED": "false",
		},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/api/health").WithPort("8080/tcp").WithStartupTimeout(120*time.Second),
			wait.ForLog("Caddy is ready!").WithStartupTimeout(60*time.Second),
		),
	})

	host, err := base.WaygatesContainer.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	env.host = host
	apiPort, err := base.WaygatesContainer.MappedPort(ctx, nat.Port("8080/tcp"))
	if err != nil {
		t.Fatalf("mapped 8080: %v", err)
	}
	base.BaseURL = fmt.Sprintf("http://%s:%s", host, apiPort.Port())
	p80, err := base.WaygatesContainer.MappedPort(ctx, nat.Port("80/tcp"))
	if err != nil {
		t.Fatalf("mapped 80: %v", err)
	}
	env.http80 = p80.Port()

	t.Cleanup(func() { env.cleanup(t) })
	base.RegisterAndLogin(t)
	return env
}

func echoRequest(netName, alias string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:          "mendhak/http-https-echo:31",
		Hostname:       alias, // surfaces in the echoed JSON so backends are distinguishable
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {alias}},
		WaitingFor:     wait.ForListeningPort("8080/tcp").WithStartupTimeout(30 * time.Second),
	}
}

func mustStart(t *testing.T, ctx context.Context, req testcontainers.ContainerRequest) testcontainers.Container {
	t.Helper()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("start container %s: %v", req.Image, err)
	}
	return c
}

func (e *TrafficEnv) cleanup(t *testing.T) {
	ctx := context.Background()
	for _, c := range e.fixtures {
		_ = c.Terminate(ctx)
	}
	e.ContainerTestEnv.Cleanup(t) // terminates waygates + app-DB, removes network
}

// --- Sync + readiness helpers ---

func (e *TrafficEnv) triggerSync(t *testing.T) {
	t.Helper()
	resp := e.MakeAuthenticatedRequest(t, http.MethodPost, "/api/sync/trigger", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("sync trigger failed: %d - %s", resp.StatusCode, string(body))
	}
}

func (e *TrafficEnv) l7URL() string { return "http://" + net.JoinHostPort(e.host, e.http80) }

// noRedirectClient returns the last response instead of following redirects.
func noRedirectClient() *http.Client {
	return &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// l7Get performs one GET against Caddy :80 with the given Host header.
func (e *TrafficEnv) l7Get(t *testing.T, hostHeader, path string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.l7URL()+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = hostHeader
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		return nil, nil
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, body
}

// waitL7 polls until a GET with the Host header returns wantStatus (≤30s).
func (e *TrafficEnv) waitL7(t *testing.T, hostHeader string, wantStatus int) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, _ := e.l7Get(t, hostHeader, "/", nil)
		if resp != nil && resp.StatusCode == wantStatus {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for L7 host %q to return %d", hostHeader, wantStatus)
}

// l4Addr returns host:mappedPort for a pre-exposed L4 pool port.
func (e *TrafficEnv) l4Addr(t *testing.T, poolPort int) string {
	t.Helper()
	mp, err := e.WaygatesContainer.MappedPort(e.ctx, nat.Port(fmt.Sprintf("%d/tcp", poolPort)))
	if err != nil {
		t.Fatalf("mapped L4 port %d: %v", poolPort, err)
	}
	return net.JoinHostPort(e.host, mp.Port())
}

// waitL4 polls until a TCP dial to addr succeeds (≤30s).
func (e *TrafficEnv) waitL4(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for L4 addr %q", addr)
}

// --- Protocol probes ---

// tcpEcho writes payload to addr and returns what comes back (read once).
func tcpEcho(t *testing.T, addr, payload string) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte(payload)); err != nil {
		return "", err
	}
	buf := make([]byte, len(payload))
	n, err := io.ReadFull(conn, buf)
	return string(buf[:n]), err
}

// pgSelect1 connects through addr to a Postgres backend and runs SELECT 1.
func pgSelect1(t *testing.T, addr string) error {
	dsn := fmt.Sprintf("postgres://waygates:waygates@%s/waygates?sslmode=disable", addr)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(ctx) }()
	var n int
	return conn.QueryRow(ctx, "SELECT 1").Scan(&n)
}

// tlsEchoHostname dials addr with TLS+SNI, makes an HTTP GET over it, and returns
// the backend hostname from the echoed JSON.
func tlsEchoHostname(t *testing.T, addr, sni string) (string, error) {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", addr, &tls.Config{ServerName: sni, InsecureSkipVerify: true})
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", sni)
	raw, _ := io.ReadAll(conn)
	return echoHostnameFromHTTP(string(raw)), nil
}

// echoHostnameFromBody extracts the backend hostname from mendhak echo JSON.
func echoHostnameFromBody(body []byte) string {
	var v struct {
		Hostname string `json:"hostname"`
		OS       struct {
			Hostname string `json:"hostname"`
		} `json:"os"`
	}
	_ = json.Unmarshal(body, &v)
	if v.Hostname != "" {
		return v.Hostname
	}
	return v.OS.Hostname
}

// echoHostnameFromHTTP splits a raw HTTP/1.1 response and parses the JSON body.
func echoHostnameFromHTTP(raw string) string {
	idx := strings.Index(raw, "\r\n\r\n")
	if idx < 0 {
		return ""
	}
	return echoHostnameFromBody([]byte(raw[idx+4:]))
}
```

- [ ] **Step 3: Write the smoke test**

Create `backend/tests/integration/traffic_l7_test.go`:

```go
//go:build traffic

package integration

import (
	"net/http"
	"testing"
)

func TestTraffic_Smoke(t *testing.T) {
	env := SetupTrafficEnvironment(t)
	resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/health", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected health 200, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 4: Add the pgx dependency**

Run: `cd backend && go get github.com/jackc/pgx/v5 && go mod tidy`
Expected: `pgx/v5` becomes a direct require in `go.mod`.

- [ ] **Step 5: Run the smoke test**

Run: `make test-traffic`
Expected: image builds; `TestTraffic_Smoke` PASS (full stack — waygates, Postgres, 2 echoes, socat, pgtarget — boots).

- [ ] **Step 6: Commit**

```bash
git add backend/tests/integration/traffic_harness.go backend/tests/integration/traffic_l7_test.go Makefile backend/go.mod backend/go.sum
git commit -m "test: scaffold proxy traffic E2E harness and smoke test"
```

---

## Task 2: L7 reverse_proxy

**Files:**
- Modify: `backend/tests/integration/traffic_l7_test.go`

- [ ] **Step 1: Add `TestTraffic_L7` with the reverse_proxy subtest**

Replace the smoke test's standalone function by folding it into a shared `TestTraffic_L7` that sets up the env once and runs subtests. Add to `traffic_l7_test.go`:

```go
func TestTraffic_L7(t *testing.T) {
	env := SetupTrafficEnvironment(t)

	t.Run("reverse_proxy", func(t *testing.T) {
		host := "rp.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "rp", "hostname": host,
			"upstreams": []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		got, body := env.l7Get(t, host, "/", nil)
		if got.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", got.StatusCode)
		}
		if hn := echoHostnameFromBody(body); hn != "echo1" {
			t.Fatalf("expected backend echo1, got %q", hn)
		}
	})
}
```

(Keep `TestTraffic_Smoke` or delete it — the env setup in `TestTraffic_L7` supersedes it. Deleting keeps the run lean.)

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L7/reverse_proxy' ./tests/integration/ -count=1 -v`
Expected: PASS — request to `:80` with `Host: rp.test.local` reaches `echo1`.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l7_test.go
git commit -m "test: L7 reverse_proxy traffic e2e"
```

---

## Task 3: L7 round-robin load balancing

**Files:**
- Modify: `backend/tests/integration/traffic_l7_test.go`

- [ ] **Step 1: Add the LB subtest inside `TestTraffic_L7`**

```go
	t.Run("round_robin_lb", func(t *testing.T) {
		host := "lb.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "lb", "hostname": host,
			"upstreams": []map[string]any{
				{"host": "echo1", "port": 8080, "scheme": "http"},
				{"host": "echo2", "port": 8080, "scheme": "http"},
			},
			"load_balancing": map[string]any{"strategy": "round_robin"},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		seen := map[string]bool{}
		for i := 0; i < 10; i++ {
			_, body := env.l7Get(t, host, "/", nil)
			seen[echoHostnameFromBody(body)] = true
		}
		if !seen["echo1"] || !seen["echo2"] {
			t.Fatalf("expected both backends, saw %v", seen)
		}
	})
```

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L7/round_robin_lb' ./tests/integration/ -count=1 -v`
Expected: PASS — 10 requests hit both `echo1` and `echo2`.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l7_test.go
git commit -m "test: L7 round-robin load balancing traffic e2e"
```

---

## Task 4: L7 redirect

**Files:**
- Modify: `backend/tests/integration/traffic_l7_test.go`

- [ ] **Step 1: Add the redirect subtest inside `TestTraffic_L7`**

```go
	t.Run("redirect", func(t *testing.T) {
		host := "rd.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "redirect", "name": "rd", "hostname": host,
			"redirect": map[string]any{
				"target": "https://target.test.local", "status_code": 301,
				"preserve_path": true, "preserve_query": true,
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusMovedPermanently)

		got, _ := env.l7Get(t, host, "/somepath", nil)
		if got.StatusCode != http.StatusMovedPermanently {
			t.Fatalf("expected 301, got %d", got.StatusCode)
		}
		loc := got.Header.Get("Location")
		if !strings.HasPrefix(loc, "https://target.test.local") {
			t.Fatalf("unexpected Location: %q", loc)
		}
	})
```

Add `"strings"` to the imports of `traffic_l7_test.go`.

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L7/redirect' ./tests/integration/ -count=1 -v`
Expected: PASS — 301 with `Location` starting `https://target.test.local`.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l7_test.go
git commit -m "test: L7 redirect traffic e2e"
```

---

## Task 5: L7 static

**Files:**
- Modify: `backend/tests/integration/traffic_harness.go` (mount a static file into the waygates container)
- Modify: `backend/tests/integration/traffic_l7_test.go`

- [ ] **Step 1: Mount a static index file into the app container**

In `SetupTrafficEnvironment`, add a second entry to the waygates `Files` slice so a known file exists to serve:

```go
		Files: []testcontainers.ContainerFile{
			{HostFilePath: findTestJSONConfig(t), ContainerFilePath: "/etc/caddy/caddy.json", FileMode: 0o644},
			{Reader: strings.NewReader("WAYGATES STATIC OK"), ContainerFilePath: "/var/www/test/index.html", FileMode: 0o644},
		},
```

Add `"strings"` to `traffic_harness.go` imports. (`ContainerFile.Reader` writes the in-memory content into the container at the given path.)

- [ ] **Step 2: Add the static subtest inside `TestTraffic_L7`**

```go
	t.Run("static", func(t *testing.T) {
		host := "st.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "static", "name": "st", "hostname": host,
			"static": map[string]any{"root_path": "/var/www/test", "index_file": "index.html"},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		_, body := env.l7Get(t, host, "/", nil)
		if !strings.Contains(string(body), "WAYGATES STATIC OK") {
			t.Fatalf("static content not served, got: %q", string(body))
		}
	})
```

- [ ] **Step 3: Run**

Run: `make test-traffic` (the mounted file requires a fresh container, so rebuild+rerun the whole suite, or at minimum re-run `-run 'TestTraffic_L7'`)
Expected: PASS — static body contains `WAYGATES STATIC OK`.

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/traffic_harness.go backend/tests/integration/traffic_l7_test.go
git commit -m "test: L7 static file serving traffic e2e"
```

---

## Task 6: L7 custom_headers

**Files:**
- Modify: `backend/tests/integration/traffic_l7_test.go`

- [ ] **Step 1: Confirm the `custom_headers` JSON shape**

Read `backend/internal/validation/proxy.go` and `backend/internal/caddy/config/*` for the `custom_headers` request struct. The builder maps to `Headers.Request` / `Headers.Response`. Use the shape the validation layer expects; the most likely form is:

```json
"custom_headers": {"response": {"X-Test": "waygates"}}
```

If validation expects a different shape (e.g. an array of `{name,value,type}`), use that instead — match the struct exactly. Document the confirmed shape in a one-line comment in the test.

- [ ] **Step 2: Add the custom_headers subtest inside `TestTraffic_L7`**

```go
	t.Run("custom_headers", func(t *testing.T) {
		host := "hdr.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "hdr", "hostname": host,
			"upstreams":      []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
			"custom_headers": map[string]any{"response": map[string]string{"X-Test": "waygates"}}, // shape confirmed in Step 1
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		got, _ := env.l7Get(t, host, "/", nil)
		if got.Header.Get("X-Test") != "waygates" {
			t.Fatalf("expected X-Test response header, headers: %v", got.Header)
		}
	})
```

- [ ] **Step 3: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L7/custom_headers' ./tests/integration/ -count=1 -v`
Expected: PASS — response carries `X-Test: waygates`.

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/traffic_l7_test.go
git commit -m "test: L7 custom response header injection traffic e2e"
```

---

## Task 7: L7 block_exploits

**Files:**
- Modify: `backend/tests/integration/traffic_l7_test.go`

- [ ] **Step 1: Identify a request the block_exploits ruleset blocks**

Read `backend/internal/caddy/config/` for how `block_exploits` builds its matchers (path traversal, known bad user-agents, query patterns). Pick one deterministic trigger — commonly a path-traversal request such as `GET /../../etc/passwd` or a flagged `User-Agent`. Use whatever the ruleset actually matches; verify by reading the generated rules.

- [ ] **Step 2: Add the block_exploits subtest inside `TestTraffic_L7`**

```go
	t.Run("block_exploits", func(t *testing.T) {
		host := "sec.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "sec", "hostname": host,
			"upstreams":      []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
			"block_exploits": true,
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK) // benign path is reachable

		// Benign request reaches the backend.
		ok, body := env.l7Get(t, host, "/", nil)
		if ok.StatusCode != http.StatusOK || echoHostnameFromBody(body) != "echo1" {
			t.Fatalf("benign request should reach echo1, got %d / %q", ok.StatusCode, echoHostnameFromBody(body))
		}
		// Malicious request is blocked (confirm exact trigger + status in Step 1).
		bad, _ := env.l7Get(t, host, "/index.php?cmd=../../etc/passwd", nil)
		if bad.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for exploit request, got %d", bad.StatusCode)
		}
	})
```

- [ ] **Step 3: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L7/block_exploits' ./tests/integration/ -count=1 -v`
Expected: PASS — benign 200 from echo1; exploit request 403.

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/traffic_l7_test.go
git commit -m "test: L7 block_exploits traffic e2e"
```

---

## Task 8: L7 ACL basic-auth

**Files:**
- Modify: `backend/tests/integration/traffic_l7_test.go`

- [ ] **Step 1: Confirm the basic-auth ACL path**

Read `backend/internal/caddy/config/acl_builder.go` and the ACL handlers to confirm that a group with a basic-auth user causes Caddy to enforce HTTP Basic Auth on the assigned proxy (unauthenticated → 401; correct credentials → upstream). Confirm the `AddBasicAuthUser` body field names (`username`, `password`).

- [ ] **Step 2: Add the ACL subtest inside `TestTraffic_L7`**

```go
	t.Run("acl_basic_auth", func(t *testing.T) {
		host := "acl.test.local"
		// Proxy
		pr := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "acl", "hostname": host,
			"upstreams": []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
		})
		var pid struct{ Data struct{ ID int } }
		env.ReadJSONResponse(t, pr, &pid)
		// Group
		gr := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/acl/groups", map[string]any{
			"name": "Traffic ACL", "description": "basic-auth e2e",
		})
		var gid struct{ Data struct{ ID int } }
		env.ReadJSONResponse(t, gr, &gid)
		// Basic-auth user
		ba := env.MakeAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/acl/groups/%d/basic-auth", gid.Data.ID), map[string]any{
			"username": "acluser", "password": "aclpass123",
		})
		_ = ba.Body.Close()
		// Assign group to proxy
		as := env.MakeAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/proxies/%d/acl", pid.Data.ID), map[string]any{
			"acl_group_id": gid.Data.ID, "path_pattern": "/*", "priority": 10,
		})
		_ = as.Body.Close()

		env.triggerSync(t)
		env.waitL7(t, host, http.StatusUnauthorized) // protected: unauthenticated → 401

		// Unauthenticated → 401
		un, _ := env.l7Get(t, host, "/", nil)
		if un.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 unauthenticated, got %d", un.StatusCode)
		}
		// Authenticated → 200 reaching echo1
		cred := base64.StdEncoding.EncodeToString([]byte("acluser:aclpass123"))
		au, body := env.l7Get(t, host, "/", map[string]string{"Authorization": "Basic " + cred})
		if au.StatusCode != http.StatusOK || echoHostnameFromBody(body) != "echo1" {
			t.Fatalf("expected authed 200 from echo1, got %d / %q", au.StatusCode, echoHostnameFromBody(body))
		}
	})
```

Add `"encoding/base64"` and `"fmt"` to the imports of `traffic_l7_test.go`.

> If Step 1 reveals the mechanism is forward-auth (redirect to login) rather than Basic Auth, change the unauthenticated expectation to `http.StatusFound` (302) and the `waitL7` target accordingly, and authenticate via the documented session flow instead of the `Authorization` header. The assertion intent — blocked then allowed — is unchanged.

- [ ] **Step 3: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L7/acl_basic_auth' ./tests/integration/ -count=1 -v`
Expected: PASS — unauthenticated 401, authenticated 200 from echo1.

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/traffic_l7_test.go
git commit -m "test: L7 ACL basic-auth traffic e2e"
```

---

## Task 9: L4 `any` (raw TCP echo)

**Files:**
- Create: `backend/tests/integration/traffic_l4_test.go`

- [ ] **Step 1: Create `TestTraffic_L4` with the TCP echo subtest**

```go
//go:build traffic

package integration

import (
	"net/http"
	"testing"
)

func TestTraffic_L4(t *testing.T) {
	env := SetupTrafficEnvironment(t)

	t.Run("any_tcp_echo", func(t *testing.T) {
		port := l4PortPool[0] // 9001
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-tcp", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "any",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "tcpecho", "port": 7000}},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		got, err := tcpEcho(t, addr, "ping-waygates")
		if err != nil {
			t.Fatalf("tcp echo: %v", err)
		}
		if got != "ping-waygates" {
			t.Fatalf("expected echo, got %q", got)
		}
	})
}
```

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L4/any_tcp_echo' ./tests/integration/ -count=1 -v`
Expected: PASS — bytes written to the L4 port are echoed back via `tcpecho`.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l4_test.go
git commit -m "test: L4 any/TCP echo traffic e2e"
```

---

## Task 10: L4 `http` matcher

**Files:**
- Modify: `backend/tests/integration/traffic_harness.go`
- Modify: `backend/tests/integration/traffic_l4_test.go`

- [ ] **Step 1: Add the plain-HTTP-through-L4 probe to `traffic_harness.go`**

```go
// httpEchoHostname does a plain HTTP GET to a raw addr (no TLS) and returns the
// backend hostname from the echoed JSON.
func httpEchoHostname(t *testing.T, addr string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + addr + "/")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return echoHostnameFromBody(body), nil
}
```

- [ ] **Step 2: Add the http-matcher subtest inside `TestTraffic_L4`**

```go
	t.Run("http_matcher", func(t *testing.T) {
		port := l4PortPool[1] // 9002
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-http", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "http",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "echo1", "port": 8080}},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		hn, err := httpEchoHostname(t, addr) // helper added in Step 1
		if err != nil {
			t.Fatalf("http through L4: %v", err)
		}
		if hn != "echo1" {
			t.Fatalf("expected echo1 via L4 http matcher, got %q", hn)
		}
	})
```

- [ ] **Step 3: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L4/http_matcher' ./tests/integration/ -count=1 -v`
Expected: PASS — HTTP through the L4 `http`-matcher port reaches `echo1`.

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/traffic_l4_test.go backend/tests/integration/traffic_harness.go
git commit -m "test: L4 http matcher traffic e2e"
```

---

## Task 11: L4 round-robin load balancing

**Files:**
- Modify: `backend/tests/integration/traffic_l4_test.go`

- [ ] **Step 1: Add the L4 LB subtest inside `TestTraffic_L4`**

```go
	t.Run("round_robin_lb", func(t *testing.T) {
		port := l4PortPool[2] // 9003
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-lb", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "http",
				"load_balancing_policy": "round_robin",
				"upstreams": []map[string]any{
					{"host": "echo1", "port": 8080},
					{"host": "echo2", "port": 8080},
				},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		seen := map[string]bool{}
		for i := 0; i < 10; i++ {
			hn, err := httpEchoHostname(t, addr)
			if err != nil {
				t.Fatalf("request %d: %v", i, err)
			}
			seen[hn] = true
		}
		if !seen["echo1"] || !seen["echo2"] {
			t.Fatalf("expected both backends, saw %v", seen)
		}
	})
```

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L4/round_robin_lb' ./tests/integration/ -count=1 -v`
Expected: PASS — 10 requests through the L4 port hit both echoes.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l4_test.go
git commit -m "test: L4 round-robin load balancing traffic e2e"
```

---

## Task 12: L4 `remote_ip` allow/deny

**Files:**
- Modify: `backend/tests/integration/traffic_l4_test.go`

- [ ] **Step 1: Add the remote_ip subtest inside `TestTraffic_L4`**

```go
	t.Run("remote_ip_allow_deny", func(t *testing.T) {
		// Allow: 0.0.0.0/0 always includes the client → echoes.
		allowPort := l4PortPool[3] // 9004
		r1 := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-allow", "listen_port": allowPort, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "remote_ip",
				"allowed_ip_ranges":     []string{"0.0.0.0/0"},
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "tcpecho", "port": 7000}},
			}},
		})
		_ = r1.Body.Close()
		// Deny: 203.0.113.0/24 (TEST-NET-3) cannot contain the client → no route → closed.
		denyPort := l4PortPool[4] // 9005
		r2 := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-deny", "listen_port": denyPort, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "remote_ip",
				"allowed_ip_ranges":     []string{"203.0.113.0/24"},
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "tcpecho", "port": 7000}},
			}},
		})
		_ = r2.Body.Close()
		env.triggerSync(t)

		allowAddr := env.l4Addr(t, allowPort)
		env.waitL4(t, allowAddr)
		if got, err := tcpEcho(t, allowAddr, "allowed"); err != nil || got != "allowed" {
			t.Fatalf("allow case should echo, got %q err %v", got, err)
		}

		// Deny case: TCP connect to Caddy may succeed, but with no matching route the
		// connection is closed, so the echo read must fail / return nothing.
		denyAddr := env.l4Addr(t, denyPort)
		if got, err := tcpEcho(t, denyAddr, "blocked"); err == nil && got == "blocked" {
			t.Fatalf("deny case should NOT echo, but it did")
		}
	})
```

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L4/remote_ip_allow_deny' ./tests/integration/ -count=1 -v`
Expected: PASS — allow port echoes; deny port does not.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l4_test.go
git commit -m "test: L4 remote_ip allow/deny traffic e2e"
```

---

## Task 13: L4 `postgres` matcher

**Files:**
- Modify: `backend/tests/integration/traffic_l4_test.go`

- [ ] **Step 1: Add the postgres subtest inside `TestTraffic_L4`**

```go
	t.Run("postgres_matcher", func(t *testing.T) {
		port := l4PortPool[5] // 9006
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-pg", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{{
				"matcher_type":          "postgres",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]any{{"host": "pgtarget", "port": 5432}},
			}},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		if err := pgSelect1(t, addr); err != nil {
			t.Fatalf("SELECT 1 through L4 postgres proxy: %v", err)
		}
	})
```

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L4/postgres_matcher' ./tests/integration/ -count=1 -v`
Expected: PASS — `SELECT 1` succeeds through the L4 `postgres`-matcher proxy.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l4_test.go
git commit -m "test: L4 postgres matcher traffic e2e"
```

---

## Task 14: L4 `tls` SNI passthrough

**Files:**
- Modify: `backend/tests/integration/traffic_l4_test.go`

- [ ] **Step 1: Add the TLS SNI subtest inside `TestTraffic_L4`**

A single L4 proxy with two `tls` routes, one per SNI, each routing to a different echo's HTTPS port (`8443`). Routing is asserted via the echoed backend hostname.

```go
	t.Run("tls_sni_passthrough", func(t *testing.T) {
		port := l4PortPool[6] // 9007
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", map[string]any{
			"name": "l4-tls", "listen_port": port, "protocol": "tcp",
			"routes": []map[string]any{
				{
					"matcher_type":          "tls",
					"sni_hostnames":         []string{"a.sni.test"},
					"tls_passthrough":       true,
					"tls_terminate":         false,
					"load_balancing_policy": "round_robin",
					"upstreams":             []map[string]any{{"host": "echo1", "port": 8443}},
				},
				{
					"matcher_type":          "tls",
					"sni_hostnames":         []string{"b.sni.test"},
					"tls_passthrough":       true,
					"tls_terminate":         false,
					"load_balancing_policy": "round_robin",
					"upstreams":             []map[string]any{{"host": "echo2", "port": 8443}},
				},
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create l4 proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		addr := env.l4Addr(t, port)
		env.waitL4(t, addr)

		hnA, err := tlsEchoHostname(t, addr, "a.sni.test")
		if err != nil {
			t.Fatalf("tls dial a: %v", err)
		}
		if hnA != "echo1" {
			t.Fatalf("SNI a.sni.test should reach echo1, got %q", hnA)
		}
		hnB, err := tlsEchoHostname(t, addr, "b.sni.test")
		if err != nil {
			t.Fatalf("tls dial b: %v", err)
		}
		if hnB != "echo2" {
			t.Fatalf("SNI b.sni.test should reach echo2, got %q", hnB)
		}
	})
```

> If route priority/ordering matters for SNI selection, set an explicit `"priority"` per route (higher = first) when the matcher would otherwise be ambiguous. Confirm from `internal/caddy/config/layer4_builder.go` whether routes are ordered by `priority`.

- [ ] **Step 2: Run**

Run: `cd backend && go test -tags traffic -run 'TestTraffic_L4/tls_sni_passthrough' ./tests/integration/ -count=1 -v`
Expected: PASS — `a.sni.test` → echo1, `b.sni.test` → echo2.

- [ ] **Step 3: Commit**

```bash
git add backend/tests/integration/traffic_l4_test.go
git commit -m "test: L4 TLS SNI passthrough traffic e2e"
```

---

## Task 15: Full-suite run + docs

**Files:**
- Modify: `docs/DEPLOYMENT.md` (or `backend/README.md`)

- [ ] **Step 1: Run the entire suite end-to-end**

Run: `make test-traffic`
Expected: image builds; `TestTraffic_L7` (7 subtests) and `TestTraffic_L4` (6 subtests) all PASS.

- [ ] **Step 2: Confirm the suite is excluded from the default test run**

Run: `cd backend && go test ./tests/integration/ -short -count=1`
Expected: builds and passes with no `TestTraffic_*` executed (build tag excludes the files).

- [ ] **Step 3: Document the suite**

Add a short section to `docs/DEPLOYMENT.md` (or `backend/README.md`):

```markdown
### Proxy traffic E2E tests

`make test-traffic` builds the `waygates-test:latest` image and runs the
end-to-end traffic suite (`backend/tests/integration/traffic_*_test.go`, build
tag `traffic`). It boots the app plus backend fixtures (HTTP/HTTPS echo, TCP
echo, Postgres) on a Docker network and drives real traffic through Caddy for
both L7 (reverse_proxy, redirect, static, load balancing, ACL, block_exploits,
custom headers) and L4 (any/TCP, http, postgres, TLS SNI passthrough, remote_ip,
load balancing) proxies. Requires Docker. Excluded from `make backend-test`.
```

- [ ] **Step 4: Commit**

```bash
git add docs/DEPLOYMENT.md
git commit -m "docs: document make test-traffic proxy E2E suite"
```

---

## Self-review notes (for the implementer)

- **Spec coverage:** Tasks 2–8 cover the 7 L7 scenarios; Tasks 9–14 cover the 6 L4 scenarios; Task 1 builds the shared harness + gating; Task 15 verifies exclusion and documents. All 13 spec scenarios map to a task.
- **Two confirm-then-implement points** (custom_headers shape, ACL mechanism) are bounded: each names the exact files to read and keeps a stable behavioral assertion. These are not open-ended placeholders.
- **Type/name consistency:** helpers (`triggerSync`, `l7Get`, `waitL7`, `l4Addr`, `waitL4`, `tcpEcho`, `pgSelect1`, `tlsEchoHostname`, `httpEchoHostname`, `echoHostnameFromBody`) are defined in Task 1/10 and used by later tasks under the same names. `TrafficEnv` embeds `*ContainerTestEnv`, so `MakeAuthenticatedRequest`/`ReadJSONResponse`/`RegisterAndLogin` are available on `env`.
- **Known iteration risks:** socat echo arg form (`EXEC:cat`), the mendhak JSON hostname field, and exact `custom_headers`/`block_exploits`/ACL shapes may need a small adjustment on first run — the TDD loop (run → adjust → rerun) absorbs these.
