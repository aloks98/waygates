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

// ProxyGroupHandler handles proxy-group-related HTTP requests, including the
// nested ACL assignment routes. ProxyGroup is a config-inheritance parent —
// NOT an ACLGroup (an auth grouping); see models/proxy_group.go.
type ProxyGroupHandler struct {
	service      service.ProxyGroupServiceInterface
	auditService service.AuditServiceInterface
	logger       *zap.Logger
}

// NewProxyGroupHandler creates a new proxy group handler
func NewProxyGroupHandler(svc service.ProxyGroupServiceInterface, auditService service.AuditServiceInterface, logger *zap.Logger) *ProxyGroupHandler {
	return &ProxyGroupHandler{
		service:      svc,
		auditService: auditService,
		logger:       logger,
	}
}

// ListGroups handles GET /api/proxy-groups
func (h *ProxyGroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	var page, limit int
	var err error

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 0 {
			utils.BadRequest(w, "Invalid page parameter: must be a non-negative integer", nil)
			return
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 || limit > 100 {
			utils.BadRequest(w, "Invalid limit parameter: must be an integer between 0 and 100", nil)
			return
		}
	}

	result, err := h.service.ListGroups(repository.ProxyGroupListParams{
		Page:   page,
		Limit:  limit,
		Search: r.URL.Query().Get("search"),
		Sort:   r.URL.Query().Get("sort"),
		Order:  r.URL.Query().Get("order"),
	})
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to list proxy groups", zap.Error(err))
		}
		utils.InternalError(w, "Failed to list proxy groups")
		return
	}

	utils.Success(w, result, "")
}

// GetGroup handles GET /api/proxy-groups/:id
func (h *ProxyGroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "Invalid proxy group ID", nil)
		return
	}

	group, err := h.service.GetGroupByID(id)
	if err != nil {
		if errors.Is(err, service.ErrGroupNotFound) {
			utils.NotFound(w, "Proxy group not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to get proxy group", zap.Int("id", id), zap.Error(err))
		}
		utils.InternalError(w, "Failed to get proxy group")
		return
	}

	utils.Success(w, group, "")
}

// CreateGroup handles POST /api/proxy-groups
func (h *ProxyGroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
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

	var group models.ProxyGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}
	if group.Name == "" {
		utils.BadRequest(w, "name is required", nil)
		return
	}

	if err := h.service.CreateGroup(&group, userID); err != nil {
		if errors.Is(err, service.ErrGroupNameConflict) {
			utils.Conflict(w, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to create proxy group",
				zap.String("name", group.Name),
				zap.Int("user_id", userID),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to create proxy group")
		return
	}

	if h.auditService != nil {
		_ = h.auditService.LogProxyGroupCreate(r.Context(), userID, &group, getClientIP(r), r.UserAgent())
	}

	utils.Created(w, group, "Proxy group created successfully")
}

// UpdateGroup handles PUT /api/proxy-groups/:id
func (h *ProxyGroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "Invalid proxy group ID", nil)
		return
	}

	existing, err := h.service.GetGroupByID(id)
	if err != nil {
		if errors.Is(err, service.ErrGroupNotFound) {
			utils.NotFound(w, "Proxy group not found")
			return
		}
		utils.InternalError(w, "Failed to get proxy group")
		return
	}

	var group models.ProxyGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}
	if group.Name == "" {
		utils.BadRequest(w, "name is required", nil)
		return
	}
	group.ID = id

	rehoming := baseDomainChanged(existing.BaseDomain, group.BaseDomain)

	if err := h.service.UpdateGroup(&group); err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			utils.NotFound(w, "Proxy group not found")
		case errors.Is(err, service.ErrBaseDomainRequiredByMembers):
			utils.Conflict(w, err.Error())
		case errors.Is(err, service.ErrGroupNameConflict):
			utils.Conflict(w, err.Error())
		case errors.Is(err, service.ErrHostnameConflict):
			utils.Conflict(w, err.Error())
		default:
			if h.logger != nil {
				h.logger.Error("Failed to update proxy group", zap.Int("id", id), zap.Error(err))
			}
			utils.InternalError(w, "Failed to update proxy group")
		}
		return
	}

	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogProxyGroupUpdate(r.Context(), userID, existing, &group, getClientIP(r), r.UserAgent())

		// A base_domain rewrite re-homes every label-addressed member's
		// materialized hostname. Log which proxies moved — without this, a mass
		// re-homing would leave no trace of what changed.
		if rehoming {
			var proxyIDs []int
			if members, mErr := h.service.ListMembers(id); mErr == nil {
				for i := range members {
					if members[i].HostnameLabel != nil {
						proxyIDs = append(proxyIDs, members[i].ID)
					}
				}
			} else if h.logger != nil {
				h.logger.Warn("Failed to list proxy group members for rehome audit log",
					zap.Int("group_id", id), zap.Error(mErr))
			}
			_ = h.auditService.LogProxyGroupRehome(r.Context(), userID, id,
				stringPtrOrEmpty(existing.BaseDomain), stringPtrOrEmpty(group.BaseDomain),
				proxyIDs, getClientIP(r), r.UserAgent())
		}
	}

	utils.Success(w, group, "Proxy group updated successfully")
}

// DeleteGroup handles DELETE /api/proxy-groups/:id
func (h *ProxyGroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "Invalid proxy group ID", nil)
		return
	}

	if err := h.service.DeleteGroup(id); err != nil {
		switch {
		case errors.Is(err, service.ErrGroupHasMembers):
			// 409 carries the member count in the message, so the UI can say
			// "7 member proxies" without a second round trip.
			utils.Conflict(w, err.Error())
		case errors.Is(err, service.ErrGroupNotFound):
			utils.NotFound(w, "Proxy group not found")
		default:
			if h.logger != nil {
				h.logger.Error("Failed to delete proxy group", zap.Int("id", id), zap.Error(err))
			}
			utils.InternalError(w, "Failed to delete proxy group")
		}
		return
	}

	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogProxyGroupDelete(r.Context(), userID, id, getClientIP(r), r.UserAgent())
	}

	utils.Success(w, nil, "Proxy group deleted successfully")
}

// =============================================================================
// ACL assignment (nested under a proxy group) — same acl:* permissions as the
// per-proxy ACL routes; the same capability, a different subject.
// =============================================================================

// AssignGroupACLRequest is the request body for assigning an ACL group to a
// proxy group.
type AssignGroupACLRequest struct {
	ACLGroupID  int    `json:"acl_group_id"`
	PathPattern string `json:"path_pattern,omitempty"`
	Priority    int    `json:"priority"`
	// Enabled mirrors AssignACLRequest (proxy_acl_handler.go): a pointer so
	// an omitted field defaults to true and an explicit false is
	// distinguishable from "not sent".
	Enabled *bool `json:"enabled,omitempty"`
}

// UpdateGroupACLRequest is the request body for updating a proxy group ACL
// assignment.
type UpdateGroupACLRequest struct {
	PathPattern string `json:"path_pattern,omitempty"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// GetGroupACL handles GET /api/proxy-groups/:id/acl
func (h *ProxyGroupHandler) GetGroupACL(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "Invalid proxy group ID", nil)
		return
	}

	assignments, err := h.service.ListACLAssignments(groupID)
	if err != nil {
		if errors.Is(err, service.ErrGroupNotFound) {
			utils.NotFound(w, "Proxy group not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to get proxy group ACL assignments", zap.Int("group_id", groupID), zap.Error(err))
		}
		utils.InternalError(w, "Failed to get proxy group ACL assignments")
		return
	}

	utils.Success(w, assignments, "")
}

// AssignACLToGroup handles POST /api/proxy-groups/:id/acl
func (h *ProxyGroupHandler) AssignACLToGroup(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	groupID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "Invalid proxy group ID", nil)
		return
	}

	var req AssignGroupACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}
	if req.ACLGroupID <= 0 {
		utils.BadRequest(w, "acl_group_id is required and must be positive", nil)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := h.service.AssignACLToGroup(groupID, req.ACLGroupID, req.PathPattern, req.Priority, enabled); err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			utils.NotFound(w, "Proxy group not found")
		case errors.Is(err, service.ErrGroupACLAssignmentExists):
			utils.Conflict(w, "This ACL group is already assigned to this proxy group")
		case errors.Is(err, service.ErrInvalidPathPattern):
			utils.BadRequest(w, "Invalid path pattern", nil)
		default:
			if h.logger != nil {
				h.logger.Error("Failed to assign ACL to proxy group",
					zap.Int("group_id", groupID), zap.Int("acl_group_id", req.ACLGroupID), zap.Error(err))
			}
			utils.InternalError(w, "Failed to assign ACL to proxy group")
		}
		return
	}

	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogEvent(r.Context(), models.AuditEvent{
			UserID:       &userID,
			Action:       models.AuditActionProxyGroupACLAssign,
			ResourceType: models.AuditResourceTypeProxyGroup,
			ResourceID:   &groupID,
			Details: map[string]interface{}{
				"acl_group_id": req.ACLGroupID,
				"path_pattern": req.PathPattern,
				"priority":     req.Priority,
			},
			IPAddress: getClientIP(r),
			UserAgent: r.UserAgent(),
			Status:    models.AuditStatusSuccess,
		})
	}

	assignments, err := h.service.ListACLAssignments(groupID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get updated proxy group ACL assignments after assign",
				zap.Int("group_id", groupID), zap.Error(err))
		}
		utils.InternalError(w, "Failed to get updated proxy group ACL assignments")
		return
	}

	utils.Created(w, assignments, "ACL assigned to proxy group successfully")
}

// UpdateGroupACLAssignment handles PUT /api/proxy-groups/:id/acl/:assignmentId
func (h *ProxyGroupHandler) UpdateGroupACLAssignment(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	groupID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "Invalid proxy group ID", nil)
		return
	}
	assignmentID, err := strconv.Atoi(chi.URLParam(r, "assignmentId"))
	if err != nil {
		utils.BadRequest(w, "Invalid assignment ID", nil)
		return
	}

	var req UpdateGroupACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	if err := h.service.UpdateGroupACLAssignment(groupID, assignmentID, req.PathPattern, req.Priority, req.Enabled); err != nil {
		switch {
		case errors.Is(err, service.ErrGroupACLAssignmentNotFound):
			utils.NotFound(w, "ACL assignment not found")
		case errors.Is(err, service.ErrInvalidPathPattern):
			utils.BadRequest(w, "Invalid path pattern", nil)
		default:
			if h.logger != nil {
				h.logger.Error("Failed to update proxy group ACL assignment",
					zap.Int("group_id", groupID), zap.Int("assignment_id", assignmentID), zap.Error(err))
			}
			utils.InternalError(w, "Failed to update proxy group ACL assignment")
		}
		return
	}

	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogEvent(r.Context(), models.AuditEvent{
			UserID:       &userID,
			Action:       models.AuditActionProxyGroupACLUpdate,
			ResourceType: models.AuditResourceTypeProxyGroup,
			ResourceID:   &groupID,
			Details: map[string]interface{}{
				"assignment_id": assignmentID,
				"path_pattern":  req.PathPattern,
				"priority":      req.Priority,
				"enabled":       req.Enabled,
			},
			IPAddress: getClientIP(r),
			UserAgent: r.UserAgent(),
			Status:    models.AuditStatusSuccess,
		})
	}

	utils.Success(w, nil, "ACL assignment updated successfully")
}

// RemoveACLFromGroup handles DELETE /api/proxy-groups/:id/acl/:aclGroupId
func (h *ProxyGroupHandler) RemoveACLFromGroup(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	groupID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "Invalid proxy group ID", nil)
		return
	}
	aclGroupID, err := strconv.Atoi(chi.URLParam(r, "aclGroupId"))
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	if err := h.service.RemoveACLFromGroup(groupID, aclGroupID); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to remove ACL from proxy group",
				zap.Int("group_id", groupID), zap.Int("acl_group_id", aclGroupID), zap.Error(err))
		}
		utils.InternalError(w, "Failed to remove ACL from proxy group")
		return
	}

	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogEvent(r.Context(), models.AuditEvent{
			UserID:       &userID,
			Action:       models.AuditActionProxyGroupACLRemove,
			ResourceType: models.AuditResourceTypeProxyGroup,
			ResourceID:   &groupID,
			Details: map[string]interface{}{
				"acl_group_id": aclGroupID,
			},
			IPAddress: getClientIP(r),
			UserAgent: r.UserAgent(),
			Status:    models.AuditStatusSuccess,
		})
	}

	utils.Success(w, nil, "ACL removed from proxy group successfully")
}

// baseDomainChanged reports whether a proxy group's base_domain is being
// changed. Duplicated (rather than exported) from
// service.baseDomainChanged — a tiny, self-contained comparison not worth
// wiring an extra cross-package export for.
func baseDomainChanged(old, next *string) bool {
	switch {
	case old == nil && next == nil:
		return false
	case old == nil || next == nil:
		return true
	default:
		return *old != *next
	}
}

// stringPtrOrEmpty returns the pointed-to string, or "" for nil. Used for
// audit log fields that are typed as plain strings (LogProxyGroupRehome's
// oldBase/newBase), not *string.
func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
