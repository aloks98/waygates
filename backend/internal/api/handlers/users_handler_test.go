package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewUsersHandler Tests
// =============================================================================

func TestNewUsersHandler(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{}
	h := NewUsersHandler(mockSvc, nil)
	require.NotNil(t, h)
	assert.Equal(t, mockSvc, h.svc)
}

// =============================================================================
// List Tests
// =============================================================================

func TestUsersHandler_List_Success(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		ListFunc: func(_ context.Context) ([]service.UserWithRole, error) {
			return []service.UserWithRole{
				{User: models.User{ID: 1, Name: "Alice", Username: "alice", Email: "alice@example.com"}, Role: "admin"},
				{User: models.User{ID: 2, Name: "Bob", Username: "bob", Email: "bob@example.com"}, Role: "operator"},
			}, nil
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool                   `json:"success"`
		Data    []service.UserWithRole `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Len(t, response.Data, 2)
}

func TestUsersHandler_List_Empty(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		ListFunc: func(_ context.Context) ([]service.UserWithRole, error) {
			return []service.UserWithRole{}, nil
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

// =============================================================================
// Get Tests
// =============================================================================

func TestUsersHandler_Get_Success(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		GetFunc: func(_ context.Context, id int) (*service.UserWithRole, error) {
			if id == 1 {
				return &service.UserWithRole{
					User: models.User{ID: 1, Name: "Alice", Username: "alice", Email: "alice@example.com"},
					Role: "admin",
				}, nil
			}
			return nil, service.ErrUserNotFound
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Get("/api/users/{id}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool                 `json:"success"`
		Data    service.UserWithRole `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, "alice", response.Data.Username)
}

func TestUsersHandler_Get_NotFound(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		GetFunc: func(_ context.Context, _ int) (*service.UserWithRole, error) {
			return nil, service.ErrUserNotFound
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Get("/api/users/{id}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/users/999", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUsersHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Get("/api/users/{id}", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/users/abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Create Tests
// =============================================================================

func TestUsersHandler_Create_Success(t *testing.T) {
	t.Parallel()
	var capturedInput service.CreateUserInput
	mockSvc := &mocks.MockUserService{
		CreateFunc: func(_ context.Context, in service.CreateUserInput, _ int, _, _ string) (*service.UserWithRole, error) {
			capturedInput = in
			return &service.UserWithRole{
				User: models.User{ID: 3, Name: in.Name, Username: in.Username, Email: in.Email},
				Role: in.Role,
			}, nil
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Charlie",
		"username": "charlie",
		"email":    "charlie@example.com",
		"role":     "operator",
		"password": "securepassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var response struct {
		Success bool                 `json:"success"`
		Data    service.UserWithRole `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, "Charlie", capturedInput.Name)
	assert.Equal(t, "operator", capturedInput.Role)
}

func TestUsersHandler_Create_InvalidRole(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{}
	h := NewUsersHandler(mockSvc, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Dave",
		"username": "dave",
		"email":    "dave@example.com",
		"role":     "superadmin",
		"password": "securepassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsersHandler_Create_MissingName(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{}
	h := NewUsersHandler(mockSvc, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"username": "dave",
		"email":    "dave@example.com",
		"role":     "operator",
		"password": "securepassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsersHandler_Create_ShortPassword(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{}
	h := NewUsersHandler(mockSvc, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Dave",
		"username": "dave",
		"email":    "dave@example.com",
		"role":     "viewer",
		"password": "short",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsersHandler_Create_UsernameTaken(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		CreateFunc: func(_ context.Context, _ service.CreateUserInput, _ int, _, _ string) (*service.UserWithRole, error) {
			return nil, service.ErrUsernameTaken
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Alice2",
		"username": "alice",
		"email":    "alice2@example.com",
		"role":     "viewer",
		"password": "securepassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp struct {
		Error struct{ Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp.Error.Message, service.ErrUsernameTaken.Error())
}

// =============================================================================
// Update Tests
// =============================================================================

func TestUsersHandler_Update_Success(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		UpdateFunc: func(_ context.Context, id int, in service.UpdateUserInput, _ int, _, _ string) (*service.UserWithRole, error) {
			return &service.UserWithRole{
				User: models.User{ID: id, Name: in.Name, Email: in.Email},
				Role: in.Role,
			}, nil
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Put("/api/users/{id}", h.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"name":   "Alice Updated",
		"email":  "alice@example.com",
		"role":   "admin",
		"active": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/users/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUsersHandler_Update_CannotModifySelf(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		UpdateFunc: func(_ context.Context, _ int, _ service.UpdateUserInput, _ int, _, _ string) (*service.UserWithRole, error) {
			return nil, service.ErrCannotModifySelf
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Put("/api/users/{id}", h.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"name":   "Self",
		"email":  "self@example.com",
		"role":   "viewer",
		"active": false,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/users/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// ResetPassword Tests
// =============================================================================

func TestUsersHandler_ResetPassword_Success(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		ResetPasswordFunc: func(_ context.Context, _ int, _ string, _ bool, _ int, _, _ string) error {
			return nil
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Post("/api/users/{id}/password", h.ResetPassword)

	body, _ := json.Marshal(map[string]interface{}{
		"password":             "newpassword123",
		"must_change_password": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users/1/password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUsersHandler_ResetPassword_UserNotFound(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		ResetPasswordFunc: func(_ context.Context, _ int, _ string, _ bool, _ int, _, _ string) error {
			return service.ErrUserNotFound
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Post("/api/users/{id}/password", h.ResetPassword)

	body, _ := json.Marshal(map[string]interface{}{
		"password": "newpassword123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users/999/password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestUsersHandler_Delete_Success(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		DeleteFunc: func(_ context.Context, _ int, _ int, _, _ string) error {
			return nil
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Delete("/api/users/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/users/2", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUsersHandler_Delete_LastAdmin(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		DeleteFunc: func(_ context.Context, _ int, _ int, _, _ string) error {
			return service.ErrLastAdmin
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Delete("/api/users/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/users/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp struct {
		Error struct{ Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp.Error.Message, service.ErrLastAdmin.Error())
}

func TestUsersHandler_Delete_CannotModifySelf(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		DeleteFunc: func(_ context.Context, _ int, _ int, _, _ string) error {
			return service.ErrCannotModifySelf
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Delete("/api/users/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/users/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsersHandler_Delete_InvalidID(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{}
	h := NewUsersHandler(mockSvc, nil)

	r := chi.NewRouter()
	r.Delete("/api/users/{id}", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/users/notanumber", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Response format test
// =============================================================================

func TestUsersHandler_List_ResponseFormat(t *testing.T) {
	t.Parallel()
	mockSvc := &mocks.MockUserService{
		ListFunc: func(_ context.Context) ([]service.UserWithRole, error) {
			return []service.UserWithRole{
				{User: models.User{ID: 1, Name: "Alice", Username: "alice"}, Role: "admin"},
			}, nil
		},
	}
	h := NewUsersHandler(mockSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	var response struct {
		Success bool                     `json:"success"`
		Message string                   `json:"message"`
		Data    []map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.Message)
	assert.NotNil(t, response.Data)
}
