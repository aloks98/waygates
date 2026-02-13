package repository

import (
	"strings"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Allowed sort fields for L4 proxy listing (whitelist to prevent SQL injection)
var l4ProxyAllowedSortFields = map[string]string{
	"id":          "id",
	"name":        "name",
	"listen_port": "listen_port",
	"protocol":    "protocol",
	"is_active":   "is_active",
	"created_at":  "created_at",
	"updated_at":  "updated_at",
}

// L4ProxyRepository handles database operations for L4 proxies
type L4ProxyRepository struct {
	db *gorm.DB
}

// NewL4ProxyRepository creates a new L4 proxy repository
func NewL4ProxyRepository(db *gorm.DB) *L4ProxyRepository {
	return &L4ProxyRepository{db: db}
}

// Create creates a new L4 proxy
func (r *L4ProxyRepository) Create(proxy *models.L4Proxy) error {
	return r.db.Create(proxy).Error
}

// GetByID retrieves an L4 proxy by ID with all related data
func (r *L4ProxyRepository) GetByID(id int) (*models.L4Proxy, error) {
	var proxy models.L4Proxy
	if err := r.db.
		Preload("Routes", func(db *gorm.DB) *gorm.DB {
			return db.Order("priority ASC, id ASC")
		}).
		Preload("Creator").
		First(&proxy, id).Error; err != nil {
		return nil, err
	}
	return &proxy, nil
}

// List returns a paginated list of L4 proxies
func (r *L4ProxyRepository) List(params L4ProxyListParams) ([]models.L4Proxy, int64, error) {
	var proxies []models.L4Proxy
	var total int64

	query := r.db.Model(&models.L4Proxy{})

	// Apply search filter on name and description
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// Apply protocol filter
	if params.Protocol != "" {
		query = query.Where("protocol = ?", strings.ToLower(params.Protocol))
	}

	// Apply is_active filter
	if params.IsActive != nil {
		query = query.Where("is_active = ?", *params.IsActive)
	}

	// Count total (before pagination)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := "created_at" // default
	if params.Sort != "" {
		if validField, ok := l4ProxyAllowedSortFields[strings.ToLower(params.Sort)]; ok {
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

	// Preload relations
	query = query.
		Preload("Routes", func(db *gorm.DB) *gorm.DB {
			return db.Order("priority ASC, id ASC")
		}).
		Preload("Creator")

	// Execute query
	if err := query.Find(&proxies).Error; err != nil {
		return nil, 0, err
	}

	return proxies, total, nil
}

// Update updates an existing L4 proxy
func (r *L4ProxyRepository) Update(proxy *models.L4Proxy) error {
	return r.db.Save(proxy).Error
}

// Delete deletes an L4 proxy by ID
// Routes are automatically cascade deleted due to the foreign key constraint
func (r *L4ProxyRepository) Delete(id int) error {
	return r.db.Delete(&models.L4Proxy{}, id).Error
}

// GetByPort retrieves an L4 proxy by listen port and protocol
// This is used to check for port conflicts when creating or updating proxies
func (r *L4ProxyRepository) GetByPort(port int, protocol string) (*models.L4Proxy, error) {
	var proxy models.L4Proxy
	if err := r.db.
		Where("listen_port = ? AND protocol = ?", port, strings.ToLower(protocol)).
		First(&proxy).Error; err != nil {
		return nil, err
	}
	return &proxy, nil
}
