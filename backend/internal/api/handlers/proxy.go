package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/proxygroup"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// groupGetter is the one capability GetProxy needs from the proxy-group layer:
// loading the proxy's group so proxygroup.Resolve can merge its settings. A
// narrow interface here (rather than the full ProxyGroupServiceInterface)
// keeps the handler's test double to a single method; it is satisfied by
// *service.ProxyGroupService without any change there.
type groupGetter interface {
	GetGroupByID(id int) (*models.ProxyGroup, error)
}

// ProxyHandler handles proxy-related HTTP requests
type ProxyHandler struct {
	service      service.ProxyServiceInterface
	groupService groupGetter
	auditService service.AuditServiceInterface
	logger       *zap.Logger
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(svc service.ProxyServiceInterface, groupService groupGetter, auditService service.AuditServiceInterface, logger *zap.Logger) *ProxyHandler {
	return &ProxyHandler{
		service:      svc,
		groupService: groupService,
		auditService: auditService,
		logger:       logger,
	}
}

// getClientIP extracts the client IP from the request, handling X-Forwarded-For
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Remove port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// ListProxies handles GET /api/proxies
func (h *ProxyHandler) ListProxies(w http.ResponseWriter, r *http.Request) {
	// Parse and validate query parameters
	var page, limit int
	var err error

	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			utils.BadRequest(w, "Invalid page parameter: must be a non-negative integer", nil)
			return
		}
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 || limit > 100 {
			utils.BadRequest(w, "Invalid limit parameter: must be an integer between 0 and 100", nil)
			return
		}
	}

	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	// Create request with defaults
	req := service.ListProxiesRequest{
		Page:   page,
		Limit:  limit,
		Search: search,
		Sort:   sort,
		Order:  order,
	}

	// Parse type filter (supports operator:value format)
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		fv := parseFilterParam(typeParam)
		switch fv.Operator {
		case OpIn, OpEq:
			if len(fv.Values) > 0 {
				req.Types = fv.Values
			} else {
				req.Types = splitAndTrim(fv.Value)
			}
		case OpNotIn, OpNot:
			if len(fv.Values) > 0 {
				req.TypesExclude = fv.Values
			} else {
				req.TypesExclude = splitAndTrim(fv.Value)
			}
		default:
			utils.BadRequest(w, "Invalid operator for type filter", nil)
			return
		}
	}

	// Parse status filter (supports operator:value format)
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		fv := parseFilterParam(statusParam)
		// Validate status value
		statusVal := fv.Value
		if statusVal != "active" && statusVal != "inactive" {
			utils.BadRequest(w, "Invalid status parameter: must be 'active' or 'inactive'", nil)
			return
		}
		switch fv.Operator {
		case OpEq:
			req.Status = statusVal
		case OpNot:
			req.StatusNot = statusVal
		default:
			utils.BadRequest(w, "Invalid operator for status filter", nil)
			return
		}
	}

	// Parse ssl_enabled filter
	if sslParam := r.URL.Query().Get("ssl_enabled"); sslParam != "" {
		switch sslParam {
		case "true":
			sslEnabled := true
			req.SSLEnabled = &sslEnabled
		case "false":
			sslEnabled := false
			req.SSLEnabled = &sslEnabled
		default:
			utils.BadRequest(w, "Invalid ssl_enabled parameter: must be 'true' or 'false'", nil)
			return
		}
	}

	// Parse target filter (searches in upstreams/redirect config)
	if target := r.URL.Query().Get("target"); target != "" {
		req.Target = target
	}

	// Parse group filter (supports the same operator:value syntax as type and
	// status): ?group=eq:3, ?group=in:1,2, ?group=not:3, ?group=eq:none (ungrouped).
	if groupParam := r.URL.Query().Get("group"); groupParam != "" {
		fv := parseFilterParam(groupParam)
		switch fv.Operator {
		case OpEq:
			if fv.Value == "none" {
				req.Ungrouped = true
			} else {
				id, err := strconv.Atoi(fv.Value)
				if err != nil {
					utils.BadRequest(w, "Invalid group parameter: must be an integer group id or 'none'", nil)
					return
				}
				req.GroupID = &id
			}
		case OpIn:
			values := fv.Values
			if len(values) == 0 {
				values = splitAndTrim(fv.Value)
			}
			ids := make([]int, 0, len(values))
			for _, v := range values {
				id, err := strconv.Atoi(v)
				if err != nil {
					utils.BadRequest(w, "Invalid group parameter: must be a comma-separated list of integer group ids", nil)
					return
				}
				ids = append(ids, id)
			}
			req.GroupIDIn = ids
		case OpNot:
			id, err := strconv.Atoi(fv.Value)
			if err != nil {
				utils.BadRequest(w, "Invalid group parameter: must be an integer group id", nil)
				return
			}
			req.GroupIDNot = &id
		default:
			utils.BadRequest(w, "Invalid operator for group filter", nil)
			return
		}
	}

	// Get proxies from service
	result, err := h.service.ListProxies(req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to list proxies",
				zap.Int("page", req.Page),
				zap.Int("limit", req.Limit),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to list proxies")
		return
	}

	// Return success response
	utils.Success(w, result, "")
}

// effectiveSource reports, per inheritable setting, where the effective value
// came from: "proxy" (explicit on this proxy), "group" (inherited), or
// "default" (neither has an opinion — the system default applies). Without
// this the edit form cannot distinguish "Inherit (currently on)" from an
// explicit "On" — the exact divergence proxygroup.Resolve exists to prevent,
// relocated into the UI if this weren't exposed.
type effectiveSource struct {
	SSLEnabled            string `json:"ssl_enabled"`
	SSLForced             string `json:"ssl_forced"`
	BlockExploits         string `json:"block_exploits"`
	TLSInsecureSkipVerify string `json:"tls_insecure_skip_verify"`
}

// effectiveView is the resolved (proxygroup.Resolve) view of a proxy's
// inheritable settings: what is actually served, as opposed to the raw
// nullable columns on the embedded Proxy.
type effectiveView struct {
	SSLEnabled            bool                 `json:"ssl_enabled"`
	SSLForced             bool                 `json:"ssl_forced"`
	BlockExploits         bool                 `json:"block_exploits"`
	TLSInsecureSkipVerify bool                 `json:"tls_insecure_skip_verify"`
	CustomHeaders         models.CustomHeaders `json:"custom_headers"`
	Source                effectiveSource      `json:"_source"`
}

// proxyDetailResponse is the GET /api/proxies/{id} response: the raw nullable
// row (so the edit form can render "inherit" correctly) plus the resolved
// `effective` view (so the overview can show what is actually served).
type proxyDetailResponse struct {
	*models.Proxy
	Effective effectiveView `json:"effective"`
}

// sourceOf reports where a resolved value came from: the proxy's own value if
// it set one, else the group's, else the system default.
func sourceOf(proxyVal, groupVal *bool) string {
	switch {
	case proxyVal != nil:
		return "proxy"
	case groupVal != nil:
		return "group"
	default:
		return "default"
	}
}

// GetProxy handles GET /api/proxies/:id
func (h *ProxyHandler) GetProxy(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	// Get proxy from service
	proxy, err := h.service.GetProxyByID(id)
	if err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to get proxy",
				zap.Int("id", id),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to get proxy")
		return
	}

	// Load the proxy's group (if any) so its settings can be merged in.
	var group *models.ProxyGroup
	if proxy.GroupID != nil && h.groupService != nil {
		group, err = h.groupService.GetGroupByID(*proxy.GroupID)
		if err != nil {
			if h.logger != nil {
				h.logger.Error("Failed to load proxy's group",
					zap.Int("id", id), zap.Int("group_id", *proxy.GroupID), zap.Error(err))
			}
			utils.InternalError(w, "Failed to resolve proxy configuration")
			return
		}
	}

	// proxygroup.Resolve is the only place a default is applied (see
	// internal/proxygroup). ACL assignments are not part of effectiveView, so
	// they are passed as nil rather than fetched.
	eff := proxygroup.Resolve(*proxy, group, nil, nil)

	var gSSLEnabled, gSSLForced, gBlockExploits, gTLSInsecure *bool
	if group != nil {
		gSSLEnabled, gSSLForced = group.SSLEnabled, group.SSLForced
		gBlockExploits, gTLSInsecure = group.BlockExploits, group.TLSInsecureSkipVerify
	}

	resp := proxyDetailResponse{
		Proxy: proxy,
		Effective: effectiveView{
			SSLEnabled:            eff.SSLEnabled,
			SSLForced:             eff.SSLForced,
			BlockExploits:         eff.BlockExploits,
			TLSInsecureSkipVerify: eff.TLSInsecureSkipVerify,
			CustomHeaders:         eff.CustomHeaders,
			Source: effectiveSource{
				SSLEnabled:            sourceOf(proxy.SSLEnabled, gSSLEnabled),
				SSLForced:             sourceOf(proxy.SSLForced, gSSLForced),
				BlockExploits:         sourceOf(proxy.BlockExploits, gBlockExploits),
				TLSInsecureSkipVerify: sourceOf(proxy.TLSInsecureSkipVerify, gTLSInsecure),
			},
		},
	}

	// Return success response
	utils.Success(w, resp, "")
}

// createProxyRequest wraps proxy with optional fields for proper tri-state
// handling. These bool fields are pointers so the handler can distinguish
// "omitted" (nil = inherit from the group, or the system default if
// ungrouped) from an explicit true/false (persist it). The model carries no
// GORM `default` tag for them, so nothing here or in the service defaults
// them — proxygroup.Resolve is the only place a default is applied.
// group_id / hostname_label need no such wrapper field: they decode straight
// through the embedded Proxy, since a plain *int/*string has no "omitted vs
// explicit zero value" ambiguity the way a *bool does.
type createProxyRequest struct {
	models.Proxy
	SSLEnabled    *bool `json:"ssl_enabled"`
	SSLForced     *bool `json:"ssl_forced"`
	BlockExploits *bool `json:"block_exploits"`
	IsActive      *bool `json:"is_active"`
}

// CreateProxy handles POST /api/proxies
func (h *ProxyHandler) CreateProxy(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by goauth middleware)
	userIDStr := chimw.UserID(r)
	if userIDStr == "" {
		utils.Unauthorized(w, "User not found in context")
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.Unauthorized(w, "Invalid user ID")
		return
	}

	// Parse request body
	var req createProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	proxy := req.Proxy

	// SSLEnabled / SSLForced / BlockExploits are tri-state: nil means inherit
	// (from the group, or the system default if ungrouped), so pass the
	// client's value straight through instead of defaulting it here.
	proxy.SSLEnabled = req.SSLEnabled
	proxy.SSLForced = req.SSLForced
	proxy.BlockExploits = req.BlockExploits
	proxy.IsActive = true

	// Create proxy via service
	if err := h.service.CreateProxy(&proxy, userID); err != nil {
		if errors.Is(err, service.ErrHostnameConflict) {
			utils.Conflict(w, "Hostname already exists")
			return
		}
		// Check if it's a Caddy error
		if service.IsCaddyError(err) {
			if h.logger != nil {
				h.logger.Error("Caddy error while creating proxy",
					zap.String("hostname", proxy.Hostname),
					zap.Int("user_id", userID),
					zap.Error(err))
			}
			utils.BadGateway(w, err.Error())
			return
		}
		// Validation errors
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	// Log audit event
	if h.auditService != nil {
		_ = h.auditService.LogProxyCreate(r.Context(), userID, &proxy, getClientIP(r), r.UserAgent())
	}

	// Return created proxy
	utils.Created(w, proxy, "Proxy created successfully")
}

// importProxyRequest is the body for POST /api/proxies/import. Each item in
// Proxies is decoded lazily so a single malformed item is reported rather than
// failing the whole request.
type importProxyRequest struct {
	DryRun  bool              `json:"dry_run"`
	Proxies []json.RawMessage `json:"proxies"`
}

const maxImportProxies = 1000

// ImportProxies handles POST /api/proxies/import
func (h *ProxyHandler) ImportProxies(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.Unauthorized(w, "Invalid user ID")
		return
	}

	var req importProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}
	if len(req.Proxies) == 0 {
		utils.BadRequest(w, "No proxies to import", nil)
		return
	}
	if len(req.Proxies) > maxImportProxies {
		utils.BadRequest(w, fmt.Sprintf("Too many proxies to import (max %d)", maxImportProxies), nil)
		return
	}

	// Decode each item the same way create does, capturing per-item decode errors.
	inputs := make([]service.ImportInput, 0, len(req.Proxies))
	for _, raw := range req.Proxies {
		var cpr createProxyRequest
		if err := json.Unmarshal(raw, &cpr); err != nil {
			inputs = append(inputs, service.ImportInput{DecodeError: "invalid item format: " + err.Error()})
			continue
		}
		proxy := cpr.Proxy
		// SSLEnabled / SSLForced / BlockExploits are tri-state: nil means
		// inherit (from the group, or the system default if ungrouped), so
		// pass the imported value straight through instead of defaulting it
		// here — the same tri-state contract as CreateProxy.
		proxy.SSLEnabled = cpr.SSLEnabled
		proxy.SSLForced = cpr.SSLForced
		proxy.BlockExploits = cpr.BlockExploits
		// Preserve is_active from the imported item (exports carry it) so an
		// exported inactive proxy imports inactive; default to active when the
		// field is absent, matching CreateProxy.
		if cpr.IsActive != nil {
			proxy.IsActive = *cpr.IsActive
		} else {
			proxy.IsActive = true
		}
		inputs = append(inputs, service.ImportInput{Proxy: &proxy})
	}

	report := h.service.ImportProxies(inputs, req.DryRun, userID)

	if h.logger != nil {
		h.logger.Info("proxy import processed",
			zap.Bool("dry_run", req.DryRun),
			zap.Int("total", report.Summary.Total),
			zap.Int("created", report.Summary.Created),
			zap.Int("user_id", userID),
		)
	}

	utils.Success(w, report, "Import processed")
}

// updateProxyRequest wraps proxy with optional tri-state fields, mirroring
// createProxyRequest. BlockExploits / TLSInsecureSkipVerify need no such
// wrapper field: they already decode straight through the embedded Proxy with
// the same nil-means-inherit semantics, since nothing here ever preserved
// them from the existing row.
//
// BREAKING (this task): an omitted ssl_enabled / ssl_forced on PUT used to
// mean "keep the existing value" (ssl_enabled) or was simply not settable at
// all (ssl_forced, always force-preserved by the service). Both now mean
// "inherit from the group, or the system default if ungrouped" — nil cannot
// mean both "keep existing" and "inherit", so this had to be a deliberate
// choice. The UI always sends all four booleans explicitly (null for
// inherit), so the wire contract is unambiguous. See docs/API.md.
type updateProxyRequest struct {
	models.Proxy
	SSLEnabled *bool `json:"ssl_enabled"`
	SSLForced  *bool `json:"ssl_forced"`
}

// UpdateProxy handles PUT /api/proxies/:id
func (h *ProxyHandler) UpdateProxy(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	// Parse request body
	var req updateProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Fetch existing proxy for audit change tracking
	existing, err := h.service.GetProxyByID(id)
	if err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		utils.InternalError(w, "Failed to get proxy")
		return
	}

	proxy := req.Proxy

	// SSLEnabled / SSLForced are tri-state: nil means inherit, not "keep
	// existing" — see the BREAKING note on updateProxyRequest.
	proxy.SSLEnabled = req.SSLEnabled
	proxy.SSLForced = req.SSLForced

	// Update proxy via service
	if err := h.service.UpdateProxy(id, &proxy); err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if errors.Is(err, service.ErrHostnameConflict) {
			utils.Conflict(w, "Hostname already exists")
			return
		}
		// Check if it's a Caddy error
		if service.IsCaddyError(err) {
			if h.logger != nil {
				h.logger.Error("Caddy error while updating proxy",
					zap.Int("id", id),
					zap.String("hostname", proxy.Hostname),
					zap.Error(err))
			}
			utils.BadGateway(w, err.Error())
			return
		}
		// Validation errors
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	// Log audit event with change tracking
	if h.auditService != nil && userID > 0 {
		changes := buildProxyChanges(existing, &proxy)
		_ = h.auditService.LogProxyUpdate(r.Context(), userID, &proxy, changes, getClientIP(r), r.UserAgent())
	}

	// Return updated proxy
	utils.Success(w, proxy, "Proxy updated successfully")
}

// DeleteProxy handles DELETE /api/proxies/:id
func (h *ProxyHandler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	// Get proxy info for audit log before deletion
	var proxyName, proxyHostname string
	if h.auditService != nil {
		if proxy, err := h.service.GetProxyByID(id); err == nil {
			proxyName = proxy.Name
			proxyHostname = proxy.Hostname
		}
	}

	// Delete proxy via service
	if err := h.service.DeleteProxy(id); err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to delete proxy",
				zap.Int("id", id),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to delete proxy")
		return
	}

	// Log audit event
	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogProxyDelete(r.Context(), userID, id, proxyName, proxyHostname, getClientIP(r), r.UserAgent())
	}

	// Return success response
	utils.Success(w, nil, "Proxy deleted successfully")
}

// EnableProxy handles POST /api/proxies/:id/enable
func (h *ProxyHandler) EnableProxy(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	// Enable proxy via service
	if err := h.service.EnableProxy(id); err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if errors.Is(err, service.ErrProxyAlreadyEnabled) {
			utils.BadRequest(w, "Proxy is already enabled", nil)
			return
		}
		// Check if it's a Caddy error
		if service.IsCaddyError(err) {
			if h.logger != nil {
				h.logger.Error("Caddy error while enabling proxy",
					zap.Int("id", id),
					zap.Error(err))
			}
			utils.BadGateway(w, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to enable proxy",
				zap.Int("id", id),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to enable proxy")
		return
	}

	// Log audit event
	if h.auditService != nil && userID > 0 {
		if proxy, err := h.service.GetProxyByID(id); err == nil {
			_ = h.auditService.LogProxyEnable(r.Context(), userID, proxy, getClientIP(r), r.UserAgent())
		}
	}

	// Return success response
	utils.Success(w, nil, "Proxy enabled successfully")
}

// DisableProxy handles POST /api/proxies/:id/disable
func (h *ProxyHandler) DisableProxy(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	// Disable proxy via service
	if err := h.service.DisableProxy(id); err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if errors.Is(err, service.ErrProxyAlreadyDisabled) {
			utils.BadRequest(w, "Proxy is already disabled", nil)
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to disable proxy",
				zap.Int("id", id),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to disable proxy")
		return
	}

	// Log audit event
	if h.auditService != nil && userID > 0 {
		if proxy, err := h.service.GetProxyByID(id); err == nil {
			_ = h.auditService.LogProxyDisable(r.Context(), userID, proxy, getClientIP(r), r.UserAgent())
		}
	}

	// Return success response
	utils.Success(w, nil, "Proxy disabled successfully")
}

// maxBulkProxyIDs caps the number of ids accepted in a single bulk request.
const maxBulkProxyIDs = 1000

// bulkProxyRequest is the request body shared by the bulk endpoints.
type bulkProxyRequest struct {
	IDs []int `json:"ids"`
}

// decodeBulkProxyIDs decodes and validates the bulk request body, writing the
// appropriate error response and returning ok=false when invalid.
func (h *ProxyHandler) decodeBulkProxyIDs(w http.ResponseWriter, r *http.Request) (ids []int, ok bool) {
	var req bulkProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return nil, false
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(w, "ids must not be empty", nil)
		return nil, false
	}
	if len(req.IDs) > maxBulkProxyIDs {
		utils.BadRequest(w, "too many ids: maximum is 1000 per request", nil)
		return nil, false
	}
	return req.IDs, true
}

// BulkEnableProxies handles POST /api/proxies/bulk/enable
func (h *ProxyHandler) BulkEnableProxies(w http.ResponseWriter, r *http.Request) {
	ids, ok := h.decodeBulkProxyIDs(w, r)
	if !ok {
		return
	}

	report := h.service.BulkSetActive(ids, true)

	if h.logger != nil {
		h.logger.Info("bulk enable proxies",
			zap.Int("requested", report.Requested),
			zap.Int("succeeded", report.Succeeded),
			zap.Int("failed", report.Failed))
	}

	utils.Success(w, report, "Bulk enable completed")
}

// BulkDisableProxies handles POST /api/proxies/bulk/disable
func (h *ProxyHandler) BulkDisableProxies(w http.ResponseWriter, r *http.Request) {
	ids, ok := h.decodeBulkProxyIDs(w, r)
	if !ok {
		return
	}

	report := h.service.BulkSetActive(ids, false)

	if h.logger != nil {
		h.logger.Info("bulk disable proxies",
			zap.Int("requested", report.Requested),
			zap.Int("succeeded", report.Succeeded),
			zap.Int("failed", report.Failed))
	}

	utils.Success(w, report, "Bulk disable completed")
}

// BulkDeleteProxies handles POST /api/proxies/bulk/delete
func (h *ProxyHandler) BulkDeleteProxies(w http.ResponseWriter, r *http.Request) {
	ids, ok := h.decodeBulkProxyIDs(w, r)
	if !ok {
		return
	}

	report := h.service.BulkDelete(ids)

	if h.logger != nil {
		h.logger.Info("bulk delete proxies",
			zap.Int("requested", report.Requested),
			zap.Int("succeeded", report.Succeeded),
			zap.Int("failed", report.Failed))
	}

	utils.Success(w, report, "Bulk delete completed")
}

// parseExportIDs parses the optional comma-separated `ids` query parameter
// (e.g. ?ids=1,2,3) into a slice of ints, writing a 400 response and returning
// ok=false on a malformed value. An empty/absent parameter yields a nil slice
// with ok=true (meaning "export all matching filters").
func parseExportIDs(w http.ResponseWriter, r *http.Request) (ids []int, ok bool) {
	idsParam := strings.TrimSpace(r.URL.Query().Get("ids"))
	if idsParam == "" {
		return nil, true
	}
	for _, part := range strings.Split(idsParam, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			utils.BadRequest(w, "Invalid ids parameter: must be a comma-separated list of integers", nil)
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

// ExportProxies handles GET /api/proxies/export. When `ids` is provided it
// exports those proxies (skipping missing ones); otherwise it exports all
// proxies matching the same filters as ListProxies (search/type/status/ssl_enabled).
// The response data is an array of export objects suitable for re-import.
func (h *ProxyHandler) ExportProxies(w http.ResponseWriter, r *http.Request) {
	ids, ok := parseExportIDs(w, r)
	if !ok {
		return
	}

	var req service.ListProxiesRequest
	req.Search = r.URL.Query().Get("search")

	// Parse type filter (supports operator:value format)
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		fv := parseFilterParam(typeParam)
		switch fv.Operator {
		case OpIn, OpEq:
			if len(fv.Values) > 0 {
				req.Types = fv.Values
			} else {
				req.Types = splitAndTrim(fv.Value)
			}
		case OpNotIn, OpNot:
			if len(fv.Values) > 0 {
				req.TypesExclude = fv.Values
			} else {
				req.TypesExclude = splitAndTrim(fv.Value)
			}
		default:
			utils.BadRequest(w, "Invalid operator for type filter", nil)
			return
		}
	}

	// Parse status filter (supports operator:value format)
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		fv := parseFilterParam(statusParam)
		statusVal := fv.Value
		if statusVal != "active" && statusVal != "inactive" {
			utils.BadRequest(w, "Invalid status parameter: must be 'active' or 'inactive'", nil)
			return
		}
		switch fv.Operator {
		case OpEq:
			req.Status = statusVal
		case OpNot:
			req.StatusNot = statusVal
		default:
			utils.BadRequest(w, "Invalid operator for status filter", nil)
			return
		}
	}

	// Parse ssl_enabled filter
	if sslParam := r.URL.Query().Get("ssl_enabled"); sslParam != "" {
		switch sslParam {
		case "true":
			sslEnabled := true
			req.SSLEnabled = &sslEnabled
		case "false":
			sslEnabled := false
			req.SSLEnabled = &sslEnabled
		default:
			utils.BadRequest(w, "Invalid ssl_enabled parameter: must be 'true' or 'false'", nil)
			return
		}
	}

	exports, err := h.service.ExportProxies(ids, req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to export proxies",
				zap.Int("requested_ids", len(ids)),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to export proxies")
		return
	}

	utils.Success(w, exports, "")
}

// GetStats handles GET /api/proxies/stats
func (h *ProxyHandler) GetStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := h.service.GetStats()
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get proxy stats", zap.Error(err))
		}
		utils.InternalError(w, "Failed to get proxy stats")
		return
	}

	utils.Success(w, stats, "")
}

// ptrChanged reports whether two pointers to a comparable type differ,
// distinguishing nil from an explicit zero value. A plain dereferenced
// comparison (treating nil as the zero value) would hide, for example, a
// transition from "inherit" (nil) to an explicit "off" (false) — exactly the
// divergence proxygroup.Resolve exists to prevent, silently dropped from the
// audit log. Used for every tri-state (nil = inherit) field.
func ptrChanged[T comparable](old, updated *T) bool {
	if old == nil && updated == nil {
		return false
	}
	if old == nil || updated == nil {
		return true
	}
	return *old != *updated
}

// buildProxyChanges compares old and new proxy values and returns a map of changes.
// Each changed field is represented as {"old": oldValue, "new": newValue}.
// Returns nil if no tracked fields changed.
func buildProxyChanges(old, updated *models.Proxy) map[string]interface{} {
	changes := make(map[string]interface{})

	// Track hostname changes
	if old.Hostname != updated.Hostname {
		changes["hostname"] = map[string]interface{}{
			"old": old.Hostname,
			"new": updated.Hostname,
		}
	}

	// Track type changes
	if old.Type != updated.Type {
		changes["type"] = map[string]interface{}{
			"old": old.Type,
			"new": updated.Type,
		}
	}

	// Track ssl_enabled changes. old/new are stored as the raw *bool (not
	// dereferenced), so the audit JSON records the tri-state honestly: `null`
	// for inherit, `true`/`false` for an explicit value.
	if ptrChanged(old.SSLEnabled, updated.SSLEnabled) {
		changes["ssl_enabled"] = map[string]interface{}{
			"old": old.SSLEnabled,
			"new": updated.SSLEnabled,
		}
	}

	// Track ssl_forced changes (settable since this task; previously always
	// force-preserved by the service, so never actually changed via the API).
	if ptrChanged(old.SSLForced, updated.SSLForced) {
		changes["ssl_forced"] = map[string]interface{}{
			"old": old.SSLForced,
			"new": updated.SSLForced,
		}
	}

	// Track group membership changes
	if ptrChanged(old.GroupID, updated.GroupID) {
		changes["group_id"] = map[string]interface{}{
			"old": old.GroupID,
			"new": updated.GroupID,
		}
	}

	// Track hostname_label changes
	if ptrChanged(old.HostnameLabel, updated.HostnameLabel) {
		changes["hostname_label"] = map[string]interface{}{
			"old": old.HostnameLabel,
			"new": updated.HostnameLabel,
		}
	}

	// Track is_active changes
	if old.IsActive != updated.IsActive {
		changes["is_active"] = map[string]interface{}{
			"old": old.IsActive,
			"new": updated.IsActive,
		}
	}

	// Track name changes
	if old.Name != updated.Name {
		changes["name"] = map[string]interface{}{
			"old": old.Name,
			"new": updated.Name,
		}
	}

	// Track upstreams changes (compare JSON representation)
	if !jsonEqual(old.Upstreams, updated.Upstreams) {
		changes["upstreams"] = map[string]interface{}{
			"old": old.Upstreams,
			"new": updated.Upstreams,
		}
	}

	// Track redirect config changes
	if !jsonEqual(old.RedirectConfig, updated.RedirectConfig) {
		changes["redirect"] = map[string]interface{}{
			"old": old.RedirectConfig,
			"new": updated.RedirectConfig,
		}
	}

	// Track description changes
	if !jsonEqual(old.Description, updated.Description) {
		changes["description"] = map[string]interface{}{
			"old": old.Description,
			"new": updated.Description,
		}
	}

	// Track block_exploits changes
	if ptrChanged(old.BlockExploits, updated.BlockExploits) {
		changes["block_exploits"] = map[string]interface{}{
			"old": old.BlockExploits,
			"new": updated.BlockExploits,
		}
	}

	// Track tls_insecure_skip_verify changes
	if ptrChanged(old.TLSInsecureSkipVerify, updated.TLSInsecureSkipVerify) {
		changes["tls_insecure_skip_verify"] = map[string]interface{}{
			"old": old.TLSInsecureSkipVerify,
			"new": updated.TLSInsecureSkipVerify,
		}
	}

	// Track load_balancing changes (strategy + health check live here)
	if !jsonEqual(old.LoadBalancing, updated.LoadBalancing) {
		changes["load_balancing"] = map[string]interface{}{
			"old": old.LoadBalancing,
			"new": updated.LoadBalancing,
		}
	}

	// Track custom_headers changes
	if !jsonEqual(old.CustomHeaders, updated.CustomHeaders) {
		changes["custom_headers"] = map[string]interface{}{
			"old": old.CustomHeaders,
			"new": updated.CustomHeaders,
		}
	}

	// Track static config changes
	if !jsonEqual(old.StaticConfig, updated.StaticConfig) {
		changes["static"] = map[string]interface{}{
			"old": old.StaticConfig,
			"new": updated.StaticConfig,
		}
	}

	if len(changes) == 0 {
		return nil
	}

	return changes
}

// jsonEqual compares two interface values by their JSON representation.
// This handles comparison of complex types like slices, maps, and JSONField.
func jsonEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Marshal both values to JSON for comparison
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)

	if errA != nil || errB != nil {
		// If marshaling fails, fall back to direct comparison
		return false
	}

	return bytes.Equal(aJSON, bJSON)
}
