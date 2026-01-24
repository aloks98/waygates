package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewStatusHandler Tests
// =============================================================================

func TestNewStatusHandler(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{}
	mockUserRepo := &mocks.MockUserRepository{}

	handler := NewStatusHandler(mockReloader, mockUserRepo)

	require.NotNil(t, handler, "handler should be created")
	assert.Equal(t, mockReloader, handler.reloader, "reloader should be set")
	assert.Equal(t, mockUserRepo, handler.userRepo, "userRepo should be set")
}

// =============================================================================
// GetStatus Tests
// =============================================================================

func TestStatusHandler_GetStatus_AllHealthy(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 5, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			CaddyStatus       string `json:"caddy_status"`
			UserSetupComplete bool   `json:"user_setup_complete"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected success to be true")
	assert.Equal(t, "healthy", response.Data.CaddyStatus)
	assert.True(t, response.Data.UserSetupComplete, "expected user_setup_complete to be true")
}

func TestStatusHandler_GetStatus_CaddyUnhealthy(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return errors.New("connection refused")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			CaddyStatus string `json:"caddy_status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.Equal(t, "unhealthy", response.Data.CaddyStatus)
}

func TestStatusHandler_GetStatus_NoUsers(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 0, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			UserSetupComplete bool `json:"user_setup_complete"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.False(t, response.Data.UserSetupComplete, "expected user_setup_complete to be false")
}

func TestStatusHandler_GetStatus_UserCountError(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 0, errors.New("database error")
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestStatusHandler_GetStatus_BothUnhealthy(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return errors.New("caddy not running")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 0, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			CaddyStatus       string `json:"caddy_status"`
			UserSetupComplete bool   `json:"user_setup_complete"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.Equal(t, "unhealthy", response.Data.CaddyStatus)
	assert.False(t, response.Data.UserSetupComplete, "expected user_setup_complete to be false")
}

func TestStatusHandler_GetStatus_ResponseFormat(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	var response struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	// Check response structure
	assert.True(t, response.Success, "expected success field")
	assert.NotEmpty(t, response.Message, "expected message field")
	assert.NotNil(t, response.Data, "expected data field")

	// Check data structure
	_, hasCaddyStatus := response.Data["caddy_status"]
	assert.True(t, hasCaddyStatus, "expected 'caddy_status' in data")
	_, hasUserSetupComplete := response.Data["user_setup_complete"]
	assert.True(t, hasUserSetupComplete, "expected 'user_setup_complete' in data")
}

func TestStatusHandler_GetStatus_SuccessMessage(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	var response struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.Equal(t, "Status retrieved successfully", response.Message)
}

func TestStatusHandler_GetStatus_CaddyTimeout(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			// Simulate context being canceled/timeout
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return errors.New("timeout")
			}
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			CaddyStatus string `json:"caddy_status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	// Caddy should be marked unhealthy on timeout
	assert.Equal(t, "unhealthy", response.Data.CaddyStatus)
}

func TestStatusHandler_GetStatus_ManyUsers(t *testing.T) {
	t.Parallel()
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(_ context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1000, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			UserSetupComplete bool `json:"user_setup_complete"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Data.UserSetupComplete, "expected user_setup_complete to be true with 1000 users")
}
