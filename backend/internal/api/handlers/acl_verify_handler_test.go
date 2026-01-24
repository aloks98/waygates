package handlers

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Unit Tests for Helper Functions
// =============================================================================

// TestExtractBaseDomainFallback tests the extractBaseDomainFallback function
func TestExtractBaseDomainFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "simple two-part domain",
			host:     "example.com",
			expected: ".example.com",
		},
		{
			name:     "three-part domain",
			host:     "app.example.com",
			expected: ".example.com",
		},
		{
			name:     "four-part domain",
			host:     "deep.app.example.com",
			expected: ".example.com",
		},
		{
			name:     "internal domain",
			host:     "app.internal.local",
			expected: ".internal.local",
		},
		{
			name:     "single part - returns empty",
			host:     "localhost",
			expected: "",
		},
		{
			name:     "empty host",
			host:     "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractBaseDomainFallback(tc.host)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestParseBasicAuthHeader tests the parseBasicAuth function
// Note: Named differently from integration test to avoid conflict
func TestParseBasicAuthHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		authHeader   string
		expectedUser string
		expectedPass string
		expectNil    bool
	}{
		{
			name:         "valid basic auth",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("user:password")),
			expectedUser: "user",
			expectedPass: "password",
			expectNil:    false,
		},
		{
			name:         "valid basic auth with colon in password",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass:with:colons")),
			expectedUser: "user",
			expectedPass: "pass:with:colons",
			expectNil:    false,
		},
		{
			name:         "valid basic auth with empty password",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("user:")),
			expectedUser: "user",
			expectedPass: "",
			expectNil:    false,
		},
		{
			name:       "missing Basic prefix",
			authHeader: base64.StdEncoding.EncodeToString([]byte("user:password")),
			expectNil:  true,
		},
		{
			name:       "Bearer token instead of Basic",
			authHeader: "Bearer sometoken",
			expectNil:  true,
		},
		{
			name:       "invalid base64",
			authHeader: "Basic not-valid-base64!!!",
			expectNil:  true,
		},
		{
			name:       "no colon in decoded string",
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("useronly")),
			expectNil:  true,
		},
		{
			name:       "empty header",
			authHeader: "",
			expectNil:  true,
		},
		{
			name:       "only Basic keyword",
			authHeader: "Basic ",
			expectNil:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parseBasicAuth(tc.authHeader)

			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedUser, result.Username)
				assert.Equal(t, tc.expectedPass, result.Password)
			}
		})
	}
}

// TestIsIPAddressUnit tests the isIPAddress function
func TestIsIPAddressUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{
			name:     "IPv4 address",
			host:     "192.168.1.1",
			expected: true,
		},
		{
			name:     "IPv4 localhost",
			host:     "127.0.0.1",
			expected: true,
		},
		{
			name:     "IPv6 address with colons",
			host:     "::1",
			expected: true,
		},
		{
			name:     "IPv6 full address",
			host:     "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: true,
		},
		{
			name:     "regular hostname",
			host:     "example.com",
			expected: false,
		},
		{
			name:     "hostname with subdomain",
			host:     "app.example.com",
			expected: false,
		},
		{
			name:     "localhost",
			host:     "localhost",
			expected: false,
		},
		{
			name:     "hostname with numbers",
			host:     "server1.example.com",
			expected: false,
		},
		{
			name:     "empty string",
			host:     "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isIPAddress(tc.host)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestExtractCookieDomainFromHostUnit tests the extractCookieDomainFromHost function
func TestExtractCookieDomainFromHostUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "simple domain with port",
			host:     "example.com:8080",
			expected: ".example.com",
		},
		{
			name:     "subdomain with port",
			host:     "app.example.com:443",
			expected: ".example.com",
		},
		{
			name:     "domain without port",
			host:     "example.com",
			expected: ".example.com",
		},
		{
			name:     "localhost with port",
			host:     "localhost:8080",
			expected: "",
		},
		{
			name:     "localhost without port",
			host:     "localhost",
			expected: "",
		},
		{
			name:     "IP address with port",
			host:     "192.168.1.1:8080",
			expected: "",
		},
		{
			name:     "IP address without port",
			host:     "192.168.1.1",
			expected: "",
		},
		{
			name:     "deep subdomain",
			host:     "deep.nested.example.com:9000",
			expected: ".example.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractCookieDomainFromHost(tc.host)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestExtractCookieDomainUnit tests the extractCookieDomain function
func TestExtractCookieDomainUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "simple HTTPS URL",
			rawURL:   "https://example.com/path",
			expected: ".example.com",
		},
		{
			name:     "URL with subdomain",
			rawURL:   "https://app.example.com/dashboard",
			expected: ".example.com",
		},
		{
			name:     "URL with deep subdomain",
			rawURL:   "https://deep.nested.example.com/path",
			expected: ".example.com",
		},
		{
			name:     "HTTP localhost",
			rawURL:   "http://localhost:8080/path",
			expected: "",
		},
		{
			name:     "IP address URL",
			rawURL:   "http://192.168.1.1:8080/api",
			expected: "",
		},
		{
			name:     "empty URL",
			rawURL:   "",
			expected: "",
		},
		{
			name:     "invalid URL",
			rawURL:   "://invalid",
			expected: "",
		},
		{
			name:     "URL with query params",
			rawURL:   "https://app.example.com/path?query=value",
			expected: ".example.com",
		},
		{
			name:     "URL with fragment",
			rawURL:   "https://app.example.com/path#section",
			expected: ".example.com",
		},
		{
			name:     "URL with port",
			rawURL:   "https://app.example.com:8443/secure",
			expected: ".example.com",
		},
		{
			name:     "relative path only",
			rawURL:   "/dashboard",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractCookieDomain(tc.rawURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestExtractBaseDomainFallbackEdgeCases tests edge cases for extractBaseDomainFallback
func TestExtractBaseDomainFallbackEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "just a dot",
			host:     ".",
			expected: "..", // Split on "." gives ["", ""], join last 2 = ".", prepend "." = ".."
		},
		{
			name:     "multiple dots only",
			host:     "...",
			expected: "..", // Split on "." gives ["", "", "", ""], join last 2 = ".", prepend "." = ".."
		},
		{
			name:     "trailing dot",
			host:     "example.com.",
			expected: ".com.", // Split gives ["example", "com", ""], join last 2 = "com.", prepend "." = ".com."
		},
		{
			name:     "leading dot",
			host:     ".example.com",
			expected: ".example.com", // Split gives ["", "example", "com"], join last 2 = "example.com", prepend "." = ".example.com"
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractBaseDomainFallback(tc.host)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestParseBasicAuthEdgeCases tests edge cases for parseBasicAuth
func TestParseBasicAuthEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		authHeader   string
		expectedUser string
		expectedPass string
		expectNil    bool
	}{
		{
			name:         "unicode username and password",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("user:password")),
			expectedUser: "user",
			expectedPass: "password",
			expectNil:    false,
		},
		{
			name:         "special characters in password",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte("user:p@ss!#$%^&*()")),
			expectedUser: "user",
			expectedPass: "p@ss!#$%^&*()",
			expectNil:    false,
		},
		{
			name:         "empty username",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte(":password")),
			expectedUser: "",
			expectedPass: "password",
			expectNil:    false,
		},
		{
			name:         "both empty",
			authHeader:   "Basic " + base64.StdEncoding.EncodeToString([]byte(":")),
			expectedUser: "",
			expectedPass: "",
			expectNil:    false,
		},
		{
			name:       "lowercase basic",
			authHeader: "basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
			expectNil:  true, // Should be case-sensitive
		},
		{
			name:       "extra spaces",
			authHeader: "Basic  " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
			expectNil:  true, // Double space makes invalid base64
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parseBasicAuth(tc.authHeader)

			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedUser, result.Username)
				assert.Equal(t, tc.expectedPass, result.Password)
			}
		})
	}
}
