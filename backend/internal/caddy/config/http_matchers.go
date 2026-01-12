// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

// MatchHost matches requests by hostname.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/host/
type MatchHost []string

// MatchPath matches requests by URI path.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/path/
type MatchPath []string

// MatchPathRE matches requests by URI path using regular expressions.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/path_regexp/
type MatchPathRE struct {
	Name    string `json:"name,omitempty"`
	Pattern string `json:"pattern"`
}

// MatchRemoteIP matches requests by client IP address.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/remote_ip/
type MatchRemoteIP struct {
	Ranges []string `json:"ranges,omitempty"`
}

// MatchHeader matches requests by header values.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/header/
type MatchHeader map[string][]string

// MatchProtocol matches requests by protocol (http, https, grpc).
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/protocol/
type MatchProtocol string

// MatchMethod matches requests by HTTP method.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/method/
type MatchMethod []string

// MatchNot negates the matchers it contains.
// See: https://caddyserver.com/docs/json/apps/http/servers/routes/match/not/
type MatchNot []MatcherSet

// NewHostMatcher creates a matcher set that matches the given hosts.
func NewHostMatcher(hosts ...string) MatcherSet {
	return MatcherSet{
		"host": MatchHost(hosts),
	}
}

// NewPathMatcher creates a matcher set that matches the given paths.
func NewPathMatcher(paths ...string) MatcherSet {
	return MatcherSet{
		"path": MatchPath(paths),
	}
}

// NewPathREMatcher creates a matcher set that matches paths using a regex pattern.
func NewPathREMatcher(name, pattern string) MatcherSet {
	return MatcherSet{
		"path_regexp": &MatchPathRE{
			Name:    name,
			Pattern: pattern,
		},
	}
}

// NewRemoteIPMatcher creates a matcher set that matches the given IP ranges.
func NewRemoteIPMatcher(ranges ...string) MatcherSet {
	return MatcherSet{
		"remote_ip": &MatchRemoteIP{
			Ranges: ranges,
		},
	}
}

// NewHeaderMatcher creates a matcher set that matches the given headers.
func NewHeaderMatcher(headers map[string][]string) MatcherSet {
	return MatcherSet{
		"header": MatchHeader(headers),
	}
}

// NewMethodMatcher creates a matcher set that matches the given HTTP methods.
func NewMethodMatcher(methods ...string) MatcherSet {
	return MatcherSet{
		"method": MatchMethod(methods),
	}
}

// NewNotMatcher creates a matcher set that negates the given matchers.
func NewNotMatcher(matchers ...MatcherSet) MatcherSet {
	return MatcherSet{
		"not": MatchNot(matchers),
	}
}

// NewHostPathMatcher creates a matcher set that matches both host and path.
func NewHostPathMatcher(host string, paths ...string) MatcherSet {
	return MatcherSet{
		"host": MatchHost{host},
		"path": MatchPath(paths),
	}
}

// CombineMatchers merges multiple matcher sets into one.
// If the same matcher type appears in multiple sets, only the first one is kept.
func CombineMatchers(matchers ...MatcherSet) MatcherSet {
	combined := make(MatcherSet)
	for _, m := range matchers {
		for k, v := range m {
			if _, exists := combined[k]; !exists {
				combined[k] = v
			}
		}
	}
	return combined
}

// AddHostToMatcher adds host matching to an existing matcher set.
func AddHostToMatcher(m MatcherSet, hosts ...string) MatcherSet {
	m["host"] = MatchHost(hosts)
	return m
}

// AddPathToMatcher adds path matching to an existing matcher set.
func AddPathToMatcher(m MatcherSet, paths ...string) MatcherSet {
	m["path"] = MatchPath(paths)
	return m
}

// AddRemoteIPToMatcher adds remote IP matching to an existing matcher set.
func AddRemoteIPToMatcher(m MatcherSet, ranges ...string) MatcherSet {
	m["remote_ip"] = &MatchRemoteIP{Ranges: ranges}
	return m
}

// StaticAssetExtensions returns common static asset file extensions.
// Used for bypassing authentication on static assets.
func StaticAssetExtensions() []string {
	return []string{
		"*.ico", "*.png", "*.jpg", "*.jpeg", "*.gif", "*.svg", "*.webp",
		"*.css", "*.js", "*.woff", "*.woff2", "*.ttf", "*.map",
	}
}

// StaticAssetPaths returns common static asset paths.
// Used for bypassing authentication on static assets.
func StaticAssetPaths() []string {
	return []string{
		"/favicon.ico",
		"/robots.txt",
		"/sitemap.xml",
	}
}

// NewStaticAssetMatcher creates a matcher for common static assets.
// This can be used to bypass authentication for static files.
func NewStaticAssetMatcher() MatcherSet {
	paths := append(StaticAssetPaths(), StaticAssetExtensions()...)
	return NewPathMatcher(paths...)
}
