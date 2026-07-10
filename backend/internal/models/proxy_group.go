package models

import "time"

// ProxyGroup is a named parent that member HTTP proxies inherit configuration
// from. It is NOT an ACLGroup: ACLGroup is an auth grouping, this is a config
// grouping. Every settings field is a pointer; nil means "the group says
// nothing about this", not "false".
type ProxyGroup struct {
	ID          int     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string  `json:"name" gorm:"column:name;type:varchar(255);uniqueIndex;not null"`
	Description *string `json:"description,omitempty" gorm:"column:description;type:text"`
	BaseDomain  *string `json:"base_domain,omitempty" gorm:"column:base_domain;type:varchar(255)"`

	// Inheritable settings. No GORM `default:` tags — a default tag drops an
	// explicit false on INSERT (see models/proxy.go:17-30).
	SSLEnabled            *bool         `json:"ssl_enabled" gorm:"column:ssl_enabled"`
	SSLForced             *bool         `json:"ssl_forced" gorm:"column:ssl_forced"`
	TLSInsecureSkipVerify *bool         `json:"tls_insecure_skip_verify" gorm:"column:tls_insecure_skip_verify"`
	BlockExploits         *bool         `json:"block_exploits" gorm:"column:block_exploits"`
	CustomHeaders         CustomHeaders `json:"custom_headers,omitempty" gorm:"column:custom_headers;type:text"`

	CreatedBy int       `json:"-" gorm:"column:created_by;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// MemberCount is computed by the repository List query, not persisted.
	MemberCount int `json:"member_count" gorm:"-"`
}

func (ProxyGroup) TableName() string { return "proxy_groups" }

// ProxyGroupListResponse is the response for listing proxy groups.
type ProxyGroupListResponse struct {
	Items      []ProxyGroup `json:"items"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	TotalPages int          `json:"total_pages"`
}

// ProxyGroupACLAssignment mirrors ProxyACLAssignment column-for-column so the
// resolver can merge the two sets without translating between shapes.
type ProxyGroupACLAssignment struct {
	ID           int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ProxyGroupID int       `json:"proxy_group_id" gorm:"column:proxy_group_id;not null;index"`
	ACLGroupID   int       `json:"acl_group_id" gorm:"column:acl_group_id;not null;index"`
	PathPattern  string    `json:"path_pattern" gorm:"column:path_pattern;type:varchar(500);not null"`
	Priority     int       `json:"priority" gorm:"column:priority;not null"`
	Enabled      bool      `json:"enabled" gorm:"column:enabled;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	ACLGroup *ACLGroup `json:"acl_group,omitempty" gorm:"foreignKey:ACLGroupID"`
}

func (ProxyGroupACLAssignment) TableName() string { return "proxy_group_acl_assignments" }
