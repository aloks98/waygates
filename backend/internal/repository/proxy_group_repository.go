package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/proxygroup"
)

// ProxyGroupRepository is the data access layer for ProxyGroup.
type ProxyGroupRepository struct {
	db *gorm.DB
}

func NewProxyGroupRepository(db *gorm.DB) *ProxyGroupRepository {
	return &ProxyGroupRepository{db: db}
}

// ProxyGroupListParams holds parameters for listing proxy groups.
type ProxyGroupListParams struct {
	Page   int
	Limit  int
	Search string
	Sort   string
	Order  string
}

func (r *ProxyGroupRepository) ListAll() ([]models.ProxyGroup, error) {
	var groups []models.ProxyGroup
	if err := r.db.Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("listing proxy groups: %w", err)
	}
	return groups, nil
}

func (r *ProxyGroupRepository) ListAllACLAssignments() ([]models.ProxyGroupACLAssignment, error) {
	var out []models.ProxyGroupACLAssignment
	if err := r.db.Find(&out).Error; err != nil {
		return nil, fmt.Errorf("listing proxy group ACL assignments: %w", err)
	}
	return out, nil
}

// List returns a paginated page of groups with MemberCount populated.
func (r *ProxyGroupRepository) List(params ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
	var groups []models.ProxyGroup
	var total int64

	query := r.db.Model(&models.ProxyGroup{})
	if params.Search != "" {
		query = query.Where("name LIKE ?", "%"+params.Search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := "created_at"
	if validField, ok := allowedGroupSortFields[params.Sort]; ok {
		sortField = validField
	}
	sortOrder := "DESC"
	if validOrder, ok := allowedSortOrders[params.Order]; ok {
		sortOrder = validOrder
	}

	offset := (params.Page - 1) * params.Limit
	if err := query.Order(sortField + " " + sortOrder).
		Offset(offset).Limit(params.Limit).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	if len(groups) > 0 {
		ids := make([]int, len(groups))
		for i := range groups {
			ids[i] = groups[i].ID
		}
		type countRow struct {
			GroupID int
			N       int
		}
		var rows []countRow
		if err := r.db.Table("proxies").
			Select("group_id, COUNT(*) AS n").
			Where("group_id IN ?", ids).
			Group("group_id").Scan(&rows).Error; err != nil {
			return nil, 0, fmt.Errorf("counting proxy group members: %w", err)
		}
		byID := make(map[int]int, len(rows))
		for _, row := range rows {
			byID[row.GroupID] = row.N
		}
		for i := range groups {
			groups[i].MemberCount = byID[groups[i].ID]
		}
	}

	return groups, total, nil
}

var allowedGroupSortFields = map[string]string{
	"id": "id", "name": "name", "base_domain": "base_domain",
	"created_at": "created_at", "updated_at": "updated_at",
}

func (r *ProxyGroupRepository) GetByID(id int) (*models.ProxyGroup, error) {
	var g models.ProxyGroup
	if err := r.db.First(&g, id).Error; err != nil {
		return nil, err
	}
	n, err := r.MemberCount(id)
	if err != nil {
		return nil, err
	}
	g.MemberCount = int(n)
	return &g, nil
}

func (r *ProxyGroupRepository) Create(g *models.ProxyGroup) error { return r.db.Create(g).Error }

// Update uses Select to write the nullable settings columns explicitly. A plain
// Save would skip nil pointers, making it impossible to clear an inherited
// value back to "the group says nothing".
func (r *ProxyGroupRepository) Update(g *models.ProxyGroup) error {
	return r.db.Model(&models.ProxyGroup{ID: g.ID}).
		Select("name", "description", "base_domain",
			"ssl_enabled", "ssl_forced", "tls_insecure_skip_verify",
			"block_exploits", "custom_headers", "updated_at").
		Updates(g).Error
}

func (r *ProxyGroupRepository) Delete(id int) error {
	return r.db.Delete(&models.ProxyGroup{}, id).Error
}

func (r *ProxyGroupRepository) MemberCount(id int) (int64, error) {
	var n int64
	err := r.db.Model(&models.Proxy{}).Where("group_id = ?", id).Count(&n).Error
	return n, err
}

func (r *ProxyGroupRepository) ListMembers(id int) ([]models.Proxy, error) {
	var out []models.Proxy
	err := r.db.Where("group_id = ?", id).Order("id ASC").Find(&out).Error
	return out, err
}

// UpdateBaseDomainTx sets the group's base_domain and recomputes every
// label-addressed member's materialized hostname, in one transaction. A
// collision trips the unique index on proxies.hostname and rolls the whole
// thing back, so a failed rename writes nothing.
func (r *ProxyGroupRepository) UpdateBaseDomainTx(groupID int, newBase *string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var members []models.Proxy
		if err := tx.Where("group_id = ? AND hostname_label IS NOT NULL", groupID).
			Find(&members).Error; err != nil {
			return fmt.Errorf("loading members: %w", err)
		}

		if newBase == nil && len(members) > 0 {
			return ErrBaseDomainRequiredByMembers
		}

		if err := tx.Model(&models.ProxyGroup{ID: groupID}).
			Select("base_domain", "updated_at").
			Updates(&models.ProxyGroup{BaseDomain: newBase}).Error; err != nil {
			return fmt.Errorf("updating base_domain: %w", err)
		}

		for i := range members {
			host := proxygroup.EffectiveHostname(*members[i].HostnameLabel, *newBase)
			if err := tx.Model(&models.Proxy{}).
				Where("id = ?", members[i].ID).
				Update("hostname", host).Error; err != nil {
				return fmt.Errorf("re-homing proxy %d to %q: %w", members[i].ID, host, err)
			}
		}
		return nil
	})
}

// ErrBaseDomainRequiredByMembers is returned when clearing base_domain would
// leave label-addressed members with no hostname.
var ErrBaseDomainRequiredByMembers = errors.New("group has label-addressed members; base_domain cannot be cleared")

func (r *ProxyGroupRepository) ListACLAssignments(groupID int) ([]models.ProxyGroupACLAssignment, error) {
	var out []models.ProxyGroupACLAssignment
	err := r.db.Preload("ACLGroup").Where("proxy_group_id = ?", groupID).
		Order("priority ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *ProxyGroupRepository) CreateACLAssignment(a *models.ProxyGroupACLAssignment) error {
	return r.db.Create(a).Error
}

func (r *ProxyGroupRepository) UpdateACLAssignment(a *models.ProxyGroupACLAssignment) error {
	return r.db.Model(&models.ProxyGroupACLAssignment{ID: a.ID}).
		Select("path_pattern", "priority", "enabled", "updated_at").Updates(a).Error
}

func (r *ProxyGroupRepository) DeleteACLAssignment(groupID, aclGroupID int) error {
	return r.db.Where("proxy_group_id = ? AND acl_group_id = ?", groupID, aclGroupID).
		Delete(&models.ProxyGroupACLAssignment{}).Error
}
