package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Allowed sort fields for ACL group listing (whitelist to prevent SQL injection)
var aclGroupAllowedSortFields = map[string]string{
	"id":         "id",
	"name":       "name",
	"created_at": "created_at",
	"updated_at": "updated_at",
}

// ACLGroupListParams holds parameters for listing ACL groups
type ACLGroupListParams struct {
	Page   int
	Limit  int
	Search string // Search in name and description
	Sort   string // created_at, name, updated_at
	Order  string // asc, desc
}

// ACLRepositoryInterface defines the interface for ACL database operations
type ACLRepositoryInterface interface {
	// ACL Groups
	CreateGroup(group *models.ACLGroup) error
	GetGroupByID(id int) (*models.ACLGroup, error)
	GetGroupByName(name string) (*models.ACLGroup, error)
	ListGroups(params ACLGroupListParams) ([]models.ACLGroup, int64, error)
	UpdateGroup(group *models.ACLGroup) error
	DeleteGroup(id int) error

	// IP Rules
	CreateIPRule(rule *models.ACLIPRule) error
	GetIPRuleByID(id int) (*models.ACLIPRule, error)
	ListIPRules(groupID int) ([]models.ACLIPRule, error)
	UpdateIPRule(rule *models.ACLIPRule) error
	DeleteIPRule(id int) error

	// Basic Auth Users
	CreateBasicAuthUser(user *models.ACLBasicAuthUser) error
	GetBasicAuthUserByID(id int) (*models.ACLBasicAuthUser, error)
	GetBasicAuthUser(groupID int, username string) (*models.ACLBasicAuthUser, error)
	ListBasicAuthUsers(groupID int) ([]models.ACLBasicAuthUser, error)
	UpdateBasicAuthUser(user *models.ACLBasicAuthUser) error
	DeleteBasicAuthUser(id int) error

	// External Providers
	CreateExternalProvider(provider *models.ACLExternalProvider) error
	GetExternalProviderByID(id int) (*models.ACLExternalProvider, error)
	ListExternalProviders(groupID int) ([]models.ACLExternalProvider, error)
	UpdateExternalProvider(provider *models.ACLExternalProvider) error
	DeleteExternalProvider(id int) error

	// Waygates Auth
	GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error)
	CreateWaygatesAuth(auth *models.ACLWaygatesAuth) error
	UpdateWaygatesAuth(auth *models.ACLWaygatesAuth) error
	DeleteWaygatesAuth(groupID int) error

	// Proxy ACL Assignments
	CreateProxyACLAssignment(assignment *models.ProxyACLAssignment) error
	GetProxyACLAssignmentByID(id int) (*models.ProxyACLAssignment, error)
	GetProxyACLAssignments(proxyID int) ([]models.ProxyACLAssignment, error)
	GetProxyACLAssignmentsByGroup(groupID int) ([]models.ProxyACLAssignment, error)
	UpdateProxyACLAssignment(assignment *models.ProxyACLAssignment) error
	DeleteProxyACLAssignment(id int) error
	DeleteProxyACLAssignmentByProxyAndGroup(proxyID, groupID int) error

	// Branding
	GetBranding() (*models.ACLBranding, error)
	UpdateBranding(branding *models.ACLBranding) error

	// Sessions
	CreateSession(session *models.ACLSession) error
	GetSessionByToken(token string) (*models.ACLSession, error)
	DeleteSession(token string) error
	DeleteExpiredSessions() (int64, error)
	DeleteUserSessions(userID int) error
	DeleteProxySessions(proxyID int) error
}

// ACLRepository handles database operations for ACL
type ACLRepository struct {
	db *gorm.DB
}

// NewACLRepository creates a new ACL repository
func NewACLRepository(db *gorm.DB) *ACLRepository {
	return &ACLRepository{db: db}
}

// Ensure ACLRepository implements ACLRepositoryInterface
var _ ACLRepositoryInterface = (*ACLRepository)(nil)

// =============================================================================
// ACL Groups
// =============================================================================

// CreateGroup creates a new ACL group
func (r *ACLRepository) CreateGroup(group *models.ACLGroup) error {
	return r.db.Create(group).Error
}

// GetGroupByID retrieves an ACL group by ID with all related data
func (r *ACLRepository) GetGroupByID(id int) (*models.ACLGroup, error) {
	var group models.ACLGroup
	if err := r.db.
		Preload("Creator").
		Preload("IPRules").
		Preload("BasicAuthUsers").
		Preload("ExternalProviders").
		Preload("WaygatesAuth").
		First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// GetGroupByName retrieves an ACL group by name
func (r *ACLRepository) GetGroupByName(name string) (*models.ACLGroup, error) {
	var group models.ACLGroup
	if err := r.db.Where("name = ?", name).First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// ListGroups returns a paginated list of ACL groups
func (r *ACLRepository) ListGroups(params ACLGroupListParams) ([]models.ACLGroup, int64, error) {
	var groups []models.ACLGroup
	var total int64

	query := r.db.Model(&models.ACLGroup{})

	// Apply search filter
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// Count total (before pagination)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := "created_at" // default
	if params.Sort != "" {
		if validField, ok := aclGroupAllowedSortFields[strings.ToLower(params.Sort)]; ok {
			sortField = validField
		}
	}

	sortOrder := "DESC" // default
	if params.Order != "" {
		if validOrder, ok := allowedSortOrders[strings.ToLower(params.Order)]; ok {
			sortOrder = validOrder
		}
	}

	query = query.Order(sortField + " " + sortOrder)

	// Apply pagination
	offset := (params.Page - 1) * params.Limit
	query = query.Offset(offset).Limit(params.Limit)

	// Preload creator relation
	query = query.Preload("Creator")

	// Execute query
	if err := query.Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

// UpdateGroup updates an existing ACL group
func (r *ACLRepository) UpdateGroup(group *models.ACLGroup) error {
	return r.db.Save(group).Error
}

// DeleteGroup deletes an ACL group by ID
// This will cascade delete related IP rules, basic auth users, external providers, and waygates auth
func (r *ACLRepository) DeleteGroup(id int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete related records first
		if err := tx.Where("acl_group_id = ?", id).Delete(&models.ACLIPRule{}).Error; err != nil {
			return err
		}
		if err := tx.Where("acl_group_id = ?", id).Delete(&models.ACLBasicAuthUser{}).Error; err != nil {
			return err
		}
		if err := tx.Where("acl_group_id = ?", id).Delete(&models.ACLExternalProvider{}).Error; err != nil {
			return err
		}
		if err := tx.Where("acl_group_id = ?", id).Delete(&models.ACLWaygatesAuth{}).Error; err != nil {
			return err
		}
		if err := tx.Where("acl_group_id = ?", id).Delete(&models.ProxyACLAssignment{}).Error; err != nil {
			return err
		}

		// Delete the group
		return tx.Delete(&models.ACLGroup{}, id).Error
	})
}

// =============================================================================
// IP Rules
// =============================================================================

// CreateIPRule creates a new IP rule
func (r *ACLRepository) CreateIPRule(rule *models.ACLIPRule) error {
	return r.db.Create(rule).Error
}

// GetIPRuleByID retrieves an IP rule by ID
func (r *ACLRepository) GetIPRuleByID(id int) (*models.ACLIPRule, error) {
	var rule models.ACLIPRule
	if err := r.db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListIPRules returns all IP rules for a group, ordered by priority
func (r *ACLRepository) ListIPRules(groupID int) ([]models.ACLIPRule, error) {
	var rules []models.ACLIPRule
	if err := r.db.Where("acl_group_id = ?", groupID).Order("priority ASC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// UpdateIPRule updates an existing IP rule
func (r *ACLRepository) UpdateIPRule(rule *models.ACLIPRule) error {
	return r.db.Save(rule).Error
}

// DeleteIPRule deletes an IP rule by ID
func (r *ACLRepository) DeleteIPRule(id int) error {
	return r.db.Delete(&models.ACLIPRule{}, id).Error
}

// =============================================================================
// Basic Auth Users
// =============================================================================

// CreateBasicAuthUser creates a new basic auth user
func (r *ACLRepository) CreateBasicAuthUser(user *models.ACLBasicAuthUser) error {
	return r.db.Create(user).Error
}

// GetBasicAuthUserByID retrieves a basic auth user by ID
func (r *ACLRepository) GetBasicAuthUserByID(id int) (*models.ACLBasicAuthUser, error) {
	var user models.ACLBasicAuthUser
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetBasicAuthUser retrieves a basic auth user by group ID and username
func (r *ACLRepository) GetBasicAuthUser(groupID int, username string) (*models.ACLBasicAuthUser, error) {
	var user models.ACLBasicAuthUser
	if err := r.db.Where("acl_group_id = ? AND username = ?", groupID, username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ListBasicAuthUsers returns all basic auth users for a group
func (r *ACLRepository) ListBasicAuthUsers(groupID int) ([]models.ACLBasicAuthUser, error) {
	var users []models.ACLBasicAuthUser
	if err := r.db.Where("acl_group_id = ?", groupID).Order("username ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateBasicAuthUser updates an existing basic auth user
func (r *ACLRepository) UpdateBasicAuthUser(user *models.ACLBasicAuthUser) error {
	return r.db.Save(user).Error
}

// DeleteBasicAuthUser deletes a basic auth user by ID
func (r *ACLRepository) DeleteBasicAuthUser(id int) error {
	return r.db.Delete(&models.ACLBasicAuthUser{}, id).Error
}

// =============================================================================
// External Providers
// =============================================================================

// CreateExternalProvider creates a new external provider
func (r *ACLRepository) CreateExternalProvider(provider *models.ACLExternalProvider) error {
	return r.db.Create(provider).Error
}

// GetExternalProviderByID retrieves an external provider by ID
func (r *ACLRepository) GetExternalProviderByID(id int) (*models.ACLExternalProvider, error) {
	var provider models.ACLExternalProvider
	if err := r.db.First(&provider, id).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

// ListExternalProviders returns all external providers for a group
func (r *ACLRepository) ListExternalProviders(groupID int) ([]models.ACLExternalProvider, error) {
	var providers []models.ACLExternalProvider
	if err := r.db.Where("acl_group_id = ?", groupID).Order("name ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// UpdateExternalProvider updates an existing external provider
func (r *ACLRepository) UpdateExternalProvider(provider *models.ACLExternalProvider) error {
	return r.db.Save(provider).Error
}

// DeleteExternalProvider deletes an external provider by ID
func (r *ACLRepository) DeleteExternalProvider(id int) error {
	return r.db.Delete(&models.ACLExternalProvider{}, id).Error
}

// =============================================================================
// Waygates Auth
// =============================================================================

// GetWaygatesAuth retrieves the Waygates auth config for a group
func (r *ACLRepository) GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error) {
	var auth models.ACLWaygatesAuth
	if err := r.db.Where("acl_group_id = ?", groupID).First(&auth).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

// CreateWaygatesAuth creates a new Waygates auth config
func (r *ACLRepository) CreateWaygatesAuth(auth *models.ACLWaygatesAuth) error {
	return r.db.Create(auth).Error
}

// UpdateWaygatesAuth updates an existing Waygates auth config
func (r *ACLRepository) UpdateWaygatesAuth(auth *models.ACLWaygatesAuth) error {
	return r.db.Save(auth).Error
}

// DeleteWaygatesAuth deletes the Waygates auth config for a group
func (r *ACLRepository) DeleteWaygatesAuth(groupID int) error {
	return r.db.Where("acl_group_id = ?", groupID).Delete(&models.ACLWaygatesAuth{}).Error
}

// =============================================================================
// Proxy ACL Assignments
// =============================================================================

// CreateProxyACLAssignment creates a new proxy ACL assignment
func (r *ACLRepository) CreateProxyACLAssignment(assignment *models.ProxyACLAssignment) error {
	return r.db.Create(assignment).Error
}

// GetProxyACLAssignmentByID retrieves a proxy ACL assignment by ID
func (r *ACLRepository) GetProxyACLAssignmentByID(id int) (*models.ProxyACLAssignment, error) {
	var assignment models.ProxyACLAssignment
	if err := r.db.First(&assignment, id).Error; err != nil {
		return nil, err
	}
	return &assignment, nil
}

// GetProxyACLAssignments returns all ACL assignments for a proxy, ordered by priority.
// This preloads all ACL group relations needed for Caddyfile generation.
func (r *ACLRepository) GetProxyACLAssignments(proxyID int) ([]models.ProxyACLAssignment, error) {
	var assignments []models.ProxyACLAssignment
	if err := r.db.
		Preload("ACLGroup").
		Preload("ACLGroup.IPRules").
		Preload("ACLGroup.BasicAuthUsers").
		Preload("ACLGroup.WaygatesAuth").
		Preload("ACLGroup.ExternalProviders").
		Where("proxy_id = ?", proxyID).
		Order("priority ASC, id ASC").
		Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

// GetProxyACLAssignmentsByGroup returns all ACL assignments for a group
func (r *ACLRepository) GetProxyACLAssignmentsByGroup(groupID int) ([]models.ProxyACLAssignment, error) {
	var assignments []models.ProxyACLAssignment
	if err := r.db.
		Preload("Proxy").
		Where("acl_group_id = ?", groupID).
		Order("priority ASC, id ASC").
		Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

// UpdateProxyACLAssignment updates an existing proxy ACL assignment
func (r *ACLRepository) UpdateProxyACLAssignment(assignment *models.ProxyACLAssignment) error {
	return r.db.Save(assignment).Error
}

// DeleteProxyACLAssignment deletes a proxy ACL assignment by ID
func (r *ACLRepository) DeleteProxyACLAssignment(id int) error {
	return r.db.Delete(&models.ProxyACLAssignment{}, id).Error
}

// DeleteProxyACLAssignmentByProxyAndGroup deletes a proxy ACL assignment by proxy ID and group ID
func (r *ACLRepository) DeleteProxyACLAssignmentByProxyAndGroup(proxyID, groupID int) error {
	return r.db.Where("proxy_id = ? AND acl_group_id = ?", proxyID, groupID).Delete(&models.ProxyACLAssignment{}).Error
}

// =============================================================================
// Branding
// =============================================================================

// GetBranding retrieves the singleton branding configuration
// If no branding exists, it creates the default configuration
func (r *ACLRepository) GetBranding() (*models.ACLBranding, error) {
	var branding models.ACLBranding
	err := r.db.First(&branding, 1).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create default branding
			branding = models.ACLBranding{
				ID:              1,
				PrimaryColor:    "#3b82f6",
				BackgroundColor: "#ffffff",
				Title:           "Login Required",
			}
			if createErr := r.db.Create(&branding).Error; createErr != nil {
				return nil, createErr
			}
			return &branding, nil
		}
		return nil, err
	}

	return &branding, nil
}

// UpdateBranding updates the singleton branding configuration
func (r *ACLRepository) UpdateBranding(branding *models.ACLBranding) error {
	// Ensure we're updating the singleton row
	branding.ID = 1
	return r.db.Save(branding).Error
}

// =============================================================================
// Sessions
// =============================================================================

// CreateSession creates a new ACL session
func (r *ACLRepository) CreateSession(session *models.ACLSession) error {
	return r.db.Create(session).Error
}

// GetSessionByToken retrieves an ACL session by token
func (r *ACLRepository) GetSessionByToken(token string) (*models.ACLSession, error) {
	var session models.ACLSession
	if err := r.db.
		Preload("User").
		Preload("Proxy").
		Where("session_token = ?", token).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession deletes an ACL session by token
func (r *ACLRepository) DeleteSession(token string) error {
	return r.db.Where("session_token = ?", token).Delete(&models.ACLSession{}).Error
}

// DeleteExpiredSessions deletes all expired sessions
func (r *ACLRepository) DeleteExpiredSessions() (int64, error) {
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&models.ACLSession{})
	return result.RowsAffected, result.Error
}

// DeleteUserSessions deletes all sessions for a user
func (r *ACLRepository) DeleteUserSessions(userID int) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.ACLSession{}).Error
}

// DeleteProxySessions deletes all sessions for a proxy
func (r *ACLRepository) DeleteProxySessions(proxyID int) error {
	return r.db.Where("proxy_id = ?", proxyID).Delete(&models.ACLSession{}).Error
}
