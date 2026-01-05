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
		Types:  []string{"reverse_proxy"},
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
	if len(params.Types) != 1 || params.Types[0] != "reverse_proxy" {
		t.Errorf("Expected Types ['reverse_proxy'], got %v", params.Types)
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
	if len(params.Types) != 0 {
		t.Errorf("Expected empty Types, got %v", params.Types)
	}
	if params.Status != "" {
		t.Errorf("Expected empty Status, got '%s'", params.Status)
	}
}

func TestProxyListParams_MultipleTypes(t *testing.T) {
	// Test with multiple types (IN query)
	params := ProxyListParams{
		Page:  1,
		Limit: 10,
		Types: []string{"reverse_proxy", "redirect", "static"},
	}

	if len(params.Types) != 3 {
		t.Errorf("Expected 3 types, got %d", len(params.Types))
	}
	if params.Types[0] != "reverse_proxy" {
		t.Errorf("Expected first type 'reverse_proxy', got '%s'", params.Types[0])
	}
	if params.Types[1] != "redirect" {
		t.Errorf("Expected second type 'redirect', got '%s'", params.Types[1])
	}
	if params.Types[2] != "static" {
		t.Errorf("Expected third type 'static', got '%s'", params.Types[2])
	}
}

func TestProxyListParams_TypesExclude(t *testing.T) {
	// Test with excluded types (NOT IN query)
	params := ProxyListParams{
		Page:         1,
		Limit:        10,
		TypesExclude: []string{"static", "redirect"},
	}

	if len(params.TypesExclude) != 2 {
		t.Errorf("Expected 2 excluded types, got %d", len(params.TypesExclude))
	}
	if params.TypesExclude[0] != "static" {
		t.Errorf("Expected first excluded type 'static', got '%s'", params.TypesExclude[0])
	}
	if params.TypesExclude[1] != "redirect" {
		t.Errorf("Expected second excluded type 'redirect', got '%s'", params.TypesExclude[1])
	}
}

func TestProxyListParams_CombinedTypesAndExclude(t *testing.T) {
	// Test with both types and excluded types (edge case)
	params := ProxyListParams{
		Page:         1,
		Limit:        10,
		Types:        []string{"reverse_proxy"},
		TypesExclude: []string{"static"},
	}

	if len(params.Types) != 1 {
		t.Errorf("Expected 1 type, got %d", len(params.Types))
	}
	if len(params.TypesExclude) != 1 {
		t.Errorf("Expected 1 excluded type, got %d", len(params.TypesExclude))
	}
}

func TestProxyListParams_StatusNot(t *testing.T) {
	// Test with status negation
	params := ProxyListParams{
		Page:      1,
		Limit:     10,
		StatusNot: "active",
	}

	if params.StatusNot != "active" {
		t.Errorf("Expected StatusNot 'active', got '%s'", params.StatusNot)
	}
	if params.Status != "" {
		t.Errorf("Expected empty Status, got '%s'", params.Status)
	}
}

func TestProxyListParams_StatusNotInactive(t *testing.T) {
	// Test with status negation for inactive
	params := ProxyListParams{
		Page:      1,
		Limit:     10,
		StatusNot: "inactive",
	}

	if params.StatusNot != "inactive" {
		t.Errorf("Expected StatusNot 'inactive', got '%s'", params.StatusNot)
	}
}

func TestProxyListParams_ComplexFilter(t *testing.T) {
	// Test with complex filter combination
	params := ProxyListParams{
		Page:         1,
		Limit:        20,
		Search:       "example",
		Types:        []string{"reverse_proxy", "redirect"},
		TypesExclude: nil, // Not excluding any
		Status:       "active",
		StatusNot:    "", // Not negating
		Sort:         "name",
		Order:        "asc",
	}

	if params.Search != "example" {
		t.Errorf("Expected Search 'example', got '%s'", params.Search)
	}
	if len(params.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(params.Types))
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

func TestProxyListParams_FilterTypeCombinations(t *testing.T) {
	testCases := []struct {
		name         string
		params       ProxyListParams
		expectTypes  int
		expectExcl   int
		expectStatus bool
		expectNot    bool
	}{
		{
			name: "types only",
			params: ProxyListParams{
				Types: []string{"reverse_proxy"},
			},
			expectTypes:  1,
			expectExcl:   0,
			expectStatus: false,
			expectNot:    false,
		},
		{
			name: "exclude only",
			params: ProxyListParams{
				TypesExclude: []string{"static", "redirect"},
			},
			expectTypes:  0,
			expectExcl:   2,
			expectStatus: false,
			expectNot:    false,
		},
		{
			name: "status active",
			params: ProxyListParams{
				Status: "active",
			},
			expectTypes:  0,
			expectExcl:   0,
			expectStatus: true,
			expectNot:    false,
		},
		{
			name: "status inactive",
			params: ProxyListParams{
				Status: "inactive",
			},
			expectTypes:  0,
			expectExcl:   0,
			expectStatus: true,
			expectNot:    false,
		},
		{
			name: "status not active",
			params: ProxyListParams{
				StatusNot: "active",
			},
			expectTypes:  0,
			expectExcl:   0,
			expectStatus: false,
			expectNot:    true,
		},
		{
			name: "status not inactive",
			params: ProxyListParams{
				StatusNot: "inactive",
			},
			expectTypes:  0,
			expectExcl:   0,
			expectStatus: false,
			expectNot:    true,
		},
		{
			name: "all filters",
			params: ProxyListParams{
				Search:       "test",
				Types:        []string{"reverse_proxy", "redirect"},
				TypesExclude: []string{"static"},
				Status:       "active",
				StatusNot:    "inactive",
			},
			expectTypes:  2,
			expectExcl:   1,
			expectStatus: true,
			expectNot:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.params.Types) != tc.expectTypes {
				t.Errorf("Expected %d types, got %d", tc.expectTypes, len(tc.params.Types))
			}
			if len(tc.params.TypesExclude) != tc.expectExcl {
				t.Errorf("Expected %d excluded types, got %d", tc.expectExcl, len(tc.params.TypesExclude))
			}
			if (tc.params.Status != "") != tc.expectStatus {
				t.Errorf("Expected status present: %v, got status: '%s'", tc.expectStatus, tc.params.Status)
			}
			if (tc.params.StatusNot != "") != tc.expectNot {
				t.Errorf("Expected status not present: %v, got statusNot: '%s'", tc.expectNot, tc.params.StatusNot)
			}
		})
	}
}

func TestProxyListParams_EmptyTypes(t *testing.T) {
	// Test that empty slice behaves correctly
	params := ProxyListParams{
		Types:        []string{},
		TypesExclude: []string{},
	}

	if len(params.Types) != 0 {
		t.Errorf("Expected empty Types slice, got %v", params.Types)
	}
	if len(params.TypesExclude) != 0 {
		t.Errorf("Expected empty TypesExclude slice, got %v", params.TypesExclude)
	}
}

func TestProxyListParams_SingleType(t *testing.T) {
	// Test with single type (backward compatibility check)
	params := ProxyListParams{
		Page:  1,
		Limit: 10,
		Types: []string{"reverse_proxy"},
	}

	if len(params.Types) != 1 {
		t.Errorf("Expected 1 type, got %d", len(params.Types))
	}
	if params.Types[0] != "reverse_proxy" {
		t.Errorf("Expected type 'reverse_proxy', got '%s'", params.Types[0])
	}
}
