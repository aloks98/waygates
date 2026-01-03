package repository

import (
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
	Page   int
	Limit  int
	Search string
	Type   string
	Status string // "active" or "inactive"
	Sort   string
	Order  string
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

	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}

	if params.Status == "active" {
		query = query.Where("is_active = ?", true)
	} else if params.Status == "inactive" {
		query = query.Where("is_active = ?", false)
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

	return proxies, total, nil
}

// GetByID retrieves a proxy by ID
func (r *ProxyRepository) GetByID(id int) (*models.Proxy, error) {
	var proxy models.Proxy
	if err := r.db.First(&proxy, id).Error; err != nil {
		return nil, err
	}
	return &proxy, nil
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

// Update updates an existing proxy
func (r *ProxyRepository) Update(proxy *models.Proxy) error {
	return r.db.Updates(proxy).Error
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
	Total         int64            `json:"total"`
	Active        int64            `json:"active"`
	Inactive      int64            `json:"inactive"`
	ByType        map[string]int64 `json:"by_type"`
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
