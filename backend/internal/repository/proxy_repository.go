package repository

import (
	"github.com/aloks98/homelab-proxy/backend/internal/models"
	"gorm.io/gorm"
)

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

	// Apply sorting
	sortField := params.Sort
	if sortField == "" {
		sortField = "created_at"
	}

	sortOrder := params.Order
	if sortOrder == "" {
		sortOrder = "desc"
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
