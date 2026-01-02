package validation

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

// GetValidator returns a singleton validator instance
func GetValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New()

		// Register custom validations
		_ = validate.RegisterValidation("username", validateUsername)
		_ = validate.RegisterValidation("hostname", validateHostname)
		_ = validate.RegisterValidation("password_strength", validatePasswordStrength)
	})
	return validate
}

// usernameRegex allows alphanumeric, underscores, and hyphens
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateUsername is a custom validator for usernames
func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	return usernameRegex.MatchString(username)
}

// validateHostname validates a hostname (no scheme, no port, no path)
func validateHostname(fl validator.FieldLevel) bool {
	hostname := fl.Field().String()

	// Check for scheme
	if strings.Contains(hostname, "://") {
		return false
	}

	// Check for port
	if strings.Contains(hostname, ":") {
		return false
	}

	// Check for path
	if strings.Contains(hostname, "/") {
		return false
	}

	// Check length
	if len(hostname) > 253 {
		return false
	}

	// Basic DNS label validation
	labels := strings.Split(hostname, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
	}

	return true
}

// validatePasswordStrength validates password meets minimum requirements
func validatePasswordStrength(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	return len(password) >= 8 && len(password) <= 128
}

// RegisterRequest is the validation struct for user registration
type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=255"`
	Username string `json:"username" validate:"required,min=3,max=50,username"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// LoginRequest is the validation struct for user login
type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required,min=1"`
	Password   string `json:"password" validate:"required,min=1"`
}

// ProxyRequest is the validation struct for proxy creation/update
type ProxyRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=255"`
	Hostname string `json:"hostname" validate:"required,hostname,max=253"`
	Type     string `json:"type" validate:"required,oneof=reverse_proxy redirect static"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateStruct validates a struct and returns a user-friendly error message
func ValidateStruct(s interface{}) error {
	v := GetValidator()
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	// Get the first validation error and return a user-friendly message
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			return &ValidationError{
				Field:   toSnakeCase(e.Field()),
				Message: getErrorMessage(e),
			}
		}
	}

	return err
}

// ValidateStructAll validates a struct and returns all errors
func ValidateStructAll(s interface{}) []ValidationError {
	v := GetValidator()
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	var errors []ValidationError
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   toSnakeCase(e.Field()),
				Message: getErrorMessage(e),
			})
		}
	}

	return errors
}

// getErrorMessage returns a user-friendly error message for a validation error
func getErrorMessage(e validator.FieldError) string {
	field := toSnakeCase(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return "invalid email format"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, e.Param())
	case "username":
		return "username can only contain letters, numbers, underscores, and hyphens"
	case "hostname":
		return "invalid hostname format (should not include scheme, port, or path)"
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "password_strength":
		return "password must be at least 8 characters"
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// toSnakeCase converts a string from PascalCase/camelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
