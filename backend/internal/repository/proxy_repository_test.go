package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aloks98/waygates/backend/internal/models"
)

// TestListProxies_SQLInjectionPrevented tests that SQL injection attempts are blocked
func TestListProxies_SQLInjectionPrevented(t *testing.T) {
	t.Parallel()
	// Test that malicious sort values are rejected/sanitized
	testCases := []struct {
		name          string
		sortInput     string
		expectedField string
		orderInput    string
		expectedOrder string
	}{
		{
			name:          "Valid sort field",
			sortInput:     "name",
			expectedField: "name",
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in sort - DROP TABLE",
			sortInput:     "name; DROP TABLE proxies;--",
			expectedField: "created_at", // Should fallback to default
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in sort - UNION SELECT",
			sortInput:     "name UNION SELECT * FROM users--",
			expectedField: "created_at", // Should fallback to default
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "SQL injection in order - malicious order",
			sortInput:     "name",
			expectedField: "name",
			orderInput:    "ASC; DELETE FROM proxies;--",
			expectedOrder: "DESC", // Should fallback to default
		},
		{
			name:          "Case insensitive sort field",
			sortInput:     "NAME",
			expectedField: "name",
			orderInput:    "DESC",
			expectedOrder: "DESC",
		},
		{
			name:          "Case insensitive order",
			sortInput:     "hostname",
			expectedField: "hostname",
			orderInput:    "ASC",
			expectedOrder: "ASC",
		},
		{
			name:          "Invalid sort field - fallback to default",
			sortInput:     "password", // Not in whitelist
			expectedField: "created_at",
			orderInput:    "asc",
			expectedOrder: "ASC",
		},
		{
			name:          "Empty sort - use default",
			sortInput:     "",
			expectedField: "created_at",
			orderInput:    "",
			expectedOrder: "DESC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test sort field validation
			sortField := "created_at" // default
			if tc.sortInput != "" {
				if validField, ok := allowedSortFields[strings.ToLower(tc.sortInput)]; ok {
					sortField = validField
				}
			}

			if sortField != tc.expectedField {
				t.Errorf("Sort field: expected '%s', got '%s'", tc.expectedField, sortField)
			}

			// Test sort order validation
			sortOrder := "DESC" // default
			if tc.orderInput != "" {
				if validOrder, ok := allowedSortOrders[strings.ToLower(tc.orderInput)]; ok {
					sortOrder = validOrder
				}
			}

			if sortOrder != tc.expectedOrder {
				t.Errorf("Sort order: expected '%s', got '%s'", tc.expectedOrder, sortOrder)
			}
		})
	}
}

// TestAllowedSortFields verifies the whitelist is correctly defined
func TestAllowedSortFields(t *testing.T) {
	t.Parallel()
	expectedFields := []string{"id", "name", "hostname", "type", "is_active", "created_at", "updated_at"}

	for _, field := range expectedFields {
		if _, ok := allowedSortFields[field]; !ok {
			t.Errorf("Expected field '%s' to be in allowedSortFields", field)
		}
	}

	// Verify dangerous fields are NOT in the whitelist
	dangerousFields := []string{"password", "created_by", "secret", "token"}
	for _, field := range dangerousFields {
		if _, ok := allowedSortFields[field]; ok {
			t.Errorf("Dangerous field '%s' should NOT be in allowedSortFields", field)
		}
	}
}

// TestAllowedSortOrders verifies the order whitelist is correctly defined
func TestAllowedSortOrders(t *testing.T) {
	t.Parallel()
	// Should only allow asc and desc
	if _, ok := allowedSortOrders["asc"]; !ok {
		t.Error("Expected 'asc' to be in allowedSortOrders")
	}
	if _, ok := allowedSortOrders["desc"]; !ok {
		t.Error("Expected 'desc' to be in allowedSortOrders")
	}

	// Should return uppercase values
	if allowedSortOrders["asc"] != "ASC" {
		t.Errorf("Expected allowedSortOrders['asc'] to be 'ASC', got '%s'", allowedSortOrders["asc"])
	}
	if allowedSortOrders["desc"] != "DESC" {
		t.Errorf("Expected allowedSortOrders['desc'] to be 'DESC', got '%s'", allowedSortOrders["desc"])
	}
}

func TestApplyACLSummaries(t *testing.T) {
	proxies := []models.Proxy{{ID: 1}, {ID: 2}, {ID: 3}}
	rows := []aclSummaryRow{
		{ProxyID: 1, Name: "vpn"},
		{ProxyID: 1, Name: "staff"},
		{ProxyID: 3, Name: "vpn"},
	}
	applyACLSummaries(proxies, rows)

	assert.Equal(t, 2, proxies[0].ACLGroupCount)
	assert.Equal(t, []string{"vpn", "staff"}, proxies[0].ACLGroupNames)
	assert.Equal(t, 0, proxies[1].ACLGroupCount)
	assert.Equal(t, []string{}, proxies[1].ACLGroupNames) // never nil
	assert.Equal(t, 1, proxies[2].ACLGroupCount)
	assert.Equal(t, []string{"vpn"}, proxies[2].ACLGroupNames)
}
