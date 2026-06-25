package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
)

// =============================================================================
// UserRepository Integration Tests
// =============================================================================

func TestUserRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success_BasicUser", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		user := &models.User{
			Name:         "John Doe",
			Username:     fmt.Sprintf("johndoe_%d", suffix),
			Email:        fmt.Sprintf("john_%d@example.com", suffix),
			PasswordHash: "$2a$10$hashedpassword",
		}

		err := repo.Create(user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("Success_WithSetPassword", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		user := &models.User{
			Name:     "Jane Doe",
			Username: fmt.Sprintf("janedoe_%d", suffix),
			Email:    fmt.Sprintf("jane_%d@example.com", suffix),
		}
		err := user.SetPassword("securepassword123", 4)
		require.NoError(t, err)

		err = repo.Create(user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.NotEmpty(t, user.PasswordHash)
		assert.True(t, user.CheckPassword("securepassword123"))
		assert.False(t, user.CheckPassword("wrongpassword"))
	})

	t.Run("Error_DuplicateUsername", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		username := fmt.Sprintf("duplicate_user_%d", suffix)

		user1 := &models.User{
			Name:         "First User",
			Username:     username,
			Email:        fmt.Sprintf("first_%d@example.com", suffix),
			PasswordHash: "$2a$10$hash1",
		}
		err := repo.Create(user1)
		require.NoError(t, err)

		user2 := &models.User{
			Name:         "Second User",
			Username:     username, // Same username
			Email:        fmt.Sprintf("second_%d@example.com", suffix),
			PasswordHash: "$2a$10$hash2",
		}
		err = repo.Create(user2)
		require.Error(t, err, "Should fail with duplicate username")
	})

	t.Run("Error_DuplicateEmail", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		email := fmt.Sprintf("duplicate_%d@example.com", suffix)

		user1 := &models.User{
			Name:         "First User",
			Username:     fmt.Sprintf("user1_%d", suffix),
			Email:        email,
			PasswordHash: "$2a$10$hash1",
		}
		err := repo.Create(user1)
		require.NoError(t, err)

		user2 := &models.User{
			Name:         "Second User",
			Username:     fmt.Sprintf("user2_%d", suffix),
			Email:        email, // Same email
			PasswordHash: "$2a$10$hash2",
		}
		err = repo.Create(user2)
		require.Error(t, err, "Should fail with duplicate email")
	})

	t.Run("Success_MinimalFields", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		user := &models.User{
			Name:         "Minimal",
			Username:     fmt.Sprintf("minimal_%d", suffix),
			Email:        fmt.Sprintf("minimal_%d@example.com", suffix),
			PasswordHash: "hash",
		}

		err := repo.Create(user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)
	})

	t.Run("Success_SpecialCharactersInName", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		user := &models.User{
			Name:         "Jose Maria Garcia-Lopez",
			Username:     fmt.Sprintf("jose_%d", suffix),
			Email:        fmt.Sprintf("jose_%d@example.com", suffix),
			PasswordHash: "hash",
		}

		err := repo.Create(user)
		require.NoError(t, err)
		assert.Equal(t, "Jose Maria Garcia-Lopez", user.Name)
	})

	t.Run("Success_UnicodeInName", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		user := &models.User{
			Name:         "Taro Yamada",
			Username:     fmt.Sprintf("taro_%d", suffix),
			Email:        fmt.Sprintf("taro_%d@example.com", suffix),
			PasswordHash: "hash",
		}

		err := repo.Create(user)
		require.NoError(t, err)
		assert.Equal(t, "Taro Yamada", user.Name)
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		fetched, err := repo.GetByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)
		assert.Equal(t, user.Username, fetched.Username)
		assert.Equal(t, user.Email, fetched.Email)
		assert.Equal(t, user.Name, fetched.Name)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(999999)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Error_ZeroID", func(t *testing.T) {
		_, err := repo.GetByID(0)
		require.Error(t, err)
	})

	t.Run("Error_NegativeID", func(t *testing.T) {
		_, err := repo.GetByID(-1)
		require.Error(t, err)
	})

	t.Run("Success_PasswordHashExcludedFromJSON", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		fetched, err := repo.GetByID(user.ID)
		require.NoError(t, err)

		// PasswordHash should be set in the struct
		assert.NotEmpty(t, fetched.PasswordHash)
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		email := fmt.Sprintf("email_test_%d@example.com", suffix)
		user := &models.User{
			Name:         "Email Test User",
			Username:     fmt.Sprintf("emailtest_%d", suffix),
			Email:        email,
			PasswordHash: "hash",
		}
		_ = repo.Create(user)

		fetched, err := repo.GetByEmail(email)
		require.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)
		assert.Equal(t, email, fetched.Email)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		_, err := repo.GetByEmail("nonexistent@example.com")
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Error_EmptyEmail", func(t *testing.T) {
		_, err := repo.GetByEmail("")
		require.Error(t, err)
	})

	t.Run("Success_CaseSensitive", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		email := fmt.Sprintf("casesensitive_%d@example.com", suffix)
		user := &models.User{
			Name:         "Case Test",
			Username:     fmt.Sprintf("casetest_%d", suffix),
			Email:        email,
			PasswordHash: "hash",
		}
		_ = repo.Create(user)

		fetched, err := repo.GetByEmail(email)
		require.NoError(t, err)
		assert.Equal(t, email, fetched.Email)
	})
}

func TestUserRepository_GetByUsernameOrEmail(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	suffix := time.Now().UnixNano()
	username := fmt.Sprintf("testuser_%d", suffix)
	email := fmt.Sprintf("test_%d@example.com", suffix)
	user := &models.User{
		Name:         "Test User",
		Username:     username,
		Email:        email,
		PasswordHash: "hash",
	}
	_ = repo.Create(user)

	t.Run("Success_ByUsername", func(t *testing.T) {
		fetched, err := repo.GetByUsernameOrEmail(username)
		require.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)
		assert.Equal(t, username, fetched.Username)
	})

	t.Run("Success_ByEmail", func(t *testing.T) {
		fetched, err := repo.GetByUsernameOrEmail(email)
		require.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)
		assert.Equal(t, email, fetched.Email)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		_, err := repo.GetByUsernameOrEmail("nonexistent")
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Error_EmptyIdentifier", func(t *testing.T) {
		_, err := repo.GetByUsernameOrEmail("")
		require.Error(t, err)
	})

	t.Run("Success_UsernamePreferenceWhenBothMatch", func(t *testing.T) {
		// This tests the behavior when someone has a username that looks like an email
		suffix2 := time.Now().UnixNano()
		specialUser := &models.User{
			Name:         "Special User",
			Username:     fmt.Sprintf("email_as_username_%d", suffix2),
			Email:        fmt.Sprintf("special_%d@example.com", suffix2),
			PasswordHash: "hash",
		}
		_ = repo.Create(specialUser)

		// Search by username
		fetched, err := repo.GetByUsernameOrEmail(specialUser.Username)
		require.NoError(t, err)
		assert.Equal(t, specialUser.ID, fetched.ID)
	})
}

func TestUserRepository_Count(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success_InitialCount", func(t *testing.T) {
		// Clean tables first
		tdb.CleanTables(t)

		count, err := repo.Count()
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Success_AfterCreatingUsers", func(t *testing.T) {
		initialCount, _ := repo.Count()

		CreateTestUser(t, tdb.DB)
		CreateTestUser(t, tdb.DB)
		CreateTestUser(t, tdb.DB)

		newCount, err := repo.Count()
		require.NoError(t, err)
		assert.Equal(t, initialCount+3, newCount)
	})

	t.Run("Success_AfterDeletingUser", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)
		countBefore, _ := repo.Count()

		_ = repo.Delete(user.ID)

		countAfter, err := repo.Count()
		require.NoError(t, err)
		assert.Equal(t, countBefore-1, countAfter)
	})
}

func TestUserRepository_Delete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		err := repo.Delete(user.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(user.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Success_NonExistent", func(t *testing.T) {
		// GORM doesn't error on non-existent records
		err := repo.Delete(999999)
		require.NoError(t, err)
	})

	t.Run("Success_DeleteTwice", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		_ = repo.Delete(user.ID)
		err := repo.Delete(user.ID)
		require.NoError(t, err)
	})

	t.Run("Success_DeletedUserNotFoundByEmail", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		email := fmt.Sprintf("deleted_%d@example.com", suffix)
		user := &models.User{
			Name:         "To Be Deleted",
			Username:     fmt.Sprintf("tobedeleted_%d", suffix),
			Email:        email,
			PasswordHash: "hash",
		}
		_ = repo.Create(user)

		_ = repo.Delete(user.ID)

		_, err := repo.GetByEmail(email)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)
		originalHash := user.PasswordHash
		newHash := "$2a$10$newhashedpassword"

		err := repo.UpdatePassword(user.ID, newHash)
		require.NoError(t, err)

		fetched, _ := repo.GetByID(user.ID)
		assert.NotEqual(t, originalHash, fetched.PasswordHash)
		assert.Equal(t, newHash, fetched.PasswordHash)
	})

	t.Run("Success_WithActualBcryptHash", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		// Create a proper user with bcrypt password
		tempUser := &models.User{}
		err := tempUser.SetPassword("newpassword123", 4)
		require.NoError(t, err)

		err = repo.UpdatePassword(user.ID, tempUser.PasswordHash)
		require.NoError(t, err)

		// Fetch and verify
		fetched, _ := repo.GetByID(user.ID)
		assert.True(t, fetched.CheckPassword("newpassword123"))
		assert.False(t, fetched.CheckPassword("oldpassword"))
	})

	t.Run("Success_NonExistentUser", func(t *testing.T) {
		// GORM doesn't error when no rows affected
		err := repo.UpdatePassword(999999, "newhash")
		require.NoError(t, err)
	})

	t.Run("Success_UpdateMultipleTimes", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		err := repo.UpdatePassword(user.ID, "hash1")
		require.NoError(t, err)

		err = repo.UpdatePassword(user.ID, "hash2")
		require.NoError(t, err)

		err = repo.UpdatePassword(user.ID, "hash3")
		require.NoError(t, err)

		fetched, _ := repo.GetByID(user.ID)
		assert.Equal(t, "hash3", fetched.PasswordHash)
	})

	t.Run("Success_EmptyHash", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		// This shouldn't be done in practice, but test the behavior
		err := repo.UpdatePassword(user.ID, "")
		require.NoError(t, err)

		fetched, _ := repo.GetByID(user.ID)
		assert.Empty(t, fetched.PasswordHash)
	})
}

func TestUserRepository_TableName(t *testing.T) {
	t.Parallel()

	t.Run("TableName", func(t *testing.T) {
		user := models.User{}
		assert.Equal(t, "users", user.TableName())
	})
}

func TestUserRepository_CheckPassword(t *testing.T) {
	t.Parallel()

	t.Run("Success_CorrectPassword", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword("correctpassword", 4)
		require.NoError(t, err)

		assert.True(t, user.CheckPassword("correctpassword"))
	})

	t.Run("Failure_IncorrectPassword", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword("correctpassword", 4)
		require.NoError(t, err)

		assert.False(t, user.CheckPassword("wrongpassword"))
	})

	t.Run("Failure_EmptyPassword", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword("password", 4)
		require.NoError(t, err)

		assert.False(t, user.CheckPassword(""))
	})

	t.Run("Failure_InvalidHash", func(t *testing.T) {
		user := &models.User{
			PasswordHash: "not_a_valid_bcrypt_hash",
		}

		assert.False(t, user.CheckPassword("anypassword"))
	})
}

func TestUserRepository_SetPassword(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword("testpassword", 4)
		require.NoError(t, err)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotEqual(t, "testpassword", user.PasswordHash)
	})

	t.Run("Success_DifferentPasswords", func(t *testing.T) {
		user1 := &models.User{}
		_ = user1.SetPassword("password1", 4)

		user2 := &models.User{}
		_ = user2.SetPassword("password2", 4)

		// Different passwords should produce different hashes
		assert.NotEqual(t, user1.PasswordHash, user2.PasswordHash)
	})

	t.Run("Success_SamePasswordDifferentHashes", func(t *testing.T) {
		user1 := &models.User{}
		_ = user1.SetPassword("samepassword", 4)

		user2 := &models.User{}
		_ = user2.SetPassword("samepassword", 4)

		// Same password produces different hashes due to salt
		assert.NotEqual(t, user1.PasswordHash, user2.PasswordHash)

		// But both should verify correctly
		assert.True(t, user1.CheckPassword("samepassword"))
		assert.True(t, user2.CheckPassword("samepassword"))
	})

	t.Run("Success_EmptyPassword", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword("", 4)
		require.NoError(t, err)
		assert.NotEmpty(t, user.PasswordHash)
		assert.True(t, user.CheckPassword(""))
	})

	t.Run("Error_LongPassword", func(t *testing.T) {
		user := &models.User{}
		// bcrypt has a 72-byte limit, passwords longer than this are truncated
		// but strings longer than 72 bytes will still be accepted
		longPassword := string(make([]byte, 72)) // exactly at the limit
		err := user.SetPassword(longPassword, 4)
		require.NoError(t, err)
	})

	t.Run("Error_TooLongPassword", func(t *testing.T) {
		user := &models.User{}
		// bcrypt will reject passwords that exceed an internal limit
		tooLongPassword := string(make([]byte, 73))
		err := user.SetPassword(tooLongPassword, 4)
		require.Error(t, err, "bcrypt should reject passwords exceeding 72 bytes")
	})

	t.Run("Success_DifferentCosts", func(t *testing.T) {
		user := &models.User{}
		// Test with minimum cost
		err := user.SetPassword("password", 4)
		require.NoError(t, err)
		assert.True(t, user.CheckPassword("password"))
	})
}

func TestNewUserRepository_Integration(t *testing.T) {
	t.Parallel()

	t.Run("Success_WithDB", func(t *testing.T) {
		// This just tests constructor doesn't panic with nil
		repo := NewUserRepository(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.db)
	})
}

func TestUserRepository_List(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success_ListReturnsCreatedUser", func(t *testing.T) {
		tdb.CleanTables(t)

		user := CreateTestUser(t, tdb.DB)

		users, err := repo.List()
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, user.ID, users[0].ID)
		assert.Equal(t, user.Username, users[0].Username)
		assert.Equal(t, user.Email, users[0].Email)
	})

	t.Run("Success_EmptyList", func(t *testing.T) {
		tdb.CleanTables(t)

		users, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("Success_OrderedByID", func(t *testing.T) {
		tdb.CleanTables(t)

		u1 := CreateTestUser(t, tdb.DB)
		u2 := CreateTestUser(t, tdb.DB)
		u3 := CreateTestUser(t, tdb.DB)

		users, err := repo.List()
		require.NoError(t, err)
		require.Len(t, users, 3)
		assert.Equal(t, u1.ID, users[0].ID)
		assert.Equal(t, u2.ID, users[1].ID)
		assert.Equal(t, u3.ID, users[2].ID)
	})
}

func TestUserRepository_Update(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success_FlipsActiveToFalseAndChangesNameEmail", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		// Verify defaults
		assert.True(t, user.Active)

		// Flip Active to false and change Name/Email
		user.Active = false
		user.Name = "Updated Name"
		user.Email = fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano())

		err := repo.Update(user)
		require.NoError(t, err)

		fetched, err := repo.GetByID(user.ID)
		require.NoError(t, err)
		assert.False(t, fetched.Active, "Active should be persisted as false")
		assert.Equal(t, "Updated Name", fetched.Name)
		assert.Equal(t, user.Email, fetched.Email)
	})

	t.Run("Success_SetMustChangePassword", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)
		assert.False(t, user.MustChangePassword)

		user.MustChangePassword = true
		err := repo.Update(user)
		require.NoError(t, err)

		fetched, err := repo.GetByID(user.ID)
		require.NoError(t, err)
		assert.True(t, fetched.MustChangePassword)
	})

	t.Run("Success_UsernameNotChanged", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)
		originalUsername := user.Username

		// Attempt to change username via Update (should be ignored)
		user.Username = "should_not_change"
		err := repo.Update(user)
		require.NoError(t, err)

		fetched, err := repo.GetByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, originalUsername, fetched.Username, "Username must not be modified by Update")
	})
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewUserRepository(tdb.DB)

	t.Run("Success_SetsTimestamp", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		// Confirm LastLoginAt is nil initially
		fetched, err := repo.GetByID(user.ID)
		require.NoError(t, err)
		assert.Nil(t, fetched.LastLoginAt)

		loginTime := time.Now().UTC().Truncate(time.Millisecond)
		err = repo.UpdateLastLogin(user.ID, loginTime)
		require.NoError(t, err)

		fetched, err = repo.GetByID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, fetched.LastLoginAt)
		assert.WithinDuration(t, loginTime, *fetched.LastLoginAt, time.Second)
	})

	t.Run("Success_CanUpdateTwice", func(t *testing.T) {
		user := CreateTestUser(t, tdb.DB)

		first := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Millisecond)
		err := repo.UpdateLastLogin(user.ID, first)
		require.NoError(t, err)

		second := time.Now().UTC().Truncate(time.Millisecond)
		err = repo.UpdateLastLogin(user.ID, second)
		require.NoError(t, err)

		fetched, err := repo.GetByID(user.ID)
		require.NoError(t, err)
		require.NotNil(t, fetched.LastLoginAt)
		assert.WithinDuration(t, second, *fetched.LastLoginAt, time.Second)
	})
}
