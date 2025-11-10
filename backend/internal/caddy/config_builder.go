package caddy

import (
	"encoding/json"
	"fmt"

	"github.com/aloks98/homelab-proxy/backend/internal/models"
)

// RouteConfig represents a Caddy route configuration
type RouteConfig struct {
	Match  []MatchConfig    `json:"match,omitempty"`
	Handle []HandlerConfig  `json:"handle"`
	ID     string           `json:"@id,omitempty"`
}

// MatchConfig represents a Caddy match configuration
type MatchConfig struct {
	Host []string `json:"host,omitempty"`
	Path []string `json:"path,omitempty"`
}

// HandlerConfig represents a Caddy handler configuration
type HandlerConfig struct {
	Handler          string                  `json:"handler"`
	ID               string                  `json:"@id,omitempty"`
	Upstreams        []UpstreamConfig        `json:"upstreams,omitempty"`
	Headers          *HeadersConfig          `json:"headers,omitempty"`         // For reverse_proxy/headers handler
	StaticHeaders    map[string][]string     `json:"-"`                          // For static_response handler (handled in MarshalJSON)
	LoadBalancing    *LoadBalancingConfig    `json:"load_balancing,omitempty"`
	HealthChecks     *HealthChecksConfig     `json:"health_checks,omitempty"`
	Transport        *TransportConfig        `json:"transport,omitempty"`    // For reverse_proxy TLS
	To               string                  `json:"to,omitempty"`           // For redirects
	StatusCode       int                     `json:"status_code,omitempty"`  // For redirects
	Root             string                  `json:"root,omitempty"`         // For file server
	IndexNames       []string                `json:"index_names,omitempty"`  // For file server
	Templates        *TemplatesConfig        `json:"templates,omitempty"`
	TryFiles         []string                `json:"try_files,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for HandlerConfig
func (h HandlerConfig) MarshalJSON() ([]byte, error) {
	type Alias HandlerConfig

	// Create a temporary map to build the JSON
	temp := make(map[string]interface{})

	// Marshal to temp struct first
	aliasBytes, err := json.Marshal((Alias)(h))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(aliasBytes, &temp); err != nil {
		return nil, err
	}

	// If StaticHeaders is set (for static_response), use it instead of Headers
	if h.StaticHeaders != nil {
		temp["headers"] = h.StaticHeaders
		delete(temp, "Headers") // Remove the nested Headers field
	}

	return json.Marshal(temp)
}

// UpstreamConfig represents a Caddy upstream configuration
type UpstreamConfig struct {
	Dial string `json:"dial"`
}

// HeadersConfig represents header manipulation
type HeadersConfig struct {
	Request  *HeaderOps `json:"request,omitempty"`
	Response *HeaderOps `json:"response,omitempty"`
}

// HeaderOps represents header operations
type HeaderOps struct {
	Set    map[string][]string `json:"set,omitempty"`
	Add    map[string][]string `json:"add,omitempty"`
	Delete []string            `json:"delete,omitempty"`
}

// LoadBalancingConfig represents load balancing configuration
type LoadBalancingConfig struct {
	SelectionPolicy interface{} `json:"selection_policy,omitempty"`
}

// HealthChecksConfig represents health check configuration
type HealthChecksConfig struct {
	Active *ActiveHealthChecks `json:"active,omitempty"`
}

// ActiveHealthChecks represents active health check configuration
type ActiveHealthChecks struct {
	Path         string `json:"path,omitempty"`
	URI          string `json:"uri,omitempty"`
	Interval     string `json:"interval,omitempty"`
	Timeout      string `json:"timeout,omitempty"`
	MaxSize      int64  `json:"max_size,omitempty"`
	Fails        int    `json:"fails,omitempty"`        // Number of consecutive failed checks before marking unhealthy
	Passes       int    `json:"passes,omitempty"`       // Number of consecutive successful checks before marking healthy
	ExpectStatus int    `json:"expect_status,omitempty"` // Expected HTTP status code
	ExpectBody   string `json:"expect_body,omitempty"`   // Expected body content (regex)
}

// TemplatesConfig represents template configuration
type TemplatesConfig struct {
	FileRoot string `json:"file_root,omitempty"`
}

// TransportConfig represents transport configuration for reverse_proxy
type TransportConfig struct {
	Protocol string     `json:"protocol"`
	TLS      *TLSConfig `json:"tls,omitempty"`
}

// TLSConfig represents TLS configuration for transport
type TLSConfig struct {
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

// BuildRouteConfig generates a Caddy route configuration from a Proxy model
func BuildRouteConfig(proxy *models.Proxy) (*RouteConfig, error) {
	route := &RouteConfig{
		ID: fmt.Sprintf("proxy_%d", proxy.ID),
		Match: []MatchConfig{
			{
				Host: []string{proxy.Hostname},
			},
		},
	}

	// Build handlers based on type
	switch proxy.Type {
	case models.ProxyTypeReverseProxy:
		handlers, err := buildReverseProxyHandlers(proxy)
		if err != nil {
			return nil, err
		}
		route.Handle = handlers

	case models.ProxyTypeRedirect:
		handlers, err := buildRedirectHandlers(proxy)
		if err != nil {
			return nil, err
		}
		route.Handle = handlers

	case models.ProxyTypeStatic:
		handlers, err := buildStaticHandlers(proxy)
		if err != nil {
			return nil, err
		}
		route.Handle = handlers

	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxy.Type)
	}

	return route, nil
}

// buildReverseProxyHandlers creates handlers for reverse_proxy type
func buildReverseProxyHandlers(proxy *models.Proxy) ([]HandlerConfig, error) {
	var handlers []HandlerConfig

	// Add security handler if block_exploits is enabled
	if proxy.BlockExploits {
		// In Caddy, we use import snippets for security
		// This will be handled at the route level, not handler level
		// We'll add a comment marker for now
	}

	// Build reverse_proxy handler
	handler := HandlerConfig{
		Handler: "reverse_proxy",
		ID:      fmt.Sprintf("handler_%d", proxy.ID),
	}

	// Add upstreams
	if proxy.Upstreams != nil {
		if upstreams, ok := proxy.Upstreams.([]interface{}); ok && len(upstreams) > 0 {
			for _, up := range upstreams {
				upstreamMap, ok := up.(map[string]interface{})
				if !ok {
					continue
				}

				host, _ := upstreamMap["host"].(string)
				port, _ := upstreamMap["port"].(float64)
				scheme, _ := upstreamMap["scheme"].(string)
				if scheme == "" {
					scheme = "http"
				}

				dial := fmt.Sprintf("%s:%d", host, int(port))
				handler.Upstreams = append(handler.Upstreams, UpstreamConfig{
					Dial: dial,
				})
			}
		}
	}

	// Add load balancing if multiple upstreams
	if len(handler.Upstreams) > 1 && len(proxy.LoadBalancing) > 0 {
		lbConfig := proxy.LoadBalancing
		strategy, _ := lbConfig["strategy"].(string)

		var selectionPolicy interface{}
		switch strategy {
		case "round_robin":
			selectionPolicy = map[string]interface{}{"policy": "round_robin"}
		case "least_conn":
			selectionPolicy = map[string]interface{}{"policy": "least_conn"}
		case "ip_hash":
			selectionPolicy = map[string]interface{}{"policy": "ip_hash"}
		case "random":
			selectionPolicy = map[string]interface{}{"policy": "random"}
		default:
			selectionPolicy = map[string]interface{}{"policy": "round_robin"}
		}

		handler.LoadBalancing = &LoadBalancingConfig{
			SelectionPolicy: selectionPolicy,
		}

		// Add health checks if configured
		if healthChecks, ok := lbConfig["health_checks"].(map[string]interface{}); ok {
			if enabled, _ := healthChecks["enabled"].(bool); enabled {
				path, _ := healthChecks["path"].(string)
				interval, _ := healthChecks["interval"].(string)
				timeout, _ := healthChecks["timeout"].(string)
				// Map user-friendly names to Caddy field names
				unhealthyThreshold, _ := healthChecks["unhealthy_threshold"].(float64)
				healthyThreshold, _ := healthChecks["healthy_threshold"].(float64)

				handler.HealthChecks = &HealthChecksConfig{
					Active: &ActiveHealthChecks{
						Path:     path,
						Interval: interval,
						Timeout:  timeout,
						Fails:    int(unhealthyThreshold), // Caddy uses "fails" instead of "unhealthy_threshold"
						Passes:   int(healthyThreshold),   // Caddy uses "passes" instead of "healthy_threshold"
					},
				}
			}
		}
	}

	// Add transport config if TLS insecure skip verify is enabled
	if proxy.TLSInsecureSkipVerify {
		handler.Transport = &TransportConfig{
			Protocol: "http",
			TLS: &TLSConfig{
				InsecureSkipVerify: true,
			},
		}
	}

	// Add custom headers
	if len(proxy.CustomHeaders) > 0 {
		headerOps := make(map[string][]string)
		for key, value := range proxy.CustomHeaders {
			if strVal, ok := value.(string); ok {
				headerOps[key] = []string{strVal}
			}
		}

		// Standard reverse proxy headers
		headerOps["X-Real-IP"] = []string{"{http.request.remote.host}"}
		headerOps["X-Forwarded-For"] = []string{"{http.request.remote.host}"}
		headerOps["X-Forwarded-Proto"] = []string{"{http.request.scheme}"}

		handler.Headers = &HeadersConfig{
			Request: &HeaderOps{
				Set: headerOps,
			},
		}
	} else {
		// Add standard headers even if no custom headers
		handler.Headers = &HeadersConfig{
			Request: &HeaderOps{
				Set: map[string][]string{
					"X-Real-IP":         {"{http.request.remote.host}"},
					"X-Forwarded-For":   {"{http.request.remote.host}"},
					"X-Forwarded-Proto": {"{http.request.scheme}"},
				},
			},
		}
	}

	handlers = append(handlers, handler)
	return handlers, nil
}

// buildRedirectHandlers creates handlers for redirect type
func buildRedirectHandlers(proxy *models.Proxy) ([]HandlerConfig, error) {
	if len(proxy.RedirectConfig) == 0 {
		return nil, fmt.Errorf("redirect_config is required for redirect type")
	}

	redirectConfig := proxy.RedirectConfig
	target, _ := redirectConfig["target"].(string)
	if target == "" {
		return nil, fmt.Errorf("redirect target is required")
	}

	statusCode, _ := redirectConfig["status_code"].(float64)
	if statusCode == 0 {
		statusCode = 302 // Default to temporary redirect
	}

	preservePath, _ := redirectConfig["preserve_path"].(bool)
	preserveQuery, _ := redirectConfig["preserve_query"].(bool)

	// Build redirect URL
	redirectTo := target
	if preservePath {
		redirectTo += "{http.request.uri.path}"
	}
	if preserveQuery && len(redirectTo) > 0 && redirectTo[len(redirectTo)-1] != '?' {
		// Only add query separator if target doesn't already have one
		redirectTo += "?"
	}
	if preserveQuery {
		redirectTo += "{http.request.uri.query}"
	}

	handler := HandlerConfig{
		Handler:    "static_response",
		ID:         fmt.Sprintf("handler_%d", proxy.ID),
		StatusCode: int(statusCode),
		StaticHeaders: map[string][]string{
			"Location": {redirectTo},
		},
	}

	return []HandlerConfig{handler}, nil
}

// buildStaticHandlers creates handlers for static type
func buildStaticHandlers(proxy *models.Proxy) ([]HandlerConfig, error) {
	if len(proxy.StaticConfig) == 0 {
		return nil, fmt.Errorf("static_config is required for static type")
	}

	var handlers []HandlerConfig
	staticConfig := proxy.StaticConfig

	rootPath, _ := staticConfig["root_path"].(string)
	if rootPath == "" {
		return nil, fmt.Errorf("root_path is required for static type")
	}

	indexFile, _ := staticConfig["index_file"].(string)
	if indexFile == "" {
		indexFile = "index.html"
	}

	templateRendering, _ := staticConfig["template_rendering"].(bool)
	tryFiles, _ := staticConfig["try_files"].([]interface{})

	// Add template handler if enabled
	if templateRendering {
		handlers = append(handlers, HandlerConfig{
			Handler: "templates",
			Templates: &TemplatesConfig{
				FileRoot: rootPath,
			},
		})
	}

	// Add rewrite handler for SPA support (try_files pattern)
	// This implements the pattern: try {path}, fallback to /index.html
	if len(tryFiles) > 0 {
		var files []string
		for _, f := range tryFiles {
			if str, ok := f.(string); ok {
				files = append(files, str)
			}
		}

		// Add rewrite handler with try_files
		handlers = append(handlers, HandlerConfig{
			Handler:  "rewrite",
			TryFiles: files,
		})
	}

	// Build file_server handler
	fileHandler := HandlerConfig{
		Handler:    "file_server",
		ID:         fmt.Sprintf("handler_%d", proxy.ID),
		Root:       rootPath,
		IndexNames: []string{indexFile},
	}

	handlers = append(handlers, fileHandler)
	return handlers, nil
}

// GetSecuritySnippetPath returns the path to import security snippet if block_exploits is enabled
func GetSecuritySnippetPath(proxy *models.Proxy) string {
	if proxy.BlockExploits && proxy.Type == models.ProxyTypeReverseProxy {
		return "snippets/security"
	}
	return ""
}
