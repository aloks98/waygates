package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aloks98/homelab-proxy/backend/internal/models"
	"github.com/aloks98/homelab-proxy/backend/internal/service"
	"github.com/aloks98/homelab-proxy/backend/internal/utils"
	"github.com/go-chi/chi/v5"
)

// ProxyHandler handles proxy-related HTTP requests
type ProxyHandler struct {
	service *service.ProxyService
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(service *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{
		service: service,
	}
}

// ListProxies handles GET /api/proxies
func (h *ProxyHandler) ListProxies(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")
	proxyType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

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
		if err == service.ErrProxyNotFound {
			utils.NotFound(w, "Proxy not found")
			return
		}
		utils.InternalError(w, "Failed to get proxy")
		return
	}

	// Return success response
	utils.Success(w, proxy, "")
}

// CreateProxy handles POST /api/proxies
func (h *ProxyHandler) CreateProxy(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var proxy models.Proxy
	if err := json.NewDecoder(r.Body).Decode(&proxy); err != nil {
		utils.BadRequest(w, "Invalid request body", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Set defaults
	proxy.SSLEnabled = true
	proxy.SSLForced = true
	proxy.IsActive = true

	// Create proxy via service
	if err := h.service.CreateProxy(&proxy); err != nil {
		if err == service.ErrHostnameConflict {
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
	var proxy models.Proxy
	if err := json.NewDecoder(r.Body).Decode(&proxy); err != nil {
		utils.BadRequest(w, "Invalid request body", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Update proxy via service
	if err := h.service.UpdateProxy(id, &proxy); err != nil {
		if err == service.ErrProxyNotFound {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if err == service.ErrHostnameConflict {
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
		if err == service.ErrProxyNotFound {
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
		if err == service.ErrProxyNotFound {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if err == service.ErrProxyAlreadyEnabled {
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
		if err == service.ErrProxyNotFound {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if err == service.ErrProxyAlreadyDisabled {
			utils.BadRequest(w, "Proxy is already disabled", nil)
			return
		}
		utils.InternalError(w, "Failed to disable proxy")
		return
	}

	// Return success response
	utils.Success(w, nil, "Proxy disabled successfully")
}
