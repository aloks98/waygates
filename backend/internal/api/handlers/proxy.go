package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ProxyHandler handles proxy-related HTTP requests
type ProxyHandler struct {
	service      service.ProxyServiceInterface
	auditService service.AuditServiceInterface
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(svc service.ProxyServiceInterface, auditService service.AuditServiceInterface) *ProxyHandler {
	return &ProxyHandler{
		service:      svc,
		auditService: auditService,
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
		utils.InternalError(w, "Failed to get proxy")
		return
	}

	// Return success response
	utils.Success(w, proxy, "")
}

// createProxyRequest wraps proxy with optional fields for proper default handling
type createProxyRequest struct {
	models.Proxy
	SSLEnabled *bool `json:"ssl_enabled"`
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

	// Set defaults - SSLEnabled defaults to true unless explicitly set to false
	if req.SSLEnabled != nil {
		proxy.SSLEnabled = *req.SSLEnabled
	} else {
		proxy.SSLEnabled = true
	}
	proxy.SSLForced = true
	proxy.IsActive = true

	// Create proxy via service
	if err := h.service.CreateProxy(&proxy, userID); err != nil {
		if errors.Is(err, service.ErrHostnameConflict) {
			utils.Conflict(w, "Hostname already exists")
			return
		}
		// Check if it's a Caddy error
		if service.IsCaddyError(err) {
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

	// Handle ssl_enabled - if explicitly provided, use it; otherwise keep existing value
	if req.SSLEnabled != nil {
		proxy.SSLEnabled = *req.SSLEnabled
	} else {
		proxy.SSLEnabled = existing.SSLEnabled
	}

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
			utils.BadGateway(w, err.Error())
			return
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

// GetStats handles GET /api/proxies/stats
func (h *ProxyHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats()
	if err != nil {
		utils.InternalError(w, "Failed to get proxy stats")
		return
	}

	utils.Success(w, stats, "")
}

// buildProxyChanges compares old and new proxy values and returns a map of changes.
// Each changed field is represented as {"old": oldValue, "new": newValue}.
// Returns nil if no tracked fields changed.
func buildProxyChanges(old, new *models.Proxy) map[string]interface{} {
	changes := make(map[string]interface{})

	// Track hostname changes
	if old.Hostname != new.Hostname {
		changes["hostname"] = map[string]interface{}{
			"old": old.Hostname,
			"new": new.Hostname,
		}
	}

	// Track type changes
	if old.Type != new.Type {
		changes["type"] = map[string]interface{}{
			"old": old.Type,
			"new": new.Type,
		}
	}

	// Track ssl_enabled changes
	if old.SSLEnabled != new.SSLEnabled {
		changes["ssl_enabled"] = map[string]interface{}{
			"old": old.SSLEnabled,
			"new": new.SSLEnabled,
		}
	}

	// Track is_active changes
	if old.IsActive != new.IsActive {
		changes["is_active"] = map[string]interface{}{
			"old": old.IsActive,
			"new": new.IsActive,
		}
	}

	// Track name changes
	if old.Name != new.Name {
		changes["name"] = map[string]interface{}{
			"old": old.Name,
			"new": new.Name,
		}
	}

	// Track upstreams changes (compare JSON representation)
	if !jsonEqual(old.Upstreams, new.Upstreams) {
		changes["upstreams"] = map[string]interface{}{
			"old": old.Upstreams,
			"new": new.Upstreams,
		}
	}

	// Track redirect config changes
	if !jsonEqual(old.RedirectConfig, new.RedirectConfig) {
		changes["redirect"] = map[string]interface{}{
			"old": old.RedirectConfig,
			"new": new.RedirectConfig,
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

	return string(aJSON) == string(bJSON)
}
