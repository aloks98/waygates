package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// BcryptCost is the bcrypt cost used for password hashing (Caddy compatible)
const BcryptCost = 14

// SessionTokenLength is the length of session tokens in bytes (before base64 encoding)
const SessionTokenLength = 32

// ACL Service Errors
var (
	ErrACLGroupNotFound         = errors.New("ACL group not found")
	ErrACLGroupNameExists       = errors.New("ACL group name already exists")
	ErrIPRuleNotFound           = errors.New("IP rule not found")
	ErrBasicAuthUserNotFound    = errors.New("basic auth user not found")
	ErrBasicAuthUserExists      = errors.New("basic auth user already exists in this group")
	ErrExternalProviderNotFound = errors.New("external provider not found")
	ErrWaygatesAuthNotFound     = errors.New("waygates auth not found")
	ErrProxyACLNotFound         = errors.New("proxy ACL assignment not found")
	ErrProxyACLExists           = errors.New("proxy ACL assignment already exists for this path")
	ErrSessionNotFound          = errors.New("session not found")
	ErrSessionExpired           = errors.New("session expired")
	ErrAccessDenied             = errors.New("access denied")
	ErrInvalidCIDR              = errors.New("invalid CIDR format")
	ErrInvalidPathPattern       = errors.New("invalid path pattern")
	ErrProxyNotFoundForHost     = errors.New("proxy not found for hostname")
)

// ListACLGroupsRequest holds parameters for listing ACL groups
type ListACLGroupsRequest struct {
	Page   int
	Limit  int
	Search string
	Sort   string
	Order  string
}

// ACLVerifyRequest is the request for verifying access
type ACLVerifyRequest struct {
	Host         string                // X-Forwarded-Host
	Path         string                // X-Forwarded-Uri
	Method       string                // X-Forwarded-Method
	RemoteIP     string                // X-Forwarded-For
	SessionToken string                // From cookie
	BasicAuth    *BasicAuthCredentials // From Authorization header
}

// BasicAuthCredentials holds basic auth credentials
type BasicAuthCredentials struct {
	Username string
	Password string
}

// ACLVerifyResponse is the response from access verification
type ACLVerifyResponse struct {
	Allowed      bool              // Whether access is allowed
	Reason       string            // Reason for denial
	RequiresAuth bool              // True if auth required but not provided
	RedirectURL  string            // URL to redirect for auth
	Headers      map[string]string // Headers to set on success (X-Auth-User, etc.)
	User         *models.User      // Authenticated user if any

	// OAuth user info (set when authenticated via OAuth)
	OAuthEmail    string // OAuth user's email
	OAuthProvider string // OAuth provider (google, github, etc.)

	// Set when user IS authenticated but fails authorization (e.g., email not in allowed list)
	// In this case, RequiresAuth should be false and we should return 403, not 401
	AuthenticatedButUnauthorized bool
	DenialDetails                string // Human-readable explanation of why access was denied
}

// groupAccessResult holds the result of evaluating access for a group
type groupAccessResult struct {
	Allowed       bool
	User          *models.User
	OAuthEmail    string
	OAuthProvider string
	// Set when user has valid session but fails restrictions
	HasValidSession              bool
	AuthenticatedButUnauthorized bool
	DenialReason                 string
}

// ACLService handles ACL business logic
type ACLService struct {
	aclRepo   repository.ACLRepositoryInterface
	proxyRepo repository.ProxyRepositoryInterface
	logger    *zap.Logger
}

// ACLServiceConfig holds configuration for ACLService
type ACLServiceConfig struct {
	ACLRepo   repository.ACLRepositoryInterface
	ProxyRepo repository.ProxyRepositoryInterface
	Logger    *zap.Logger
}

// NewACLService creates a new ACL service
func NewACLService(cfg ACLServiceConfig) *ACLService {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	return &ACLService{
		aclRepo:   cfg.ACLRepo,
		proxyRepo: cfg.ProxyRepo,
		logger:    cfg.Logger.Named("acl-service"),
	}
}

// =============================================================================
// Group Management
// =============================================================================

// CreateGroup creates a new ACL group
func (s *ACLService) CreateGroup(group *models.ACLGroup, createdBy int) error {
	// Validate group
	if err := group.Validate(); err != nil {
		return err
	}

	// Check name uniqueness
	existing, err := s.aclRepo.GetGroupByName(group.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("checking group name uniqueness: %w", err)
	}
	if existing != nil {
		return ErrACLGroupNameExists
	}

	// Set creator
	group.CreatedBy = createdBy

	// Set default combination mode if not specified
	if group.CombinationMode == "" {
		group.CombinationMode = models.ACLCombinationModeAny
	}

	if err := s.aclRepo.CreateGroup(group); err != nil {
		return fmt.Errorf("creating ACL group: %w", err)
	}

	s.logger.Info("ACL group created",
		zap.Int("id", group.ID),
		zap.String("name", group.Name),
		zap.Int("created_by", createdBy))

	return nil
}

// GetGroup retrieves an ACL group by ID with all related data
func (s *ACLService) GetGroup(id int) (*models.ACLGroup, error) {
	group, err := s.aclRepo.GetGroupByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrACLGroupNotFound
		}
		return nil, fmt.Errorf("getting ACL group: %w", err)
	}
	return group, nil
}

// GetGroupByName retrieves an ACL group by name
func (s *ACLService) GetGroupByName(name string) (*models.ACLGroup, error) {
	group, err := s.aclRepo.GetGroupByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrACLGroupNotFound
		}
		return nil, fmt.Errorf("getting ACL group by name: %w", err)
	}
	return group, nil
}

// ListGroups returns a paginated list of ACL groups
func (s *ACLService) ListGroups(params ListACLGroupsRequest) (*models.ACLGroupListResponse, error) {
	// Validate and set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	groups, total, err := s.aclRepo.ListGroups(repository.ACLGroupListParams{
		Page:   params.Page,
		Limit:  params.Limit,
		Search: params.Search,
		Sort:   params.Sort,
		Order:  params.Order,
	})
	if err != nil {
		return nil, fmt.Errorf("listing ACL groups: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &models.ACLGroupListResponse{
		Items:      groups,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// UpdateGroup updates an existing ACL group
func (s *ACLService) UpdateGroup(id int, updates *models.ACLGroup) error {
	// Get existing group
	existing, err := s.aclRepo.GetGroupByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Check name uniqueness if name is being changed
	if updates.Name != existing.Name {
		existingByName, err := s.aclRepo.GetGroupByName(updates.Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("checking group name uniqueness: %w", err)
		}
		if existingByName != nil {
			return ErrACLGroupNameExists
		}
	}

	// Apply updates while preserving immutable fields
	existing.Name = updates.Name
	existing.Description = updates.Description
	if updates.CombinationMode != "" {
		existing.CombinationMode = updates.CombinationMode
	}

	// Validate updated group
	if err := existing.Validate(); err != nil {
		return err
	}

	if err := s.aclRepo.UpdateGroup(existing); err != nil {
		return fmt.Errorf("updating ACL group: %w", err)
	}

	s.logger.Info("ACL group updated",
		zap.Int("id", id),
		zap.String("name", existing.Name))

	return nil
}

// DeleteGroup deletes an ACL group and all related data
func (s *ACLService) DeleteGroup(id int) error {
	// Check if group exists
	_, err := s.aclRepo.GetGroupByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	if err := s.aclRepo.DeleteGroup(id); err != nil {
		return fmt.Errorf("deleting ACL group: %w", err)
	}

	s.logger.Info("ACL group deleted", zap.Int("id", id))

	return nil
}

// =============================================================================
// IP Rules
// =============================================================================

// AddIPRule adds an IP rule to an ACL group
func (s *ACLService) AddIPRule(groupID int, rule *models.ACLIPRule) error {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Validate CIDR format
	if err := validateCIDR(rule.CIDR); err != nil {
		return err
	}

	// Validate rule type and other fields
	if err := rule.Validate(); err != nil {
		return err
	}

	rule.ACLGroupID = groupID

	if err := s.aclRepo.CreateIPRule(rule); err != nil {
		return fmt.Errorf("creating IP rule: %w", err)
	}

	s.logger.Info("IP rule added",
		zap.Int("id", rule.ID),
		zap.Int("group_id", groupID),
		zap.String("cidr", rule.CIDR),
		zap.String("rule_type", rule.RuleType))

	return nil
}

// UpdateIPRule updates an existing IP rule
func (s *ACLService) UpdateIPRule(id int, rule *models.ACLIPRule) error {
	// Get existing rule
	existing, err := s.aclRepo.GetIPRuleByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrIPRuleNotFound
		}
		return fmt.Errorf("getting IP rule: %w", err)
	}

	// Validate CIDR format if changed
	if rule.CIDR != existing.CIDR {
		if err := validateCIDR(rule.CIDR); err != nil {
			return err
		}
	}

	// Apply updates
	existing.RuleType = rule.RuleType
	existing.CIDR = rule.CIDR
	existing.Description = rule.Description
	existing.Priority = rule.Priority

	// Validate
	if err := existing.Validate(); err != nil {
		return err
	}

	if err := s.aclRepo.UpdateIPRule(existing); err != nil {
		return fmt.Errorf("updating IP rule: %w", err)
	}

	s.logger.Info("IP rule updated",
		zap.Int("id", id),
		zap.String("cidr", existing.CIDR))

	return nil
}

// DeleteIPRule deletes an IP rule
func (s *ACLService) DeleteIPRule(id int) error {
	// Check if rule exists
	_, err := s.aclRepo.GetIPRuleByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrIPRuleNotFound
		}
		return fmt.Errorf("getting IP rule: %w", err)
	}

	if err := s.aclRepo.DeleteIPRule(id); err != nil {
		return fmt.Errorf("deleting IP rule: %w", err)
	}

	s.logger.Info("IP rule deleted", zap.Int("id", id))

	return nil
}

// =============================================================================
// Basic Auth
// =============================================================================

// AddBasicAuthUser adds a basic auth user to an ACL group
func (s *ACLService) AddBasicAuthUser(groupID int, username, password string) error {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Validate inputs
	if username == "" {
		return models.ErrBasicAuthUsernameEmpty
	}
	if password == "" {
		return models.ErrBasicAuthPasswordEmpty
	}

	// Check if user already exists in this group
	existing, err := s.aclRepo.GetBasicAuthUser(groupID, username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("checking basic auth user: %w", err)
	}
	if existing != nil {
		return ErrBasicAuthUserExists
	}

	user := &models.ACLBasicAuthUser{
		ACLGroupID: groupID,
		Username:   username,
	}

	// Hash password with bcrypt cost 14 (Caddy compatible)
	if err := user.SetPassword(password, BcryptCost); err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.aclRepo.CreateBasicAuthUser(user); err != nil {
		return fmt.Errorf("creating basic auth user: %w", err)
	}

	s.logger.Info("Basic auth user added",
		zap.Int("id", user.ID),
		zap.Int("group_id", groupID),
		zap.String("username", username))

	return nil
}

// UpdateBasicAuthPassword updates a basic auth user's password
func (s *ACLService) UpdateBasicAuthPassword(id int, password string) error {
	// Get existing user
	user, err := s.aclRepo.GetBasicAuthUserByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBasicAuthUserNotFound
		}
		return fmt.Errorf("getting basic auth user: %w", err)
	}

	// Validate password
	if password == "" {
		return models.ErrBasicAuthPasswordEmpty
	}

	// Hash new password
	if err := user.SetPassword(password, BcryptCost); err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.aclRepo.UpdateBasicAuthUser(user); err != nil {
		return fmt.Errorf("updating basic auth user: %w", err)
	}

	s.logger.Info("Basic auth password updated",
		zap.Int("id", id),
		zap.String("username", user.Username))

	return nil
}

// DeleteBasicAuthUser deletes a basic auth user
func (s *ACLService) DeleteBasicAuthUser(id int) error {
	// Check if user exists
	_, err := s.aclRepo.GetBasicAuthUserByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBasicAuthUserNotFound
		}
		return fmt.Errorf("getting basic auth user: %w", err)
	}

	if err := s.aclRepo.DeleteBasicAuthUser(id); err != nil {
		return fmt.Errorf("deleting basic auth user: %w", err)
	}

	s.logger.Info("Basic auth user deleted", zap.Int("id", id))

	return nil
}

// =============================================================================
// External Providers
// =============================================================================

// AddExternalProvider adds an external auth provider to an ACL group
func (s *ACLService) AddExternalProvider(groupID int, provider *models.ACLExternalProvider) error {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Validate provider
	if err := provider.Validate(); err != nil {
		return err
	}

	provider.ACLGroupID = groupID

	if err := s.aclRepo.CreateExternalProvider(provider); err != nil {
		return fmt.Errorf("creating external provider: %w", err)
	}

	s.logger.Info("External provider added",
		zap.Int("id", provider.ID),
		zap.Int("group_id", groupID),
		zap.String("name", provider.Name),
		zap.String("type", provider.ProviderType))

	return nil
}

// UpdateExternalProvider updates an external auth provider
func (s *ACLService) UpdateExternalProvider(id int, provider *models.ACLExternalProvider) error {
	// Get existing provider
	existing, err := s.aclRepo.GetExternalProviderByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrExternalProviderNotFound
		}
		return fmt.Errorf("getting external provider: %w", err)
	}

	// Apply updates
	existing.ProviderType = provider.ProviderType
	existing.Name = provider.Name
	existing.VerifyURL = provider.VerifyURL
	existing.AuthRedirectURL = provider.AuthRedirectURL
	existing.HeadersToCopy = provider.HeadersToCopy

	// Validate
	if err := existing.Validate(); err != nil {
		return err
	}

	if err := s.aclRepo.UpdateExternalProvider(existing); err != nil {
		return fmt.Errorf("updating external provider: %w", err)
	}

	s.logger.Info("External provider updated",
		zap.Int("id", id),
		zap.String("name", existing.Name))

	return nil
}

// DeleteExternalProvider deletes an external auth provider
func (s *ACLService) DeleteExternalProvider(id int) error {
	// Check if provider exists
	_, err := s.aclRepo.GetExternalProviderByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrExternalProviderNotFound
		}
		return fmt.Errorf("getting external provider: %w", err)
	}

	if err := s.aclRepo.DeleteExternalProvider(id); err != nil {
		return fmt.Errorf("deleting external provider: %w", err)
	}

	s.logger.Info("External provider deleted", zap.Int("id", id))

	return nil
}

// =============================================================================
// Waygates Auth Config
// =============================================================================

// GetWaygatesAuth retrieves the Waygates auth config for a group
func (s *ACLService) GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error) {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrACLGroupNotFound
		}
		return nil, fmt.Errorf("getting ACL group: %w", err)
	}

	auth, err := s.aclRepo.GetWaygatesAuth(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWaygatesAuthNotFound
		}
		return nil, fmt.Errorf("getting waygates auth: %w", err)
	}

	return auth, nil
}

// ConfigureWaygatesAuth creates or updates the Waygates auth config for a group
func (s *ACLService) ConfigureWaygatesAuth(groupID int, config *models.ACLWaygatesAuth) error {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Check if config already exists
	existing, err := s.aclRepo.GetWaygatesAuth(groupID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("checking waygates auth: %w", err)
	}

	config.ACLGroupID = groupID

	// Set default session TTL if not specified
	if config.SessionTTL == 0 {
		config.SessionTTL = 86400 // 24 hours
	}

	if existing != nil {
		// Update existing config
		config.ID = existing.ID
		config.CreatedAt = existing.CreatedAt
		if err := s.aclRepo.UpdateWaygatesAuth(config); err != nil {
			return fmt.Errorf("updating waygates auth: %w", err)
		}
		s.logger.Info("Waygates auth updated", zap.Int("group_id", groupID))
	} else {
		// Create new config
		if err := s.aclRepo.CreateWaygatesAuth(config); err != nil {
			return fmt.Errorf("creating waygates auth: %w", err)
		}
		s.logger.Info("Waygates auth created", zap.Int("group_id", groupID))
	}

	return nil
}

// =============================================================================
// Proxy Assignment
// =============================================================================

// AssignToProxy assigns an ACL group to a proxy
func (s *ACLService) AssignToProxy(proxyID, groupID int, pathPattern string, priority int) error {
	// Check proxy exists
	_, err := s.proxyRepo.GetByID(proxyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProxyNotFound
		}
		return fmt.Errorf("getting proxy: %w", err)
	}

	// Check group exists
	_, err = s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Validate path pattern
	if err := validatePathPattern(pathPattern); err != nil {
		return err
	}

	// Set default path pattern if empty
	if pathPattern == "" {
		pathPattern = "/*"
	}

	assignment := &models.ProxyACLAssignment{
		ProxyID:     proxyID,
		ACLGroupID:  groupID,
		PathPattern: pathPattern,
		Priority:    priority,
		Enabled:     true,
	}

	if err := s.aclRepo.CreateProxyACLAssignment(assignment); err != nil {
		// Check for duplicate constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "duplicate") {
			return ErrProxyACLExists
		}
		return fmt.Errorf("creating proxy ACL assignment: %w", err)
	}

	s.logger.Info("Proxy ACL assignment created",
		zap.Int("id", assignment.ID),
		zap.Int("proxy_id", proxyID),
		zap.Int("group_id", groupID),
		zap.String("path_pattern", pathPattern))

	return nil
}

// UpdateProxyAssignment updates a proxy ACL assignment
func (s *ACLService) UpdateProxyAssignment(id int, pathPattern string, priority int, enabled bool) error {
	// Validate path pattern if provided
	if pathPattern != "" {
		if err := validatePathPattern(pathPattern); err != nil {
			return err
		}
	}

	// Fetch existing assignment to preserve ProxyID and ACLGroupID
	existing, err := s.aclRepo.GetProxyACLAssignmentByID(id)
	if err != nil {
		return fmt.Errorf("getting proxy ACL assignment: %w", err)
	}

	// Update only the fields we want to change
	existing.PathPattern = pathPattern
	existing.Priority = priority
	existing.Enabled = enabled

	if err := s.aclRepo.UpdateProxyACLAssignment(existing); err != nil {
		return fmt.Errorf("updating proxy ACL assignment: %w", err)
	}

	s.logger.Info("Proxy ACL assignment updated",
		zap.Int("id", id),
		zap.String("path_pattern", pathPattern),
		zap.Bool("enabled", enabled))

	return nil
}

// RemoveFromProxy removes an ACL group from a proxy
func (s *ACLService) RemoveFromProxy(proxyID, groupID int) error {
	if err := s.aclRepo.DeleteProxyACLAssignmentByProxyAndGroup(proxyID, groupID); err != nil {
		return fmt.Errorf("removing proxy ACL assignment: %w", err)
	}

	s.logger.Info("Proxy ACL assignment removed",
		zap.Int("proxy_id", proxyID),
		zap.Int("group_id", groupID))

	return nil
}

// GetProxyACL returns all ACL assignments for a proxy
func (s *ACLService) GetProxyACL(proxyID int) ([]models.ProxyACLAssignment, error) {
	assignments, err := s.aclRepo.GetProxyACLAssignments(proxyID)
	if err != nil {
		return nil, fmt.Errorf("getting proxy ACL assignments: %w", err)
	}
	return assignments, nil
}

// GetGroupUsage returns all proxies that use a specific ACL group
func (s *ACLService) GetGroupUsage(groupID int) ([]models.ProxyACLAssignment, error) {
	assignments, err := s.aclRepo.GetProxyACLAssignmentsByGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("getting group usage: %w", err)
	}
	return assignments, nil
}

// =============================================================================
// Branding
// =============================================================================

// GetBranding retrieves the login page branding configuration
func (s *ACLService) GetBranding() (*models.ACLBranding, error) {
	branding, err := s.aclRepo.GetBranding()
	if err != nil {
		return nil, fmt.Errorf("getting branding: %w", err)
	}
	return branding, nil
}

// UpdateBranding updates the login page branding configuration
func (s *ACLService) UpdateBranding(branding *models.ACLBranding) error {
	if err := s.aclRepo.UpdateBranding(branding); err != nil {
		return fmt.Errorf("updating branding: %w", err)
	}

	s.logger.Info("Branding updated")

	return nil
}

// =============================================================================
// OAuth Provider Restrictions
// =============================================================================

// ErrOAuthProviderRestrictionNotFound is returned when the OAuth provider restriction is not found
var ErrOAuthProviderRestrictionNotFound = errors.New("OAuth provider restriction not found")

// GetOAuthProviderRestrictions returns all OAuth provider restrictions for a group
func (s *ACLService) GetOAuthProviderRestrictions(groupID int) ([]models.ACLOAuthProviderRestriction, error) {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrACLGroupNotFound
		}
		return nil, fmt.Errorf("getting ACL group: %w", err)
	}

	restrictions, err := s.aclRepo.GetOAuthProviderRestrictions(groupID)
	if err != nil {
		return nil, fmt.Errorf("getting OAuth provider restrictions: %w", err)
	}

	return restrictions, nil
}

// SetOAuthProviderRestriction creates or updates an OAuth provider restriction for a group.
// If a restriction for the provider already exists, it will be updated.
func (s *ACLService) SetOAuthProviderRestriction(groupID int, provider string, emails, domains []string, enabled bool) error {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Validate provider name
	if provider == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	// Check if restriction already exists
	existing, err := s.aclRepo.GetOAuthProviderRestriction(groupID, provider)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("checking OAuth provider restriction: %w", err)
	}

	if existing != nil {
		// Update existing restriction
		existing.AllowedEmails = emails
		existing.AllowedDomains = domains
		existing.Enabled = enabled

		if err := s.aclRepo.UpdateOAuthProviderRestriction(existing); err != nil {
			return fmt.Errorf("updating OAuth provider restriction: %w", err)
		}

		s.logger.Info("OAuth provider restriction updated",
			zap.Int("group_id", groupID),
			zap.String("provider", provider),
			zap.Bool("enabled", enabled))
	} else {
		// Create new restriction
		restriction := &models.ACLOAuthProviderRestriction{
			ACLGroupID:     groupID,
			Provider:       provider,
			AllowedEmails:  emails,
			AllowedDomains: domains,
			Enabled:        enabled,
		}

		if err := s.aclRepo.CreateOAuthProviderRestriction(restriction); err != nil {
			return fmt.Errorf("creating OAuth provider restriction: %w", err)
		}

		s.logger.Info("OAuth provider restriction created",
			zap.Int("group_id", groupID),
			zap.String("provider", provider),
			zap.Bool("enabled", enabled))
	}

	return nil
}

// DeleteOAuthProviderRestriction deletes an OAuth provider restriction for a group
func (s *ACLService) DeleteOAuthProviderRestriction(groupID int, provider string) error {
	// Check group exists
	_, err := s.aclRepo.GetGroupByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("getting ACL group: %w", err)
	}

	// Check if restriction exists
	_, err = s.aclRepo.GetOAuthProviderRestriction(groupID, provider)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOAuthProviderRestrictionNotFound
		}
		return fmt.Errorf("getting OAuth provider restriction: %w", err)
	}

	if err := s.aclRepo.DeleteOAuthProviderRestriction(groupID, provider); err != nil {
		return fmt.Errorf("deleting OAuth provider restriction: %w", err)
	}

	s.logger.Info("OAuth provider restriction deleted",
		zap.Int("group_id", groupID),
		zap.String("provider", provider))

	return nil
}

// =============================================================================
// Access Verification (for forward_auth)
// =============================================================================

// VerifyAccess verifies if a request should be allowed based on ACL rules
func (s *ACLService) VerifyAccess(request *ACLVerifyRequest) (*ACLVerifyResponse, error) {
	response := &ACLVerifyResponse{
		Allowed: false,
		Headers: make(map[string]string),
	}

	// 1. Find proxy by hostname
	proxy, err := s.proxyRepo.GetByHostname(request.Host)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Reason = "proxy not found for hostname"
			return response, nil
		}
		return nil, fmt.Errorf("getting proxy by hostname: %w", err)
	}

	// 2. Get ACL assignments for proxy (ordered by priority)
	assignments, err := s.aclRepo.GetProxyACLAssignments(proxy.ID)
	if err != nil {
		return nil, fmt.Errorf("getting proxy ACL assignments: %w", err)
	}

	// If no ACL assignments, allow by default
	if len(assignments) == 0 {
		response.Allowed = true
		return response, nil
	}

	// 3. Filter assignments by path match
	matchingAssignments := s.filterAssignmentsByPath(assignments, request.Path)
	if len(matchingAssignments) == 0 {
		response.Allowed = true
		return response, nil
	}

	// 4. Process each matching assignment
	for _, assignment := range matchingAssignments {
		if !assignment.Enabled {
			continue
		}

		// Get full group with related data
		group, err := s.aclRepo.GetGroupByID(assignment.ACLGroupID)
		if err != nil {
			s.logger.Warn("Failed to get ACL group for assignment",
				zap.Int("assignment_id", assignment.ID),
				zap.Int("group_id", assignment.ACLGroupID),
				zap.Error(err))
			continue
		}

		// Evaluate access for this group
		accessResult, err := s.evaluateGroupAccess(group, request)
		if err != nil {
			s.logger.Warn("Error evaluating group access",
				zap.Int("group_id", group.ID),
				zap.Error(err))
			continue
		}

		if accessResult.Allowed {
			response.Allowed = true
			response.User = accessResult.User
			response.OAuthEmail = accessResult.OAuthEmail
			response.OAuthProvider = accessResult.OAuthProvider

			// Set headers based on auth type
			if accessResult.User != nil {
				response.Headers["X-Auth-User"] = accessResult.User.Username
				response.Headers["X-Auth-User-ID"] = fmt.Sprintf("%d", accessResult.User.ID)
				response.Headers["X-Auth-User-Email"] = accessResult.User.Email
			} else if accessResult.OAuthEmail != "" {
				// OAuth user (no Waygates user account)
				response.Headers["X-Auth-User-Email"] = accessResult.OAuthEmail
				response.Headers["X-Auth-Provider"] = accessResult.OAuthProvider
			}
			return response, nil
		}

		// Track if user was authenticated but unauthorized
		// This is important for providing proper feedback instead of a redirect loop
		if accessResult.AuthenticatedButUnauthorized {
			response.AuthenticatedButUnauthorized = true
			response.OAuthEmail = accessResult.OAuthEmail
			response.OAuthProvider = accessResult.OAuthProvider
			response.User = accessResult.User
			if accessResult.DenialReason != "" {
				response.DenialDetails = accessResult.DenialReason
			}
		}
	}

	// No access granted
	response.Reason = "access denied by ACL rules"

	// If user is authenticated but unauthorized, don't redirect to login (it would be a loop)
	// Instead, return 403 with the denial details
	if response.AuthenticatedButUnauthorized {
		response.RequiresAuth = false
		if response.DenialDetails == "" {
			response.DenialDetails = "Your account is not authorized to access this resource"
		}
	} else {
		// User is not authenticated - redirect to login
		response.RequiresAuth = true

		// Get branding for redirect URL
		branding, _ := s.aclRepo.GetBranding()
		if branding != nil {
			// Construct login redirect URL
			response.RedirectURL = fmt.Sprintf("/auth/login?redirect=%s%s", request.Host, request.Path)
		}
	}

	return response, nil
}

// evaluateGroupAccess evaluates access rules for a specific ACL group
func (s *ACLService) evaluateGroupAccess(group *models.ACLGroup, request *ACLVerifyRequest) (*groupAccessResult, error) {
	result := &groupAccessResult{Allowed: false}

	// First check IP rules (deny rules take priority)
	ipResult, bypass := s.evaluateIPRules(group.IPRules, request.RemoteIP, group.CombinationMode)

	// If IP is denied, reject immediately
	if ipResult == ipRuleDeny {
		return result, nil
	}

	// If IP bypass and combination mode allows it, grant access without further auth
	if ipResult == ipRuleBypass && group.CombinationMode == models.ACLCombinationModeIPBypass {
		result.Allowed = true
		return result, nil
	}

	// If IP is explicitly allowed (not just bypass), it might be sufficient
	if ipResult == ipRuleAllow && group.CombinationMode == models.ACLCombinationModeIPBypass {
		result.Allowed = true
		return result, nil
	}

	// Check authentication methods based on combination mode
	var authResults []bool

	// Check session token (Waygates auth or OAuth)
	// OAuth sessions should be validated even when WaygatesAuth.Enabled is false,
	// as long as there's a WaygatesAuth config with OAuth restrictions
	if request.SessionToken != "" && group.WaygatesAuth != nil {
		session, err := s.aclRepo.GetSessionByToken(request.SessionToken)
		if err == nil && !session.IsExpired() {
			result.HasValidSession = true

			// Check if this is an OAuth session (has email and provider)
			if session.Email != nil && session.Provider != nil {
				result.OAuthEmail = *session.Email
				result.OAuthProvider = *session.Provider
				// Also set user if available (OAuth user may have Waygates account)
				if session.User != nil {
					result.User = session.User
				}

				// OAuth session - check OAuth restrictions
				// OAuth sessions are validated regardless of WaygatesAuth.Enabled
				if s.isOAuthUserAllowed(*session.Email, *session.Provider, group.WaygatesAuth, group.OAuthProviderRestrictions) {
					authResults = append(authResults, true)
				} else {
					// User is authenticated but fails OAuth restrictions
					result.AuthenticatedButUnauthorized = true
					result.DenialReason = s.buildOAuthDenialReason(*session.Email, *session.Provider, group.WaygatesAuth, group.OAuthProviderRestrictions)
				}
			} else if session.User != nil && group.WaygatesAuth.Enabled {
				// Pure Waygates user session (no OAuth info)
				// This requires WaygatesAuth.Enabled since it's a username/password session
				result.User = session.User
				if s.isUserAllowedByWaygatesAuth(session.User, group.WaygatesAuth) {
					authResults = append(authResults, true)
				} else {
					// User is authenticated but fails restrictions
					result.AuthenticatedButUnauthorized = true
					result.DenialReason = "Your account is not authorized to access this resource"
				}
			}
		}
	}

	// Check basic auth
	if request.BasicAuth != nil && len(group.BasicAuthUsers) > 0 {
		for _, authUser := range group.BasicAuthUsers {
			if authUser.Username == request.BasicAuth.Username {
				if authUser.CheckPassword(request.BasicAuth.Password) {
					authResults = append(authResults, true)
					break
				}
			}
		}
	}

	// Evaluate based on combination mode
	switch group.CombinationMode {
	case models.ACLCombinationModeAny:
		// If IP is allowed/bypassed, that's enough
		if bypass || ipResult == ipRuleAllow {
			result.Allowed = true
			return result, nil
		}
		// Or if any auth method passed
		for _, passed := range authResults {
			if passed {
				result.Allowed = true
				return result, nil
			}
		}

	case models.ACLCombinationModeAll:
		// All configured methods must pass
		// For simplicity, if there are auth users and basic auth passed, that's one requirement
		// If there's waygates auth and session is valid, that's another requirement
		// IP rules are handled separately
		hasAuthRequirements := len(group.BasicAuthUsers) > 0 || (group.WaygatesAuth != nil && group.WaygatesAuth.Enabled)
		if !hasAuthRequirements {
			// No auth requirements, IP check was enough
			if ipResult != ipRuleDeny {
				result.Allowed = true
				return result, nil
			}
		}
		// Need all auth results to pass
		if len(authResults) > 0 {
			allPassed := true
			for _, passed := range authResults {
				if !passed {
					allPassed = false
					break
				}
			}
			if allPassed {
				result.Allowed = true
				return result, nil
			}
		}

	case models.ACLCombinationModeIPBypass:
		// IP bypass already handled above
		// Otherwise, need auth
		for _, passed := range authResults {
			if passed {
				result.Allowed = true
				return result, nil
			}
		}
	}

	return result, nil
}

// IP rule evaluation result
type ipRuleResult int

const (
	ipRuleNoMatch ipRuleResult = iota
	ipRuleAllow
	ipRuleDeny
	ipRuleBypass
)

// evaluateIPRules evaluates IP rules against a remote IP
func (s *ACLService) evaluateIPRules(rules []models.ACLIPRule, remoteIP string, combinationMode string) (ipRuleResult, bool) {
	if len(rules) == 0 {
		return ipRuleNoMatch, false
	}

	// Parse the remote IP
	ip := net.ParseIP(remoteIP)
	if ip == nil {
		// Try to extract IP from potential "ip:port" format
		host, _, err := net.SplitHostPort(remoteIP)
		if err == nil {
			ip = net.ParseIP(host)
		}
	}

	if ip == nil {
		s.logger.Warn("Failed to parse remote IP", zap.String("remote_ip", remoteIP))
		return ipRuleNoMatch, false
	}

	// Process rules by priority (already sorted by repository)
	// Deny rules should be checked first
	for _, rule := range rules {
		if rule.RuleType == models.ACLIPRuleTypeDeny {
			_, network, err := net.ParseCIDR(rule.CIDR)
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				return ipRuleDeny, false
			}
		}
	}

	// Then check allow/bypass rules
	for _, rule := range rules {
		_, network, err := net.ParseCIDR(rule.CIDR)
		if err != nil {
			continue
		}

		if network.Contains(ip) {
			switch rule.RuleType {
			case models.ACLIPRuleTypeAllow:
				return ipRuleAllow, false
			case models.ACLIPRuleTypeBypass:
				return ipRuleBypass, true
			}
		}
	}

	return ipRuleNoMatch, false
}

// isUserAllowedByWaygatesAuth checks if a user is allowed by Waygates auth rules
func (s *ACLService) isUserAllowedByWaygatesAuth(user *models.User, auth *models.ACLWaygatesAuth) bool {
	// If no restrictions, allow
	if len(auth.AllowedUsers) == 0 && len(auth.AllowedRoles) == 0 && len(auth.AllowedEmailPatterns) == 0 {
		return true
	}

	// Check allowed users
	for _, allowedUser := range auth.AllowedUsers {
		if user.Username == allowedUser || user.Email == allowedUser {
			return true
		}
	}

	// Check allowed email patterns
	for _, pattern := range auth.AllowedEmailPatterns {
		if matchEmailPattern(user.Email, pattern) {
			return true
		}
	}

	// Note: Role checking would require role information on the user
	// For now, we skip role checks

	return false
}

// isOAuthUserAllowed checks if an OAuth user (email + provider) is allowed by the group's OAuth restrictions.
// It first checks per-provider restrictions, and if none exist or are enabled for the provider,
// falls back to global restrictions in WaygatesAuth.
func (s *ACLService) isOAuthUserAllowed(email, provider string, auth *models.ACLWaygatesAuth, providerRestrictions []models.ACLOAuthProviderRestriction) bool {
	// Step 1: Check if provider is in global AllowedProviders (if set)
	if len(auth.AllowedProviders) > 0 {
		providerAllowed := false
		for _, allowedProvider := range auth.AllowedProviders {
			if strings.EqualFold(provider, allowedProvider) {
				providerAllowed = true
				break
			}
		}
		if !providerAllowed {
			return false
		}
	}

	// Step 2: Look for per-provider restriction for this specific provider
	var perProviderRestriction *models.ACLOAuthProviderRestriction
	for i := range providerRestrictions {
		if strings.EqualFold(providerRestrictions[i].Provider, provider) {
			perProviderRestriction = &providerRestrictions[i]
			break
		}
	}

	// Step 3: If per-provider restriction found and enabled, use it
	if perProviderRestriction != nil && perProviderRestriction.Enabled {
		return s.checkEmailAgainstRestrictions(email, perProviderRestriction.AllowedEmails, perProviderRestriction.AllowedDomains)
	}

	// Step 4: Fall back to global AllowedEmails/AllowedDomains
	return s.checkEmailAgainstRestrictions(email, auth.AllowedEmails, auth.AllowedDomains)
}

// checkEmailAgainstRestrictions checks if an email matches the allowed emails or domains.
// If no restrictions are set (both empty), returns true (allow all).
func (s *ACLService) checkEmailAgainstRestrictions(email string, allowedEmails, allowedDomains []string) bool {
	// If no email restrictions, allow
	if len(allowedEmails) == 0 && len(allowedDomains) == 0 {
		return true
	}

	// Check allowed emails (exact match)
	for _, allowedEmail := range allowedEmails {
		if strings.EqualFold(email, allowedEmail) {
			return true
		}
	}

	// Check allowed domains (e.g., "@company.com")
	for _, domain := range allowedDomains {
		// Normalize domain (ensure it starts with @)
		normalizedDomain := domain
		if !strings.HasPrefix(domain, "@") {
			normalizedDomain = "@" + domain
		}
		if strings.HasSuffix(strings.ToLower(email), strings.ToLower(normalizedDomain)) {
			return true
		}
	}

	return false
}

// buildOAuthDenialReason creates a human-readable explanation for why an OAuth user was denied
func (s *ACLService) buildOAuthDenialReason(email, provider string, auth *models.ACLWaygatesAuth, providerRestrictions []models.ACLOAuthProviderRestriction) string {
	// Check if provider is restricted globally
	if len(auth.AllowedProviders) > 0 {
		providerAllowed := false
		for _, allowedProvider := range auth.AllowedProviders {
			if strings.EqualFold(provider, allowedProvider) {
				providerAllowed = true
				break
			}
		}
		if !providerAllowed {
			return fmt.Sprintf("OAuth provider '%s' is not allowed for this resource. Allowed providers: %s",
				provider, strings.Join(auth.AllowedProviders, ", "))
		}
	}

	// Check if there's a per-provider restriction
	var perProviderRestriction *models.ACLOAuthProviderRestriction
	for i := range providerRestrictions {
		if strings.EqualFold(providerRestrictions[i].Provider, provider) {
			perProviderRestriction = &providerRestrictions[i]
			break
		}
	}

	// Use per-provider restrictions if found and enabled, otherwise use global
	var allowedEmails, allowedDomains []string
	restrictionSource := "global"
	if perProviderRestriction != nil && perProviderRestriction.Enabled {
		allowedEmails = perProviderRestriction.AllowedEmails
		allowedDomains = perProviderRestriction.AllowedDomains
		restrictionSource = fmt.Sprintf("provider-specific (%s)", provider)
	} else {
		allowedEmails = auth.AllowedEmails
		allowedDomains = auth.AllowedDomains
	}

	// Provider is allowed, so the issue is with email/domain
	if len(allowedEmails) > 0 || len(allowedDomains) > 0 {
		var restrictions []string
		if len(allowedEmails) > 0 {
			restrictions = append(restrictions, fmt.Sprintf("specific emails (%d configured)", len(allowedEmails)))
		}
		if len(allowedDomains) > 0 {
			restrictions = append(restrictions, fmt.Sprintf("email domains: %s", strings.Join(allowedDomains, ", ")))
		}
		return fmt.Sprintf("Your email (%s) is not authorized to access this resource. Access is restricted to: %s (%s restrictions)",
			email, strings.Join(restrictions, " or "), restrictionSource)
	}

	return fmt.Sprintf("Your account (%s via %s) is not authorized to access this resource", email, provider)
}

// matchEmailPattern checks if an email matches a pattern (supports wildcard *)
func matchEmailPattern(email, pattern string) bool {
	// Simple pattern matching with * wildcard
	if pattern == "*" {
		return true
	}

	// Handle patterns like *@domain.com
	if strings.HasPrefix(pattern, "*@") {
		domain := strings.TrimPrefix(pattern, "*@")
		return strings.HasSuffix(email, "@"+domain)
	}

	// Exact match
	return email == pattern
}

// filterAssignmentsByPath filters assignments to only those matching the request path
func (s *ACLService) filterAssignmentsByPath(assignments []models.ProxyACLAssignment, path string) []models.ProxyACLAssignment {
	var matching []models.ProxyACLAssignment

	for _, assignment := range assignments {
		if matchPath(assignment.PathPattern, path) {
			matching = append(matching, assignment)
		}
	}

	return matching
}

// matchPath checks if a path matches a pattern
func matchPath(pattern, path string) bool {
	// Handle exact match
	if pattern == path {
		return true
	}

	// Handle wildcard patterns
	if pattern == "/*" || pattern == "*" {
		return true
	}

	// Handle prefix patterns like /api/*
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}

	// Handle patterns like /api/*/users
	if strings.Contains(pattern, "*") {
		// Convert to a simple regex-like match
		parts := strings.Split(pattern, "*")
		remaining := path
		for i, part := range parts {
			if part == "" {
				continue
			}
			idx := strings.Index(remaining, part)
			if idx == -1 {
				return false
			}
			if i == 0 && idx != 0 {
				return false // First part must match at start
			}
			remaining = remaining[idx+len(part):]
		}
		return true
	}

	return false
}

// =============================================================================
// Session Management
// =============================================================================

// CreateSessionParams holds parameters for creating a session
type CreateSessionParams struct {
	UserID    *int   // Waygates user ID (nil for OAuth-only sessions)
	ProxyID   *int   // Optional proxy ID
	IP        string // Client IP address
	UserAgent string // Client user agent
	TTL       int    // Session TTL in seconds (default 24h)
	// OAuth fields (for OAuth-only sessions)
	Email    string // OAuth user's email
	Provider string // OAuth provider (google, github, etc.)
}

// CreateSession creates a new ACL session for a Waygates user
func (s *ACLService) CreateSession(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
	return s.CreateSessionWithParams(CreateSessionParams{
		UserID:    &userID,
		ProxyID:   proxyID,
		IP:        ip,
		UserAgent: userAgent,
		TTL:       ttl,
	})
}

// CreateOAuthSession creates a new ACL session for an OAuth user (without Waygates account)
func (s *ACLService) CreateOAuthSession(email, provider string, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
	return s.CreateSessionWithParams(CreateSessionParams{
		ProxyID:   proxyID,
		IP:        ip,
		UserAgent: userAgent,
		TTL:       ttl,
		Email:     email,
		Provider:  provider,
	})
}

// CreateSessionWithParams creates a new ACL session with flexible parameters
func (s *ACLService) CreateSessionWithParams(params CreateSessionParams) (*models.ACLSession, error) {
	// Generate secure random token
	token, err := generateSecureToken(SessionTokenLength)
	if err != nil {
		return nil, fmt.Errorf("generating session token: %w", err)
	}

	// Set default TTL if not specified
	if params.TTL <= 0 {
		params.TTL = 86400 // 24 hours
	}

	session := &models.ACLSession{
		SessionToken: token,
		UserID:       params.UserID,
		ProxyID:      params.ProxyID,
		ExpiresAt:    time.Now().Add(time.Duration(params.TTL) * time.Second),
	}

	if params.IP != "" {
		session.IPAddress = &params.IP
	}
	if params.UserAgent != "" {
		session.UserAgent = &params.UserAgent
	}

	// Set OAuth fields if provided
	if params.Email != "" {
		session.Email = &params.Email
	}
	if params.Provider != "" {
		session.Provider = &params.Provider
	}

	if err := s.aclRepo.CreateSession(session); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	// Log session creation
	if params.UserID != nil {
		s.logger.Info("ACL session created for user",
			zap.Int("user_id", *params.UserID),
			zap.Int("ttl_seconds", params.TTL))
	} else if params.Email != "" {
		s.logger.Info("ACL session created for OAuth user",
			zap.String("email", params.Email),
			zap.String("provider", params.Provider),
			zap.Int("ttl_seconds", params.TTL))
	}

	return session, nil
}

// ValidateSession validates a session token and returns the session
func (s *ACLService) ValidateSession(token string) (*models.ACLSession, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	session, err := s.aclRepo.GetSessionByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("getting session: %w", err)
	}

	if session.IsExpired() {
		// Clean up expired session
		_ = s.aclRepo.DeleteSession(token)
		return nil, ErrSessionExpired
	}

	return session, nil
}

// RevokeSession revokes a session by token
func (s *ACLService) RevokeSession(token string) error {
	if err := s.aclRepo.DeleteSession(token); err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}

	s.logger.Info("ACL session revoked")

	return nil
}

// RevokeUserSessions revokes all sessions for a user
func (s *ACLService) RevokeUserSessions(userID int) error {
	if err := s.aclRepo.DeleteUserSessions(userID); err != nil {
		return fmt.Errorf("revoking user sessions: %w", err)
	}

	s.logger.Info("All user sessions revoked", zap.Int("user_id", userID))

	return nil
}

// CleanupExpiredSessions removes all expired sessions
func (s *ACLService) CleanupExpiredSessions() (int64, error) {
	count, err := s.aclRepo.DeleteExpiredSessions()
	if err != nil {
		return 0, fmt.Errorf("cleaning up expired sessions: %w", err)
	}

	if count > 0 {
		s.logger.Info("Expired sessions cleaned up", zap.Int64("count", count))
	}

	return count, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// validateCIDR validates that a string is a valid CIDR notation
func validateCIDR(cidr string) error {
	// Handle single IP addresses by appending /32 or /128
	if !strings.Contains(cidr, "/") {
		ip := net.ParseIP(cidr)
		if ip == nil {
			return ErrInvalidCIDR
		}
		// It's a valid IP but not CIDR - this is acceptable
		return nil
	}

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return ErrInvalidCIDR
	}

	return nil
}

// validatePathPattern validates a path pattern
func validatePathPattern(pattern string) error {
	if pattern == "" {
		return nil // Empty is valid, will default to /*
	}

	// Must start with /
	if !strings.HasPrefix(pattern, "/") && pattern != "*" {
		return ErrInvalidPathPattern
	}

	// Check for invalid characters
	if strings.ContainsAny(pattern, "?#") {
		return ErrInvalidPathPattern
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
