package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimit_ContentLengthExceeded(t *testing.T) {
	handler := BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.NewReader("this is a very long body that exceeds the limit")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(len("this is a very long body that exceeds the limit"))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Request body too large") {
		t.Errorf("Expected error message about body size, got %s", w.Body.String())
	}
}

func TestBodyLimit_ContentLengthWithinLimit(t *testing.T) {
	nextCalled := false
	handler := BodyLimit(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.NewReader("small body")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(len("small body"))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Expected next handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBodyLimit_NoBody(t *testing.T) {
	nextCalled := false
	handler := BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Expected next handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBodyLimit_ZeroContentLength(t *testing.T) {
	nextCalled := false
	handler := BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	req.ContentLength = 0

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Expected next handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBodyLimit_MaxBytesReader(t *testing.T) {
	// Test that the body is wrapped with MaxBytesReader
	handler := BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read more than the limit
		buf := make([]byte, 1024)
		_, err := r.Body.Read(buf)
		if err != nil && err != io.EOF {
			// MaxBytesReader returns an error when limit is exceeded
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with unknown content length but large body
	largeBody := bytes.Repeat([]byte("x"), 100)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(largeBody))
	req.ContentLength = -1 // Unknown content length

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// The MaxBytesReader should limit the read
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", w.Code)
	}
}

func TestConstants(t *testing.T) {
	if DefaultBodyLimit != 1*1024*1024 {
		t.Errorf("Expected DefaultBodyLimit to be 1MB, got %d", DefaultBodyLimit)
	}

	if LargeBodyLimit != 10*1024*1024 {
		t.Errorf("Expected LargeBodyLimit to be 10MB, got %d", LargeBodyLimit)
	}
}

func TestBodyLimit_ExactLimit(t *testing.T) {
	// Test body exactly at the limit
	nextCalled := false
	handler := BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	body := "1234567890" // Exactly 10 bytes
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.ContentLength = 10

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Expected next handler to be called when body is exactly at limit")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBodyLimit_OneOverLimit(t *testing.T) {
	handler := BodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := "12345678901" // 11 bytes, one over
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.ContentLength = 11

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
