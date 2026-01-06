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
// SettingsRepository Integration Tests
// =============================================================================

func TestSettingsRepository_Set(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success_NewSetting", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("new_setting_%d", suffix)

		err := repo.Set(key, "test_value")
		require.NoError(t, err)

		setting, err := repo.Get(key)
		require.NoError(t, err)
		assert.Equal(t, "test_value", setting.Value)
	})

	t.Run("Success_UpdateExisting", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("update_setting_%d", suffix)

		// Create initial value
		err := repo.Set(key, "initial_value")
		require.NoError(t, err)

		// Update value
		err = repo.Set(key, "updated_value")
		require.NoError(t, err)

		setting, _ := repo.Get(key)
		assert.Equal(t, "updated_value", setting.Value)
	})

	t.Run("Success_EmptyValue", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("empty_value_%d", suffix)

		err := repo.Set(key, "")
		require.NoError(t, err)

		setting, _ := repo.Get(key)
		assert.Equal(t, "", setting.Value)
	})

	t.Run("Success_LongValue", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("long_value_%d", suffix)
		// Use valid UTF-8 characters instead of null bytes
		longValue := ""
		for i := 0; i < 10000; i++ {
			longValue += "a"
		}

		err := repo.Set(key, longValue)
		require.NoError(t, err)

		setting, _ := repo.Get(key)
		assert.Len(t, setting.Value, 10000)
	})

	t.Run("Success_JSONValue", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("json_value_%d", suffix)
		jsonValue := `{"key": "value", "nested": {"a": 1, "b": true}}`

		err := repo.Set(key, jsonValue)
		require.NoError(t, err)

		setting, _ := repo.Get(key)
		assert.Equal(t, jsonValue, setting.Value)
	})

	t.Run("Success_SpecialCharacters", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("special_chars_%d", suffix)
		specialValue := `Test with "quotes" and 'apostrophes' and \backslash\ and newlines
and tabs	done`

		err := repo.Set(key, specialValue)
		require.NoError(t, err)

		setting, _ := repo.Get(key)
		assert.Equal(t, specialValue, setting.Value)
	})

	t.Run("Success_UnicodeValue", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("unicode_%d", suffix)
		unicodeValue := "Hello World"

		err := repo.Set(key, unicodeValue)
		require.NoError(t, err)

		setting, _ := repo.Get(key)
		assert.Equal(t, unicodeValue, setting.Value)
	})

	t.Run("Success_NestedKey", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("parent.child.grandchild_%d", suffix)

		err := repo.Set(key, "nested_value")
		require.NoError(t, err)

		setting, _ := repo.Get(key)
		assert.Equal(t, "nested_value", setting.Value)
	})
}

func TestSettingsRepository_Get(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("get_test_%d", suffix)
		_ = repo.Set(key, "get_value")

		setting, err := repo.Get(key)
		require.NoError(t, err)
		assert.Equal(t, key, setting.Key)
		assert.Equal(t, "get_value", setting.Value)
		assert.NotZero(t, setting.UpdatedAt)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		_, err := repo.Get("nonexistent_key")
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Error_EmptyKey", func(t *testing.T) {
		_, err := repo.Get("")
		require.Error(t, err)
	})
}

func TestSettingsRepository_GetValue(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success_ExistingKey", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("getvalue_%d", suffix)
		_ = repo.Set(key, "actual_value")

		value := repo.GetValue(key, "default")
		assert.Equal(t, "actual_value", value)
	})

	t.Run("Success_NonExistentKey_ReturnsDefault", func(t *testing.T) {
		value := repo.GetValue("nonexistent_key", "default_value")
		assert.Equal(t, "default_value", value)
	})

	t.Run("Success_EmptyDefault", func(t *testing.T) {
		value := repo.GetValue("another_nonexistent", "")
		assert.Equal(t, "", value)
	})

	t.Run("Success_ExistingEmpty_ReturnsEmpty", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("empty_setting_%d", suffix)
		_ = repo.Set(key, "")

		value := repo.GetValue(key, "default")
		assert.Equal(t, "", value, "Should return empty string, not default")
	})

	t.Run("Success_DifferentDefaults", func(t *testing.T) {
		v1 := repo.GetValue("missing_1", "default_1")
		v2 := repo.GetValue("missing_2", "default_2")

		assert.Equal(t, "default_1", v1)
		assert.Equal(t, "default_2", v2)
	})
}

func TestSettingsRepository_GetAll(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success_EmptyDB", func(t *testing.T) {
		tdb.CleanTables(t)

		all, err := repo.GetAll()
		require.NoError(t, err)
		assert.NotNil(t, all)
		assert.Empty(t, all)
	})

	t.Run("Success_WithData", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		keys := []string{
			fmt.Sprintf("all_key1_%d", suffix),
			fmt.Sprintf("all_key2_%d", suffix),
			fmt.Sprintf("all_key3_%d", suffix),
		}

		for i, key := range keys {
			_ = repo.Set(key, fmt.Sprintf("value_%d", i))
		}

		all, err := repo.GetAll()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(all), 3)

		for i, key := range keys {
			assert.Equal(t, fmt.Sprintf("value_%d", i), all[key])
		}
	})

	t.Run("Success_ReturnsMap", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		_ = repo.Set(fmt.Sprintf("map_test_%d", suffix), "value")

		all, err := repo.GetAll()
		require.NoError(t, err)
		assert.IsType(t, map[string]string{}, all)
	})
}

func TestSettingsRepository_Delete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("delete_test_%d", suffix)
		_ = repo.Set(key, "to_delete")

		err := repo.Delete(key)
		require.NoError(t, err)

		_, err = repo.Get(key)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Success_NonExistent", func(t *testing.T) {
		err := repo.Delete("nonexistent_to_delete")
		require.NoError(t, err)
	})

	t.Run("Success_DeleteTwice", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("double_delete_%d", suffix)
		_ = repo.Set(key, "value")

		_ = repo.Delete(key)
		err := repo.Delete(key)
		require.NoError(t, err)
	})

	t.Run("Success_DeleteAndRecreate", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("recreate_%d", suffix)

		_ = repo.Set(key, "initial")
		_ = repo.Delete(key)
		_ = repo.Set(key, "recreated")

		setting, err := repo.Get(key)
		require.NoError(t, err)
		assert.Equal(t, "recreated", setting.Value)
	})
}

func TestSettingsRepository_GetNotFoundSettings(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success_DefaultValues", func(t *testing.T) {
		tdb.CleanTables(t)

		settings, err := repo.GetNotFoundSettings()
		require.NoError(t, err)
		assert.Equal(t, "default", settings.Mode)
		assert.Equal(t, "", settings.RedirectURL)
	})

	t.Run("Success_WithCustomMode", func(t *testing.T) {
		_ = repo.Set(models.SettingNotFoundMode, "redirect")

		settings, err := repo.GetNotFoundSettings()
		require.NoError(t, err)
		assert.Equal(t, "redirect", settings.Mode)
	})

	t.Run("Success_WithCustomRedirectURL", func(t *testing.T) {
		_ = repo.Set(models.SettingNotFoundMode, "redirect")
		_ = repo.Set(models.SettingNotFoundRedirectURL, "https://example.com/404")

		settings, err := repo.GetNotFoundSettings()
		require.NoError(t, err)
		assert.Equal(t, "redirect", settings.Mode)
		assert.Equal(t, "https://example.com/404", settings.RedirectURL)
	})
}

func TestSettingsRepository_SetNotFoundSettings(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success_SetRedirectMode", func(t *testing.T) {
		settings := &models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://example.com/404",
		}

		err := repo.SetNotFoundSettings(settings)
		require.NoError(t, err)

		retrieved, _ := repo.GetNotFoundSettings()
		assert.Equal(t, "redirect", retrieved.Mode)
		assert.Equal(t, "https://example.com/404", retrieved.RedirectURL)
	})

	t.Run("Success_SetDefaultMode", func(t *testing.T) {
		// First set to redirect
		_ = repo.SetNotFoundSettings(&models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://example.com",
		})

		// Then set back to default
		settings := &models.NotFoundSettings{
			Mode:        "default",
			RedirectURL: "",
		}

		err := repo.SetNotFoundSettings(settings)
		require.NoError(t, err)

		retrieved, _ := repo.GetNotFoundSettings()
		assert.Equal(t, "default", retrieved.Mode)
		assert.Equal(t, "", retrieved.RedirectURL)
	})

	t.Run("Success_UpdateRedirectURL", func(t *testing.T) {
		_ = repo.SetNotFoundSettings(&models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://old.example.com",
		})

		err := repo.SetNotFoundSettings(&models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://new.example.com",
		})
		require.NoError(t, err)

		retrieved, _ := repo.GetNotFoundSettings()
		assert.Equal(t, "https://new.example.com", retrieved.RedirectURL)
	})
}

func TestSettingsRepository_TableName(t *testing.T) {
	t.Parallel()

	t.Run("TableName", func(t *testing.T) {
		setting := models.Setting{}
		assert.Equal(t, "settings", setting.TableName())
	})
}

func TestSettingsRepository_Constants(t *testing.T) {
	t.Parallel()

	t.Run("SettingNotFoundMode", func(t *testing.T) {
		assert.Equal(t, "not_found.mode", models.SettingNotFoundMode)
	})

	t.Run("SettingNotFoundRedirectURL", func(t *testing.T) {
		assert.Equal(t, "not_found.redirect_url", models.SettingNotFoundRedirectURL)
	})
}

func TestNewSettingsRepository_Integration(t *testing.T) {
	t.Parallel()

	t.Run("Success_WithNilDB", func(t *testing.T) {
		repo := NewSettingsRepository(nil)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.db)
	})
}

func TestSettingsRepository_Upsert(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("Success_InsertThenUpdate", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("upsert_test_%d", suffix)

		// Insert
		err := repo.Set(key, "v1")
		require.NoError(t, err)

		setting1, _ := repo.Get(key)
		assert.Equal(t, "v1", setting1.Value)
		time1 := setting1.UpdatedAt

		// Small delay to ensure timestamp changes
		time.Sleep(10 * time.Millisecond)

		// Update
		err = repo.Set(key, "v2")
		require.NoError(t, err)

		setting2, _ := repo.Get(key)
		assert.Equal(t, "v2", setting2.Value)
		assert.True(t, setting2.UpdatedAt.After(time1) || setting2.UpdatedAt.Equal(time1))
	})
}

func TestSettingsRepository_ConcurrentAccess(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)
	suffix := time.Now().UnixNano()
	key := fmt.Sprintf("concurrent_%d", suffix)

	t.Run("ConcurrentReads", func(t *testing.T) {
		_ = repo.Set(key, "concurrent_value")

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				value := repo.GetValue(key, "default")
				assert.Equal(t, "concurrent_value", value)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestSettingsRepository_NotFoundSettings_Struct(t *testing.T) {
	t.Parallel()

	t.Run("DefaultValues", func(t *testing.T) {
		settings := &models.NotFoundSettings{}
		assert.Equal(t, "", settings.Mode)
		assert.Equal(t, "", settings.RedirectURL)
	})

	t.Run("WithValues", func(t *testing.T) {
		settings := &models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://example.com/404",
		}
		assert.Equal(t, "redirect", settings.Mode)
		assert.Equal(t, "https://example.com/404", settings.RedirectURL)
	})
}

func TestSettingsRepository_EdgeCases(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewSettingsRepository(tdb.DB)

	t.Run("KeyWithSpaces", func(t *testing.T) {
		// Note: Keys with spaces might not be ideal but should still work
		key := "key with spaces"
		err := repo.Set(key, "value")
		require.NoError(t, err)

		setting, err := repo.Get(key)
		require.NoError(t, err)
		assert.Equal(t, "value", setting.Value)
	})

	t.Run("KeyWithSpecialCharacters", func(t *testing.T) {
		key := "key.with.dots_and-dashes"
		err := repo.Set(key, "value")
		require.NoError(t, err)

		setting, err := repo.Get(key)
		require.NoError(t, err)
		assert.Equal(t, "value", setting.Value)
	})

	t.Run("Base64EncodedValue", func(t *testing.T) {
		// PostgreSQL text columns don't support null bytes, so use base64 encoding
		// for binary-like data in practice
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("binary_%d", suffix)
		base64Value := "AAECBA==" // base64 encoded binary

		err := repo.Set(key, base64Value)
		require.NoError(t, err)

		setting, err := repo.Get(key)
		require.NoError(t, err)
		assert.Equal(t, base64Value, setting.Value)
	})

	t.Run("ValueWithSQLInjectionAttempt", func(t *testing.T) {
		suffix := time.Now().UnixNano()
		key := fmt.Sprintf("sql_injection_%d", suffix)
		injectionValue := "'; DROP TABLE settings; --"

		err := repo.Set(key, injectionValue)
		require.NoError(t, err)

		setting, err := repo.Get(key)
		require.NoError(t, err)
		assert.Equal(t, injectionValue, setting.Value)

		// Verify table still exists
		all, err := repo.GetAll()
		require.NoError(t, err)
		assert.NotNil(t, all)
	})
}
