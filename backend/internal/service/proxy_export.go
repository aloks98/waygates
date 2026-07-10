package service

import (
	"fmt"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// exportListLimit is the page size used when exporting all proxies matching a
// filter. The List repository path caps the limit at 100 per page, so the
// export pages through the result set rather than requesting everything at once.
const exportListLimit = 100

// ProxyExport is the import/export representation of a proxy. It mirrors the
// frontend export item: it drops server-managed fields (id, timestamps,
// created_by, ssl_forced) while keeping everything needed to recreate the proxy,
// including is_active so an exported inactive proxy imports inactive.
type ProxyExport struct {
	Type                  string                `json:"type"`
	Name                  string                `json:"name"`
	Hostname              string                `json:"hostname"`
	Description           *string               `json:"description,omitempty"`
	SSLEnabled            bool                  `json:"ssl_enabled"`
	IsActive              bool                  `json:"is_active"`
	Upstreams             interface{}           `json:"upstreams,omitempty"`
	LoadBalancing         models.JSONField      `json:"load_balancing,omitempty"`
	BlockExploits         bool                  `json:"block_exploits"`
	TLSInsecureSkipVerify bool                  `json:"tls_insecure_skip_verify"`
	CustomHeaders         *models.CustomHeaders `json:"custom_headers,omitempty"`
	RedirectConfig        models.JSONField      `json:"redirect,omitempty"`
	StaticConfig          models.JSONField      `json:"static,omitempty"`
}

// derefBool reports the pointed-to value, treating a nil *bool as false. This
// is a TEMPORARY shim: models.Proxy's inheritable booleans are now *bool
// (nil = inherit from group / system default), but the export contract still
// carries plain bools. A group-inheriting proxy therefore currently exports
// its inherited fields as false rather than resolving them; revisit once
// import/export gains inheritance awareness.
func derefBool(b *bool) bool { return b != nil && *b }

// newProxyExport builds a ProxyExport from a Proxy model.
func newProxyExport(p models.Proxy) ProxyExport {
	export := ProxyExport{
		Type:                  p.Type,
		Name:                  p.Name,
		Hostname:              p.Hostname,
		Description:           p.Description,
		SSLEnabled:            derefBool(p.SSLEnabled),
		IsActive:              p.IsActive,
		Upstreams:             p.Upstreams,
		LoadBalancing:         p.LoadBalancing,
		BlockExploits:         derefBool(p.BlockExploits),
		TLSInsecureSkipVerify: derefBool(p.TLSInsecureSkipVerify),
		RedirectConfig:        p.RedirectConfig,
		StaticConfig:          p.StaticConfig,
	}
	if !p.CustomHeaders.IsEmpty() {
		headers := p.CustomHeaders
		export.CustomHeaders = &headers
	}
	return export
}

// ExportProxies returns the export representation of the requested proxies. When
// ids is non-empty it fetches those proxies by id, silently skipping any that no
// longer exist; otherwise it returns all proxies matching the given list filters
// (paging through the full result set, ignoring pagination).
func (s *ProxyService) ExportProxies(ids []int, filters ListProxiesRequest) ([]ProxyExport, error) {
	if len(ids) > 0 {
		proxies, err := s.repo.GetByIDs(ids)
		if err != nil {
			return nil, fmt.Errorf("failed to get proxies for export: %w", err)
		}
		exports := make([]ProxyExport, 0, len(proxies))
		for i := range proxies {
			exports = append(exports, newProxyExport(proxies[i]))
		}
		return exports, nil
	}

	exports := make([]ProxyExport, 0)
	page := 1
	for {
		proxies, total, err := s.repo.List(repository.ProxyListParams{
			Page:         page,
			Limit:        exportListLimit,
			Search:       filters.Search,
			Types:        filters.Types,
			TypesExclude: filters.TypesExclude,
			Status:       filters.Status,
			StatusNot:    filters.StatusNot,
			SSLEnabled:   filters.SSLEnabled,
			Target:       filters.Target,
			Sort:         filters.Sort,
			Order:        filters.Order,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list proxies for export: %w", err)
		}
		for i := range proxies {
			exports = append(exports, newProxyExport(proxies[i]))
		}
		if len(proxies) == 0 || int64(len(exports)) >= total {
			break
		}
		page++
	}
	return exports, nil
}
