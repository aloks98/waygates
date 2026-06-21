package service

import (
	"fmt"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// l4ExportListLimit is the page size used when exporting all L4 proxies matching
// a filter. The List repository path caps the limit at 100 per page, so the
// export pages through the result set rather than requesting everything at once.
const l4ExportListLimit = 100

// L4UpstreamExport is the export representation of a single L4 upstream. It
// drops nothing (weight is optional/omitempty to match the frontend shape).
type L4UpstreamExport struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Weight *int   `json:"weight,omitempty"`
}

// L4RouteExport is the export representation of a single L4 route. It drops
// server-managed fields (id, l4_proxy_id, created_at, updated_at) while keeping
// everything needed to recreate the route. Optional fields use omitempty to
// match the frontend CreateL4RouteRequest shape.
type L4RouteExport struct {
	Priority             int                `json:"priority"`
	MatcherType          string             `json:"matcher_type"`
	SNIHostnames         []string           `json:"sni_hostnames,omitempty"`
	AllowedIPRanges      []string           `json:"allowed_ip_ranges,omitempty"`
	RegexPattern         *string            `json:"regex_pattern,omitempty"`
	Upstreams            []L4UpstreamExport `json:"upstreams"`
	LoadBalancingPolicy  string             `json:"load_balancing_policy"`
	TLSTerminate         bool               `json:"tls_terminate"`
	TLSPassthrough       bool               `json:"tls_passthrough"`
	ProxyProtocolVersion *string            `json:"proxy_protocol_version,omitempty"`
}

// L4Export is the import/export representation of an L4 proxy. It mirrors the
// frontend toL4ExportPayload shape: server-managed fields (id, created_at,
// updated_at, created_by) are dropped; everything needed to recreate the proxy
// is kept, including is_active so an exported inactive proxy imports inactive.
type L4Export struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	ListenPort  int             `json:"listen_port"`
	Protocol    string          `json:"protocol"`
	IsActive    bool            `json:"is_active"`
	Routes      []L4RouteExport `json:"routes"`
}

// newL4Export builds an L4Export from an L4Proxy model.
func newL4Export(p models.L4Proxy) L4Export {
	routes := make([]L4RouteExport, 0, len(p.Routes))
	for i := range p.Routes {
		r := &p.Routes[i]
		upstreams := make([]L4UpstreamExport, 0, len(r.Upstreams))
		for j := range r.Upstreams {
			u := &r.Upstreams[j]
			upstreams = append(upstreams, L4UpstreamExport{
				Host:   u.Host,
				Port:   u.Port,
				Weight: u.Weight,
			})
		}

		var sniHostnames []string
		if len(r.SNIHostnames) > 0 {
			sniHostnames = []string(r.SNIHostnames)
		}
		var allowedIPRanges []string
		if len(r.AllowedIPRanges) > 0 {
			allowedIPRanges = []string(r.AllowedIPRanges)
		}

		routes = append(routes, L4RouteExport{
			Priority:             r.Priority,
			MatcherType:          r.MatcherType,
			SNIHostnames:         sniHostnames,
			AllowedIPRanges:      allowedIPRanges,
			RegexPattern:         r.RegexPattern,
			Upstreams:            upstreams,
			LoadBalancingPolicy:  r.LoadBalancingPolicy,
			TLSTerminate:         r.TLSTerminate,
			TLSPassthrough:       r.TLSPassthrough,
			ProxyProtocolVersion: r.ProxyProtocolVersion,
		})
	}

	return L4Export{
		Name:        p.Name,
		Description: p.Description,
		ListenPort:  p.ListenPort,
		Protocol:    p.Protocol,
		IsActive:    p.IsActive,
		Routes:      routes,
	}
}

// ExportL4Proxies returns the export representation of the requested L4 proxies.
// When ids is non-empty it fetches those proxies by id, silently skipping any
// that no longer exist; otherwise it returns all L4 proxies matching the given
// list filters (protocol/search), paging through the full result set.
func (s *L4ProxyService) ExportL4Proxies(ids []int, filters ListL4ProxiesRequest) ([]L4Export, error) {
	if len(ids) > 0 {
		proxies, err := s.repo.GetByIDs(ids)
		if err != nil {
			return nil, fmt.Errorf("failed to get l4 proxies for export: %w", err)
		}
		exports := make([]L4Export, 0, len(proxies))
		for i := range proxies {
			exports = append(exports, newL4Export(proxies[i]))
		}
		return exports, nil
	}

	exports := make([]L4Export, 0)
	page := 1
	for {
		proxies, total, err := s.repo.List(repository.L4ProxyListParams{
			Page:     page,
			Limit:    l4ExportListLimit,
			Search:   filters.Search,
			Protocol: filters.Protocol,
			IsActive: filters.IsActive,
			Sort:     filters.Sort,
			Order:    filters.Order,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list l4 proxies for export: %w", err)
		}
		for i := range proxies {
			exports = append(exports, newL4Export(proxies[i]))
		}
		if len(proxies) == 0 || int64(len(exports)) >= total {
			break
		}
		page++
	}
	return exports, nil
}
