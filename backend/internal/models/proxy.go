package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Proxy represents a proxy configuration
type Proxy struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Type        string    `json:"type" gorm:"type:varchar(50);not null"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Hostname    string    `json:"hostname" gorm:"type:varchar(255);uniqueIndex;not null"`
	Description *string   `json:"description,omitempty" gorm:"type:text"`
	SSLEnabled  bool      `json:"ssl_enabled" gorm:"default:true;not null"`
	SSLForced   bool      `json:"ssl_forced" gorm:"default:true;not null"`
	IsActive    bool      `json:"is_active" gorm:"default:true;not null"`
	CreatedBy   int       `json:"-" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Type-specific fields (stored as JSON in database)
	Upstreams             interface{} `json:"upstreams,omitempty" gorm:"type:text;serializer:json"`
	LoadBalancing         JSONField   `json:"load_balancing,omitempty" gorm:"type:text"`
	BlockExploits         bool        `json:"block_exploits" gorm:"default:true;not null"`
	TLSInsecureSkipVerify bool        `json:"tls_insecure_skip_verify" gorm:"default:false;not null"`
	CustomHeaders         JSONField   `json:"custom_headers,omitempty" gorm:"type:text"`
	RedirectConfig        JSONField   `json:"redirect,omitempty" gorm:"type:text;column:redirect_config"`
	StaticConfig          JSONField   `json:"static,omitempty" gorm:"type:text;column:static_config"`

	// Relations (will be populated when needed)
	Creator *User `json:"created_by,omitempty" gorm:"foreignKey:CreatedBy"`
}

// TableName specifies the table name for GORM
func (Proxy) TableName() string {
	return "proxies"
}

// JSONField is a custom type for storing JSON in database
type JSONField map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (j JSONField) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database retrieval
func (j *JSONField) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (j JSONField) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(j))
}

// UnmarshalJSON implements custom JSON unmarshaling
func (j *JSONField) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*j = nil
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	*j = result
	return nil
}



// ProxyListResponse is the response for listing proxies
type ProxyListResponse struct {
	Proxies    []Proxy    `json:"proxies"`
	Pagination Pagination `json:"pagination"`
}

// Pagination contains pagination metadata
type Pagination struct {
	CurrentPage  int  `json:"current_page"`
	TotalPages   int  `json:"total_pages"`
	TotalItems   int64 `json:"total_items"`
	ItemsPerPage int  `json:"items_per_page"`
	HasNext      bool `json:"has_next"`
	HasPrev      bool `json:"has_prev"`
}

// ProxyType constants
const (
	ProxyTypeReverseProxy = "reverse_proxy"
	ProxyTypeRedirect     = "redirect"
	ProxyTypeStatic       = "static"
)

// Validate performs basic validation on the proxy
func (p *Proxy) Validate() error {
	if p.Type != ProxyTypeReverseProxy && p.Type != ProxyTypeRedirect && p.Type != ProxyTypeStatic {
		return ErrInvalidProxyType
	}

	if p.Name == "" {
		return ErrProxyNameRequired
	}

	if p.Hostname == "" {
		return ErrProxyHostnameRequired
	}

	// Type-specific validation
	switch p.Type {
	case ProxyTypeReverseProxy:
		if p.Upstreams == nil {
			return ErrUpstreamsRequired
		}
		// Check if Upstreams is a slice/array
		if upstreams, ok := p.Upstreams.([]interface{}); ok {
			if len(upstreams) == 0 {
				return ErrUpstreamsRequired
			}
		}
	case ProxyTypeRedirect:
		if len(p.RedirectConfig) == 0 {
			return ErrRedirectConfigRequired
		}
	case ProxyTypeStatic:
		if len(p.StaticConfig) == 0 {
			return ErrStaticConfigRequired
		}
	}

	return nil
}

// Custom errors
var (
	ErrInvalidProxyType        = &ValidationError{Message: "invalid proxy type"}
	ErrProxyNameRequired       = &ValidationError{Message: "proxy name is required"}
	ErrProxyHostnameRequired   = &ValidationError{Message: "proxy hostname is required"}
	ErrUpstreamsRequired       = &ValidationError{Message: "upstreams are required for reverse_proxy type"}
	ErrRedirectConfigRequired  = &ValidationError{Message: "redirect configuration is required for redirect type"}
	ErrStaticConfigRequired    = &ValidationError{Message: "static configuration is required for static type"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
