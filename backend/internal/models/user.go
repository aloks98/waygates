package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user of the system
type User struct {
	ID                 int        `json:"id" gorm:"primaryKey;autoIncrement"`
	Name               string     `json:"name" gorm:"type:varchar(255);not null"`
	Username           string     `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
	Email              string     `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash       string     `json:"-" gorm:"type:varchar(255);not null"`
	Active             bool       `json:"active" gorm:"not null;default:true"`
	MustChangePassword bool       `json:"must_change_password" gorm:"not null;default:false"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	CreatedAt          time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// SetPassword hashes the password and sets it on the user model
func (u *User) SetPassword(password string, cost int) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies if the provided password matches the hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
