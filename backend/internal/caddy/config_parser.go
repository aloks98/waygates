package caddy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aloks98/waygates/backend/internal/models"
)

// ParseRouteToProxy converts a Caddy route configuration back to a Proxy model
// This is useful for syncing database with actual Caddy configuration
func ParseRouteToProxy(route *RouteConfig) (*models.Proxy, error) {
	proxy := &models.Proxy{}

	// Extract hostname from match
	if len(route.Match) > 0 && len(route.Match[0].Host) > 0 {
		proxy.Hostname = route.Match[0].Host[0]
	} else {
		return nil, fmt.Errorf("no hostname found in route")
	}

	// Extract proxy name from ID or use hostname
	if route.ID != "" {
		proxy.Name = route.ID
	} else {
		proxy.Name = proxy.Hostname
	}

	// Determine proxy type and parse handlers
	if len(route.Handle) == 0 {
		return nil, fmt.Errorf("no handlers found in route")
	}

	// Parse based on handler type
	mainHandler := route.Handle[len(route.Handle)-1] // Last handler is usually the main one

	switch mainHandler.Handler {
	case "reverse_proxy":
		if err := parseReverseProxyHandler(&mainHandler, proxy); err != nil {
			return nil, err
		}
		proxy.Type = models.ProxyTypeReverseProxy

	case "static_response":
		if err := parseRedirectHandler(&mainHandler, proxy); err != nil {
			return nil, err
		}
		proxy.Type = models.ProxyTypeRedirect

	case "file_server":
		if err := parseStaticHandler(route.Handle, proxy); err != nil {
			return nil, err
		}
		proxy.Type = models.ProxyTypeStatic

	default:
		return nil, fmt.Errorf("unsupported handler type: %s", mainHandler.Handler)
	}

	// Set defaults
	proxy.SSLEnabled = true
	proxy.SSLForced = true
	proxy.IsActive = true

	return proxy, nil
}

// parseReverseProxyHandler parses reverse_proxy handler into Proxy model
func parseReverseProxyHandler(handler *HandlerConfig, proxy *models.Proxy) error {
	// Parse upstreams
	upstreams := make([]interface{}, 0)
	for _, upstream := range handler.Upstreams {
		// Parse dial string (e.g., "192.168.1.100:8080")
		parts := strings.Split(upstream.Dial, ":")
		if len(parts) != 2 {
			continue
		}

		host := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		upstreams = append(upstreams, map[string]interface{}{
			"host":   host,
			"port":   port,
			"scheme": "http", // Default, we don't store scheme in dial
		})
	}

	proxy.Upstreams = upstreams

	// Parse load balancing
	if handler.LoadBalancing != nil {
		lbConfig := make(map[string]interface{})

		// Map Caddy selection policy to our strategy
		// SelectionPolicy can be either a string or a map[string]interface{} with "policy" key
		strategy := "round_robin"
		switch sp := handler.LoadBalancing.SelectionPolicy.(type) {
		case string:
			strategy = sp
		case map[string]interface{}:
			if policy, ok := sp["policy"].(string); ok {
				strategy = policy
			}
		}
		// Normalize known strategies
		switch strategy {
		case "round_robin", "least_conn", "ip_hash", "random":
			// Valid strategy, keep as-is
		default:
			strategy = "round_robin"
		}
		lbConfig["strategy"] = strategy

		// Parse health checks
		if handler.HealthChecks != nil && handler.HealthChecks.Active != nil {
			hc := handler.HealthChecks.Active
			lbConfig["health_checks"] = map[string]interface{}{
				"enabled":             true,
				"path":                hc.Path,
				"interval":            hc.Interval,
				"timeout":             hc.Timeout,
				"unhealthy_threshold": hc.Fails,  // Map Caddy's "fails" to our "unhealthy_threshold"
				"healthy_threshold":   hc.Passes, // Map Caddy's "passes" to our "healthy_threshold"
			}
		}

		proxy.LoadBalancing = lbConfig
	}

	// Parse custom headers (skip standard proxy headers)
	if handler.Headers != nil && handler.Headers.Request != nil && handler.Headers.Request.Set != nil {
		customHeaders := make(map[string]interface{})
		standardHeaders := map[string]bool{
			"X-Real-IP":         true,
			"X-Forwarded-For":   true,
			"X-Forwarded-Proto": true,
		}

		for key, values := range handler.Headers.Request.Set {
			if !standardHeaders[key] && len(values) > 0 {
				customHeaders[key] = values[0]
			}
		}

		if len(customHeaders) > 0 {
			proxy.CustomHeaders = customHeaders
		}
	}

	// Parse transport config for TLS insecure skip verify
	if handler.Transport != nil && handler.Transport.TLS != nil {
		proxy.TLSInsecureSkipVerify = handler.Transport.TLS.InsecureSkipVerify
	}

	// Default block_exploits to true for reverse proxy
	proxy.BlockExploits = true

	return nil
}

// parseRedirectHandler parses static_response (redirect) handler into Proxy model
func parseRedirectHandler(handler *HandlerConfig, proxy *models.Proxy) error {
	redirectConfig := make(map[string]interface{})

	// Get status code
	redirectConfig["status_code"] = handler.StatusCode
	if handler.StatusCode == 0 {
		redirectConfig["status_code"] = 302
	}

	// Parse Location header for target
	// For static_response, Caddy returns headers as a direct map (StaticHeaders)
	// not the nested Headers.Response.Set format used by reverse_proxy
	var locations []string
	if handler.StaticHeaders != nil {
		if locs, ok := handler.StaticHeaders["Location"]; ok {
			locations = locs
		}
	}
	// Also check the nested format for backwards compatibility
	if len(locations) == 0 && handler.Headers != nil && handler.Headers.Response != nil && handler.Headers.Response.Set != nil {
		if locs, ok := handler.Headers.Response.Set["Location"]; ok {
			locations = locs
		}
	}

	if len(locations) > 0 {
		target := locations[0]

		// Determine if path/query are preserved
		preservePath := strings.Contains(target, "{http.request.uri.path}")
		preserveQuery := strings.Contains(target, "{http.request.uri.query}")

		// Remove placeholders to get base target
		target = strings.ReplaceAll(target, "{http.request.uri.path}", "")
		target = strings.ReplaceAll(target, "?{http.request.uri.query}", "")
		target = strings.ReplaceAll(target, "{http.request.uri.query}", "")

		redirectConfig["target"] = target
		redirectConfig["preserve_path"] = preservePath
		redirectConfig["preserve_query"] = preserveQuery
	}

	proxy.RedirectConfig = redirectConfig
	return nil
}

// parseStaticHandler parses file_server handler into Proxy model
func parseStaticHandler(handlers []HandlerConfig, proxy *models.Proxy) error {
	staticConfig := make(map[string]interface{})

	var fileHandler *HandlerConfig
	var hasTemplates bool
	var tryFilesConfigured bool
	var tryFilesFallback string
	var catchAllPage string

	for i := range handlers {
		if handlers[i].Handler == "templates" {
			hasTemplates = true
		}
		if handlers[i].Handler == "file_server" {
			fileHandler = &handlers[i]
		}
		// Check for the subroute handler that implements try_files
		if handlers[i].Handler == "subroute" && len(handlers[i].Routes) > 0 {
			// Check if the subroute contains the SPA rewrite logic
			subroute := handlers[i].Routes[0]
			if len(subroute.Handle) > 0 && subroute.Handle[0].Handler == "rewrite" {
				// This is likely the SPA rewrite subroute
				tryFilesConfigured = true
				tryFilesFallback = subroute.Handle[0].URI
			}
		}
		// Check for the unconditional rewrite for catch-all page
		if handlers[i].Handler == "rewrite" {
			// An unconditional rewrite at this level is the catch-all page
			catchAllPage = handlers[i].URI
		}
	}

	if fileHandler == nil {
		return fmt.Errorf("no file_server handler found")
	}

	// Parse root path
	staticConfig["root_path"] = fileHandler.Root

	// Parse index file
	if len(fileHandler.IndexNames) > 0 {
		staticConfig["index_file"] = fileHandler.IndexNames[0]
	} else {
		staticConfig["index_file"] = "index.html"
	}

	// Parse template rendering
	staticConfig["template_rendering"] = hasTemplates

	// Reconstruct try_files if found
	if tryFilesConfigured {
		staticConfig["try_files"] = []interface{}{tryFilesFallback}
	}

	// Reconstruct catch_all_page if found
	if catchAllPage != "" {
		staticConfig["catch_all_page"] = catchAllPage
	}

	// Default browse to false
	staticConfig["browse"] = false

	proxy.StaticConfig = staticConfig
	return nil
}

// SimplifyProxyForUI converts a Proxy model to a simplified structure for UI display
// This removes internal fields and formats data in a user-friendly way
func SimplifyProxyForUI(proxy *models.Proxy) map[string]interface{} {
	simplified := map[string]interface{}{
		"id":          proxy.ID,
		"type":        proxy.Type,
		"name":        proxy.Name,
		"hostname":    proxy.Hostname,
		"ssl_enabled": proxy.SSLEnabled,
		"ssl_forced":  proxy.SSLForced,
		"is_active":   proxy.IsActive,
		"created_at":  proxy.CreatedAt,
		"updated_at":  proxy.UpdatedAt,
	}

	if proxy.Description != nil {
		simplified["description"] = *proxy.Description
	}

	// Add type-specific fields
	switch proxy.Type {
	case models.ProxyTypeReverseProxy:
		simplified["upstreams"] = proxy.Upstreams
		if len(proxy.LoadBalancing) > 0 {
			simplified["load_balancing"] = proxy.LoadBalancing
		}
		simplified["block_exploits"] = proxy.BlockExploits
		if len(proxy.CustomHeaders) > 0 {
			simplified["custom_headers"] = proxy.CustomHeaders
		}

	case models.ProxyTypeRedirect:
		if len(proxy.RedirectConfig) > 0 {
			simplified["redirect"] = proxy.RedirectConfig
		}

	case models.ProxyTypeStatic:
		if len(proxy.StaticConfig) > 0 {
			simplified["static"] = proxy.StaticConfig
		}
	}

	return simplified
}

// BuildCaddyfileSnippet generates a Caddyfile snippet for documentation/preview
// This is useful for showing users what the configuration looks like
func BuildCaddyfileSnippet(proxy *models.Proxy) string {
	var snippet strings.Builder

	// Write hostname
	snippet.WriteString(proxy.Hostname + " {\n")

	// Add security import if needed
	if proxy.BlockExploits && proxy.Type == models.ProxyTypeReverseProxy {
		snippet.WriteString("    import snippets/security\n\n")
	}

	// Build type-specific config
	switch proxy.Type {
	case models.ProxyTypeReverseProxy:
		snippet.WriteString("    reverse_proxy ")

		// Add upstreams
		if proxy.Upstreams != nil {
			if upstreams, ok := proxy.Upstreams.([]interface{}); ok {
				upstreamStrings := []string{}
				for _, up := range upstreams {
					if upMap, ok := up.(map[string]interface{}); ok {
						host, _ := upMap["host"].(string)
						port, _ := upMap["port"].(float64)
						upstreamStrings = append(upstreamStrings, fmt.Sprintf("%s:%d", host, int(port)))
					}
				}
				snippet.WriteString(strings.Join(upstreamStrings, " "))
			}
		}

		snippet.WriteString(" {\n")

		// Add load balancing
		if len(proxy.LoadBalancing) > 0 {
			if strategy, ok := proxy.LoadBalancing["strategy"].(string); ok {
				snippet.WriteString(fmt.Sprintf("        lb_policy %s\n", strategy))
			}
		}

		// Add headers
		snippet.WriteString("        header_up X-Real-IP {remote_host}\n")
		snippet.WriteString("        header_up X-Forwarded-For {remote_host}\n")
		snippet.WriteString("        header_up X-Forwarded-Proto {scheme}\n")

		snippet.WriteString("    }\n")

	case models.ProxyTypeRedirect:
		if len(proxy.RedirectConfig) > 0 {
			target, _ := proxy.RedirectConfig["target"].(string)
			statusCode, _ := proxy.RedirectConfig["status_code"].(float64)
			if statusCode == 0 {
				statusCode = 302
			}
			snippet.WriteString(fmt.Sprintf("    redir %s %d\n", target, int(statusCode)))
		}

	case models.ProxyTypeStatic:
		if len(proxy.StaticConfig) > 0 {
			rootPath, _ := proxy.StaticConfig["root_path"].(string)
			snippet.WriteString(fmt.Sprintf("    root * %s\n", rootPath))

			if templateRendering, _ := proxy.StaticConfig["template_rendering"].(bool); templateRendering {
				snippet.WriteString("    templates\n")
			}

			snippet.WriteString("    file_server\n")
		}
	}

	snippet.WriteString("}\n")

	return snippet.String()
}
