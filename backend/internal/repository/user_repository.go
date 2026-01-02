package repository

import (
	"github.com/aloks98/homelab-proxy/backend/internal/models"
	"gorm.io/gorm"
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
