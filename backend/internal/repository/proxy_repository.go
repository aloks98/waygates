package repository

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Allowed sort fields for proxy listing (whitelist to prevent SQL injection)
var allowedSortFields = map[string]string{
	"id":         "id",
	"name":       "name",
	"hostname":   "hostname",
	"type":       "type",
	"is_active":  "is_active",
	"created_at": "created_at",
	"updated_at": "updated_at",
}

// Allowed sort orders (whitelist to prevent SQL injection)
var allowedSortOrders = map[string]string{
	"asc":  "ASC",
	"desc": "DESC",
}

// ProxyRepository handles database operations for proxies
type ProxyRepository struct {
	db *gorm.DB
}

// NewProxyRepository creates a new proxy repository
func NewProxyRepository(db *gorm.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

// ProxyListParams holds parameters for listing proxies
type ProxyListParams struct {
	Page         int
	Limit        int
	Search       string
	Types        []string // Filter by multiple types (IN query)
	TypesExclude []string // Exclude types (NOT IN query)
	Status       string   // "active" or "inactive"
	StatusNot    string   // Exclude status
	SSLEnabled   *bool    // Filter by SSL enabled (nil = no filter)
	Target       string   // Filter by target/upstream address (searches in upstreams JSON)
	Sort         string
	Order        string
}

// List returns a paginated list of proxies
func (r *ProxyRepository) List(params ProxyListParams) ([]models.Proxy, int64, error) {
	var proxies []models.Proxy
	var total int64

	query := r.db.Model(&models.Proxy{})

	// Apply filters
	if params.Search != "" {
		query = query.Where("hostname LIKE ? OR name LIKE ?",
			"%"+params.Search+"%",
			"%"+params.Search+"%")
	}

	// Type filters
	if len(params.Types) > 0 {
		query = query.Where("type IN ?", params.Types)
	}
	if len(params.TypesExclude) > 0 {
		query = query.Where("type NOT IN ?", params.TypesExclude)
	}

	// Status filters
	if params.Status == "active" {
		query = query.Where("is_active = ?", true)
	} else if params.Status == "inactive" {
		query = query.Where("is_active = ?", false)
	}
	if params.StatusNot == "active" {
		query = query.Where("is_active = ?", false)
	} else if params.StatusNot == "inactive" {
		query = query.Where("is_active = ?", true)
	}

	// SSL enabled filter
	if params.SSLEnabled != nil {
		query = query.Where("ssl_enabled = ?", *params.SSLEnabled)
	}

	// Target filter (searches in upstreams JSON and redirect_config)
	if params.Target != "" {
		targetPattern := "%" + params.Target + "%"
		query = query.Where("upstreams LIKE ? OR redirect_config LIKE ?", targetPattern, targetPattern)
	}

	// Count total (before pagination)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := "created_at" // default
	if params.Sort != "" {
		if validField, ok := allowedSortFields[strings.ToLower(params.Sort)]; ok {
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

	// Execute query
	if err := query.Find(&proxies).Error; err != nil {
		return nil, 0, err
	}

	if len(proxies) > 0 {
		ids := make([]int, len(proxies))
		for i := range proxies {
			ids[i] = proxies[i].ID
		}
		var rows []aclSummaryRow
		if err := r.db.
			Table("proxy_acl_assignments").
			Select("proxy_acl_assignments.proxy_id, acl_groups.name").
			Joins("JOIN acl_groups ON acl_groups.id = proxy_acl_assignments.acl_group_id").
			Where("proxy_acl_assignments.proxy_id IN ?", ids).
			Order("proxy_acl_assignments.priority ASC, proxy_acl_assignments.id ASC").
			Scan(&rows).Error; err != nil {
			return nil, 0, fmt.Errorf("loading proxy ACL summaries: %w", err)
		}
		applyACLSummaries(proxies, rows)
	}

	return proxies, total, nil
}

// aclSummaryRow is one (proxy, acl group name) pair from the summary query.
type aclSummaryRow struct {
	ProxyID int
	Name    string
}

// applyACLSummaries populates ACLGroupCount/ACLGroupNames on each proxy from the
// flat rows. Every proxy gets a non-nil names slice (empty when unprotected).
func applyACLSummaries(proxies []models.Proxy, rows []aclSummaryRow) {
	byProxy := make(map[int][]string, len(proxies))
	for _, row := range rows {
		byProxy[row.ProxyID] = append(byProxy[row.ProxyID], row.Name)
	}
	for i := range proxies {
		names := byProxy[proxies[i].ID]
		if names == nil {
			names = []string{}
		}
		proxies[i].ACLGroupNames = names
		proxies[i].ACLGroupCount = len(names)
	}
}

// GetByID retrieves a proxy by ID
func (r *ProxyRepository) GetByID(id int) (*models.Proxy, error) {
	var proxy models.Proxy
	if err := r.db.First(&proxy, id).Error; err != nil {
		return nil, err
	}
	return &proxy, nil
}

// GetByIDs retrieves multiple proxies in a single query (WHERE id IN (...)).
// Missing ids are simply absent from the result — no error — so the caller gets
// only the proxies that still exist. Result order is not guaranteed.
func (r *ProxyRepository) GetByIDs(ids []int) ([]models.Proxy, error) {
	if len(ids) == 0 {
		return []models.Proxy{}, nil
	}
	var proxies []models.Proxy
	if err := r.db.Where("id IN ?", ids).Find(&proxies).Error; err != nil {
		return nil, err
	}
	return proxies, nil
}

// GetByHostname retrieves a proxy by hostname
func (r *ProxyRepository) GetByHostname(hostname string) (*models.Proxy, error) {
	var proxy models.Proxy
	if err := r.db.Where("hostname = ?", hostname).First(&proxy).Error; err != nil {
		return nil, err
	}
	return &proxy, nil
}

// Create creates a new proxy
func (r *ProxyRepository) Create(proxy *models.Proxy) error {
	return r.db.Create(proxy).Error
}

// Update updates an existing proxy. It uses Save rather than Updates so that
// zero-value fields (e.g. disabling SSL, or clearing type-specific config when
// a proxy's type changes) are written instead of silently skipped. Callers are
// expected to pass a fully populated proxy (see ProxyService.UpdateProxy).
func (r *ProxyRepository) Update(proxy *models.Proxy) error {
	return r.db.Save(proxy).Error
}

// Delete deletes a proxy by ID
func (r *ProxyRepository) Delete(id int) error {
	return r.db.Delete(&models.Proxy{}, id).Error
}

// UpdateStatus updates the is_active status of a proxy
func (r *ProxyRepository) UpdateStatus(id int, isActive bool) error {
	return r.db.Model(&models.Proxy{}).Where("id = ?", id).Update("is_active", isActive).Error
}

// HostnameExists checks if a hostname already exists (excluding a specific ID)
func (r *ProxyRepository) HostnameExists(hostname string, excludeID int) (bool, error) {
	var count int64
	query := r.db.Model(&models.Proxy{}).Where("hostname = ?", hostname)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// ProxyStats holds statistics about proxies
type ProxyStats struct {
	Total    int64            `json:"total"`
	Active   int64            `json:"active"`
	Inactive int64            `json:"inactive"`
	ByType   map[string]int64 `json:"by_type"`
}

// GetStats returns statistics about proxies
func (r *ProxyRepository) GetStats() (*ProxyStats, error) {
	stats := &ProxyStats{
		ByType: make(map[string]int64),
	}

	// Get total count
	if err := r.db.Model(&models.Proxy{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Get active count
	if err := r.db.Model(&models.Proxy{}).Where("is_active = ?", true).Count(&stats.Active).Error; err != nil {
		return nil, err
	}

	// Calculate inactive
	stats.Inactive = stats.Total - stats.Active

	// Get counts by type
	type TypeCount struct {
		Type  string
		Count int64
	}
	var typeCounts []TypeCount
	if err := r.db.Model(&models.Proxy{}).
		Select("type, count(*) as count").
		Group("type").
		Scan(&typeCounts).Error; err != nil {
		return nil, err
	}

	for _, tc := range typeCounts {
		stats.ByType[tc.Type] = tc.Count
	}

	return stats, nil
}
