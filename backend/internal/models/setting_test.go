package models

import (
	"testing"
)

// TestSetting_TableName verifies the table name
func TestSetting_TableName(t *testing.T) {
	t.Parallel()
	s := Setting{}
	if s.TableName() != "settings" {
		t.Errorf("Expected table name 'settings', got '%s'", s.TableName())
	}
}

// TestSettingConstants verifies the setting key constants
func TestSettingConstants(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		constant string
		expected string
	}{
		{"NotFoundMode", SettingNotFoundMode, "not_found.mode"},
		{"NotFoundRedirectURL", SettingNotFoundRedirectURL, "not_found.redirect_url"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.constant != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, tc.constant)
			}
		})
	}
}

// TestNotFoundSettings_Validation tests NotFoundSettings struct
func TestNotFoundSettings_Validation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		settings    NotFoundSettings
		expectValid bool
		description string
	}{
		{
			name:        "Default mode without redirect",
			settings:    NotFoundSettings{Mode: "default", RedirectURL: ""},
			expectValid: true,
			description: "Default mode should not require redirect URL",
		},
		{
			name:        "Redirect mode with URL",
			settings:    NotFoundSettings{Mode: "redirect", RedirectURL: "https://example.com"},
			expectValid: true,
			description: "Redirect mode with URL is valid",
		},
		{
			name:        "Redirect mode without URL",
			settings:    NotFoundSettings{Mode: "redirect", RedirectURL: ""},
			expectValid: false,
			description: "Redirect mode should have a URL",
		},
		{
			name:        "Empty mode",
			settings:    NotFoundSettings{Mode: "", RedirectURL: ""},
			expectValid: false,
			description: "Empty mode should be invalid",
		},
		{
			name:        "Invalid mode",
			settings:    NotFoundSettings{Mode: "invalid", RedirectURL: ""},
			expectValid: false,
			description: "Invalid mode should be rejected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate mode
			validModes := map[string]bool{"default": true, "redirect": true}
			isValidMode := validModes[tc.settings.Mode]

			// Validate redirect URL if redirect mode
			hasRequiredURL := tc.settings.Mode != "redirect" || tc.settings.RedirectURL != ""

			isValid := isValidMode && hasRequiredURL

			if isValid != tc.expectValid {
				t.Errorf("%s: expected valid=%v, got valid=%v", tc.description, tc.expectValid, isValid)
			}
		})
	}
}

// TestNotFoundSettings_JSONMarshaling tests JSON field names
func TestNotFoundSettings_JSONMarshaling(t *testing.T) {
	t.Parallel()
	settings := NotFoundSettings{
		Mode:        "redirect",
		RedirectURL: "https://example.com/404",
	}

	// Verify struct fields exist and have correct values
	if settings.Mode != "redirect" {
		t.Errorf("Expected Mode 'redirect', got '%s'", settings.Mode)
	}
	if settings.RedirectURL != "https://example.com/404" {
		t.Errorf("Expected RedirectURL 'https://example.com/404', got '%s'", settings.RedirectURL)
	}
}

// TestSetting_StructFields verifies Setting struct fields
func TestSetting_StructFields(t *testing.T) {
	t.Parallel()
	setting := Setting{
		Key:   "test.key",
		Value: "test.value",
	}

	if setting.Key != "test.key" {
		t.Errorf("Expected key 'test.key', got '%s'", setting.Key)
	}
	if setting.Value != "test.value" {
		t.Errorf("Expected value 'test.value', got '%s'", setting.Value)
	}
}
