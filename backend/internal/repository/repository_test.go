package repository

import (
	"testing"
)

func TestNewSettingsRepository(t *testing.T) {
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewSettingsRepository(nil)
	if repo == nil {
		t.Fatal("Expected repository to be created")
	}
	if repo.db != nil {
		t.Error("Expected db to be nil")
	}
}

func TestNewProxyRepository(t *testing.T) {
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewProxyRepository(nil)
	if repo == nil {
		t.Fatal("Expected repository to be created")
	}
	if repo.db != nil {
		t.Error("Expected db to be nil")
	}
}

func TestNewUserRepository(t *testing.T) {
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewUserRepository(nil)
	if repo == nil {
		t.Fatal("Expected repository to be created")
	}
	if repo.db != nil {
		t.Error("Expected db to be nil")
	}
}

func TestProxyStats_Structure(t *testing.T) {
	// Test that ProxyStats can be initialized properly
	stats := ProxyStats{
		Total:    10,
		Active:   7,
		Inactive: 3,
		ByType:   make(map[string]int64),
	}

	if stats.Total != 10 {
		t.Errorf("Expected Total 10, got %d", stats.Total)
	}
	if stats.Active != 7 {
		t.Errorf("Expected Active 7, got %d", stats.Active)
	}
	if stats.Inactive != 3 {
		t.Errorf("Expected Inactive 3, got %d", stats.Inactive)
	}

	// Test ByType map
	stats.ByType["reverse_proxy"] = 5
	stats.ByType["redirect"] = 3
	stats.ByType["static"] = 2

	if stats.ByType["reverse_proxy"] != 5 {
		t.Errorf("Expected ByType['reverse_proxy'] 5, got %d", stats.ByType["reverse_proxy"])
	}
}

func TestProxyListParams_Structure(t *testing.T) {
	params := ProxyListParams{
		Page:   1,
		Limit:  10,
		Search: "test",
		Type:   "reverse_proxy",
		Status: "active",
		Sort:   "name",
		Order:  "asc",
	}

	if params.Page != 1 {
		t.Errorf("Expected Page 1, got %d", params.Page)
	}
	if params.Limit != 10 {
		t.Errorf("Expected Limit 10, got %d", params.Limit)
	}
	if params.Search != "test" {
		t.Errorf("Expected Search 'test', got '%s'", params.Search)
	}
	if params.Type != "reverse_proxy" {
		t.Errorf("Expected Type 'reverse_proxy', got '%s'", params.Type)
	}
	if params.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", params.Status)
	}
	if params.Sort != "name" {
		t.Errorf("Expected Sort 'name', got '%s'", params.Sort)
	}
	if params.Order != "asc" {
		t.Errorf("Expected Order 'asc', got '%s'", params.Order)
	}
}

func TestProxyListParams_Defaults(t *testing.T) {
	// Test zero value defaults
	params := ProxyListParams{}

	if params.Page != 0 {
		t.Errorf("Expected default Page 0, got %d", params.Page)
	}
	if params.Limit != 0 {
		t.Errorf("Expected default Limit 0, got %d", params.Limit)
	}
	if params.Search != "" {
		t.Errorf("Expected empty Search, got '%s'", params.Search)
	}
	if params.Type != "" {
		t.Errorf("Expected empty Type, got '%s'", params.Type)
	}
	if params.Status != "" {
		t.Errorf("Expected empty Status, got '%s'", params.Status)
	}
}
