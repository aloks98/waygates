package models

import (
	"time"
)

// Setting represents a key-value configuration setting
type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey;type:varchar(255)"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Setting) TableName() string {
	return "settings"
}

// NotFoundSettings represents the 404 page configuration
type NotFoundSettings struct {
	Mode        string `json:"mode"`         // "default" or "redirect"
	RedirectURL string `json:"redirect_url"` // URL to redirect when mode is "redirect"
}

// Settings keys
const (
	SettingNotFoundMode        = "not_found.mode"
	SettingNotFoundRedirectURL = "not_found.redirect_url"
)
