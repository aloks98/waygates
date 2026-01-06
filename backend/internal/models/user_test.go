package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestUser_TableName verifies the table name
func TestUser_TableName(t *testing.T) {
	t.Parallel()
	u := User{}
	assert.Equal(t, "users", u.TableName())
}

// TestUser_SetPassword tests the SetPassword function
func TestUser_SetPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		cost     int
		wantErr  bool
	}{
		{
			name:     "success - valid password",
			password: "securePassword123",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "success - password with special characters",
			password: "p@$$w0rd!#$%^&*()",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "success - short password",
			password: "abc",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "success - empty password",
			password: "",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "success - password with unicode characters",
			password: "пароль密码🔐",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "success - password with whitespace",
			password: "   password with spaces   ",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "success - password with newlines",
			password: "password\nwith\nnewlines",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "error - password exceeds bcrypt max length (73 bytes)",
			password: strings.Repeat("a", 73),
			cost:     bcrypt.MinCost,
			wantErr:  true, // bcrypt returns ErrPasswordTooLong for > 72 bytes
		},
		{
			name:     "success - password at bcrypt max length (72 bytes)",
			password: strings.Repeat("x", 72),
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "success - cost below MinCost gets normalized",
			password: "password",
			cost:     bcrypt.MinCost - 1,
			wantErr:  false, // bcrypt normalizes low cost to MinCost
		},
		{
			name:     "error - invalid cost too high",
			password: "password",
			cost:     bcrypt.MaxCost + 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := &User{}
			err := u.SetPassword(tt.password, tt.cost)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, u.PasswordHash)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, u.PasswordHash)
			// Verify the hash is a valid bcrypt hash
			assert.True(t, strings.HasPrefix(u.PasswordHash, "$2a$") || strings.HasPrefix(u.PasswordHash, "$2b$"))
		})
	}
}

// TestUser_SetPassword_DifferentCosts tests SetPassword with different bcrypt costs
func TestUser_SetPassword_DifferentCosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cost int
	}{
		{"minimum cost", bcrypt.MinCost},
		{"default cost", bcrypt.DefaultCost},
		{"cost 10", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := &User{}
			err := u.SetPassword("testpassword", tt.cost)

			require.NoError(t, err)
			assert.NotEmpty(t, u.PasswordHash)

			// Verify the password can be checked
			assert.True(t, u.CheckPassword("testpassword"))
		})
	}
}

// TestUser_SetPassword_HashUniqueness verifies that different calls produce different hashes
func TestUser_SetPassword_HashUniqueness(t *testing.T) {
	t.Parallel()

	u1 := &User{}
	u2 := &User{}

	require.NoError(t, u1.SetPassword("samepassword", bcrypt.MinCost))
	require.NoError(t, u2.SetPassword("samepassword", bcrypt.MinCost))

	// bcrypt should produce different hashes each time due to random salt
	assert.NotEqual(t, u1.PasswordHash, u2.PasswordHash)
}

// TestUser_CheckPassword tests the CheckPassword function
func TestUser_CheckPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupPassword  string
		checkPassword  string
		expectedResult bool
	}{
		{
			name:           "success - correct password",
			setupPassword:  "correctPassword123",
			checkPassword:  "correctPassword123",
			expectedResult: true,
		},
		{
			name:           "failure - wrong password",
			setupPassword:  "correctPassword123",
			checkPassword:  "wrongPassword456",
			expectedResult: false,
		},
		{
			name:           "failure - empty password when set is not empty",
			setupPassword:  "correctPassword123",
			checkPassword:  "",
			expectedResult: false,
		},
		{
			name:           "success - empty password matches empty",
			setupPassword:  "",
			checkPassword:  "",
			expectedResult: true,
		},
		{
			name:           "failure - case sensitive mismatch",
			setupPassword:  "Password",
			checkPassword:  "password",
			expectedResult: false,
		},
		{
			name:           "failure - password with extra whitespace",
			setupPassword:  "password",
			checkPassword:  " password ",
			expectedResult: false,
		},
		{
			name:           "success - password with special characters",
			setupPassword:  "p@$$w0rd!#$%",
			checkPassword:  "p@$$w0rd!#$%",
			expectedResult: true,
		},
		{
			name:           "success - password with unicode",
			setupPassword:  "пароль密码🔐",
			checkPassword:  "пароль密码🔐",
			expectedResult: true,
		},
		{
			name:           "failure - similar unicode characters",
			setupPassword:  "пароль",
			checkPassword:  "парoль", // 'o' is latin, not cyrillic
			expectedResult: false,
		},
		{
			name:           "success - password with newlines",
			setupPassword:  "pass\nword",
			checkPassword:  "pass\nword",
			expectedResult: true,
		},
		{
			name:           "failure - newline vs space",
			setupPassword:  "pass\nword",
			checkPassword:  "pass word",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := &User{}
			err := u.SetPassword(tt.setupPassword, bcrypt.MinCost)
			require.NoError(t, err)

			result := u.CheckPassword(tt.checkPassword)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestUser_CheckPassword_EmptyHash tests CheckPassword when hash is not set
func TestUser_CheckPassword_EmptyHash(t *testing.T) {
	t.Parallel()

	u := &User{}
	// User without any password hash set
	result := u.CheckPassword("anypassword")
	assert.False(t, result)
}

// TestUser_CheckPassword_InvalidHash tests CheckPassword with invalid hash
func TestUser_CheckPassword_InvalidHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hash string
	}{
		{"empty hash", ""},
		{"random string", "not-a-valid-hash"},
		{"malformed bcrypt prefix", "$2a$invalid"},
		{"truncated hash", "$2a$10$"},
		{"hash with wrong length", "$2a$10$abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := &User{PasswordHash: tt.hash}
			result := u.CheckPassword("anypassword")
			assert.False(t, result)
		})
	}
}

// TestUser_SetAndCheckPassword_Integration tests the full flow of setting and checking passwords
func TestUser_SetAndCheckPassword_Integration(t *testing.T) {
	t.Parallel()

	u := &User{
		Name:     "Test User",
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Set password
	err := u.SetPassword("mySecurePassword123", bcrypt.MinCost)
	require.NoError(t, err)

	// Check correct password
	assert.True(t, u.CheckPassword("mySecurePassword123"))

	// Check incorrect password
	assert.False(t, u.CheckPassword("wrongPassword"))

	// Change password
	err = u.SetPassword("newPassword456", bcrypt.MinCost)
	require.NoError(t, err)

	// Old password should no longer work
	assert.False(t, u.CheckPassword("mySecurePassword123"))

	// New password should work
	assert.True(t, u.CheckPassword("newPassword456"))
}

// TestUser_StructFields tests User struct field assignment
func TestUser_StructFields(t *testing.T) {
	t.Parallel()

	u := User{
		ID:       1,
		Name:     "John Doe",
		Username: "johndoe",
		Email:    "john@example.com",
	}

	assert.Equal(t, 1, u.ID)
	assert.Equal(t, "John Doe", u.Name)
	assert.Equal(t, "johndoe", u.Username)
	assert.Equal(t, "john@example.com", u.Email)
}
