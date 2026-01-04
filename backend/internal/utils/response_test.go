package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		data           interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Simple object",
			statusCode:     http.StatusOK,
			data:           map[string]string{"key": "value"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"key":"value"}`,
		},
		{
			name:           "Success response",
			statusCode:     http.StatusOK,
			data:           SuccessResponse{Success: true, Message: "OK"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"success":true,"message":"OK"}`,
		},
		{
			name:           "Error response",
			statusCode:     http.StatusBadRequest,
			data:           ErrorResponse{Success: false, Error: ErrorDetail{Code: "TEST", Message: "test error"}},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"success":false,"error":{"code":"TEST","message":"test error"}}`,
		},
		{
			name:           "Empty data",
			statusCode:     http.StatusNoContent,
			data:           nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSON(w, tt.statusCode, tt.data)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Error("Expected Content-Type application/json")
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestJSON_MarshalError(t *testing.T) {
	w := httptest.NewRecorder()

	// channels cannot be marshaled to JSON
	invalidData := make(chan int)
	JSON(w, http.StatusOK, invalidData)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	expectedBody := `{"success":false,"error":{"code":"INTERNAL_ERROR","message":"Failed to encode response"}}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]int{"count": 42}
	Success(w, data, "Data retrieved successfully")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Message != "Data retrieved successfully" {
		t.Errorf("Expected message 'Data retrieved successfully', got %s", response.Message)
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]int{"id": 1}
	Created(w, data, "Resource created")

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Message != "Resource created" {
		t.Errorf("Expected message 'Resource created', got %s", response.Message)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusTeapot, "TEAPOT", "I'm a teapot", map[string]string{"detail": "brew coffee"})

	if w.Code != http.StatusTeapot {
		t.Errorf("Expected status 418, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}
	if response.Error.Code != "TEAPOT" {
		t.Errorf("Expected code 'TEAPOT', got %s", response.Error.Code)
	}
	if response.Error.Message != "I'm a teapot" {
		t.Errorf("Expected message \"I'm a teapot\", got %s", response.Error.Message)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "Invalid input", []string{"field1 is required"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected code 'VALIDATION_ERROR', got %s", response.Error.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w, "Invalid credentials")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "UNAUTHORIZED" {
		t.Errorf("Expected code 'UNAUTHORIZED', got %s", response.Error.Code)
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "Access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "FORBIDDEN" {
		t.Errorf("Expected code 'FORBIDDEN', got %s", response.Error.Code)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "Resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "NOT_FOUND" {
		t.Errorf("Expected code 'NOT_FOUND', got %s", response.Error.Code)
	}
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	Conflict(w, "Resource already exists")

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "CONFLICT" {
		t.Errorf("Expected code 'CONFLICT', got %s", response.Error.Code)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w, "Something went wrong")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("Expected code 'INTERNAL_ERROR', got %s", response.Error.Code)
	}
}

func TestBadGateway(t *testing.T) {
	w := httptest.NewRecorder()
	BadGateway(w, "Upstream service unavailable")

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "EXTERNAL_SERVICE_ERROR" {
		t.Errorf("Expected code 'EXTERNAL_SERVICE_ERROR', got %s", response.Error.Code)
	}
}
