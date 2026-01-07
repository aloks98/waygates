package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ProxyACLHandler handles proxy ACL assignment HTTP requests
type ProxyACLHandler struct {
	aclService   service.ACLServiceInterface
	syncService  service.SyncServiceInterface
	auditService service.AuditServiceInterface
	logger       *zap.Logger
}

// NewProxyACLHandler creates a new proxy ACL handler
func NewProxyACLHandler(aclService service.ACLServiceInterface, syncService service.SyncServiceInterface, auditService service.AuditServiceInterface, logger *zap.Logger) *ProxyACLHandler {
	return &ProxyACLHandler{
		aclService:   aclService,
		syncService:  syncService,
		auditService: auditService,
		logger:       logger,
	}
}

// =============================================================================
// Request/Response Types
// =============================================================================

// AssignACLRequest is the request body for assigning an ACL group to a proxy
type AssignACLRequest struct {
	ACLGroupID  int    `json:"acl_group_id"`
	PathPattern string `json:"path_pattern,omitempty"`
	Priority    int    `json:"priority"`
}

// UpdateProxyACLRequest is the request body for updating a proxy ACL assignment
type UpdateProxyACLRequest struct {
	PathPattern string `json:"path_pattern,omitempty"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// =============================================================================
// Handlers
// =============================================================================

// GetProxyACL handles GET /api/proxies/:id/acl
// Returns all ACL assignments for a proxy
func (h *ProxyACLHandler) GetProxyACL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	proxyID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	assignments, err := h.aclService.GetProxyACL(proxyID)
	if err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to get proxy ACL assignments",
				zap.Int("proxy_id", proxyID),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to get proxy ACL assignments")
		return
	}

	utils.Success(w, assignments, "")
}

// AssignACLToProxy handles POST /api/proxies/:id/acl
// Assigns an ACL group to a proxy
func (h *ProxyACLHandler) AssignACLToProxy(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	proxyID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	// Parse request body
	var req AssignACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.ACLGroupID <= 0 {
		utils.BadRequest(w, "acl_group_id is required and must be positive", nil)
		return
	}

	// Set default path pattern
	pathPattern := req.PathPattern
	if pathPattern == "" {
		pathPattern = "/*"
	}

	// Assign ACL group to proxy
	if err := h.aclService.AssignToProxy(proxyID, req.ACLGroupID, pathPattern, req.Priority); err != nil {
		if errors.Is(err, service.ErrProxyNotFound) {
			utils.NotFound(w, "Proxy not found")
			return
		}
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		if errors.Is(err, service.ErrProxyACLExists) {
			utils.Conflict(w, "ACL assignment already exists for this path pattern")
			return
		}
		if errors.Is(err, service.ErrInvalidPathPattern) {
			utils.BadRequest(w, "Invalid path pattern", nil)
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to assign ACL to proxy",
				zap.Int("proxy_id", proxyID),
				zap.Int("acl_group_id", req.ACLGroupID),
				zap.String("path_pattern", pathPattern),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to assign ACL to proxy")
		return
	}

	// Get updated assignments to return
	assignments, err := h.aclService.GetProxyACL(proxyID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get updated ACL assignments after assign",
				zap.Int("proxy_id", proxyID),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to get updated ACL assignments")
		return
	}

	// Trigger Caddy sync to update the proxy's Caddyfile with ACL config
	if h.syncService != nil {
		if err := h.syncService.SyncProxyByID(proxyID); err != nil {
			if h.logger != nil {
				h.logger.Warn("Failed to sync proxy after ACL assignment",
					zap.Int("proxy_id", proxyID),
					zap.Error(err))
			}
			// Don't fail the request, the ACL was assigned successfully
		}
	}

	utils.Created(w, assignments, "ACL assigned to proxy successfully")
}

// UpdateProxyACLAssignment handles PUT /api/proxies/:id/acl/:assignmentId
// Updates a specific ACL assignment for a proxy
func (h *ProxyACLHandler) UpdateProxyACLAssignment(w http.ResponseWriter, r *http.Request) {
	proxyIDStr := chi.URLParam(r, "id")
	proxyID, err := strconv.Atoi(proxyIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	assignmentIDStr := chi.URLParam(r, "assignmentId")
	assignmentID, err := strconv.Atoi(assignmentIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid assignment ID", nil)
		return
	}

	// Parse request body
	var req UpdateProxyACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Update the assignment
	if err := h.aclService.UpdateProxyAssignment(assignmentID, req.PathPattern, req.Priority, req.Enabled); err != nil {
		if errors.Is(err, service.ErrProxyACLNotFound) {
			utils.NotFound(w, "ACL assignment not found")
			return
		}
		if errors.Is(err, service.ErrInvalidPathPattern) {
			utils.BadRequest(w, "Invalid path pattern", nil)
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to update ACL assignment",
				zap.Int("assignment_id", assignmentID),
				zap.String("path_pattern", req.PathPattern),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to update ACL assignment")
		return
	}

	// Trigger Caddy sync to update the proxy's Caddyfile with ACL config
	if h.syncService != nil {
		if err := h.syncService.SyncProxyByID(proxyID); err != nil {
			if h.logger != nil {
				h.logger.Warn("Failed to sync proxy after ACL assignment update",
					zap.Int("proxy_id", proxyID),
					zap.Error(err))
			}
		}
	}

	utils.Success(w, nil, "ACL assignment updated successfully")
}

// RemoveACLFromProxy handles DELETE /api/proxies/:id/acl/:groupId
// Removes an ACL group assignment from a proxy
func (h *ProxyACLHandler) RemoveACLFromProxy(w http.ResponseWriter, r *http.Request) {
	proxyIDStr := chi.URLParam(r, "id")
	proxyID, err := strconv.Atoi(proxyIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid proxy ID", nil)
		return
	}

	groupIDStr := chi.URLParam(r, "groupId")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Remove ACL group from proxy
	if err := h.aclService.RemoveFromProxy(proxyID, groupID); err != nil {
		if errors.Is(err, service.ErrProxyACLNotFound) {
			utils.NotFound(w, "ACL assignment not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to remove ACL from proxy",
				zap.Int("proxy_id", proxyID),
				zap.Int("group_id", groupID),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to remove ACL from proxy")
		return
	}

	// Trigger Caddy sync to update the proxy's Caddyfile (remove ACL config)
	if h.syncService != nil {
		if err := h.syncService.SyncProxyByID(proxyID); err != nil {
			if h.logger != nil {
				h.logger.Warn("Failed to sync proxy after ACL removal",
					zap.Int("proxy_id", proxyID),
					zap.Error(err))
			}
		}
	}

	utils.Success(w, nil, "ACL removed from proxy successfully")
}
