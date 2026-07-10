// Package proxygroup resolves a proxy's effective configuration by merging its
// ProxyGroup's defaults underneath its own explicit values.
//
// It lives at internal/proxygroup rather than internal/service/proxygroup
// because internal/caddy/config imports it, and the Caddy builder must not
// depend on the service layer.
package proxygroup

import (
	"github.com/aloks98/waygates/backend/internal/models"
)

// System defaults, applied when neither the proxy nor its group has an opinion.
// These are lifted verbatim from the defaults ProxyHandler.CreateProxy applied
// before this package existed (handlers/proxy.go:235-248). They are NOT
// all-false. Changing DefaultSSLEnabled or DefaultBlockExploits to false is a
// security regression: new proxies would be served over plaintext with exploit
// blocking off.
const (
	DefaultSSLEnabled            = true
	DefaultSSLForced             = true
	DefaultBlockExploits         = true
	DefaultTLSInsecureSkipVerify = false
)

// EffectiveProxy is a proxy with every inheritable value decided. It is the only
// type the Caddy builder accepts, so an unresolved models.Proxy cannot reach
// config generation.
type EffectiveProxy struct {
	ID       int
	Type     string
	Name     string
	Hostname string
	IsActive bool

	SSLEnabled            bool
	SSLForced             bool
	BlockExploits         bool
	TLSInsecureSkipVerify bool

	Upstreams      interface{}
	LoadBalancing  models.JSONField
	CustomHeaders  models.CustomHeaders
	RedirectConfig models.JSONField
	StaticConfig   models.JSONField

	// ACL is the merged assignment set, expressed in the same row type the
	// Caddy builder already consumes. Rows inherited from the group carry
	// ID == 0; the builder's existing Enabled filter implements opt-out.
	ACL []models.ProxyACLAssignment

	// GroupID is nil for an ungrouped proxy. Carried for display and logging.
	GroupID *int
}

// EffectiveHostname composes a label-addressed proxy's full hostname.
func EffectiveHostname(label, baseDomain string) string {
	return label + "." + baseDomain
}

// Resolve merges a group's defaults into a proxy. g may be nil.
//
// Scalars: proxy value if non-nil, else group value if non-nil, else the system
// default. Headers: per-key union, proxy wins. ACL: union by ACLGroupID, the
// proxy's row winning wholesale. LoadBalancing, RedirectConfig, StaticConfig,
// Hostname, Upstreams, Type, Name and IsActive are never inherited.
//
// Resolve does not mutate its arguments.
func Resolve(
	p models.Proxy,
	g *models.ProxyGroup,
	proxyACL []models.ProxyACLAssignment,
	groupACL []models.ProxyGroupACLAssignment,
) EffectiveProxy {
	var (
		gSSLEnabled, gSSLForced, gBlockExploits, gTLSInsecure *bool
		gHeaders                                              models.CustomHeaders
	)
	if g != nil {
		gSSLEnabled, gSSLForced = g.SSLEnabled, g.SSLForced
		gBlockExploits, gTLSInsecure = g.BlockExploits, g.TLSInsecureSkipVerify
		gHeaders = g.CustomHeaders
	}

	return EffectiveProxy{
		ID:       p.ID,
		Type:     p.Type,
		Name:     p.Name,
		Hostname: p.Hostname,
		IsActive: p.IsActive,
		GroupID:  p.GroupID,

		SSLEnabled:            resolveBool(p.SSLEnabled, gSSLEnabled, DefaultSSLEnabled),
		SSLForced:             resolveBool(p.SSLForced, gSSLForced, DefaultSSLForced),
		BlockExploits:         resolveBool(p.BlockExploits, gBlockExploits, DefaultBlockExploits),
		TLSInsecureSkipVerify: resolveBool(p.TLSInsecureSkipVerify, gTLSInsecure, DefaultTLSInsecureSkipVerify),

		Upstreams:      p.Upstreams,
		LoadBalancing:  p.LoadBalancing,
		CustomHeaders:  mergeHeaders(gHeaders, p.CustomHeaders),
		RedirectConfig: p.RedirectConfig,
		StaticConfig:   p.StaticConfig,

		ACL: mergeACL(p.ID, proxyACL, groupACL),
	}
}

// resolveBool walks proxy -> group -> system default.
func resolveBool(proxy, group *bool, systemDefault bool) bool {
	if proxy != nil {
		return *proxy
	}
	if group != nil {
		return *group
	}
	return systemDefault
}

// mergeHeaders unions the group's headers under the proxy's, per key and per
// direction. It allocates fresh maps so neither input is mutated.
func mergeHeaders(group, proxy models.CustomHeaders) models.CustomHeaders {
	return models.CustomHeaders{
		Request:  mergeStringMap(group.Request, proxy.Request),
		Response: mergeStringMap(group.Response, proxy.Response),
	}
}

func mergeStringMap(base, over map[string]string) map[string]string {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

// mergeACL unions the group's assignments under the proxy's, keyed by
// ACLGroupID. A proxy row wins wholesale — never field-wise, which would
// produce a row neither side authored. Inherited rows are synthesized onto the
// proxy with ID 0 marking their provenance.
//
// Disabled rows are kept, not dropped: the Caddy builder's SetACLAssignments
// already skips Enabled == false, and keeping the row is what lets a proxy opt
// out of an inherited ACL by re-assigning it with enabled = false.
func mergeACL(
	proxyID int,
	proxyACL []models.ProxyACLAssignment,
	groupACL []models.ProxyGroupACLAssignment,
) []models.ProxyACLAssignment {
	if len(proxyACL) == 0 && len(groupACL) == 0 {
		return nil
	}

	claimed := make(map[int]bool, len(proxyACL))
	for _, a := range proxyACL {
		claimed[a.ACLGroupID] = true
	}

	out := make([]models.ProxyACLAssignment, 0, len(proxyACL)+len(groupACL))
	for _, a := range groupACL {
		if claimed[a.ACLGroupID] {
			continue // the proxy overrides this one wholesale
		}
		out = append(out, models.ProxyACLAssignment{
			ID:          0, // inherited
			ProxyID:     proxyID,
			ACLGroupID:  a.ACLGroupID,
			PathPattern: a.PathPattern,
			Priority:    a.Priority,
			Enabled:     a.Enabled,
		})
	}
	out = append(out, proxyACL...)
	return out
}
