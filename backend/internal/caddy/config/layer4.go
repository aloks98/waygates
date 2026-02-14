// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

// L4 protocol constants for listen addresses.
const (
	L4ProtocolTCP = "tcp"
	L4ProtocolUDP = "udp"
)

// NewLayer4App creates a new Layer4 application configuration.
func NewLayer4App() *Layer4App {
	return &Layer4App{
		Servers: make(map[string]*L4Server),
	}
}

// NewL4Server creates a new L4 server with the given listen addresses.
func NewL4Server(listen ...string) *L4Server {
	return &L4Server{
		Listen: listen,
		Routes: make([]*L4Route, 0),
	}
}

// NewL4Route creates a new L4 route.
func NewL4Route() *L4Route {
	return &L4Route{
		Match:  make([]L4MatcherSet, 0),
		Handle: make([]L4Handler, 0),
	}
}

// AddServer adds a server to the Layer4 app.
func (a *Layer4App) AddServer(name string, server *L4Server) *Layer4App {
	if a.Servers == nil {
		a.Servers = make(map[string]*L4Server)
	}
	a.Servers[name] = server
	return a
}

// AddRoute adds a route to the L4 server.
func (s *L4Server) AddRoute(route *L4Route) *L4Server {
	s.Routes = append(s.Routes, route)
	return s
}

// AddRoutes adds multiple routes to the L4 server.
func (s *L4Server) AddRoutes(routes ...*L4Route) *L4Server {
	s.Routes = append(s.Routes, routes...)
	return s
}

// AddMatch adds a matcher set to the L4 route.
func (r *L4Route) AddMatch(matchers ...L4MatcherSet) *L4Route {
	r.Match = append(r.Match, matchers...)
	return r
}

// AddHandler adds a handler to the L4 route.
func (r *L4Route) AddHandler(handler L4Handler) *L4Route {
	r.Handle = append(r.Handle, handler)
	return r
}

// L4ListenAddress formats a listen address for L4 proxy.
// Format: "protocol/host:port" e.g., "tcp/:8080", "udp/0.0.0.0:53"
func L4ListenAddress(protocol, host string, port int) string {
	if host == "" {
		return protocol + "/:" + itoa(port)
	}
	return protocol + "/" + host + ":" + itoa(port)
}

// L4TCPListenAddress creates a TCP listen address.
func L4TCPListenAddress(host string, port int) string {
	return L4ListenAddress(L4ProtocolTCP, host, port)
}

// L4UDPListenAddress creates a UDP listen address.
func L4UDPListenAddress(host string, port int) string {
	return L4ListenAddress(L4ProtocolUDP, host, port)
}

// NewL4ProxyRoute creates a route for proxying to the given upstreams.
func NewL4ProxyRoute(upstreams []*L4Upstream) *L4Route {
	handler := NewL4ProxyHandler(upstreams...)

	route := NewL4Route()
	route.AddHandler(ToL4Handler(handler))

	return route
}

// NewL4TLSProxyRoute creates a route for TLS passthrough proxying.
func NewL4TLSProxyRoute(sniHostnames []string, upstreams []*L4Upstream) *L4Route {
	handler := NewL4ProxyHandler(upstreams...)

	route := NewL4Route()
	if len(sniHostnames) > 0 {
		route.AddMatch(NewL4TLSMatcher(sniHostnames))
	}
	route.AddHandler(ToL4Handler(handler))

	return route
}

// NewL4TLSTerminateRoute creates a route that terminates TLS and proxies to upstream.
func NewL4TLSTerminateRoute(sniHostnames []string, upstreams []*L4Upstream) *L4Route {
	route := NewL4Route()

	if len(sniHostnames) > 0 {
		route.AddMatch(NewL4TLSMatcher(sniHostnames))
	}

	// Add TLS termination handler first
	tlsHandler := NewL4TLSHandler()
	route.AddHandler(ToL4Handler(tlsHandler))

	// Then proxy handler
	proxyHandler := NewL4ProxyHandler(upstreams...)
	route.AddHandler(ToL4Handler(proxyHandler))

	return route
}

// GroupL4RoutesByPort groups L4 routes by their listen port for server creation.
func GroupL4RoutesByPort(routes map[int][]*L4Route) map[int]*L4Server {
	servers := make(map[int]*L4Server)
	for port, routeList := range routes {
		server := NewL4Server()
		server.AddRoutes(routeList...)
		servers[port] = server
	}
	return servers
}
