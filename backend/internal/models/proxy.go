package models

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
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
	Upstreams             interface{}   `json:"upstreams,omitempty" gorm:"type:text;serializer:json"`
	LoadBalancing         JSONField     `json:"load_balancing,omitempty" gorm:"type:text"`
	BlockExploits         bool          `json:"block_exploits" gorm:"default:true;not null"`
	TLSInsecureSkipVerify bool          `json:"tls_insecure_skip_verify" gorm:"default:false;not null"`
	CustomHeaders         CustomHeaders `json:"custom_headers,omitempty" gorm:"type:text"`
	RedirectConfig        JSONField     `json:"redirect,omitempty" gorm:"type:text;column:redirect_config"`
	StaticConfig          JSONField     `json:"static,omitempty" gorm:"type:text;column:static_config"`

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

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	if len(bytes) == 0 {
		*j = nil
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

// CustomHeaders holds custom HTTP headers to set on the request forwarded to the
// upstream (Request) and/or on the response returned to the client (Response).
// It is stored as JSON text. To stay backward compatible it accepts BOTH the
// nested {"request":{...},"response":{...}} shape and the legacy flat
// {"X-Header":"value"} map (interpreted as request headers), and always
// serializes back as the nested shape.
type CustomHeaders struct {
	Request  map[string]string `json:"request,omitempty"`
	Response map[string]string `json:"response,omitempty"`
}

// IsEmpty reports whether no headers are configured in either direction.
func (c CustomHeaders) IsEmpty() bool {
	return len(c.Request) == 0 && len(c.Response) == 0
}

// UnmarshalJSON accepts the nested shape or a legacy flat map. HTTP header values
// are always strings, so a top-level value that is a JSON object marks the nested
// shape; otherwise the whole object is a flat request-header map.
func (c *CustomHeaders) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*c = CustomHeaders{}
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	nested := len(raw) > 0
	for key, val := range raw {
		if key != "request" && key != "response" {
			nested = false
			break
		}
		v := strings.TrimSpace(string(val))
		if len(v) == 0 || v[0] != '{' {
			nested = false
			break
		}
	}

	if nested {
		type alias CustomHeaders // break the UnmarshalJSON recursion
		var a alias
		if err := json.Unmarshal(data, &a); err != nil {
			return err
		}
		*c = CustomHeaders(a)
		return nil
	}

	flat := make(map[string]string)
	if err := json.Unmarshal(data, &flat); err != nil {
		return err
	}
	*c = CustomHeaders{Request: flat}
	return nil
}

// Value implements driver.Valuer: store the nested JSON shape (SQL NULL if empty).
func (c CustomHeaders) Value() (driver.Value, error) {
	if c.IsEmpty() {
		return nil, nil
	}
	type alias CustomHeaders
	return json.Marshal(alias(c))
}

// Scan implements sql.Scanner: load JSON text, tolerating legacy flat rows.
func (c *CustomHeaders) Scan(value interface{}) error {
	if value == nil {
		*c = CustomHeaders{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	if len(bytes) == 0 {
		*c = CustomHeaders{}
		return nil
	}
	return c.UnmarshalJSON(bytes)
}

// ProxyListResponse is the response for listing proxies
type ProxyListResponse struct {
	Items      []Proxy `json:"items"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
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

	if len(p.Name) > 255 {
		return ErrProxyNameTooLong
	}

	if p.Hostname == "" {
		return ErrProxyHostnameRequired
	}

	// Validate hostname format
	if err := validateHostname(p.Hostname); err != nil {
		return err
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

// validateHostname validates that a hostname is properly formatted
func validateHostname(hostname string) error {
	hostname = strings.TrimSpace(hostname)

	// Check length
	if len(hostname) > 253 {
		return ErrHostnameTooLong
	}

	// Check for scheme
	if strings.Contains(hostname, "://") {
		return ErrHostnameHasScheme
	}

	// Check for port
	if strings.Contains(hostname, ":") {
		return ErrHostnameHasPort
	}

	// Check for path
	if strings.Contains(hostname, "/") {
		return ErrHostnameHasPath
	}

	// Basic DNS label validation
	labels := strings.Split(hostname, ".")
	for _, label := range labels {
		if label == "" {
			return ErrHostnameEmptyLabel
		}
		if len(label) > 63 {
			return ErrHostnameLabelTooLong
		}
	}

	return nil
}

// Custom errors
var (
	ErrInvalidProxyType       = &ValidationError{Message: "invalid proxy type"}
	ErrProxyNameRequired      = &ValidationError{Message: "proxy name is required"}
	ErrProxyNameTooLong       = &ValidationError{Message: "proxy name must be at most 255 characters"}
	ErrProxyHostnameRequired  = &ValidationError{Message: "proxy hostname is required"}
	ErrHostnameTooLong        = &ValidationError{Message: "hostname must be at most 253 characters"}
	ErrHostnameHasScheme      = &ValidationError{Message: "hostname should not include a scheme (http:// or https://)"}
	ErrHostnameHasPort        = &ValidationError{Message: "hostname should not include a port"}
	ErrHostnameHasPath        = &ValidationError{Message: "hostname should not include a path"}
	ErrHostnameEmptyLabel     = &ValidationError{Message: "hostname contains empty label"}
	ErrHostnameLabelTooLong   = &ValidationError{Message: "hostname label exceeds 63 characters"}
	ErrUpstreamsRequired      = &ValidationError{Message: "upstreams are required for reverse_proxy type"}
	ErrRedirectConfigRequired = &ValidationError{Message: "redirect configuration is required for redirect type"}
	ErrStaticConfigRequired   = &ValidationError{Message: "static configuration is required for static type"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
