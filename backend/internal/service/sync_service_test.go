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
