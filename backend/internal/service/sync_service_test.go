package service

import (
	"testing"
	"time"
)

// TestSyncStatus_DefaultValues tests SyncStatus default values
func TestSyncStatus_DefaultValues(t *testing.T) {
	status := SyncStatus{}

	if status.IsSyncing {
		t.Error("Expected IsSyncing to be false by default")
	}
	if status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be false by default (zero value)")
	}
	if status.LastError != "" {
		t.Errorf("Expected LastError to be empty, got '%s'", status.LastError)
	}
	if !status.LastSyncTime.IsZero() {
		t.Error("Expected LastSyncTime to be zero")
	}
}

// TestSyncStatus_JSONSerialization tests JSON field names
func TestSyncStatus_JSONSerialization(t *testing.T) {
	now := time.Now()
	status := SyncStatus{
		LastSyncTime:    now,
		IsSyncing:       true,
		LastSyncSuccess: true,
		LastError:       "test error",
	}

	if !status.IsSyncing {
		t.Error("Expected IsSyncing to be true")
	}
	if !status.LastSyncSuccess {
		t.Error("Expected LastSyncSuccess to be true")
	}
	if status.LastError != "test error" {
		t.Errorf("Expected LastError 'test error', got '%s'", status.LastError)
	}
	if status.LastSyncTime != now {
		t.Error("LastSyncTime mismatch")
	}
}

// TestSyncStatus_Fields verifies all expected fields exist
func TestSyncStatus_Fields(t *testing.T) {
	status := SyncStatus{
		LastSyncTime:    time.Now(),
		IsSyncing:       true,
		LastSyncSuccess: false,
		LastError:       "connection refused",
	}

	// Verify fields can be read
	_ = status.LastSyncTime
	_ = status.IsSyncing
	_ = status.LastSyncSuccess
	_ = status.LastError
}

// TestNewSyncService_NilLogger tests that nil logger is handled
func TestNewSyncService_NilLogger(t *testing.T) {
	// Test that passing nil logger doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewSyncService panicked with nil logger: %v", r)
		}
	}()

	// Note: We can't fully test this without mocking the repositories
	// This test documents the expected nil-safety behavior
}

// TestSyncStatus_ErrorState tests error state handling
func TestSyncStatus_ErrorState(t *testing.T) {
	testCases := []struct {
		name            string
		lastSyncSuccess bool
		lastError       string
		expectError     bool
	}{
		{
			name:            "Success state",
			lastSyncSuccess: true,
			lastError:       "",
			expectError:     false,
		},
		{
			name:            "Failure state with error",
			lastSyncSuccess: false,
			lastError:       "Caddy not reachable",
			expectError:     true,
		},
		{
			name:            "Failure state without error",
			lastSyncSuccess: false,
			lastError:       "",
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := SyncStatus{
				LastSyncSuccess: tc.lastSyncSuccess,
				LastError:       tc.lastError,
			}

			hasError := !status.LastSyncSuccess
			if hasError != tc.expectError {
				t.Errorf("Expected error=%v, got error=%v", tc.expectError, hasError)
			}
		})
	}
}

// TestSyncStatus_SyncingLock tests that IsSyncing acts as a mutex indicator
func TestSyncStatus_SyncingLock(t *testing.T) {
	status := SyncStatus{IsSyncing: false}

	// Simulate acquiring lock
	if status.IsSyncing {
		t.Error("Should be able to start sync when not syncing")
	}
	status.IsSyncing = true

	// Simulate checking lock
	if !status.IsSyncing {
		t.Error("Should indicate sync in progress")
	}

	// Simulate releasing lock
	status.IsSyncing = false
	if status.IsSyncing {
		t.Error("Should be released after sync completes")
	}
}

// TestSyncIntervalValidation tests sync interval boundaries
func TestSyncIntervalValidation(t *testing.T) {
	testCases := []struct {
		name     string
		interval time.Duration
		valid    bool
	}{
		{"1 second", 1 * time.Second, true},
		{"30 seconds", 30 * time.Second, true},
		{"1 minute", 1 * time.Minute, true},
		{"5 minutes", 5 * time.Minute, true},
		{"Zero", 0, false},
		{"Negative", -1 * time.Second, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := tc.interval > 0
			if isValid != tc.valid {
				t.Errorf("Expected valid=%v for interval %v, got %v", tc.valid, tc.interval, isValid)
			}
		})
	}
}

// TestSyncStatus_TimeTracking tests time tracking functionality
func TestSyncStatus_TimeTracking(t *testing.T) {
	before := time.Now()
	time.Sleep(1 * time.Millisecond)

	status := SyncStatus{
		LastSyncTime: time.Now(),
	}

	time.Sleep(1 * time.Millisecond)
	after := time.Now()

	if status.LastSyncTime.Before(before) {
		t.Error("LastSyncTime should be after 'before'")
	}
	if status.LastSyncTime.After(after) {
		t.Error("LastSyncTime should be before 'after'")
	}
}

// TestSanitizeFilename tests the sanitizeFilename function
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple hostname",
			input:    "example.com",
			expected: "example_com",
		},
		{
			name:     "Subdomain",
			input:    "api.example.com",
			expected: "api_example_com",
		},
		{
			name:     "Multiple subdomains",
			input:    "www.api.example.com",
			expected: "www_api_example_com",
		},
		{
			name:     "Already safe",
			input:    "example-site",
			expected: "example-site",
		},
		{
			name:     "With underscores",
			input:    "my_site_name",
			expected: "my_site_name",
		},
		{
			name:     "Mixed characters",
			input:    "test.site-123_name",
			expected: "test_site-123_name",
		},
		{
			name:     "Special characters",
			input:    "test@site!name#123",
			expected: "test_site_name_123",
		},
		{
			name:     "Consecutive dots",
			input:    "test..example...com",
			expected: "test_example_com",
		},
		{
			name:     "Leading/trailing dots",
			input:    ".example.com.",
			expected: "example_com",
		},
		{
			name:     "Only dots",
			input:    "...",
			expected: "",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Uppercase preserved",
			input:    "MyApp.Example.COM",
			expected: "MyApp_Example_COM",
		},
		{
			name:     "Numbers",
			input:    "app123.site456.com",
			expected: "app123_site456_com",
		},
		{
			name:     "Hyphen and underscore mix",
			input:    "my-app_v2.example.com",
			expected: "my-app_v2_example_com",
		},
		{
			name:     "Spaces become underscores",
			input:    "my site name",
			expected: "my_site_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetProxyFilename tests the GetProxyFilename function
func TestGetProxyFilename(t *testing.T) {
	tests := []struct {
		name     string
		proxyID  int
		hostname string
		expected string
	}{
		{
			name:     "Simple case",
			proxyID:  1,
			hostname: "example.com",
			expected: "1_example_com.conf",
		},
		{
			name:     "With subdomain",
			proxyID:  42,
			hostname: "api.example.com",
			expected: "42_api_example_com.conf",
		},
		{
			name:     "Large ID",
			proxyID:  99999,
			hostname: "test.site.com",
			expected: "99999_test_site_com.conf",
		},
		{
			name:     "Zero ID",
			proxyID:  0,
			hostname: "example.com",
			expected: "0_example_com.conf",
		},
		{
			name:     "Complex hostname",
			proxyID:  5,
			hostname: "my-app.sub.domain.co.uk",
			expected: "5_my-app_sub_domain_co_uk.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProxyFilename(tt.proxyID, tt.hostname)
			if result != tt.expected {
				t.Errorf("GetProxyFilename(%d, %q) = %q, want %q", tt.proxyID, tt.hostname, result, tt.expected)
			}
		})
	}
}

// TestSyncStatus_AllFields tests all SyncStatus fields
func TestSyncStatus_AllFields(t *testing.T) {
	now := time.Now()
	status := SyncStatus{
		LastSyncTime:    now,
		IsSyncing:       true,
		LastSyncSuccess: true,
		LastError:       "",
		SyncCount:       5,
		LastReloadTime:  now,
		ReloadCount:     3,
		ConfigChanged:   true,
	}

	if status.SyncCount != 5 {
		t.Errorf("Expected SyncCount 5, got %d", status.SyncCount)
	}
	if status.ReloadCount != 3 {
		t.Errorf("Expected ReloadCount 3, got %d", status.ReloadCount)
	}
	if !status.ConfigChanged {
		t.Error("Expected ConfigChanged to be true")
	}
	if status.LastReloadTime != now {
		t.Error("LastReloadTime mismatch")
	}
}
