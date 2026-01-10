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

// ACLHandler handles ACL group management HTTP requests
type ACLHandler struct {
	aclService   service.ACLServiceInterface
	aclRepo      repository.ACLRepositoryInterface
	syncService  service.SyncServiceInterface
	auditService service.AuditServiceInterface
	logger       *zap.Logger
}

// NewACLHandler creates a new ACL handler
func NewACLHandler(aclService service.ACLServiceInterface, aclRepo repository.ACLRepositoryInterface, syncService service.SyncServiceInterface, auditService service.AuditServiceInterface, logger *zap.Logger) *ACLHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ACLHandler{
		aclService:   aclService,
		aclRepo:      aclRepo,
		syncService:  syncService,
		logger:       logger.Named("acl-handler"),
		auditService: auditService,
	}
}

// syncProxiesUsingGroup triggers a Caddy sync for all proxies that use the given ACL group.
// This ensures that changes to ACL group content (IP rules, basic auth, etc.) are reflected
// in the Caddy configuration without requiring manual proxy re-save.
func (h *ACLHandler) syncProxiesUsingGroup(groupID int) {
	if h.syncService == nil {
		return
	}

	assignments, err := h.aclService.GetGroupUsage(groupID)
	if err != nil {
		h.logger.Warn("Failed to get proxies using ACL group for sync",
			zap.Int("group_id", groupID),
			zap.Error(err))
		return
	}

	for _, assignment := range assignments {
		if err := h.syncService.SyncProxyByID(assignment.ProxyID); err != nil {
			h.logger.Warn("Failed to sync proxy after ACL group change",
				zap.Int("proxy_id", assignment.ProxyID),
				zap.Int("group_id", groupID),
				zap.Error(err))
			// Continue syncing other proxies even if one fails
		}
	}

	if len(assignments) > 0 {
		h.logger.Info("Synced proxies after ACL group content change",
			zap.Int("group_id", groupID),
			zap.Int("proxy_count", len(assignments)))
	}
}

// =============================================================================
// Request/Response Types
// =============================================================================

// AuthOptionsResponse contains union of auth options from all ACL groups for a proxy
type AuthOptionsResponse struct {
	Hostname         string               `json:"hostname"`
	ProxyID          int64                `json:"proxy_id"`
	WaygatesAuth     *UnionWaygatesAuth   `json:"waygates_auth,omitempty"`
	OAuthProviders   []UnionOAuthProvider `json:"oauth_providers,omitempty"`
	BasicAuthEnabled bool                 `json:"basic_auth_enabled"`
	RequiresAuth     bool                 `json:"requires_auth"`
}

// UnionWaygatesAuth represents Waygates auth configuration in the union response
type UnionWaygatesAuth struct {
	Enabled bool `json:"enabled"`
}

// UnionOAuthProvider represents an OAuth provider in the union response
type UnionOAuthProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// CreateGroupRequest is the request body for creating an ACL group
type CreateGroupRequest struct {
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	CombinationMode string  `json:"combination_mode,omitempty"`
}

// UpdateGroupRequest is the request body for updating an ACL group
type UpdateGroupRequest struct {
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	CombinationMode string  `json:"combination_mode,omitempty"`
}

// AddIPRuleRequest is the request body for adding an IP rule
type AddIPRuleRequest struct {
	RuleType    string  `json:"rule_type"`
	CIDR        string  `json:"cidr"`
	Description *string `json:"description,omitempty"`
	Priority    int     `json:"priority"`
}

// UpdateIPRuleRequest is the request body for updating an IP rule
type UpdateIPRuleRequest struct {
	RuleType    string  `json:"rule_type"`
	CIDR        string  `json:"cidr"`
	Description *string `json:"description,omitempty"`
	Priority    int     `json:"priority"`
}

// AddBasicAuthUserRequest is the request body for adding a basic auth user
type AddBasicAuthUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateBasicAuthUserRequest is the request body for updating a basic auth user
type UpdateBasicAuthUserRequest struct {
	Password string `json:"password"`
}

// AddExternalProviderRequest is the request body for adding an external provider
type AddExternalProviderRequest struct {
	ProviderType    string   `json:"provider_type"`
	Name            string   `json:"name"`
	VerifyURL       string   `json:"verify_url"`
	AuthRedirectURL *string  `json:"auth_redirect_url,omitempty"`
	HeadersToCopy   []string `json:"headers_to_copy,omitempty"`
}

// UpdateExternalProviderRequest is the request body for updating an external provider
type UpdateExternalProviderRequest struct {
	ProviderType    string   `json:"provider_type"`
	Name            string   `json:"name"`
	VerifyURL       string   `json:"verify_url"`
	AuthRedirectURL *string  `json:"auth_redirect_url,omitempty"`
	HeadersToCopy   []string `json:"headers_to_copy,omitempty"`
}

// ConfigureWaygatesAuthRequest is the request body for configuring Waygates auth
type ConfigureWaygatesAuthRequest struct {
	Enabled              bool     `json:"enabled"`
	AllowedUsers         []string `json:"allowed_users,omitempty"`
	AllowedRoles         []string `json:"allowed_roles,omitempty"`
	AllowedEmailPatterns []string `json:"allowed_email_patterns,omitempty"`
	Require2FA           bool     `json:"require_2fa"`
	SessionTTL           int      `json:"session_ttl"`
	// OAuth user restrictions
	AllowedEmails    []string `json:"allowed_emails,omitempty"`
	AllowedDomains   []string `json:"allowed_domains,omitempty"`
	AllowedProviders []string `json:"allowed_providers,omitempty"`
}

// UpdateBrandingRequest is the request body for updating branding
type UpdateBrandingRequest struct {
	LogoURL         *string `json:"logo_url,omitempty"`
	PrimaryColor    string  `json:"primary_color"`
	BackgroundColor string  `json:"background_color"`
	Title           string  `json:"title"`
	Subtitle        *string `json:"subtitle,omitempty"`
	FooterText      *string `json:"footer_text,omitempty"`
	CustomCSS       *string `json:"custom_css,omitempty"`
}

// =============================================================================
// Group Handlers
// =============================================================================

// ListGroups handles GET /api/acl/groups
func (h *ACLHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var page, limit int
	var err error

	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			utils.BadRequest(w, "Invalid page parameter: must be a positive integer", nil)
			return
		}
	} else {
		page = 1
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			utils.BadRequest(w, "Invalid limit parameter: must be an integer between 1 and 100", nil)
			return
		}
	} else {
		limit = 20
	}

	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	req := service.ListACLGroupsRequest{
		Page:   page,
		Limit:  limit,
		Search: search,
		Sort:   sort,
		Order:  order,
	}

	result, err := h.aclService.ListGroups(req)
	if err != nil {
		utils.InternalError(w, "Failed to list ACL groups")
		return
	}

	utils.Success(w, result, "")
}

// CreateGroup handles POST /api/acl/groups
func (h *ACLHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
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
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		utils.BadRequest(w, "Name is required", nil)
		return
	}

	// Set default combination mode
	combinationMode := req.CombinationMode
	if combinationMode == "" {
		combinationMode = models.ACLCombinationModeAny
	}

	// Validate combination mode
	if combinationMode != models.ACLCombinationModeAny &&
		combinationMode != models.ACLCombinationModeAll &&
		combinationMode != models.ACLCombinationModeIPBypass {
		utils.BadRequest(w, "Invalid combination_mode: must be 'any', 'all', or 'ip_bypass'", nil)
		return
	}

	group := &models.ACLGroup{
		Name:            req.Name,
		Description:     req.Description,
		CombinationMode: combinationMode,
	}

	if err := h.aclService.CreateGroup(group, userID); err != nil {
		if errors.Is(err, service.ErrACLGroupNameExists) {
			utils.Conflict(w, "ACL group with this name already exists")
			return
		}
		// Handle validation errors
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		utils.InternalError(w, "Failed to create ACL group")
		return
	}

	// TODO: Add audit logging for ACL group creation

	utils.Created(w, group, "ACL group created successfully")
}

// GetGroup handles GET /api/acl/groups/:id
func (h *ACLHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	group, err := h.aclService.GetGroup(id)
	if err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		utils.InternalError(w, "Failed to get ACL group")
		return
	}

	utils.Success(w, group, "")
}

// UpdateGroup handles PUT /api/acl/groups/:id
func (h *ACLHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Parse request body
	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		utils.BadRequest(w, "Name is required", nil)
		return
	}

	// Validate combination mode if provided
	if req.CombinationMode != "" &&
		req.CombinationMode != models.ACLCombinationModeAny &&
		req.CombinationMode != models.ACLCombinationModeAll &&
		req.CombinationMode != models.ACLCombinationModeIPBypass {
		utils.BadRequest(w, "Invalid combination_mode: must be 'any', 'all', or 'ip_bypass'", nil)
		return
	}

	updates := &models.ACLGroup{
		Name:            req.Name,
		Description:     req.Description,
		CombinationMode: req.CombinationMode,
	}

	if err := h.aclService.UpdateGroup(id, updates); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		if errors.Is(err, service.ErrACLGroupNameExists) {
			utils.Conflict(w, "ACL group with this name already exists")
			return
		}
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		utils.InternalError(w, "Failed to update ACL group")
		return
	}

	// Get updated group to return
	group, err := h.aclService.GetGroup(id)
	if err != nil {
		utils.InternalError(w, "Failed to get updated ACL group")
		return
	}

	utils.Success(w, group, "ACL group updated successfully")
}

// DeleteGroup handles DELETE /api/acl/groups/:id
func (h *ACLHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Get proxies using this ACL group BEFORE deletion so we can sync them after
	var proxyIDsToSync []int
	if h.syncService != nil {
		assignments, err := h.aclService.GetGroupUsage(id)
		if err != nil {
			h.logger.Warn("Failed to get proxies using ACL group before deletion",
				zap.Int("group_id", id),
				zap.Error(err))
		} else {
			for _, assignment := range assignments {
				proxyIDsToSync = append(proxyIDsToSync, assignment.ProxyID)
			}
		}
	}

	if err := h.aclService.DeleteGroup(id); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		utils.InternalError(w, "Failed to delete ACL group")
		return
	}

	// Sync all proxies that were using this ACL group to remove ACL config from Caddyfiles
	for _, proxyID := range proxyIDsToSync {
		if err := h.syncService.SyncProxyByID(proxyID); err != nil {
			h.logger.Warn("Failed to sync proxy after ACL group deletion",
				zap.Int("proxy_id", proxyID),
				zap.Int("group_id", id),
				zap.Error(err))
			// Continue syncing other proxies even if one fails
		}
	}

	if len(proxyIDsToSync) > 0 {
		h.logger.Info("Synced proxies after ACL group deletion",
			zap.Int("group_id", id),
			zap.Int("proxy_count", len(proxyIDsToSync)))
	}

	utils.Success(w, nil, "ACL group deleted successfully")
}

// =============================================================================
// IP Rule Handlers
// =============================================================================

// ListIPRules handles GET /api/acl/groups/:id/ip-rules
func (h *ACLHandler) ListIPRules(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Verify group exists
	_, err = h.aclService.GetGroup(groupID)
	if err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		utils.InternalError(w, "Failed to get ACL group")
		return
	}

	// Get IP rules from repository
	rules, err := h.aclRepo.ListIPRules(groupID)
	if err != nil {
		utils.InternalError(w, "Failed to list IP rules")
		return
	}

	utils.Success(w, rules, "")
}

// AddIPRule handles POST /api/acl/groups/:id/ip-rules
func (h *ACLHandler) AddIPRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Parse request body
	var req AddIPRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.RuleType == "" {
		utils.BadRequest(w, "rule_type is required", nil)
		return
	}
	if req.CIDR == "" {
		utils.BadRequest(w, "cidr is required", nil)
		return
	}

	// Validate rule type
	if req.RuleType != models.ACLIPRuleTypeAllow &&
		req.RuleType != models.ACLIPRuleTypeDeny &&
		req.RuleType != models.ACLIPRuleTypeBypass {
		utils.BadRequest(w, "Invalid rule_type: must be 'allow', 'deny', or 'bypass'", nil)
		return
	}

	rule := &models.ACLIPRule{
		RuleType:    req.RuleType,
		CIDR:        req.CIDR,
		Description: req.Description,
		Priority:    req.Priority,
	}

	if err := h.aclService.AddIPRule(groupID, rule); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		if errors.Is(err, service.ErrInvalidCIDR) {
			utils.BadRequest(w, "Invalid CIDR format", nil)
			return
		}
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		h.logger.Error("Failed to add IP rule",
			zap.Int("group_id", groupID),
			zap.String("cidr", req.CIDR),
			zap.Error(err))
		utils.InternalError(w, "Failed to add IP rule")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Created(w, rule, "IP rule added successfully")
}

// UpdateIPRule handles PUT /api/acl/ip-rules/:id
func (h *ACLHandler) UpdateIPRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid IP rule ID", nil)
		return
	}

	// Get existing rule to obtain groupID for sync
	existingRule, err := h.aclRepo.GetIPRuleByID(id)
	if err != nil {
		utils.NotFound(w, "IP rule not found")
		return
	}
	groupID := existingRule.ACLGroupID

	// Parse request body
	var req UpdateIPRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.RuleType == "" {
		utils.BadRequest(w, "rule_type is required", nil)
		return
	}
	if req.CIDR == "" {
		utils.BadRequest(w, "cidr is required", nil)
		return
	}

	// Validate rule type
	if req.RuleType != models.ACLIPRuleTypeAllow &&
		req.RuleType != models.ACLIPRuleTypeDeny &&
		req.RuleType != models.ACLIPRuleTypeBypass {
		utils.BadRequest(w, "Invalid rule_type: must be 'allow', 'deny', or 'bypass'", nil)
		return
	}

	rule := &models.ACLIPRule{
		RuleType:    req.RuleType,
		CIDR:        req.CIDR,
		Description: req.Description,
		Priority:    req.Priority,
	}

	if err := h.aclService.UpdateIPRule(id, rule); err != nil {
		if errors.Is(err, service.ErrIPRuleNotFound) {
			utils.NotFound(w, "IP rule not found")
			return
		}
		if errors.Is(err, service.ErrInvalidCIDR) {
			utils.BadRequest(w, "Invalid CIDR format", nil)
			return
		}
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		utils.InternalError(w, "Failed to update IP rule")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, rule, "IP rule updated successfully")
}

// DeleteIPRule handles DELETE /api/acl/ip-rules/:id
func (h *ACLHandler) DeleteIPRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid IP rule ID", nil)
		return
	}

	// Get existing rule to obtain groupID for sync before deletion
	existingRule, err := h.aclRepo.GetIPRuleByID(id)
	if err != nil {
		utils.NotFound(w, "IP rule not found")
		return
	}
	groupID := existingRule.ACLGroupID

	if err := h.aclService.DeleteIPRule(id); err != nil {
		if errors.Is(err, service.ErrIPRuleNotFound) {
			utils.NotFound(w, "IP rule not found")
			return
		}
		utils.InternalError(w, "Failed to delete IP rule")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, nil, "IP rule deleted successfully")
}

// =============================================================================
// Basic Auth Handlers
// =============================================================================

// ListBasicAuthUsers handles GET /api/acl/groups/:id/basic-auth
func (h *ACLHandler) ListBasicAuthUsers(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Verify group exists
	_, err = h.aclService.GetGroup(groupID)
	if err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		utils.InternalError(w, "Failed to get ACL group")
		return
	}

	// Get basic auth users from repository
	users, err := h.aclRepo.ListBasicAuthUsers(groupID)
	if err != nil {
		utils.InternalError(w, "Failed to list basic auth users")
		return
	}

	utils.Success(w, users, "")
}

// AddBasicAuthUser handles POST /api/acl/groups/:id/basic-auth
func (h *ACLHandler) AddBasicAuthUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Parse request body
	var req AddBasicAuthUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.Username == "" {
		utils.BadRequest(w, "username is required", nil)
		return
	}
	if req.Password == "" {
		utils.BadRequest(w, "password is required", nil)
		return
	}
	if len(req.Password) < 8 {
		utils.BadRequest(w, "password must be at least 8 characters", nil)
		return
	}

	if err := h.aclService.AddBasicAuthUser(groupID, req.Username, req.Password); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		if errors.Is(err, service.ErrBasicAuthUserExists) {
			utils.Conflict(w, "Basic auth user already exists in this group")
			return
		}
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		utils.InternalError(w, "Failed to add basic auth user")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Created(w, map[string]string{"username": req.Username}, "Basic auth user added successfully")
}

// UpdateBasicAuthUser handles PUT /api/acl/basic-auth/:id
func (h *ACLHandler) UpdateBasicAuthUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid basic auth user ID", nil)
		return
	}

	// Get existing user to obtain groupID for sync
	existingUser, err := h.aclRepo.GetBasicAuthUserByID(id)
	if err != nil {
		utils.NotFound(w, "Basic auth user not found")
		return
	}
	groupID := existingUser.ACLGroupID

	// Parse request body
	var req UpdateBasicAuthUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.Password == "" {
		utils.BadRequest(w, "password is required", nil)
		return
	}
	if len(req.Password) < 8 {
		utils.BadRequest(w, "password must be at least 8 characters", nil)
		return
	}

	if err := h.aclService.UpdateBasicAuthPassword(id, req.Password); err != nil {
		if errors.Is(err, service.ErrBasicAuthUserNotFound) {
			utils.NotFound(w, "Basic auth user not found")
			return
		}
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		utils.InternalError(w, "Failed to update basic auth user")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, nil, "Basic auth user password updated successfully")
}

// DeleteBasicAuthUser handles DELETE /api/acl/basic-auth/:id
func (h *ACLHandler) DeleteBasicAuthUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid basic auth user ID", nil)
		return
	}

	// Get existing user to obtain groupID for sync before deletion
	existingUser, err := h.aclRepo.GetBasicAuthUserByID(id)
	if err != nil {
		utils.NotFound(w, "Basic auth user not found")
		return
	}
	groupID := existingUser.ACLGroupID

	if err := h.aclService.DeleteBasicAuthUser(id); err != nil {
		if errors.Is(err, service.ErrBasicAuthUserNotFound) {
			utils.NotFound(w, "Basic auth user not found")
			return
		}
		utils.InternalError(w, "Failed to delete basic auth user")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, nil, "Basic auth user deleted successfully")
}

// =============================================================================
// External Provider Handlers
// =============================================================================

// ListExternalProviders handles GET /api/acl/groups/:id/providers
func (h *ACLHandler) ListExternalProviders(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Verify group exists
	_, err = h.aclService.GetGroup(groupID)
	if err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		utils.InternalError(w, "Failed to get ACL group")
		return
	}

	// Get external providers from repository
	providers, err := h.aclRepo.ListExternalProviders(groupID)
	if err != nil {
		utils.InternalError(w, "Failed to list external providers")
		return
	}

	utils.Success(w, providers, "")
}

// AddExternalProvider handles POST /api/acl/groups/:id/providers
func (h *ACLHandler) AddExternalProvider(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Parse request body
	var req AddExternalProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.ProviderType == "" {
		utils.BadRequest(w, "provider_type is required", nil)
		return
	}
	if req.Name == "" {
		utils.BadRequest(w, "name is required", nil)
		return
	}
	if req.VerifyURL == "" {
		utils.BadRequest(w, "verify_url is required", nil)
		return
	}

	// Validate provider type
	if req.ProviderType != models.ACLProviderTypeAuthelia &&
		req.ProviderType != models.ACLProviderTypeAuthentik &&
		req.ProviderType != models.ACLProviderTypeCustom {
		utils.BadRequest(w, "Invalid provider_type: must be 'authelia', 'authentik', or 'custom'", nil)
		return
	}

	provider := &models.ACLExternalProvider{
		ProviderType:    req.ProviderType,
		Name:            req.Name,
		VerifyURL:       req.VerifyURL,
		AuthRedirectURL: req.AuthRedirectURL,
		HeadersToCopy:   req.HeadersToCopy,
	}

	if err := h.aclService.AddExternalProvider(groupID, provider); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		utils.InternalError(w, "Failed to add external provider")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Created(w, provider, "External provider added successfully")
}

// UpdateExternalProvider handles PUT /api/acl/providers/:id
func (h *ACLHandler) UpdateExternalProvider(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid external provider ID", nil)
		return
	}

	// Get existing provider to obtain groupID for sync
	existingProvider, err := h.aclRepo.GetExternalProviderByID(id)
	if err != nil {
		utils.NotFound(w, "External provider not found")
		return
	}
	groupID := existingProvider.ACLGroupID

	// Parse request body
	var req UpdateExternalProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.ProviderType == "" {
		utils.BadRequest(w, "provider_type is required", nil)
		return
	}
	if req.Name == "" {
		utils.BadRequest(w, "name is required", nil)
		return
	}
	if req.VerifyURL == "" {
		utils.BadRequest(w, "verify_url is required", nil)
		return
	}

	// Validate provider type
	if req.ProviderType != models.ACLProviderTypeAuthelia &&
		req.ProviderType != models.ACLProviderTypeAuthentik &&
		req.ProviderType != models.ACLProviderTypeCustom {
		utils.BadRequest(w, "Invalid provider_type: must be 'authelia', 'authentik', or 'custom'", nil)
		return
	}

	provider := &models.ACLExternalProvider{
		ProviderType:    req.ProviderType,
		Name:            req.Name,
		VerifyURL:       req.VerifyURL,
		AuthRedirectURL: req.AuthRedirectURL,
		HeadersToCopy:   req.HeadersToCopy,
	}

	if err := h.aclService.UpdateExternalProvider(id, provider); err != nil {
		if errors.Is(err, service.ErrExternalProviderNotFound) {
			utils.NotFound(w, "External provider not found")
			return
		}
		var validationErr *models.ValidationError
		if errors.As(err, &validationErr) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		utils.InternalError(w, "Failed to update external provider")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, provider, "External provider updated successfully")
}

// DeleteExternalProvider handles DELETE /api/acl/providers/:id
func (h *ACLHandler) DeleteExternalProvider(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid external provider ID", nil)
		return
	}

	// Get existing provider to obtain groupID for sync before deletion
	existingProvider, err := h.aclRepo.GetExternalProviderByID(id)
	if err != nil {
		utils.NotFound(w, "External provider not found")
		return
	}
	groupID := existingProvider.ACLGroupID

	if err := h.aclService.DeleteExternalProvider(id); err != nil {
		if errors.Is(err, service.ErrExternalProviderNotFound) {
			utils.NotFound(w, "External provider not found")
			return
		}
		utils.InternalError(w, "Failed to delete external provider")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, nil, "External provider deleted successfully")
}

// =============================================================================
// Waygates Auth Handlers
// =============================================================================

// GetWaygatesAuth handles GET /api/acl/groups/:id/waygates-auth
func (h *ACLHandler) GetWaygatesAuth(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	auth, err := h.aclService.GetWaygatesAuth(groupID)
	if err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		if errors.Is(err, service.ErrWaygatesAuthNotFound) {
			// Return empty config - this is not an error, just not configured yet
			utils.Success(w, nil, "")
			return
		}
		utils.InternalError(w, "Failed to get Waygates auth configuration")
		return
	}

	utils.Success(w, auth, "")
}

// ConfigureWaygatesAuth handles PUT /api/acl/groups/:id/waygates-auth
func (h *ACLHandler) ConfigureWaygatesAuth(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Parse request body
	var req ConfigureWaygatesAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	config := &models.ACLWaygatesAuth{
		Enabled:              req.Enabled,
		AllowedUsers:         req.AllowedUsers,
		AllowedRoles:         req.AllowedRoles,
		AllowedEmailPatterns: req.AllowedEmailPatterns,
		Require2FA:           req.Require2FA,
		SessionTTL:           req.SessionTTL,
		AllowedEmails:        req.AllowedEmails,
		AllowedDomains:       req.AllowedDomains,
		AllowedProviders:     req.AllowedProviders,
	}

	if err := h.aclService.ConfigureWaygatesAuth(groupID, config); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		h.logger.Error("Failed to configure Waygates auth", zap.Int("group_id", groupID), zap.Error(err))
		utils.InternalError(w, "Failed to configure Waygates auth")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, config, "Waygates auth configured successfully")
}

// =============================================================================
// Branding Handlers
// =============================================================================

// GetBranding handles GET /api/acl/branding
func (h *ACLHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	branding, err := h.aclService.GetBranding()
	if err != nil {
		utils.InternalError(w, "Failed to get branding configuration")
		return
	}

	utils.Success(w, branding, "")
}

// UpdateBranding handles PUT /api/acl/branding
func (h *ACLHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req UpdateBrandingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Set defaults if not provided
	if req.PrimaryColor == "" {
		req.PrimaryColor = "#3b82f6"
	}
	if req.BackgroundColor == "" {
		req.BackgroundColor = "#ffffff"
	}
	if req.Title == "" {
		req.Title = "Login Required"
	}

	branding := &models.ACLBranding{
		LogoURL:         req.LogoURL,
		PrimaryColor:    req.PrimaryColor,
		BackgroundColor: req.BackgroundColor,
		Title:           req.Title,
		Subtitle:        req.Subtitle,
		FooterText:      req.FooterText,
		CustomCSS:       req.CustomCSS,
	}

	if err := h.aclService.UpdateBranding(branding); err != nil {
		utils.InternalError(w, "Failed to update branding configuration")
		return
	}

	utils.Success(w, branding, "Branding updated successfully")
}

// GetGroupUsage handles GET /api/acl/groups/:id/usage
func (h *ACLHandler) GetGroupUsage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	// Verify group exists
	_, err = h.aclService.GetGroup(groupID)
	if err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		utils.InternalError(w, "Failed to get ACL group")
		return
	}

	// Get usage from service
	assignments, err := h.aclService.GetGroupUsage(groupID)
	if err != nil {
		utils.InternalError(w, "Failed to get ACL group usage")
		return
	}

	utils.Success(w, assignments, "")
}

// =============================================================================
// Auth Options Handler
// =============================================================================

// GetAuthOptions returns the union of auth options for a hostname.
// This is a PUBLIC endpoint used by the login page to determine what auth methods are available.
// GET /api/acl/options?hostname=example.com
func (h *ACLHandler) GetAuthOptions(w http.ResponseWriter, r *http.Request) {
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		utils.BadRequest(w, "hostname query parameter is required", nil)
		return
	}

	options, err := h.aclService.GetAuthOptionsForProxy(hostname)
	if err != nil {
		h.logger.Warn("failed to get auth options", zap.String("hostname", hostname), zap.Error(err))
		utils.NotFound(w, "proxy not found for hostname")
		return
	}

	utils.Success(w, options, "")
}

// =============================================================================
// OAuth Provider Restriction Handlers
// =============================================================================

// Valid OAuth providers
var validOAuthProviders = map[string]bool{
	"google":    true,
	"github":    true,
	"microsoft": true,
	"gitlab":    true,
}

// SetOAuthProviderRestrictionRequest is the request body for setting OAuth provider restrictions
type SetOAuthProviderRestrictionRequest struct {
	AllowedEmails  []string `json:"allowed_emails"`
	AllowedDomains []string `json:"allowed_domains"`
	Enabled        bool     `json:"enabled"`
}

// GetOAuthProviderRestrictions handles GET /api/acl/groups/{id}/oauth-restrictions
func (h *ACLHandler) GetOAuthProviderRestrictions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	restrictions, err := h.aclService.GetOAuthProviderRestrictions(groupID)
	if err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		h.logger.Error("Failed to get OAuth provider restrictions",
			zap.Int("group_id", groupID),
			zap.Error(err))
		utils.InternalError(w, "Failed to get OAuth provider restrictions")
		return
	}

	utils.Success(w, restrictions, "")
}

// SetOAuthProviderRestriction handles PUT /api/acl/groups/{id}/oauth-restrictions/{provider}
func (h *ACLHandler) SetOAuthProviderRestriction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	provider := chi.URLParam(r, "provider")
	if provider == "" {
		utils.BadRequest(w, "Provider is required", nil)
		return
	}

	// Validate provider is one of the valid OAuth providers
	if !validOAuthProviders[provider] {
		utils.BadRequest(w, "Invalid provider: must be one of 'google', 'github', 'microsoft', 'gitlab'", nil)
		return
	}

	// Parse request body
	var req SetOAuthProviderRestrictionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	if err := h.aclService.SetOAuthProviderRestriction(groupID, provider, req.AllowedEmails, req.AllowedDomains, req.Enabled); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		h.logger.Error("Failed to set OAuth provider restriction",
			zap.Int("group_id", groupID),
			zap.String("provider", provider),
			zap.Error(err))
		utils.InternalError(w, "Failed to set OAuth provider restriction")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, map[string]interface{}{
		"provider":        provider,
		"allowed_emails":  req.AllowedEmails,
		"allowed_domains": req.AllowedDomains,
		"enabled":         req.Enabled,
	}, "OAuth provider restriction set successfully")
}

// DeleteOAuthProviderRestriction handles DELETE /api/acl/groups/{id}/oauth-restrictions/{provider}
func (h *ACLHandler) DeleteOAuthProviderRestriction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid ACL group ID", nil)
		return
	}

	provider := chi.URLParam(r, "provider")
	if provider == "" {
		utils.BadRequest(w, "Provider is required", nil)
		return
	}

	// Validate provider is one of the valid OAuth providers
	if !validOAuthProviders[provider] {
		utils.BadRequest(w, "Invalid provider: must be one of 'google', 'github', 'microsoft', 'gitlab'", nil)
		return
	}

	if err := h.aclService.DeleteOAuthProviderRestriction(groupID, provider); err != nil {
		if errors.Is(err, service.ErrACLGroupNotFound) {
			utils.NotFound(w, "ACL group not found")
			return
		}
		if errors.Is(err, service.ErrOAuthProviderRestrictionNotFound) {
			utils.NotFound(w, "OAuth provider restriction not found")
			return
		}
		h.logger.Error("Failed to delete OAuth provider restriction",
			zap.Int("group_id", groupID),
			zap.String("provider", provider),
			zap.Error(err))
		utils.InternalError(w, "Failed to delete OAuth provider restriction")
		return
	}

	// Sync all proxies using this ACL group
	h.syncProxiesUsingGroup(groupID)

	utils.Success(w, nil, "OAuth provider restriction deleted successfully")
}
