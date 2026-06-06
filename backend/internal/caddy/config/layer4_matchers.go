// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

// L4 Matcher module names as constants.
// See: https://github.com/mholt/caddy-l4
const (
	L4MatcherTLS      = "tls"
	L4MatcherSSH      = "ssh"
	L4MatcherPostgres = "postgres"
	L4MatcherHTTP     = "http"
	L4MatcherRDP      = "rdp"
	L4MatcherSocks5   = "socks5"
	L4MatcherRemoteIP = "remote_ip"
	L4MatcherRegexp   = "regexp"
	L4MatcherNot      = "not"
)

// L4MatcherSet is a set of matchers that must all match for an L4 route.
type L4MatcherSet map[string]interface{}

// L4Handler represents any L4 handler configuration.
type L4Handler map[string]interface{}

// L4MatchTLS matches TLS connections by SNI hostname.
// See: https://github.com/mholt/caddy-l4#tls
type L4MatchTLS struct {
	SNI []string `json:"sni,omitempty"`
}

// L4MatchSSH matches SSH protocol connections.
// See: https://github.com/mholt/caddy-l4#ssh
type L4MatchSSH struct{}

// L4MatchPostgres matches PostgreSQL protocol connections.
// See: https://github.com/mholt/caddy-l4#postgres
type L4MatchPostgres struct{}

// L4MatchHTTP is caddy-l4's http matcher value. Despite its auto-generated
// docs describing an object, the layer4.matchers.http module deserializes into
// caddyhttp.RawMatcherSets: a JSON ARRAY of caddyhttp matcher-set objects
// (e.g. [{"host":["h1","h2"]}]). An empty array means "match any HTTP
// connection". See https://github.com/mholt/caddy-l4 modules/l4http.
type L4MatchHTTP []map[string]interface{}

// L4MatchRDP matches RDP (Remote Desktop Protocol) connections.
// See: https://github.com/mholt/caddy-l4#rdp
type L4MatchRDP struct{}

// L4MatchSOCKS5 matches SOCKS5 protocol connections.
// See: https://github.com/mholt/caddy-l4#socks5
type L4MatchSOCKS5 struct{}

// L4MatchRemoteIP matches connections by client IP address.
// See: https://github.com/mholt/caddy-l4#remote_ip
type L4MatchRemoteIP struct {
	Ranges []string `json:"ranges,omitempty"`
}

// L4MatchRegexp matches connection data by regular expression.
// See: https://github.com/mholt/caddy-l4#regexp
type L4MatchRegexp struct {
	Pattern string `json:"pattern,omitempty"`
	Count   int    `json:"count,omitempty"` // Number of bytes to read for matching
}

// L4MatchNot negates the matchers it contains.
type L4MatchNot []L4MatcherSet

// NewL4TLSMatcher creates a matcher set that matches TLS connections by SNI.
func NewL4TLSMatcher(sni []string) L4MatcherSet {
	return L4MatcherSet{
		L4MatcherTLS: &L4MatchTLS{SNI: sni},
	}
}

// NewL4TLSMatcherAny creates a matcher set that matches any TLS connection.
func NewL4TLSMatcherAny() L4MatcherSet {
	return L4MatcherSet{
		L4MatcherTLS: &L4MatchTLS{},
	}
}

// NewL4SSHMatcher creates a matcher set that matches SSH connections.
func NewL4SSHMatcher() L4MatcherSet {
	return L4MatcherSet{
		L4MatcherSSH: &L4MatchSSH{},
	}
}

// NewL4PostgresMatcher creates a matcher set that matches PostgreSQL connections.
func NewL4PostgresMatcher() L4MatcherSet {
	return L4MatcherSet{
		L4MatcherPostgres: &L4MatchPostgres{},
	}
}

// NewL4HTTPMatcher creates a matcher set that matches HTTP connections at L4.
// With no hosts it produces {"http": []} (match any HTTP connection); with
// hosts it produces {"http": [{"host": ["h1", ...]}]} using the caddyhttp host
// matcher, mirroring caddyhttp.RawMatcherSets.
func NewL4HTTPMatcher(hosts ...string) L4MatcherSet {
	sets := L4MatchHTTP{}
	if len(hosts) > 0 {
		sets = append(sets, map[string]interface{}{"host": hosts})
	}
	return L4MatcherSet{
		L4MatcherHTTP: sets,
	}
}

// NewL4RDPMatcher creates a matcher set that matches RDP connections.
func NewL4RDPMatcher() L4MatcherSet {
	return L4MatcherSet{
		L4MatcherRDP: &L4MatchRDP{},
	}
}

// NewL4SOCKS5Matcher creates a matcher set that matches SOCKS5 connections.
func NewL4SOCKS5Matcher() L4MatcherSet {
	return L4MatcherSet{
		L4MatcherSocks5: &L4MatchSOCKS5{},
	}
}

// NewL4RemoteIPMatcher creates a matcher set that matches by client IP ranges.
func NewL4RemoteIPMatcher(ranges []string) L4MatcherSet {
	return L4MatcherSet{
		L4MatcherRemoteIP: &L4MatchRemoteIP{Ranges: ranges},
	}
}

// NewL4RegexpMatcher creates a matcher set that matches by regex pattern.
func NewL4RegexpMatcher(pattern string, count int) L4MatcherSet {
	matcher := &L4MatchRegexp{Pattern: pattern}
	if count > 0 {
		matcher.Count = count
	}
	return L4MatcherSet{
		L4MatcherRegexp: matcher,
	}
}

// NewL4NotMatcher creates a matcher set that negates the given matchers.
func NewL4NotMatcher(matchers ...L4MatcherSet) L4MatcherSet {
	return L4MatcherSet{
		L4MatcherNot: L4MatchNot(matchers),
	}
}

// CombineL4Matchers merges multiple L4 matcher sets into one.
func CombineL4Matchers(matchers ...L4MatcherSet) L4MatcherSet {
	combined := make(L4MatcherSet)
	for _, m := range matchers {
		for k, v := range m {
			if _, exists := combined[k]; !exists {
				combined[k] = v
			}
		}
	}
	return combined
}

// AddRemoteIPToL4Matcher adds remote IP matching to an existing L4 matcher set.
func AddRemoteIPToL4Matcher(m L4MatcherSet, ranges []string) L4MatcherSet {
	m[L4MatcherRemoteIP] = &L4MatchRemoteIP{Ranges: ranges}
	return m
}

// MapMatcherTypeToL4Matcher maps model matcher type to Caddy L4 matcher.
// Returns nil for "any" type as it doesn't require a matcher.
func MapMatcherTypeToL4Matcher(matcherType string, sniHostnames, ipRanges []string, regexPattern string) L4MatcherSet {
	switch matcherType {
	case "any":
		// "any" matches all - no matcher needed
		if len(ipRanges) > 0 {
			return NewL4RemoteIPMatcher(ipRanges)
		}
		return nil
	case "tls":
		matcher := NewL4TLSMatcher(sniHostnames)
		if len(ipRanges) > 0 {
			AddRemoteIPToL4Matcher(matcher, ipRanges)
		}
		return matcher
	case "ssh":
		matcher := NewL4SSHMatcher()
		if len(ipRanges) > 0 {
			AddRemoteIPToL4Matcher(matcher, ipRanges)
		}
		return matcher
	case "postgres":
		matcher := NewL4PostgresMatcher()
		if len(ipRanges) > 0 {
			AddRemoteIPToL4Matcher(matcher, ipRanges)
		}
		return matcher
	case "http":
		// The L4 http matcher performs protocol detection only: it matches ANY
		// HTTP connection and intentionally ignores sniHostnames. SNI is a
		// TLS-layer concept (see the "tls" case) and HTTP host-based routing
		// belongs in the L7 stack, not at layer 4. We therefore call
		// NewL4HTTPMatcher() with no host args (match-any); any sni_hostnames
		// configured on an http-matcher route are silently ignored by design.
		matcher := NewL4HTTPMatcher()
		if len(ipRanges) > 0 {
			AddRemoteIPToL4Matcher(matcher, ipRanges)
		}
		return matcher
	case "rdp":
		matcher := NewL4RDPMatcher()
		if len(ipRanges) > 0 {
			AddRemoteIPToL4Matcher(matcher, ipRanges)
		}
		return matcher
	case "socks5":
		matcher := NewL4SOCKS5Matcher()
		if len(ipRanges) > 0 {
			AddRemoteIPToL4Matcher(matcher, ipRanges)
		}
		return matcher
	case "remote_ip":
		return NewL4RemoteIPMatcher(ipRanges)
	case "regexp":
		matcher := NewL4RegexpMatcher(regexPattern, 0)
		if len(ipRanges) > 0 {
			AddRemoteIPToL4Matcher(matcher, ipRanges)
		}
		return matcher
	default:
		return nil
	}
}
