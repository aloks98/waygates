package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSettingsRepository(t *testing.T) {
	t.Parallel()
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewSettingsRepository(nil)
	require.NotNil(t, repo, "Expected repository to be created")
	require.Nil(t, repo.db, "Expected db to be nil")
}

func TestNewProxyRepository(t *testing.T) {
	t.Parallel()
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewProxyRepository(nil)
	require.NotNil(t, repo, "Expected repository to be created")
	require.Nil(t, repo.db, "Expected db to be nil")
}

func TestNewUserRepository(t *testing.T) {
	t.Parallel()
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewUserRepository(nil)
	require.NotNil(t, repo, "Expected repository to be created")
	require.Nil(t, repo.db, "Expected db to be nil")
}

func TestProxyStats_Structure(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	sslEnabled := true
	params := ProxyListParams{
		Page:         1,
		Limit:        10,
		Search:       "test",
		Types:        []string{"reverse_proxy", "redirect"},
		TypesExclude: []string{"static"},
		Status:       "active",
		StatusNot:    "inactive",
		SSLEnabled:   &sslEnabled,
		Target:       "localhost:8080",
		Sort:         "name",
		Order:        "asc",
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
	if len(params.Types) != 2 || params.Types[0] != "reverse_proxy" {
		t.Errorf("Expected Types ['reverse_proxy', 'redirect'], got %v", params.Types)
	}
	if len(params.TypesExclude) != 1 || params.TypesExclude[0] != "static" {
		t.Errorf("Expected TypesExclude ['static'], got %v", params.TypesExclude)
	}
	if params.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", params.Status)
	}
	if params.StatusNot != "inactive" {
		t.Errorf("Expected StatusNot 'inactive', got '%s'", params.StatusNot)
	}
	if params.SSLEnabled == nil || *params.SSLEnabled != true {
		t.Errorf("Expected SSLEnabled true, got %v", params.SSLEnabled)
	}
	if params.Target != "localhost:8080" {
		t.Errorf("Expected Target 'localhost:8080', got '%s'", params.Target)
	}
	if params.Sort != "name" {
		t.Errorf("Expected Sort 'name', got '%s'", params.Sort)
	}
	if params.Order != "asc" {
		t.Errorf("Expected Order 'asc', got '%s'", params.Order)
	}
}

func TestProxyListParams_Defaults(t *testing.T) {
	t.Parallel()
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
	if len(params.Types) != 0 {
		t.Errorf("Expected empty Types, got %v", params.Types)
	}
	if len(params.TypesExclude) != 0 {
		t.Errorf("Expected empty TypesExclude, got %v", params.TypesExclude)
	}
	if params.Status != "" {
		t.Errorf("Expected empty Status, got '%s'", params.Status)
	}
	if params.StatusNot != "" {
		t.Errorf("Expected empty StatusNot, got '%s'", params.StatusNot)
	}
	if params.SSLEnabled != nil {
		t.Errorf("Expected nil SSLEnabled, got %v", params.SSLEnabled)
	}
	if params.Target != "" {
		t.Errorf("Expected empty Target, got '%s'", params.Target)
	}
}
