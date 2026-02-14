// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

import (
	"fmt"
	"sort"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// L4Builder builds Layer 4 configurations from L4Proxy models.
type L4Builder struct {
	logger *zap.Logger
}

// NewL4Builder creates a new L4 builder.
func NewL4Builder(logger *zap.Logger) *L4Builder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &L4Builder{
		logger: logger,
	}
}

// BuildL4Config builds a Layer4App configuration from L4Proxy models.
// Only active proxies with routes are included in the configuration.
func (b *L4Builder) BuildL4Config(proxies []models.L4Proxy) (*Layer4App, error) {
	app := NewLayer4App()

	for i := range proxies {
		proxy := &proxies[i]

		// Skip inactive proxies
		if !proxy.IsActive {
			b.logger.Debug("skipping inactive L4 proxy",
				zap.Int("proxy_id", proxy.ID),
				zap.String("name", proxy.Name),
			)
			continue
		}

		// Skip proxies with no routes
		if len(proxy.Routes) == 0 {
			b.logger.Debug("skipping L4 proxy with no routes",
				zap.Int("proxy_id", proxy.ID),
				zap.String("name", proxy.Name),
			)
			continue
		}

		server, err := b.buildServer(proxy)
		if err != nil {
			b.logger.Warn("failed to build L4 server",
				zap.Int("proxy_id", proxy.ID),
				zap.String("name", proxy.Name),
				zap.Error(err),
			)
			continue
		}

		// Use proxy name as server key for readability
		serverName := b.formatServerName(proxy)
		app.AddServer(serverName, server)

		b.logger.Debug("built L4 server",
			zap.String("server_name", serverName),
			zap.Int("proxy_id", proxy.ID),
			zap.Int("route_count", len(server.Routes)),
		)
	}

	return app, nil
}

// buildServer creates an L4Server from an L4Proxy model.
func (b *L4Builder) buildServer(proxy *models.L4Proxy) (*L4Server, error) {
	// Create listen address based on protocol and port
	listenAddr := b.buildListenAddress(proxy.Protocol, proxy.ListenPort)
	server := NewL4Server(listenAddr)

	// Sort routes by priority (lower number = higher priority)
	sortedRoutes := make([]models.L4Route, len(proxy.Routes))
	copy(sortedRoutes, proxy.Routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return sortedRoutes[i].Priority < sortedRoutes[j].Priority
	})

	// Build routes
	for i := range sortedRoutes {
		route, err := b.buildRoute(&sortedRoutes[i])
		if err != nil {
			b.logger.Warn("failed to build L4 route",
				zap.Int("route_id", sortedRoutes[i].ID),
				zap.Error(err),
			)
			continue
		}
		server.AddRoute(route)
	}

	return server, nil
}

// buildRoute creates an L4Route from an L4Route model.
func (b *L4Builder) buildRoute(modelRoute *models.L4Route) (*L4Route, error) {
	route := NewL4Route()

	// Build matchers based on matcher type
	matcher := b.buildMatcher(modelRoute)
	if matcher != nil {
		route.AddMatch(matcher)
	}

	// Add TLS termination handler if enabled (must come before proxy handler)
	if modelRoute.TLSTerminate {
		tlsHandler := b.buildTLSHandler(modelRoute)
		route.AddHandler(ToL4Handler(tlsHandler))
	}

	// Build and add proxy handler
	proxyHandler, err := b.buildProxyHandler(modelRoute)
	if err != nil {
		return nil, fmt.Errorf("building proxy handler: %w", err)
	}
	route.AddHandler(ToL4Handler(proxyHandler))

	return route, nil
}

// buildMatcher creates an L4MatcherSet from route configuration.
func (b *L4Builder) buildMatcher(route *models.L4Route) L4MatcherSet {
	var sniHostnames []string
	if len(route.SNIHostnames) > 0 {
		sniHostnames = route.SNIHostnames
	}

	var ipRanges []string
	if len(route.AllowedIPRanges) > 0 {
		ipRanges = route.AllowedIPRanges
	}

	var regexPattern string
	if route.RegexPattern != nil {
		regexPattern = *route.RegexPattern
	}

	return MapMatcherTypeToL4Matcher(route.MatcherType, sniHostnames, ipRanges, regexPattern)
}

// buildTLSHandler creates an L4TLSHandler for TLS termination.
func (b *L4Builder) buildTLSHandler(route *models.L4Route) *L4TLSHandler {
	handler := NewL4TLSHandler()

	// If SNI hostnames are specified, add connection policies
	if len(route.SNIHostnames) > 0 {
		policy := NewL4TLSConnPolicy(route.SNIHostnames)
		handler.WithConnectionPolicies(policy)
	}

	return handler
}

// buildProxyHandler creates an L4ProxyHandler from route configuration.
func (b *L4Builder) buildProxyHandler(route *models.L4Route) (*L4ProxyHandler, error) {
	if len(route.Upstreams) == 0 {
		return nil, fmt.Errorf("route has no upstreams")
	}

	// Convert model upstreams to config upstreams
	upstreams := make([]*L4Upstream, 0, len(route.Upstreams))
	for _, upstream := range route.Upstreams {
		upstreams = append(upstreams, NewL4UpstreamFromHostPort(upstream.Host, upstream.Port))
	}

	handler := NewL4ProxyHandler(upstreams...)

	// Add load balancing policy
	if route.LoadBalancingPolicy != "" {
		l4Policy := MapLBPolicyToL4(route.LoadBalancingPolicy)
		handler.WithLoadBalancing(l4Policy)
	}

	// Add proxy protocol if specified
	if route.ProxyProtocolVersion != nil && *route.ProxyProtocolVersion != "" {
		handler.WithProxyProtocol(*route.ProxyProtocolVersion)
	}

	return handler, nil
}

// buildListenAddress creates a listen address string for the given protocol and port.
func (b *L4Builder) buildListenAddress(protocol string, port int) string {
	switch protocol {
	case models.L4ProtocolUDP:
		return L4UDPListenAddress("", port)
	case models.L4ProtocolTCP:
		fallthrough
	default:
		return L4TCPListenAddress("", port)
	}
}

// formatServerName creates a descriptive server name from the proxy.
func (b *L4Builder) formatServerName(proxy *models.L4Proxy) string {
	return fmt.Sprintf("%s_%s_%d", proxy.Name, proxy.Protocol, proxy.ListenPort)
}

// BuildL4Routes builds routes for a single L4Proxy.
// This is useful when you need to build routes without creating a full Layer4App.
func (b *L4Builder) BuildL4Routes(proxy *models.L4Proxy) ([]*L4Route, error) {
	if !proxy.IsActive {
		return nil, nil
	}

	if len(proxy.Routes) == 0 {
		return nil, nil
	}

	// Sort routes by priority
	sortedRoutes := make([]models.L4Route, len(proxy.Routes))
	copy(sortedRoutes, proxy.Routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return sortedRoutes[i].Priority < sortedRoutes[j].Priority
	})

	routes := make([]*L4Route, 0, len(sortedRoutes))
	for i := range sortedRoutes {
		route, err := b.buildRoute(&sortedRoutes[i])
		if err != nil {
			b.logger.Warn("failed to build L4 route",
				zap.Int("route_id", sortedRoutes[i].ID),
				zap.Error(err),
			)
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

// BuildL4Server builds a single L4Server for a proxy.
// Returns nil if the proxy is inactive or has no routes.
func (b *L4Builder) BuildL4Server(proxy *models.L4Proxy) (*L4Server, error) {
	if !proxy.IsActive {
		return nil, nil
	}

	if len(proxy.Routes) == 0 {
		return nil, nil
	}

	return b.buildServer(proxy)
}

// MergeL4Servers merges multiple L4 proxies listening on the same port.
// This is useful when multiple proxies share the same port with different matchers.
func (b *L4Builder) MergeL4Servers(proxies []models.L4Proxy) (map[string]*L4Server, error) {
	// Group proxies by protocol:port
	portGroups := make(map[string][]models.L4Proxy)
	for i := range proxies {
		proxy := &proxies[i]
		if !proxy.IsActive || len(proxy.Routes) == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%d", proxy.Protocol, proxy.ListenPort)
		portGroups[key] = append(portGroups[key], *proxy)
	}

	servers := make(map[string]*L4Server)
	for key, group := range portGroups {
		// Build routes from all proxies in this group
		var allRoutes []*L4Route
		var protocol string
		var port int

		for i := range group {
			proxy := &group[i]
			protocol = proxy.Protocol
			port = proxy.ListenPort

			routes, err := b.BuildL4Routes(proxy)
			if err != nil {
				b.logger.Warn("failed to build routes for proxy",
					zap.Int("proxy_id", proxy.ID),
					zap.Error(err),
				)
				continue
			}
			allRoutes = append(allRoutes, routes...)
		}

		if len(allRoutes) == 0 {
			continue
		}

		// Create server with merged routes
		listenAddr := b.buildListenAddress(protocol, port)
		server := NewL4Server(listenAddr)
		server.AddRoutes(allRoutes...)

		servers[key] = server
	}

	return servers, nil
}
