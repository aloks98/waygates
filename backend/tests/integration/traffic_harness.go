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

// Timeouts used throughout the traffic harness. Centralised here so they can be
// tuned in one place rather than scattered as inline literals.
const (
	startupTimeout     = 120 * time.Second // waygates app HTTP/log readiness
	dbStartupTimeout   = 60 * time.Second  // Postgres "ready to accept connections"
	caddyReadyTimeout  = 60 * time.Second  // waygates "Caddy is ready!" log line
	echoStartupTimeout = 30 * time.Second  // echo/socat fixture listening port
	l7ReadyTimeout     = 60 * time.Second  // poll for L7 host to return wanted status
	l4ReadyTimeout     = 30 * time.Second  // poll for L4 TCP dial to succeed
	l7PollInterval     = 500 * time.Millisecond
	l4PollInterval     = 500 * time.Millisecond
	httpClientTimeout  = 5 * time.Second  // l7Get / readiness probe HTTP client
	probeDialTimeout   = 2 * time.Second  // single L4 dial attempt during waitL4
	tcpEchoTimeout     = 3 * time.Second  // tcpEcho dial + read/write deadline
	pgSelectTimeout    = 10 * time.Second // pgSelect1 connect + query
	tlsProbeTimeout    = 5 * time.Second  // tlsEchoHostname dial + deadline
	cleanupTermTimeout = 60 * time.Second // bound for terminating fixtures on cleanup
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
		WaitingFor:     wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(dbStartupTimeout),
	})

	// Register cleanup as early as possible so any container started below is
	// still terminated if a later mustStart fails (t.Fatalf would otherwise leak
	// already-started fixtures). cleanup tolerates a partially-populated env.
	t.Cleanup(func() { env.cleanup(t) })

	// --- Fixtures ---
	// Each fixture is appended to env.fixtures immediately after it starts so a
	// failure in a later mustStart still tears down the earlier ones.
	echo1 := mustStart(t, ctx, echoRequest(netName, "echo1"))
	env.fixtures = append(env.fixtures, echo1)
	echo2 := mustStart(t, ctx, echoRequest(netName, "echo2"))
	env.fixtures = append(env.fixtures, echo2)
	tcpecho := mustStart(t, ctx, testcontainers.ContainerRequest{
		Image: "alpine/socat",
		// alpine/socat declares no EXPOSE in its image, so the port must be listed
		// explicitly for testcontainers to map it and for ForListeningPort to work.
		ExposedPorts:   []string{"7000/tcp"},
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {"tcpecho"}},
		Cmd:            []string{"TCP-LISTEN:7000,fork,reuseaddr", "EXEC:cat"},
		WaitingFor:     wait.ForListeningPort("7000/tcp").WithStartupTimeout(echoStartupTimeout),
	})
	env.fixtures = append(env.fixtures, tcpecho)
	pgtarget := mustStart(t, ctx, testcontainers.ContainerRequest{
		Image:          "postgres:16-alpine",
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {"pgtarget"}},
		Env:            map[string]string{"POSTGRES_USER": "waygates", "POSTGRES_PASSWORD": "waygates", "POSTGRES_DB": "waygates"},
		WaitingFor:     wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(dbStartupTimeout),
	})
	env.fixtures = append(env.fixtures, pgtarget)

	// --- App under test ---
	exposed := []string{"8080/tcp", "80/tcp", "443/tcp"}
	for _, p := range l4PortPool {
		exposed = append(exposed, fmt.Sprintf("%d/tcp", p))
	}
	base.WaygatesContainer = mustStart(t, ctx, testcontainers.ContainerRequest{
		Image:        "waygates-test:latest",
		ExposedPorts: exposed,
		Networks:     []string{netName},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      findTestJSONConfig(t),
				ContainerFilePath: "/etc/caddy/caddy.json",
				FileMode:          0o644,
			},
			{
				// A known static file for the L7 static-serving subtest to fetch.
				Reader:            strings.NewReader("WAYGATES STATIC OK"),
				ContainerFilePath: "/var/www/test/index.html",
				FileMode:          0o644,
			},
		},
		Env: map[string]string{
			"DB_HOST": "postgres", "DB_PORT": "5432", "DB_USER": "waygates",
			"DB_PASSWORD": "waygates", "DB_NAME": "waygates",
			"JWT_SECRET":        "test-secret-key-that-is-at-least-32-characters-long",
			"JWT_ACCESS_EXPIRY": "15m", "JWT_REFRESH_EXPIRY": "24h",
			"BCRYPT_COST": "4", "CORS_ORIGINS": "*",
			"RBAC_PATH": "/app/backend/rbac.yaml", "CADDY_EMAIL": "test@example.com",
			"CADDY_DISABLE_AUTO_HTTPS": "true",
			"CLOUDFLARE_EMAIL":         "test@example.com", "CLOUDFLARE_API_TOKEN": "dummy-token-for-testing",
			"LOG_LEVEL": "debug", "LOG_FORMAT": "console", "UI_ENABLED": "false",
		},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/api/health").WithPort("8080/tcp").WithStartupTimeout(startupTimeout),
			wait.ForLog("Caddy is ready!").WithStartupTimeout(caddyReadyTimeout),
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

	base.RegisterAndLogin(t)
	return env
}

func echoRequest(netName, alias string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:          "mendhak/http-https-echo:31",
		Hostname:       alias, // surfaces in the echoed JSON so backends are distinguishable
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {alias}},
		WaitingFor:     wait.ForListeningPort("8080/tcp").WithStartupTimeout(echoStartupTimeout),
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
	// Bound the terminate calls so a hung docker daemon cannot block cleanup
	// (and therefore the whole test process) indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTermTimeout)
	defer cancel()
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
		Timeout:       httpClientTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// l7Probe performs one GET against Caddy :80 with the given Host header,
// tolerating transport errors by returning (nil, nil). It is intended for
// readiness polling (waitL7) where the backend may not be reachable yet.
func (e *TrafficEnv) l7Probe(t *testing.T, hostHeader, path string, headers map[string]string) (*http.Response, []byte) {
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

// l7Get performs one GET against Caddy :80 with the given Host header and fails
// the test on a transport error, so assertion callers never receive a nil
// response silently. Use l7Probe for tolerant readiness polling.
func (e *TrafficEnv) l7Get(t *testing.T, hostHeader, path string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	resp, body := e.l7Probe(t, hostHeader, path, headers)
	if resp == nil {
		t.Fatalf("l7Get transport error: host=%q path=%q via %s", hostHeader, path, e.l7URL())
	}
	return resp, body
}

// waitL7 polls until a GET with the Host header returns wantStatus.
func (e *TrafficEnv) waitL7(t *testing.T, hostHeader string, wantStatus int) {
	t.Helper()
	deadline := time.Now().Add(l7ReadyTimeout)
	for time.Now().Before(deadline) {
		resp, _ := e.l7Probe(t, hostHeader, "/", nil)
		if resp != nil && resp.StatusCode == wantStatus {
			return
		}
		time.Sleep(l7PollInterval)
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

// waitL4 polls until a TCP dial to addr succeeds.
func (e *TrafficEnv) waitL4(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(l4ReadyTimeout)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, probeDialTimeout)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(l4PollInterval)
	}
	t.Fatalf("timed out waiting for L4 addr %q", addr)
}

// --- Protocol probes ---

// tcpEcho writes payload to addr and returns what comes back (read once).
func tcpEcho(_ *testing.T, addr, payload string) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, tcpEchoTimeout)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(tcpEchoTimeout))
	if _, err := conn.Write([]byte(payload)); err != nil {
		return "", err
	}
	buf := make([]byte, len(payload))
	n, err := io.ReadFull(conn, buf)
	return string(buf[:n]), err
}

// pgSelect1 connects through addr to a Postgres backend and runs SELECT 1.
func pgSelect1(_ *testing.T, addr string) error {
	dsn := fmt.Sprintf("postgres://waygates:waygates@%s/waygates?sslmode=disable", addr)
	ctx, cancel := context.WithTimeout(context.Background(), pgSelectTimeout)
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
func tlsEchoHostname(_ *testing.T, addr, sni string) (string, error) {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: tlsProbeTimeout}, "tcp", addr, &tls.Config{ServerName: sni, InsecureSkipVerify: true})
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(tlsProbeTimeout))
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", sni)
	raw, _ := io.ReadAll(conn)
	return echoHostnameFromHTTP(string(raw)), nil
}

// echoHostnameFromBody extracts the backend container hostname from mendhak echo
// JSON. mendhak's top-level "hostname" field reflects the *request* Host header
// (e.g. "rp.test.local"), not the backend; the container hostname (set via the
// fixture's Hostname, e.g. "echo1"/"echo2") is under "os.hostname". We therefore
// prefer os.hostname so the assertion identifies which backend actually answered,
// and only fall back to the top-level field if os.hostname is absent.
func echoHostnameFromBody(body []byte) string {
	var v struct {
		Hostname string `json:"hostname"`
		OS       struct {
			Hostname string `json:"hostname"`
		} `json:"os"`
	}
	_ = json.Unmarshal(body, &v)
	if v.OS.Hostname != "" {
		return v.OS.Hostname
	}
	return v.Hostname
}

// echoHostnameFromHTTP splits a raw HTTP/1.1 response and parses the JSON body.
func echoHostnameFromHTTP(raw string) string {
	idx := strings.Index(raw, "\r\n\r\n")
	if idx < 0 {
		return ""
	}
	return echoHostnameFromBody([]byte(raw[idx+4:]))
}
