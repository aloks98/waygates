package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ProxyHandler handles proxy-related HTTP requests
type ProxyHandler struct {
	service service.ProxyServiceInterface
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(svc service.ProxyServiceInterface) *ProxyHandler {
	return &ProxyHandler{
		service: svc,
	}
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
	proxyType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	// Validate status if provided
	if status != "" && status != "active" && status != "inactive" {
		utils.BadRequest(w, "Invalid status parameter: must be 'active' or 'inactive'", nil)
		return
	}

	// Create request
	req := service.ListProxiesRequest{
		Page:   page,
		Limit:  limit,
		Search: search,
		Type:   proxyType,
		Status: status,
		Sort:   sort,
		Order:  order,
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

	proxy := req.Proxy

	// Handle ssl_enabled - if explicitly provided, use it; otherwise keep existing value
	if req.SSLEnabled != nil {
		proxy.SSLEnabled = *req.SSLEnabled
	} else {
		// Fetch existing proxy to preserve ssl_enabled
		existing, err := h.service.GetProxyByID(id)
		if err != nil {
			if errors.Is(err, service.ErrProxyNotFound) {
				utils.NotFound(w, "Proxy not found")
				return
			}
			utils.InternalError(w, "Failed to get proxy")
			return
		}
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

	// Return updated proxy
	utils.Success(w, proxy, "Proxy updated successfully")
}

// DeleteProxy handles DELETE /api/proxies/:id
func (h *ProxyHandler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
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

	// Return success response
	utils.Success(w, nil, "Proxy deleted successfully")
}

// EnableProxy handles POST /api/proxies/:id/enable
func (h *ProxyHandler) EnableProxy(w http.ResponseWriter, r *http.Request) {
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

	// Return success response
	utils.Success(w, nil, "Proxy enabled successfully")
}

// DisableProxy handles POST /api/proxies/:id/disable
func (h *ProxyHandler) DisableProxy(w http.ResponseWriter, r *http.Request) {
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
