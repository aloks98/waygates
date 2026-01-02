package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestProxy_Validate tests the Proxy validation function
func TestProxy_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		proxy       Proxy
		expectedErr error
	}{
		{
			name: "Valid reverse_proxy",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test Proxy",
				Hostname:  "test.example.com",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: nil,
		},
		{
			name: "Valid redirect",
			proxy: Proxy{
				Type:     ProxyTypeRedirect,
				Name:     "Redirect Proxy",
				Hostname: "redirect.example.com",
				RedirectConfig: JSONField{
					"url":  "https://target.example.com",
					"code": 301,
				},
			},
			expectedErr: nil,
		},
		{
			name: "Valid static",
			proxy: Proxy{
				Type:     ProxyTypeStatic,
				Name:     "Static Proxy",
				Hostname: "static.example.com",
				StaticConfig: JSONField{
					"root": "/var/www/html",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Invalid proxy type",
			proxy: Proxy{
				Type:     "invalid_type",
				Name:     "Test",
				Hostname: "test.example.com",
			},
			expectedErr: ErrInvalidProxyType,
		},
		{
			name: "Empty name",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "",
				Hostname:  "test.example.com",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrProxyNameRequired,
		},
		{
			name: "Name too long",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      strings.Repeat("a", 256),
				Hostname:  "test.example.com",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrProxyNameTooLong,
		},
		{
			name: "Empty hostname",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  "",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrProxyHostnameRequired,
		},
		{
			name: "Hostname with scheme",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  "https://test.example.com",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrHostnameHasScheme,
		},
		{
			name: "Hostname with port",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  "test.example.com:8080",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrHostnameHasPort,
		},
		{
			name: "Hostname with path",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  "test.example.com/path",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrHostnameHasPath,
		},
		{
			name: "Hostname too long",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  strings.Repeat("a", 254),
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrHostnameTooLong,
		},
		{
			name: "Hostname with empty label",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  "test..example.com",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrHostnameEmptyLabel,
		},
		{
			name: "Hostname label too long",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  strings.Repeat("a", 64) + ".example.com",
				Upstreams: []interface{}{"http://localhost:8080"},
			},
			expectedErr: ErrHostnameLabelTooLong,
		},
		{
			name: "Reverse proxy without upstreams",
			proxy: Proxy{
				Type:     ProxyTypeReverseProxy,
				Name:     "Test",
				Hostname: "test.example.com",
			},
			expectedErr: ErrUpstreamsRequired,
		},
		{
			name: "Reverse proxy with empty upstreams",
			proxy: Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  "test.example.com",
				Upstreams: []interface{}{},
			},
			expectedErr: ErrUpstreamsRequired,
		},
		{
			name: "Redirect without config",
			proxy: Proxy{
				Type:     ProxyTypeRedirect,
				Name:     "Test",
				Hostname: "test.example.com",
			},
			expectedErr: ErrRedirectConfigRequired,
		},
		{
			name: "Static without config",
			proxy: Proxy{
				Type:     ProxyTypeStatic,
				Name:     "Test",
				Hostname: "test.example.com",
			},
			expectedErr: ErrStaticConfigRequired,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.proxy.Validate()

			if tc.expectedErr == nil {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tc.expectedErr)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("Expected error %v, got %v", tc.expectedErr, err)
				}
			}
		})
	}
}

// TestProxy_ValidateHostname tests hostname validation edge cases
func TestProxy_ValidateHostname(t *testing.T) {
	testCases := []struct {
		name     string
		hostname string
		valid    bool
	}{
		{"Simple hostname", "example.com", true},
		{"Subdomain", "app.example.com", true},
		{"Deep subdomain", "a.b.c.example.com", true},
		{"Single label", "localhost", true},
		{"With numbers", "app123.example.com", true},
		{"With hyphens", "my-app.example.com", true},
		{"Maximum length label (63)", strings.Repeat("a", 63), true},
		{"Maximum length hostname (253)", strings.Repeat("a.", 126) + "a", true},
		{"HTTP scheme", "http://example.com", false},
		{"HTTPS scheme", "https://example.com", false},
		{"With port", "example.com:443", false},
		{"With path", "example.com/path", false},
		{"Empty label at start", ".example.com", false},
		{"Empty label in middle", "app..example.com", false},
		{"Empty label at end", "example.com.", false},
		{"Label too long (64)", strings.Repeat("a", 64) + ".com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proxy := Proxy{
				Type:      ProxyTypeReverseProxy,
				Name:      "Test",
				Hostname:  tc.hostname,
				Upstreams: []interface{}{"http://localhost:8080"},
			}

			err := proxy.Validate()

			if tc.valid && err != nil {
				t.Errorf("Expected valid hostname, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Errorf("Expected invalid hostname, got no error")
			}
		})
	}
}

// TestJSONField_MarshalUnmarshal tests JSONField serialization
func TestJSONField_MarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name     string
		input    JSONField
		expected string
	}{
		{
			name:     "Simple map",
			input:    JSONField{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "Nested map",
			input:    JSONField{"outer": map[string]interface{}{"inner": "value"}},
			expected: `{"outer":{"inner":"value"}}`,
		},
		{
			name:     "With numbers",
			input:    JSONField{"count": float64(42)},
			expected: `{"count":42}`,
		},
		{
			name:     "With boolean",
			input:    JSONField{"active": true},
			expected: `{"active":true}`,
		},
		{
			name:     "Nil field",
			input:    nil,
			expected: "null",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test marshal
			data, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if string(data) != tc.expected {
				t.Errorf("Marshal: expected %s, got %s", tc.expected, string(data))
			}

			// Test unmarshal
			var result JSONField
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Compare (nil case)
			if tc.input == nil {
				if result != nil {
					t.Errorf("Unmarshal: expected nil, got %v", result)
				}
				return
			}

			// Compare keys
			for k, v := range tc.input {
				if result[k] == nil {
					t.Errorf("Unmarshal: missing key %s", k)
					continue
				}
				// Simple comparison for strings
				if sv, ok := v.(string); ok {
					if result[k] != sv {
						t.Errorf("Unmarshal: key %s expected %v, got %v", k, v, result[k])
					}
				}
			}
		})
	}
}

// TestJSONField_Value tests database value conversion
func TestJSONField_Value(t *testing.T) {
	testCases := []struct {
		name        string
		input       JSONField
		expectNil   bool
	}{
		{"Nil field", nil, true},
		{"Empty map", JSONField{}, false},
		{"With data", JSONField{"key": "value"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := tc.input.Value()
			if err != nil {
				t.Fatalf("Value() error: %v", err)
			}

			if tc.expectNil && val != nil {
				t.Errorf("Expected nil, got %v", val)
			}
			if !tc.expectNil && val == nil {
				t.Error("Expected non-nil value")
			}
		})
	}
}

// TestJSONField_Scan tests database scan conversion
func TestJSONField_Scan(t *testing.T) {
	testCases := []struct {
		name      string
		input     interface{}
		expectNil bool
	}{
		{"Nil value", nil, true},
		{"Empty JSON", []byte("{}"), false},
		{"With data", []byte(`{"key":"value"}`), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var j JSONField
			err := j.Scan(tc.input)
			if err != nil {
				t.Fatalf("Scan() error: %v", err)
			}

			if tc.expectNil && j != nil {
				t.Errorf("Expected nil, got %v", j)
			}
			if !tc.expectNil && j == nil {
				t.Error("Expected non-nil value")
			}
		})
	}
}

// TestProxy_TableName tests the table name
func TestProxy_TableName(t *testing.T) {
	proxy := Proxy{}
	if proxy.TableName() != "proxies" {
		t.Errorf("Expected table name 'proxies', got '%s'", proxy.TableName())
	}
}

// TestProxyTypes tests proxy type constants
func TestProxyTypes(t *testing.T) {
	if ProxyTypeReverseProxy != "reverse_proxy" {
		t.Errorf("Expected ProxyTypeReverseProxy to be 'reverse_proxy', got '%s'", ProxyTypeReverseProxy)
	}
	if ProxyTypeRedirect != "redirect" {
		t.Errorf("Expected ProxyTypeRedirect to be 'redirect', got '%s'", ProxyTypeRedirect)
	}
	if ProxyTypeStatic != "static" {
		t.Errorf("Expected ProxyTypeStatic to be 'static', got '%s'", ProxyTypeStatic)
	}
}

// TestValidationError tests the ValidationError type
func TestValidationError(t *testing.T) {
	err := &ValidationError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", err.Error())
	}
}

// TestPagination tests Pagination struct
func TestPagination(t *testing.T) {
	p := Pagination{
		CurrentPage:  1,
		TotalPages:   5,
		TotalItems:   100,
		ItemsPerPage: 20,
		HasNext:      true,
		HasPrev:      false,
	}

	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage 1, got %d", p.CurrentPage)
	}
	if p.TotalPages != 5 {
		t.Errorf("Expected TotalPages 5, got %d", p.TotalPages)
	}
	if p.TotalItems != 100 {
		t.Errorf("Expected TotalItems 100, got %d", p.TotalItems)
	}
	if !p.HasNext {
		t.Error("Expected HasNext to be true")
	}
	if p.HasPrev {
		t.Error("Expected HasPrev to be false")
	}
}
