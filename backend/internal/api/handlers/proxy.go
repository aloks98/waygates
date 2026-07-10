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
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ProxyHandler handles proxy-related HTTP requests
type ProxyHandler struct {
	service      service.ProxyServiceInterface
	auditService service.AuditServiceInterface
	logger       *zap.Logger
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(svc service.ProxyServiceInterface, auditService service.AuditServiceInterface, logger *zap.Logger) *ProxyHandler {
	return &ProxyHandler{
		service:      svc,
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

	// Return success response
	utils.Success(w, proxy, "")
}

// createProxyRequest wraps proxy with optional fields for proper default handling.
// These bool fields are pointers so the handler can distinguish "omitted" (apply
// the secure default) from an explicit false (persist it). The model no longer
// carries GORM `default` tags for them, so the handler owns the defaulting.
type createProxyRequest struct {
	models.Proxy
	SSLEnabled    *bool `json:"ssl_enabled"`
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

	// SSLEnabled / BlockExploits are tri-state: nil means inherit (from group,
	// or the system default if ungrouped), so pass the client's value straight
	// through instead of defaulting it here.
	proxy.SSLEnabled = req.SSLEnabled
	proxy.BlockExploits = req.BlockExploits
	proxy.SSLForced = nil
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
		// SSLEnabled / BlockExploits are tri-state: nil means inherit (from
		// group, or the system default if ungrouped), so pass the imported
		// value straight through instead of defaulting it here.
		proxy.SSLEnabled = cpr.SSLEnabled
		proxy.BlockExploits = cpr.BlockExploits
		proxy.SSLForced = nil
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

// updateProxyRequest wraps proxy with optional fields for proper handling
type updateProxyRequest struct {
	models.Proxy
	SSLEnabled *bool `json:"ssl_enabled"`
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

	// Fetch existing proxy for change tracking and ssl_enabled preservation
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

	// SSLEnabled is tri-state: nil now means inherit, not "keep existing".
	// TODO(task-6): revisit update semantics — an omitted field previously
	// preserved the existing value; it now means inherit-from-group.
	proxy.SSLEnabled = req.SSLEnabled

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

// derefBool reports the pointed-to value, treating a nil *bool as false. This
// is a transitional shim for audit diffing only — it does NOT resolve
// inheritance (nil means "inherit", not "false"). Task 3 removes the
// equivalent helper in caddy/config once the builder is retyped to accept
// only resolved proxies.
func derefBool(b *bool) bool { return b != nil && *b }

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

	// Track ssl_enabled changes
	if derefBool(old.SSLEnabled) != derefBool(updated.SSLEnabled) {
		changes["ssl_enabled"] = map[string]interface{}{
			"old": old.SSLEnabled,
			"new": updated.SSLEnabled,
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
	if derefBool(old.BlockExploits) != derefBool(updated.BlockExploits) {
		changes["block_exploits"] = map[string]interface{}{
			"old": old.BlockExploits,
			"new": updated.BlockExploits,
		}
	}

	// Track tls_insecure_skip_verify changes
	if derefBool(old.TLSInsecureSkipVerify) != derefBool(updated.TLSInsecureSkipVerify) {
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
