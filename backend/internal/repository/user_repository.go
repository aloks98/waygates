package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	return r.GetByUsernameOrEmail(email)
}

// GetByUsernameOrEmail retrieves a user by username or email
func (r *UserRepository) GetByUsernameOrEmail(identifier string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id int) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Count returns the total number of users
func (r *UserRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(id int) error {
	return r.db.Delete(&models.User{}, id).Error
}

// UpdatePassword updates a user's password hash
func (r *UserRepository) UpdatePassword(id int, passwordHash string) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

// List returns all users ordered by ID ascending.
func (r *UserRepository) List() ([]models.User, error) {
	var users []models.User
	if err := r.db.Order("id ASC").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// Update writes the mutable profile fields (NOT username or password).
// Select is required so a false Active is persisted (GORM skips zero-values otherwise).
func (r *UserRepository) Update(user *models.User) error {
	if err := r.db.Model(user).
		Select("Name", "Email", "Active", "MustChangePassword").
		Updates(user).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// UpdateLastLogin sets the last_login_at timestamp for the given user ID.
func (r *UserRepository) UpdateLastLogin(id int, t time.Time) error {
	if err := r.db.Model(&models.User{}).Where("id = ?", id).
		Update("last_login_at", t).Error; err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}
