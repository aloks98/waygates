package validation

import (
	"testing"
)

func TestValidateRegisterRequest(t *testing.T) {
	testCases := []struct {
		name      string
		req       RegisterRequest
		expectErr bool
		errField  string
	}{
		{
			name: "Valid request",
			req: RegisterRequest{
				Name:     "John Doe",
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "password123",
			},
			expectErr: false,
		},
		{
			name: "Empty name",
			req: RegisterRequest{
				Name:     "",
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "password123",
			},
			expectErr: true,
			errField:  "name",
		},
		{
			name: "Username too short",
			req: RegisterRequest{
				Name:     "John Doe",
				Username: "jo",
				Email:    "john@example.com",
				Password: "password123",
			},
			expectErr: true,
			errField:  "username",
		},
		{
			name: "Username with invalid characters",
			req: RegisterRequest{
				Name:     "John Doe",
				Username: "john@doe",
				Email:    "john@example.com",
				Password: "password123",
			},
			expectErr: true,
			errField:  "username",
		},
		{
			name: "Invalid email",
			req: RegisterRequest{
				Name:     "John Doe",
				Username: "johndoe",
				Email:    "not-an-email",
				Password: "password123",
			},
			expectErr: true,
			errField:  "email",
		},
		{
			name: "Password too short",
			req: RegisterRequest{
				Name:     "John Doe",
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "pass",
			},
			expectErr: true,
			errField:  "password",
		},
		{
			name: "Username with underscore and hyphen",
			req: RegisterRequest{
				Name:     "John Doe",
				Username: "john_doe-123",
				Email:    "john@example.com",
				Password: "password123",
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStruct(&tc.req)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error for field '%s', got nil", tc.errField)
				} else if validErr, ok := err.(*ValidationError); ok {
					if validErr.Field != tc.errField {
						t.Errorf("Expected error field '%s', got '%s'", tc.errField, validErr.Field)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	testCases := []struct {
		name      string
		hostname  string
		expectErr bool
	}{
		{"Valid hostname", "example.com", false},
		{"Valid subdomain", "app.example.com", false},
		{"Hostname with scheme", "https://example.com", true},
		{"Hostname with port", "example.com:8080", true},
		{"Hostname with path", "example.com/path", true},
		{"Empty label", "example..com", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := ProxyRequest{
				Name:     "Test",
				Hostname: tc.hostname,
				Type:     "reverse_proxy",
			}
			err := ValidateStruct(&req)

			if tc.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateProxyType(t *testing.T) {
	testCases := []struct {
		name      string
		proxyType string
		expectErr bool
	}{
		{"Valid reverse_proxy", "reverse_proxy", false},
		{"Valid redirect", "redirect", false},
		{"Valid static", "static", false},
		{"Invalid type", "invalid", true},
		{"Empty type", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := ProxyRequest{
				Name:     "Test",
				Hostname: "example.com",
				Type:     tc.proxyType,
			}
			err := ValidateStruct(&req)

			if tc.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Name", "name"},
		{"Username", "username"},
		{"Email", "email"},
		{"RefreshToken", "refresh_token"},
		{"AccessToken", "access_token"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := toSnakeCase(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
