package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ProxyACLHandler handles proxy ACL assignment HTTP requests
type ProxyACLHandler struct {
	aclService   service.ACLServiceInterface
	aclRepo      repository.ACLRepositoryInterface
	proxyRepo    repository.ProxyRepositoryInterface
	syncService  service.SyncServiceInterface
	auditService service.AuditServiceInterface
	logger       *zap.Logger
}

// NewProxyACLHandler creates a new proxy ACL handler
func NewProxyACLHandler(aclService service.ACLServiceInterface, aclRepo repository.ACLRepositoryInterface, proxyRepo repository.ProxyRepositoryInterface, syncService service.SyncServiceInterface, auditService service.AuditServiceInterface, logger *zap.Logger) *ProxyACLHandler {
	return &ProxyACLHandler{
		aclService:   aclService,
		aclRepo:      aclRepo,
		proxyRepo:    proxyRepo,
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
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

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

	// Get proxy and group names for audit logging
	proxyName := ""
	if h.proxyRepo != nil {
		if proxy, err := h.proxyRepo.GetByID(proxyID); err == nil && proxy != nil {
			proxyName = proxy.Name
		}
	}
	groupName := ""
	if group, err := h.aclService.GetGroup(req.ACLGroupID); err == nil && group != nil {
		groupName = group.Name
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
			utils.Conflict(w, "This ACL group is already assigned to this proxy")
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

	// Audit logging for ACL assignment
	if h.auditService != nil {
		_ = h.auditService.LogACLAssignmentCreate(r.Context(), userID, proxyID, proxyName, req.ACLGroupID, groupName, pathPattern, getClientIP(r), r.UserAgent())
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
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

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

	// Get old assignment for change tracking
	var oldAssignment *models.ProxyACLAssignment
	if h.aclRepo != nil {
		oldAssignment, _ = h.aclRepo.GetProxyACLAssignmentByID(assignmentID)
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

	// Audit logging for ACL assignment update
	if h.auditService != nil && h.aclRepo != nil {
		newAssignment, _ := h.aclRepo.GetProxyACLAssignmentByID(assignmentID)
		if newAssignment != nil {
			changes := buildProxyACLAssignmentChanges(oldAssignment, newAssignment)
			_ = h.auditService.LogACLAssignmentUpdate(r.Context(), userID, newAssignment, changes, getClientIP(r), r.UserAgent())
		}
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
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

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

	// Get proxy and group names for audit logging before deletion
	proxyName := ""
	if h.proxyRepo != nil {
		if proxy, err := h.proxyRepo.GetByID(proxyID); err == nil && proxy != nil {
			proxyName = proxy.Name
		}
	}
	groupName := ""
	if group, err := h.aclService.GetGroup(groupID); err == nil && group != nil {
		groupName = group.Name
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

	// Audit logging for ACL assignment deletion
	if h.auditService != nil {
		_ = h.auditService.LogACLAssignmentDelete(r.Context(), userID, proxyID, proxyName, groupID, groupName, getClientIP(r), r.UserAgent())
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

// buildProxyACLAssignmentChanges builds a map of changes between old and updated proxy ACL assignment
func buildProxyACLAssignmentChanges(old, updated *models.ProxyACLAssignment) map[string]interface{} {
	changes := make(map[string]interface{})

	if old == nil {
		return changes
	}

	if old.PathPattern != updated.PathPattern {
		changes["path_pattern"] = map[string]interface{}{
			"old": old.PathPattern,
			"new": updated.PathPattern,
		}
	}

	if old.Priority != updated.Priority {
		changes["priority"] = map[string]interface{}{
			"old": old.Priority,
			"new": updated.Priority,
		}
	}

	if old.Enabled != updated.Enabled {
		changes["enabled"] = map[string]interface{}{
			"old": old.Enabled,
			"new": updated.Enabled,
		}
	}

	return changes
}
