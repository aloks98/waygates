package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
	"github.com/aloks98/waygates/backend/internal/validation"
)

// L4ProxyHandler handles L4 proxy-related HTTP requests
type L4ProxyHandler struct {
	service service.L4ProxyServiceInterface
	logger  *zap.Logger
}

// NewL4ProxyHandler creates a new L4 proxy handler
func NewL4ProxyHandler(svc service.L4ProxyServiceInterface, logger *zap.Logger) *L4ProxyHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &L4ProxyHandler{
		service: svc,
		logger:  logger.Named("l4-proxy-handler"),
	}
}

// List handles GET /api/l4-proxies
func (h *L4ProxyHandler) List(w http.ResponseWriter, r *http.Request) {
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
	protocol := r.URL.Query().Get("protocol")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	// Validate protocol if provided
	if protocol != "" && protocol != "tcp" && protocol != "udp" {
		utils.BadRequest(w, "Invalid protocol parameter: must be 'tcp' or 'udp'", nil)
		return
	}

	// Parse is_active filter
	var isActive *bool
	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		switch isActiveStr {
		case "true":
			active := true
			isActive = &active
		case "false":
			active := false
			isActive = &active
		default:
			utils.BadRequest(w, "Invalid is_active parameter: must be 'true' or 'false'", nil)
			return
		}
	}

	// Create request
	req := &service.ListL4ProxiesRequest{
		Page:     page,
		Limit:    limit,
		Search:   search,
		Protocol: protocol,
		IsActive: isActive,
		Sort:     sort,
		Order:    order,
	}

	// Get L4 proxies from service
	result, err := h.service.List(req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to list L4 proxies",
				zap.Int("page", req.Page),
				zap.Int("limit", req.Limit),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to list L4 proxies")
		return
	}

	// Return success response
	utils.Success(w, result, "")
}

// Create handles POST /api/l4-proxies
func (h *L4ProxyHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	var req validation.CreateL4ProxyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode create l4 proxy request",
			zap.Int("user_id", userID),
			zap.Error(err))
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	h.logger.Debug("received create l4 proxy request",
		zap.String("name", req.Name),
		zap.Int("listen_port", req.ListenPort),
		zap.String("protocol", req.Protocol),
		zap.Int("route_count", len(req.Routes)),
		zap.Int("user_id", userID))

	// Validate the request
	if err := validation.ValidateStruct(&req); err != nil {
		h.logger.Debug("l4 proxy request validation failed",
			zap.String("name", req.Name),
			zap.Error(err))
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	// Convert DTO to service request
	serviceReq := dtoToCreateL4ProxyRequest(&req)

	// Create L4 proxy via service
	proxy, err := h.service.Create(serviceReq, userID)
	if err != nil {
		if errors.Is(err, service.ErrL4ProxyPortConflict) {
			h.logger.Info("l4 proxy creation failed: port conflict",
				zap.String("name", req.Name),
				zap.Int("listen_port", req.ListenPort),
				zap.String("protocol", req.Protocol))
			utils.Conflict(w, "Port and protocol combination already in use")
			return
		}
		// Validation errors from the model
		h.logger.Debug("l4 proxy creation failed",
			zap.String("name", req.Name),
			zap.Error(err))
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	h.logger.Info("l4 proxy created via API",
		zap.Int("id", proxy.ID),
		zap.String("name", proxy.Name),
		zap.Int("listen_port", proxy.ListenPort),
		zap.String("protocol", proxy.Protocol),
		zap.Int("user_id", userID))

	// Return created proxy
	utils.Created(w, proxy, "L4 proxy created successfully")
}

// Get handles GET /api/l4-proxies/{id}
func (h *L4ProxyHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid L4 proxy ID", nil)
		return
	}

	// Get L4 proxy from service
	proxy, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrL4ProxyNotFound) {
			utils.NotFound(w, "L4 proxy not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to get L4 proxy",
				zap.Int("id", id),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to get L4 proxy")
		return
	}

	// Return success response
	utils.Success(w, proxy, "")
}

// Update handles PUT /api/l4-proxies/{id}
func (h *L4ProxyHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid L4 proxy ID", nil)
		return
	}

	// Parse request body
	var req validation.UpdateL4ProxyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode update l4 proxy request",
			zap.Int("id", id),
			zap.Error(err))
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	h.logger.Debug("received update l4 proxy request",
		zap.Int("id", id),
		zap.String("name", req.Name),
		zap.Int("listen_port", req.ListenPort),
		zap.String("protocol", req.Protocol),
		zap.Int("route_count", len(req.Routes)))

	// Validate the request
	if err := validation.ValidateStruct(&req); err != nil {
		h.logger.Debug("l4 proxy update request validation failed",
			zap.Int("id", id),
			zap.Error(err))
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	// Convert DTO to service request
	serviceReq := dtoToUpdateL4ProxyRequest(&req)

	// Update L4 proxy via service
	proxy, err := h.service.Update(id, serviceReq)
	if err != nil {
		if errors.Is(err, service.ErrL4ProxyNotFound) {
			h.logger.Debug("l4 proxy update failed: not found", zap.Int("id", id))
			utils.NotFound(w, "L4 proxy not found")
			return
		}
		if errors.Is(err, service.ErrL4ProxyPortConflict) {
			h.logger.Info("l4 proxy update failed: port conflict",
				zap.Int("id", id),
				zap.Int("listen_port", req.ListenPort),
				zap.String("protocol", req.Protocol))
			utils.Conflict(w, "Port and protocol combination already in use")
			return
		}
		// Validation errors from the model
		h.logger.Debug("l4 proxy update failed",
			zap.Int("id", id),
			zap.Error(err))
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	h.logger.Info("l4 proxy updated via API",
		zap.Int("id", proxy.ID),
		zap.String("name", proxy.Name),
		zap.Int("listen_port", proxy.ListenPort),
		zap.String("protocol", proxy.Protocol))

	// Return updated proxy
	utils.Success(w, proxy, "L4 proxy updated successfully")
}

// Delete handles DELETE /api/l4-proxies/{id}
func (h *L4ProxyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid L4 proxy ID", nil)
		return
	}

	// Delete L4 proxy via service
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, service.ErrL4ProxyNotFound) {
			utils.NotFound(w, "L4 proxy not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to delete L4 proxy",
				zap.Int("id", id),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to delete L4 proxy")
		return
	}

	// Return success response
	utils.Success(w, nil, "L4 proxy deleted successfully")
}

// ToggleActive handles PATCH /api/l4-proxies/{id}/toggle
func (h *L4ProxyHandler) ToggleActive(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid L4 proxy ID", nil)
		return
	}

	// Toggle L4 proxy active status via service
	proxy, err := h.service.ToggleActive(id)
	if err != nil {
		if errors.Is(err, service.ErrL4ProxyNotFound) {
			utils.NotFound(w, "L4 proxy not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to toggle L4 proxy status",
				zap.Int("id", id),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to toggle L4 proxy status")
		return
	}

	// Return success response with the updated proxy
	var message string
	if proxy.IsActive {
		message = "L4 proxy enabled successfully"
	} else {
		message = "L4 proxy disabled successfully"
	}
	utils.Success(w, proxy, message)
}

// GetStats handles GET /api/l4-proxies/stats
func (h *L4ProxyHandler) GetStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := h.service.GetStats()
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get L4 proxy stats", zap.Error(err))
		}
		utils.InternalError(w, "Failed to get L4 proxy stats")
		return
	}

	utils.Success(w, stats, "")
}

// dtoToCreateL4ProxyRequest converts a CreateL4ProxyRequestDTO to a service.CreateL4ProxyRequest
func dtoToCreateL4ProxyRequest(dto *validation.CreateL4ProxyRequestDTO) *service.CreateL4ProxyRequest {
	// Default is_active to true if not provided
	isActive := true
	if dto.IsActive != nil {
		isActive = *dto.IsActive
	}

	req := &service.CreateL4ProxyRequest{
		Name:        dto.Name,
		Description: dto.Description,
		ListenPort:  dto.ListenPort,
		Protocol:    dto.Protocol,
		IsActive:    isActive,
		Routes:      make([]service.CreateL4RouteRequest, 0, len(dto.Routes)),
	}

	for i := range dto.Routes {
		req.Routes = append(req.Routes, dtoToCreateL4RouteRequest(&dto.Routes[i]))
	}

	return req
}

// dtoToUpdateL4ProxyRequest converts an UpdateL4ProxyRequestDTO to a service.UpdateL4ProxyRequest
func dtoToUpdateL4ProxyRequest(dto *validation.UpdateL4ProxyRequestDTO) *service.UpdateL4ProxyRequest {
	req := &service.UpdateL4ProxyRequest{
		Name:        dto.Name,
		Description: dto.Description,
		ListenPort:  dto.ListenPort,
		Protocol:    dto.Protocol,
		IsActive:    dto.IsActive,
		Routes:      make([]service.CreateL4RouteRequest, 0, len(dto.Routes)),
	}

	for i := range dto.Routes {
		req.Routes = append(req.Routes, dtoToCreateL4RouteRequest(&dto.Routes[i]))
	}

	return req
}

// dtoToCreateL4RouteRequest converts a CreateL4RouteRequestDTO to a service.CreateL4RouteRequest
func dtoToCreateL4RouteRequest(dto *validation.CreateL4RouteRequestDTO) service.CreateL4RouteRequest {
	upstreams := make([]service.L4UpstreamRequest, 0, len(dto.Upstreams))
	for i := range dto.Upstreams {
		upstreams = append(upstreams, service.L4UpstreamRequest{
			Host:   dto.Upstreams[i].Host,
			Port:   dto.Upstreams[i].Port,
			Weight: dto.Upstreams[i].Weight,
		})
	}

	return service.CreateL4RouteRequest{
		Priority:             dto.Priority,
		MatcherType:          dto.MatcherType,
		SNIHostnames:         dto.SNIHostnames,
		AllowedIPRanges:      dto.AllowedIPRanges,
		RegexPattern:         dto.RegexPattern,
		Upstreams:            upstreams,
		LoadBalancingPolicy:  dto.LoadBalancingPolicy,
		TLSTerminate:         dto.TLSTerminate,
		TLSPassthrough:       dto.TLSPassthrough,
		ProxyProtocolVersion: dto.ProxyProtocolVersion,
	}
}
