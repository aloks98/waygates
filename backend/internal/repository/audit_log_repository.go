package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Allowed sort fields for audit log listing (whitelist to prevent SQL injection)
var auditLogAllowedSortFields = map[string]string{
	"id":            "id",
	"action":        "action",
	"resource_type": "resource_type",
	"resource_name": "resource_name",
	"status":        "status",
	"created_at":    "created_at",
	"user_id":       "user_id",
}

// AuditLogRepository handles database operations for audit logs
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// AuditLogListParams holds parameters for listing audit logs
type AuditLogListParams struct {
	Page                 int
	Limit                int
	Search               string
	Actions              []string // Multiple actions (IN filter)
	ActionsExclude       []string // Multiple actions to exclude (NOT IN filter)
	ResourceTypes        []string // Multiple resource types (IN filter)
	ResourceTypesExclude []string // Multiple resource types to exclude (NOT IN filter)
	UserID               *int
	Status               string
	StatusExclude        string // Status to exclude
	IPAddress            string // Exact match
	IPAddressContains    string // LIKE %value%
	IPAddressStartsWith  string // LIKE value%
	IPAddressEndsWith    string // LIKE %value
	IPAddressNot         string // Not equal
	DateFrom             *time.Time
	DateTo               *time.Time
	Sort                 string
	Order                string
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

// GetByID retrieves an audit log by ID
func (r *AuditLogRepository) GetByID(id int) (*models.AuditLog, error) {
	var log models.AuditLog
	if err := r.db.Preload("User").First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// List returns a paginated list of audit logs with filters
func (r *AuditLogRepository) List(params AuditLogListParams) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{})

	// Apply filters
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		query = query.Where("action LIKE ? OR resource_name LIKE ?", searchPattern, searchPattern)
	}

	if len(params.Actions) > 0 {
		query = query.Where("action IN ?", params.Actions)
	}

	if len(params.ActionsExclude) > 0 {
		query = query.Where("action NOT IN ?", params.ActionsExclude)
	}

	if len(params.ResourceTypes) > 0 {
		query = query.Where("resource_type IN ?", params.ResourceTypes)
	}

	if len(params.ResourceTypesExclude) > 0 {
		query = query.Where("resource_type NOT IN ?", params.ResourceTypesExclude)
	}

	if params.IPAddress != "" {
		query = query.Where("ip_address = ?", params.IPAddress)
	}

	if params.IPAddressContains != "" {
		query = query.Where("ip_address LIKE ?", "%"+params.IPAddressContains+"%")
	}

	if params.IPAddressStartsWith != "" {
		query = query.Where("ip_address LIKE ?", params.IPAddressStartsWith+"%")
	}

	if params.IPAddressEndsWith != "" {
		query = query.Where("ip_address LIKE ?", "%"+params.IPAddressEndsWith)
	}

	if params.IPAddressNot != "" {
		query = query.Where("ip_address != ?", params.IPAddressNot)
	}

	if params.UserID != nil {
		query = query.Where("user_id = ?", *params.UserID)
	}

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if params.StatusExclude != "" {
		query = query.Where("status != ?", params.StatusExclude)
	}

	if params.DateFrom != nil {
		query = query.Where("created_at >= ?", *params.DateFrom)
	}

	if params.DateTo != nil {
		query = query.Where("created_at <= ?", *params.DateTo)
	}

	// Count total (before pagination)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := "created_at" // default
	if params.Sort != "" {
		if validField, ok := auditLogAllowedSortFields[strings.ToLower(params.Sort)]; ok {
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

	// Preload user relation
	query = query.Preload("User")

	// Execute query
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetStats returns aggregate statistics for audit logs
func (r *AuditLogRepository) GetStats() (*models.AuditLogStats, error) {
	stats := &models.AuditLogStats{
		ByAction:       make(map[string]int64),
		ByStatus:       make(map[string]int64),
		ByResourceType: make(map[string]int64),
	}

	// Get total count
	if err := r.db.Model(&models.AuditLog{}).Count(&stats.TotalLogs).Error; err != nil {
		return nil, err
	}

	// Get counts by action
	type ActionCount struct {
		Action string
		Count  int64
	}
	var actionCounts []ActionCount
	if err := r.db.Model(&models.AuditLog{}).
		Select("action, count(*) as count").
		Group("action").
		Scan(&actionCounts).Error; err != nil {
		return nil, err
	}
	for _, ac := range actionCounts {
		stats.ByAction[ac.Action] = ac.Count
	}

	// Get counts by status
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := r.db.Model(&models.AuditLog{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	// Get counts by resource type
	type ResourceTypeCount struct {
		ResourceType string
		Count        int64
	}
	var resourceTypeCounts []ResourceTypeCount
	if err := r.db.Model(&models.AuditLog{}).
		Select("resource_type, count(*) as count").
		Where("resource_type IS NOT NULL").
		Group("resource_type").
		Scan(&resourceTypeCounts).Error; err != nil {
		return nil, err
	}
	for _, rtc := range resourceTypeCounts {
		stats.ByResourceType[rtc.ResourceType] = rtc.Count
	}

	// Get recent activity (last 10 entries)
	if err := r.db.Model(&models.AuditLog{}).
		Preload("User").
		Order("created_at DESC").
		Limit(10).
		Find(&stats.RecentActivity).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// DeleteOlderThan deletes audit logs older than the specified time
// This could be used for future retention policies
func (r *AuditLogRepository) DeleteOlderThan(before time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", before).Delete(&models.AuditLog{})
	return result.RowsAffected, result.Error
}

// CountByAction returns the count of logs for a specific action
func (r *AuditLogRepository) CountByAction(action string) (int64, error) {
	var count int64
	err := r.db.Model(&models.AuditLog{}).Where("action = ?", action).Count(&count).Error
	return count, err
}

// CountByUserID returns the count of logs for a specific user
func (r *AuditLogRepository) CountByUserID(userID int) (int64, error) {
	var count int64
	err := r.db.Model(&models.AuditLog{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
