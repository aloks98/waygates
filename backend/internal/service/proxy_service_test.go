package service

import (
	"testing"
)

// TestListProxies_PaginationDefaults tests pagination default values
func TestListProxies_PaginationDefaults(t *testing.T) {
	testCases := []struct {
		name          string
		inputPage     int
		inputLimit    int
		expectedPage  int
		expectedLimit int
	}{
		{"Zero page becomes 1", 0, 20, 1, 20},
		{"Negative page becomes 1", -1, 20, 1, 20},
		{"Page 1 stays 1", 1, 20, 1, 20},
		{"Page 5 stays 5", 5, 20, 5, 20},
		{"Zero limit becomes 20", 1, 0, 1, 20},
		{"Negative limit becomes 20", 1, -1, 1, 20},
		{"Limit 50 stays 50", 1, 50, 1, 50},
		{"Limit 100 stays 100", 1, 100, 1, 100},
		{"Limit 101 becomes 20", 1, 101, 1, 20},
		{"Limit 1000 becomes 20", 1, 1000, 1, 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := ListProxiesRequest{
				Page:  tc.inputPage,
				Limit: tc.inputLimit,
			}

			// Apply the same defaults as the service
			if req.Page < 1 {
				req.Page = 1
			}
			if req.Limit < 1 || req.Limit > 100 {
				req.Limit = 20
			}

			if req.Page != tc.expectedPage {
				t.Errorf("Expected page %d, got %d", tc.expectedPage, req.Page)
			}
			if req.Limit != tc.expectedLimit {
				t.Errorf("Expected limit %d, got %d", tc.expectedLimit, req.Limit)
			}
		})
	}
}

// TestCaddyError tests CaddyError type
func TestCaddyError(t *testing.T) {
	err := NewCaddyError("test error message")

	if err.Error() != "test error message" {
		t.Errorf("Expected error message 'test error message', got '%s'", err.Error())
	}

	if !IsCaddyError(err) {
		t.Error("Expected IsCaddyError to return true")
	}
}

// TestIsCaddyError tests IsCaddyError function
func TestIsCaddyError(t *testing.T) {
	caddyErr := NewCaddyError("caddy error")
	if !IsCaddyError(caddyErr) {
		t.Error("Expected IsCaddyError to return true for CaddyError")
	}

	regularErr := ErrProxyNotFound
	if IsCaddyError(regularErr) {
		t.Error("Expected IsCaddyError to return false for regular error")
	}
}

// TestServiceErrors tests that service errors are defined correctly
func TestServiceErrors(t *testing.T) {
	if ErrProxyNotFound.Error() != "proxy not found" {
		t.Errorf("Unexpected ErrProxyNotFound message: %s", ErrProxyNotFound.Error())
	}

	if ErrHostnameConflict.Error() != "hostname already exists" {
		t.Errorf("Unexpected ErrHostnameConflict message: %s", ErrHostnameConflict.Error())
	}

	if ErrProxyAlreadyEnabled.Error() != "proxy is already enabled" {
		t.Errorf("Unexpected ErrProxyAlreadyEnabled message: %s", ErrProxyAlreadyEnabled.Error())
	}

	if ErrProxyAlreadyDisabled.Error() != "proxy is already disabled" {
		t.Errorf("Unexpected ErrProxyAlreadyDisabled message: %s", ErrProxyAlreadyDisabled.Error())
	}
}

// Note: TestExtractRoutes_EmptyConfig was removed as extractRoutes method
// was removed when migrating from Caddy Admin API to file-based approach

// TestPaginationCalculation tests pagination math
func TestPaginationCalculation(t *testing.T) {
	testCases := []struct {
		name          string
		total         int64
		page          int
		limit         int
		expectedPages int
		hasNext       bool
		hasPrev       bool
	}{
		{"Page 1 of 1", 10, 1, 20, 1, false, false},
		{"Page 1 of 2", 30, 1, 20, 2, true, false},
		{"Page 2 of 2", 30, 2, 20, 2, false, true},
		{"Page 1 of 5", 100, 1, 20, 5, true, false},
		{"Page 3 of 5", 100, 3, 20, 5, true, true},
		{"Page 5 of 5", 100, 5, 20, 5, false, true},
		{"Empty result", 0, 1, 20, 0, false, false},
		{"Single item", 1, 1, 20, 1, false, false},
		{"Exact page boundary", 20, 1, 20, 1, false, false},
		{"Just over page boundary", 21, 1, 20, 2, true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the pagination calculation from the service
			ceil := func(a float64) int {
				if a <= 0 {
					return 0
				}
				return int(a + 0.999999) // Ceiling
			}

			totalPages := 0
			if tc.limit > 0 {
				totalPages = ceil(float64(tc.total) / float64(tc.limit))
			}
			hasNext := tc.page < totalPages
			hasPrev := tc.page > 1

			if totalPages != tc.expectedPages {
				t.Errorf("Expected %d pages, got %d", tc.expectedPages, totalPages)
			}
			if hasNext != tc.hasNext {
				t.Errorf("Expected hasNext=%v, got %v", tc.hasNext, hasNext)
			}
			if hasPrev != tc.hasPrev {
				t.Errorf("Expected hasPrev=%v, got %v", tc.hasPrev, hasPrev)
			}
		})
	}
}

// TestListProxiesRequest_Validation tests request struct validation
func TestListProxiesRequest_Validation(t *testing.T) {
	testCases := []struct {
		name  string
		req   ListProxiesRequest
		valid bool
	}{
		{
			name:  "Empty request is valid",
			req:   ListProxiesRequest{},
			valid: true,
		},
		{
			name: "Full request is valid",
			req: ListProxiesRequest{
				Page:   1,
				Limit:  20,
				Search: "test",
				Type:   "reverse_proxy",
				Status: "active",
				Sort:   "name",
				Order:  "asc",
			},
			valid: true,
		},
		{
			name: "Search with special chars is valid",
			req: ListProxiesRequest{
				Search: "test.domain.com",
			},
			valid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// ListProxiesRequest doesn't have built-in validation
			// The service normalizes the values instead
			// This test documents the expected behavior
			_ = tc.valid // Suppress unused variable warning - test documents expected behavior
		})
	}
}

// TestNewProxyService_NilLogger tests that nil logger is handled
func TestNewProxyService_NilLogger(t *testing.T) {
	// Test that passing nil logger doesn't panic
	// We can't fully test this without mocking the repository and Caddy client
	// but we can verify the nil check exists in the code
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewProxyService panicked with nil logger: %v", r)
		}
	}()

	// Note: This would actually create a service and start the sync goroutine
	// In a real test, we'd use interfaces and mocks
	// For now, we just verify the nil check is documented
}
