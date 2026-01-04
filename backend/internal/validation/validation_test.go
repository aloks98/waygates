package validation

import (
	"errors"
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
				} else {
					var validErr *ValidationError
					if errors.As(err, &validErr) && validErr.Field != tc.errField {
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

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "username",
		Message: "username is required",
	}

	expected := "username: username is required"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestValidateStructAll(t *testing.T) {
	// Test with multiple errors
	req := RegisterRequest{
		Name:     "",        // Error: required
		Username: "a",       // Error: too short
		Email:    "invalid", // Error: invalid email
		Password: "short",   // Error: too short
	}

	errs := ValidateStructAll(&req)
	if len(errs) == 0 {
		t.Error("Expected validation errors, got none")
	}

	// Should have at least 3 errors (name, username, email or password)
	if len(errs) < 3 {
		t.Errorf("Expected at least 3 errors, got %d", len(errs))
	}
}

func TestValidateStructAll_NoErrors(t *testing.T) {
	req := RegisterRequest{
		Name:     "John Doe",
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "password123",
	}

	errs := ValidateStructAll(&req)
	if errs != nil {
		t.Errorf("Expected no errors, got %v", errs)
	}
}

func TestValidateLoginRequest(t *testing.T) {
	testCases := []struct {
		name      string
		req       LoginRequest
		expectErr bool
	}{
		{
			name: "Valid login",
			req: LoginRequest{
				Identifier: "johndoe",
				Password:   "password123",
			},
			expectErr: false,
		},
		{
			name: "Empty identifier",
			req: LoginRequest{
				Identifier: "",
				Password:   "password123",
			},
			expectErr: true,
		},
		{
			name: "Empty password",
			req: LoginRequest{
				Identifier: "johndoe",
				Password:   "",
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStruct(&tc.req)

			if tc.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestGetValidator_Singleton(t *testing.T) {
	v1 := GetValidator()
	v2 := GetValidator()

	if v1 != v2 {
		t.Error("GetValidator should return the same instance")
	}
}

func TestValidateHostname_LongLabel(t *testing.T) {
	// Label longer than 63 characters
	longLabel := "a"
	for i := 0; i < 64; i++ {
		longLabel += "a"
	}
	hostname := longLabel + ".example.com"

	req := ProxyRequest{
		Name:     "Test",
		Hostname: hostname,
		Type:     "reverse_proxy",
	}
	err := ValidateStruct(&req)

	if err == nil {
		t.Error("Expected error for long label, got nil")
	}
}

func TestValidateHostname_MaxLength(t *testing.T) {
	// Hostname longer than 253 characters
	hostname := ""
	for i := 0; i < 260; i++ {
		hostname += "a"
	}

	req := ProxyRequest{
		Name:     "Test",
		Hostname: hostname,
		Type:     "reverse_proxy",
	}
	err := ValidateStruct(&req)

	if err == nil {
		t.Error("Expected error for long hostname, got nil")
	}
}

func TestValidateProxyRequest_NameMaxLength(t *testing.T) {
	// Name longer than 255 characters
	longName := ""
	for i := 0; i < 260; i++ {
		longName += "a"
	}

	req := ProxyRequest{
		Name:     longName,
		Hostname: "example.com",
		Type:     "reverse_proxy",
	}
	err := ValidateStruct(&req)

	if err == nil {
		t.Error("Expected error for long name, got nil")
	}

	var validErr *ValidationError
	if errors.As(err, &validErr) {
		if validErr.Field != "name" {
			t.Errorf("Expected error field 'name', got '%s'", validErr.Field)
		}
	}
}
