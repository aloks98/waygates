// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

// Handler module names as constants.
const (
	HandlerReverseProxy   = "reverse_proxy"
	HandlerStaticResponse = "static_response"
	HandlerFileServer     = "file_server"
	HandlerSubroute       = "subroute"
	HandlerForwardAuth    = "forward_auth"
	HandlerAuthentication = "authentication"
	HandlerHeaders        = "headers"
	HandlerRewrite        = "rewrite"
	HandlerError          = "error"
)

// ReverseProxyHandler configures the reverse_proxy handler.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/reverse_proxy/
type ReverseProxyHandler struct {
	Handler         string         `json:"handler"` // Must be "reverse_proxy"
	Upstreams       []*Upstream    `json:"upstreams,omitempty"`
	LoadBalancing   *LoadBalancing `json:"load_balancing,omitempty"`
	HealthChecks    *HealthChecks  `json:"health_checks,omitempty"`
	Transport       *HTTPTransport `json:"transport,omitempty"`
	Headers         *HeadersConfig `json:"headers,omitempty"`
	FlushInterval   Duration       `json:"flush_interval,omitempty"`
	BufferRequests  bool           `json:"buffer_requests,omitempty"`
	BufferResponses bool           `json:"buffer_responses,omitempty"`
	MaxBufferSize   int64          `json:"max_buffer_size,omitempty"`
	TrustedProxies  []string       `json:"trusted_proxies,omitempty"`
}

// Duration is a custom type for Caddy duration strings.
type Duration string

// Upstream represents a single upstream server.
type Upstream struct {
	Dial        string `json:"dial,omitempty"`
	MaxRequests int    `json:"max_requests,omitempty"`
	LookupSRV   string `json:"lookup_srv,omitempty"`
}

// LoadBalancing configures load balancing for reverse proxy.
type LoadBalancing struct {
	SelectionPolicy *SelectionPolicy `json:"selection_policy,omitempty"`
	TryDuration     Duration         `json:"try_duration,omitempty"`
	TryInterval     Duration         `json:"try_interval,omitempty"`
	RetryMatch      []MatcherSet     `json:"retry_match,omitempty"`
}

// SelectionPolicy configures the upstream selection policy.
type SelectionPolicy struct {
	Policy string `json:"policy,omitempty"` // round_robin, least_conn, random, first, ip_hash, uri_hash, header
	Header string `json:"header,omitempty"` // For header policy
}

// HealthChecks configures health checking for upstreams.
type HealthChecks struct {
	Active  *ActiveHealthCheck  `json:"active,omitempty"`
	Passive *PassiveHealthCheck `json:"passive,omitempty"`
}

// ActiveHealthCheck configures active health checking.
type ActiveHealthCheck struct {
	Path         string              `json:"path,omitempty"`
	URI          string              `json:"uri,omitempty"`
	Port         int                 `json:"port,omitempty"`
	Headers      map[string][]string `json:"headers,omitempty"`
	Interval     Duration            `json:"interval,omitempty"`
	Timeout      Duration            `json:"timeout,omitempty"`
	MaxSize      int64               `json:"max_size,omitempty"`
	ExpectStatus int                 `json:"expect_status,omitempty"`
	ExpectBody   string              `json:"expect_body,omitempty"`
}

// PassiveHealthCheck configures passive health checking.
type PassiveHealthCheck struct {
	FailDuration     Duration `json:"fail_duration,omitempty"`
	MaxFails         int      `json:"max_fails,omitempty"`
	UnhealthyStatus  []int    `json:"unhealthy_status,omitempty"`
	UnhealthyLatency Duration `json:"unhealthy_latency,omitempty"`
}

// HTTPTransport configures the HTTP transport for reverse proxy.
type HTTPTransport struct {
	Resolver              *DNSResolver `json:"resolver,omitempty"`
	TLS                   *TLSConfig   `json:"tls,omitempty"`
	KeepAlive             *KeepAlive   `json:"keep_alive,omitempty"`
	Compression           bool         `json:"compression,omitempty"`
	MaxConnsPerHost       int          `json:"max_conns_per_host,omitempty"`
	MaxIdleConnsPerHost   int          `json:"max_idle_conns_per_host,omitempty"`
	MaxResponseHeaderSize int64        `json:"max_response_header_size,omitempty"`
	DialTimeout           Duration     `json:"dial_timeout,omitempty"`
	ReadBufferSize        int          `json:"read_buffer_size,omitempty"`
	WriteBufferSize       int          `json:"write_buffer_size,omitempty"`
	ResponseHeaderTimeout Duration     `json:"response_header_timeout,omitempty"`
	ExpectContinueTimeout Duration     `json:"expect_continue_timeout,omitempty"`
	Versions              []string     `json:"versions,omitempty"`
}

// DNSResolver configures DNS resolution for reverse proxy.
type DNSResolver struct {
	Addresses []string `json:"addresses,omitempty"`
}

// TLSConfig configures TLS for upstream connections.
type TLSConfig struct {
	RootCAPool               []string `json:"root_ca_pool,omitempty"`
	RootCAPemFiles           []string `json:"root_ca_pem_files,omitempty"`
	ClientCertificateFile    string   `json:"client_certificate_file,omitempty"`
	ClientCertificateKeyFile string   `json:"client_certificate_key_file,omitempty"`
	InsecureSkipVerify       bool     `json:"insecure_skip_verify,omitempty"`
	ServerName               string   `json:"server_name,omitempty"`
	Renegotiation            string   `json:"renegotiation,omitempty"`
}

// KeepAlive configures keep-alive for HTTP transport.
type KeepAlive struct {
	Enabled             *bool    `json:"enabled,omitempty"`
	ProbeInterval       Duration `json:"probe_interval,omitempty"`
	MaxIdleConns        int      `json:"max_idle_conns,omitempty"`
	MaxIdleConnsPerHost int      `json:"max_idle_conns_per_host,omitempty"`
	IdleConnTimeout     Duration `json:"idle_conn_timeout,omitempty"`
}

// HeadersConfig configures header manipulation for reverse proxy.
type HeadersConfig struct {
	Request  *HeaderOps `json:"request,omitempty"`
	Response *HeaderOps `json:"response,omitempty"`
}

// HeaderOps defines header operations.
type HeaderOps struct {
	Set     map[string][]string        `json:"set,omitempty"`
	Add     map[string][]string        `json:"add,omitempty"`
	Delete  []string                   `json:"delete,omitempty"`
	Replace map[string][]ReplacementOp `json:"replace,omitempty"`
}

// ReplacementOp defines a header replacement operation.
type ReplacementOp struct {
	Search       string `json:"search,omitempty"`
	SearchRegexp string `json:"search_regexp,omitempty"`
	Replace      string `json:"replace,omitempty"`
}

// StaticResponseHandler configures the static_response handler.
// Used for redirects, error responses, and simple static content.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/static_response/
type StaticResponseHandler struct {
	Handler    string              `json:"handler"` // Must be "static_response"
	StatusCode int                 `json:"status_code,omitempty"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Body       string              `json:"body,omitempty"`
	Close      bool                `json:"close,omitempty"`
}

// FileServerHandler configures the file_server handler.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/file_server/
type FileServerHandler struct {
	Handler       string   `json:"handler"` // Must be "file_server"
	Root          string   `json:"root,omitempty"`
	IndexNames    []string `json:"index_names,omitempty"`
	Browse        *Browse  `json:"browse,omitempty"`
	CanonicalURIs bool     `json:"canonical_uris,omitempty"`
	PassThru      bool     `json:"pass_thru,omitempty"`
	Hide          []string `json:"hide,omitempty"`
}

// Browse configures directory browsing for file_server.
type Browse struct {
	TemplateFile string `json:"template_file,omitempty"`
}

// SubrouteHandler configures the subroute handler for nested routing.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/subroute/
type SubrouteHandler struct {
	Handler string       `json:"handler"` // Must be "subroute"
	Routes  []*HTTPRoute `json:"routes,omitempty"`
}

// ForwardAuthHandler configures forward_auth for authentication.
// See: https://caddyserver.com/docs/caddyfile/directives/forward_auth
type ForwardAuthHandler struct {
	Handler        string            `json:"handler"` // Must be "reverse_proxy"
	Upstreams      []*Upstream       `json:"upstreams,omitempty"`
	Headers        *HeadersConfig    `json:"headers,omitempty"`
	HandleResponse []*HandleResponse `json:"handle_response,omitempty"`
	RewriteHeaders *RewriteHeaders   `json:"rewrite,omitempty"`
}

// HandleResponse configures how to handle responses from forward_auth.
type HandleResponse struct {
	Match      *ResponseMatch `json:"match,omitempty"`
	Routes     []*HTTPRoute   `json:"routes,omitempty"`
	StatusCode int            `json:"status_code,omitempty"`
}

// ResponseMatch matches response attributes.
type ResponseMatch struct {
	StatusCode []int               `json:"status_code,omitempty"`
	Headers    map[string][]string `json:"headers,omitempty"`
}

// RewriteHeaders configures header rewriting.
type RewriteHeaders struct {
	Method string `json:"method,omitempty"`
	URI    string `json:"uri,omitempty"`
}

// AuthenticationHandler configures HTTP Basic Authentication.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/authentication/
type AuthenticationHandler struct {
	Handler   string                   `json:"handler"` // Must be "authentication"
	Providers *AuthenticationProviders `json:"providers,omitempty"`
}

// AuthenticationProviders contains authentication provider configurations.
type AuthenticationProviders struct {
	HTTPBasic *HTTPBasicAuth `json:"http_basic,omitempty"`
}

// HTTPBasicAuth configures HTTP Basic Authentication.
type HTTPBasicAuth struct {
	Accounts  []*BasicAuthAccount `json:"accounts,omitempty"`
	Realm     string              `json:"realm,omitempty"`
	HashCache *HashCache          `json:"hash_cache,omitempty"`
}

// BasicAuthAccount represents a user account for basic auth.
type BasicAuthAccount struct {
	Username string `json:"username"`
	Password string `json:"password"` // bcrypt hash
}

// HashCache configures caching for password hash verification.
type HashCache struct {
	Enabled bool `json:"enabled,omitempty"`
}

// HeadersHandler configures the headers handler.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/headers/
type HeadersHandler struct {
	Handler  string     `json:"handler"` // Must be "headers"
	Request  *HeaderOps `json:"request,omitempty"`
	Response *HeaderOps `json:"response,omitempty"`
}

// RewriteHandler configures the rewrite handler.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/rewrite/
type RewriteHandler struct {
	Handler         string                 `json:"handler"` // Must be "rewrite"
	Method          string                 `json:"method,omitempty"`
	URI             string                 `json:"uri,omitempty"`
	StripPathPrefix string                 `json:"strip_path_prefix,omitempty"`
	StripPathSuffix string                 `json:"strip_path_suffix,omitempty"`
	URISubstring    []SubstringReplacement `json:"uri_substring,omitempty"`
	PathRegexp      []RegexpReplacement    `json:"path_regexp,omitempty"`
}

// SubstringReplacement configures substring replacement in rewrite.
type SubstringReplacement struct {
	Find    string `json:"find"`
	Replace string `json:"replace"`
	Limit   int    `json:"limit,omitempty"`
}

// RegexpReplacement configures regexp replacement in rewrite.
type RegexpReplacement struct {
	Find    string `json:"find"`
	Replace string `json:"replace"`
}

// ErrorHandler configures the error handler.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/handle/error/
type ErrorHandler struct {
	Handler    string `json:"handler"` // Must be "error"
	Error      string `json:"error,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

// NewReverseProxyHandler creates a new reverse proxy handler.
func NewReverseProxyHandler(upstreams ...*Upstream) *ReverseProxyHandler {
	return &ReverseProxyHandler{
		Handler:   HandlerReverseProxy,
		Upstreams: upstreams,
	}
}

// NewUpstream creates a new upstream configuration.
func NewUpstream(dial string) *Upstream {
	return &Upstream{
		Dial: dial,
	}
}

// NewUpstreamFromHostPort creates an upstream from host and port.
func NewUpstreamFromHostPort(host string, port int) *Upstream {
	return &Upstream{
		Dial: formatDialAddress(host, port),
	}
}

// formatDialAddress formats a dial address from host and port.
func formatDialAddress(host string, port int) string {
	if port == 0 {
		return host
	}
	return host + ":" + itoa(port)
}

// itoa is a simple int to string conversion.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	// Simple implementation for positive integers
	var buf [20]byte
	n := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		buf[n] = '-'
	}
	return string(buf[n:])
}

// WithLoadBalancing adds load balancing to a reverse proxy handler.
func (h *ReverseProxyHandler) WithLoadBalancing(policy string) *ReverseProxyHandler {
	h.LoadBalancing = &LoadBalancing{
		SelectionPolicy: &SelectionPolicy{
			Policy: policy,
		},
	}
	return h
}

// WithHealthChecks adds health checks to a reverse proxy handler.
func (h *ReverseProxyHandler) WithHealthChecks(path string, interval, timeout Duration) *ReverseProxyHandler {
	h.HealthChecks = &HealthChecks{
		Active: &ActiveHealthCheck{
			Path:     path,
			Interval: interval,
			Timeout:  timeout,
		},
	}
	return h
}

// WithTLSTransport adds TLS transport configuration to a reverse proxy handler.
func (h *ReverseProxyHandler) WithTLSTransport(insecureSkipVerify bool) *ReverseProxyHandler {
	h.Transport = &HTTPTransport{
		TLS: &TLSConfig{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}
	return h
}

// WithHeaders adds header configuration to a reverse proxy handler.
func (h *ReverseProxyHandler) WithHeaders(request, response *HeaderOps) *ReverseProxyHandler {
	h.Headers = &HeadersConfig{
		Request:  request,
		Response: response,
	}
	return h
}

// NewStaticResponseHandler creates a static response handler.
func NewStaticResponseHandler(statusCode int, body string) *StaticResponseHandler {
	return &StaticResponseHandler{
		Handler:    HandlerStaticResponse,
		StatusCode: statusCode,
		Body:       body,
	}
}

// NewRedirectHandler creates a redirect response handler.
func NewRedirectHandler(location string, statusCode int) *StaticResponseHandler {
	return &StaticResponseHandler{
		Handler:    HandlerStaticResponse,
		StatusCode: statusCode,
		Headers: map[string][]string{
			"Location": {location},
		},
	}
}

// NewFileServerHandler creates a new file server handler.
func NewFileServerHandler(root string) *FileServerHandler {
	return &FileServerHandler{
		Handler: HandlerFileServer,
		Root:    root,
	}
}

// WithIndexNames sets the index file names for a file server handler.
func (h *FileServerHandler) WithIndexNames(names ...string) *FileServerHandler {
	h.IndexNames = names
	return h
}

// WithBrowse enables directory browsing for a file server handler.
func (h *FileServerHandler) WithBrowse(templateFile string) *FileServerHandler {
	h.Browse = &Browse{
		TemplateFile: templateFile,
	}
	return h
}

// NewSubrouteHandler creates a new subroute handler.
func NewSubrouteHandler(routes ...*HTTPRoute) *SubrouteHandler {
	return &SubrouteHandler{
		Handler: HandlerSubroute,
		Routes:  routes,
	}
}

// NewAuthenticationHandler creates a new authentication handler with basic auth.
func NewAuthenticationHandler(accounts []*BasicAuthAccount, realm string) *AuthenticationHandler {
	return &AuthenticationHandler{
		Handler: HandlerAuthentication,
		Providers: &AuthenticationProviders{
			HTTPBasic: &HTTPBasicAuth{
				Accounts: accounts,
				Realm:    realm,
			},
		},
	}
}

// NewBasicAuthAccount creates a new basic auth account.
func NewBasicAuthAccount(username, passwordHash string) *BasicAuthAccount {
	return &BasicAuthAccount{
		Username: username,
		Password: passwordHash,
	}
}

// NewHeadersHandler creates a new headers handler.
func NewHeadersHandler(request, response *HeaderOps) *HeadersHandler {
	return &HeadersHandler{
		Handler:  HandlerHeaders,
		Request:  request,
		Response: response,
	}
}

// NewRequestHeaderOps creates header operations for requests.
func NewRequestHeaderOps() *HeaderOps {
	return &HeaderOps{
		Set: make(map[string][]string),
		Add: make(map[string][]string),
	}
}

// SetHeader sets a header value.
func (h *HeaderOps) SetHeader(name string, values ...string) *HeaderOps {
	if h.Set == nil {
		h.Set = make(map[string][]string)
	}
	h.Set[name] = values
	return h
}

// AddHeader adds a header value.
func (h *HeaderOps) AddHeader(name string, values ...string) *HeaderOps {
	if h.Add == nil {
		h.Add = make(map[string][]string)
	}
	h.Add[name] = values
	return h
}

// DeleteHeader adds a header to delete.
func (h *HeaderOps) DeleteHeader(names ...string) *HeaderOps {
	h.Delete = append(h.Delete, names...)
	return h
}

// StandardProxyHeaders returns header operations for standard proxy headers.
// These headers inform the upstream server about the original request.
func StandardProxyHeaders() *HeaderOps {
	return &HeaderOps{
		Set: map[string][]string{
			"X-Real-IP":         {"{http.request.remote.host}"},
			"X-Forwarded-For":   {"{http.request.remote.host}"},
			"X-Forwarded-Proto": {"{http.request.scheme}"},
			"X-Forwarded-Host":  {"{http.request.host}"},
		},
	}
}

// ToHTTPHandler converts a typed handler to the generic HTTPHandler map.
func ToHTTPHandler(handler interface{}) HTTPHandler {
	switch h := handler.(type) {
	case *ReverseProxyHandler:
		return HTTPHandler{
			"handler":        h.Handler,
			"upstreams":      h.Upstreams,
			"load_balancing": h.LoadBalancing,
			"health_checks":  h.HealthChecks,
			"transport":      h.Transport,
			"headers":        h.Headers,
		}
	case *StaticResponseHandler:
		result := HTTPHandler{
			"handler":     h.Handler,
			"status_code": h.StatusCode,
		}
		if h.Body != "" {
			result["body"] = h.Body
		}
		if len(h.Headers) > 0 {
			result["headers"] = h.Headers
		}
		return result
	case *FileServerHandler:
		result := HTTPHandler{
			"handler": h.Handler,
		}
		if h.Root != "" {
			result["root"] = h.Root
		}
		if len(h.IndexNames) > 0 {
			result["index_names"] = h.IndexNames
		}
		if h.Browse != nil {
			result["browse"] = h.Browse
		}
		return result
	case *SubrouteHandler:
		return HTTPHandler{
			"handler": h.Handler,
			"routes":  h.Routes,
		}
	case *AuthenticationHandler:
		return HTTPHandler{
			"handler":   h.Handler,
			"providers": h.Providers,
		}
	case *HeadersHandler:
		return HTTPHandler{
			"handler":  h.Handler,
			"request":  h.Request,
			"response": h.Response,
		}
	default:
		return nil
	}
}
