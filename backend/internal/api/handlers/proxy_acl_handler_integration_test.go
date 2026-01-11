package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
)

// =============================================================================
// Test Setup
// =============================================================================

// setupProxyACLTestRouter creates a router with proxy ACL handler for testing
func setupProxyACLTestRouter(mockService *MockACLService) *chi.Mux {
	handler := NewProxyACLHandler(mockService, nil, nil, nil)
	r := chi.NewRouter()

	r.Get("/api/proxies/{id}/acl", handler.GetProxyACL)
	r.Post("/api/proxies/{id}/acl", handler.AssignACLToProxy)
	r.Put("/api/proxies/{id}/acl/{assignmentId}", handler.UpdateProxyACLAssignment)
	r.Delete("/api/proxies/{id}/acl/{groupId}", handler.RemoveACLFromProxy)

	return r
}

// =============================================================================
// GetProxyACL Tests
// =============================================================================

func TestProxyACLHandler_GetProxyACL_Success(t *testing.T) {
	mockService := &MockACLService{
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{
					ID:          1,
					ProxyID:     proxyID,
					ACLGroupID:  1,
					PathPattern: "/*",
					Priority:    0,
					Enabled:     true,
					ACLGroup: &models.ACLGroup{
						ID:              1,
						Name:            "Default ACL",
						CombinationMode: models.ACLCombinationModeAny,
					},
				},
				{
					ID:          2,
					ProxyID:     proxyID,
					ACLGroupID:  2,
					PathPattern: "/api/*",
					Priority:    10,
					Enabled:     true,
					ACLGroup: &models.ACLGroup{
						ID:              2,
						Name:            "API ACL",
						CombinationMode: models.ACLCombinationModeAll,
					},
				},
			}, nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1/acl", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])

	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestProxyACLHandler_GetProxyACL_NoAssignments(t *testing.T) {
	mockService := &MockACLService{
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1/acl", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])

	data := response["data"].([]interface{})
	assert.Empty(t, data)
}

func TestProxyACLHandler_GetProxyACL_ProxyNotFound(t *testing.T) {
	mockService := &MockACLService{
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return nil, service.ErrProxyNotFound
		},
	}

	r := setupProxyACLTestRouter(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/999/acl", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyACLHandler_GetProxyACL_InvalidProxyID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/invalid/acl", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// AssignACLToProxy Tests
// =============================================================================

func TestProxyACLHandler_AssignACLToProxy_Success(t *testing.T) {
	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			return nil
		},
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{
					ID:          1,
					ProxyID:     proxyID,
					ACLGroupID:  1,
					PathPattern: "/*",
					Priority:    0,
					Enabled:     true,
				},
			}, nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"acl_group_id": 1, "path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_DefaultPathPattern(t *testing.T) {
	var capturedPathPattern string
	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			capturedPathPattern = pathPattern
			return nil
		},
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	// No path_pattern specified
	body := `{"acl_group_id": 1, "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/*", capturedPathPattern)
}

func TestProxyACLHandler_AssignACLToProxy_DuplicatePath(t *testing.T) {
	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			return service.ErrProxyACLExists
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"acl_group_id": 1, "path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_ProxyNotFound(t *testing.T) {
	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			return service.ErrProxyNotFound
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"acl_group_id": 1, "path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/999/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_GroupNotFound(t *testing.T) {
	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			return service.ErrACLGroupNotFound
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"acl_group_id": 999, "path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_MissingACLGroupID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{"path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_InvalidACLGroupID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{"acl_group_id": 0, "path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_NegativeACLGroupID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{"acl_group_id": -1, "path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_InvalidPathPattern(t *testing.T) {
	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			return service.ErrInvalidPathPattern
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"acl_group_id": 1, "path_pattern": "invalid?pattern", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_InvalidJSON(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_AssignACLToProxy_InvalidProxyID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{"acl_group_id": 1, "path_pattern": "/*", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/invalid/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// UpdateProxyACLAssignment Tests
// =============================================================================

func TestProxyACLHandler_UpdateProxyACLAssignment_Success(t *testing.T) {
	mockService := &MockACLService{
		UpdateProxyAssignmentFunc: func(id int, pathPattern string, priority int, enabled bool) error {
			return nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"path_pattern": "/api/*", "priority": 10, "enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1/acl/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProxyACLHandler_UpdateProxyACLAssignment_NotFound(t *testing.T) {
	mockService := &MockACLService{
		UpdateProxyAssignmentFunc: func(id int, pathPattern string, priority int, enabled bool) error {
			return service.ErrProxyACLNotFound
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"path_pattern": "/api/*", "priority": 10, "enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1/acl/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyACLHandler_UpdateProxyACLAssignment_InvalidPathPattern(t *testing.T) {
	mockService := &MockACLService{
		UpdateProxyAssignmentFunc: func(id int, pathPattern string, priority int, enabled bool) error {
			return service.ErrInvalidPathPattern
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"path_pattern": "invalid?pattern", "priority": 10, "enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1/acl/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_UpdateProxyACLAssignment_DisableAssignment(t *testing.T) {
	var capturedEnabled bool
	mockService := &MockACLService{
		UpdateProxyAssignmentFunc: func(id int, pathPattern string, priority int, enabled bool) error {
			capturedEnabled = enabled
			return nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"path_pattern": "/*", "priority": 0, "enabled": false}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1/acl/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, capturedEnabled)
}

func TestProxyACLHandler_UpdateProxyACLAssignment_InvalidProxyID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{"path_pattern": "/*", "priority": 0, "enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/invalid/acl/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_UpdateProxyACLAssignment_InvalidAssignmentID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{"path_pattern": "/*", "priority": 0, "enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1/acl/invalid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_UpdateProxyACLAssignment_InvalidJSON(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1/acl/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// RemoveACLFromProxy Tests
// =============================================================================

func TestProxyACLHandler_RemoveACLFromProxy_Success(t *testing.T) {
	mockService := &MockACLService{
		RemoveFromProxyFunc: func(proxyID, groupID int) error {
			return nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1/acl/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProxyACLHandler_RemoveACLFromProxy_NotFound(t *testing.T) {
	mockService := &MockACLService{
		RemoveFromProxyFunc: func(proxyID, groupID int) error {
			return service.ErrProxyACLNotFound
		},
	}

	r := setupProxyACLTestRouter(mockService)
	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1/acl/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyACLHandler_RemoveACLFromProxy_InvalidProxyID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/invalid/acl/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyACLHandler_RemoveACLFromProxy_InvalidGroupID(t *testing.T) {
	r := setupProxyACLTestRouter(&MockACLService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1/acl/invalid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestProxyACLHandler_FullWorkflow(t *testing.T) {
	// Track state
	assignments := make(map[int]models.ProxyACLAssignment)
	nextID := 1

	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			assignments[nextID] = models.ProxyACLAssignment{
				ID:          nextID,
				ProxyID:     proxyID,
				ACLGroupID:  groupID,
				PathPattern: pathPattern,
				Priority:    priority,
				Enabled:     true,
			}
			nextID++
			return nil
		},
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			result := []models.ProxyACLAssignment{}
			for _, a := range assignments {
				if a.ProxyID == proxyID {
					result = append(result, a)
				}
			}
			return result, nil
		},
		UpdateProxyAssignmentFunc: func(id int, pathPattern string, priority int, enabled bool) error {
			if a, ok := assignments[id]; ok {
				a.PathPattern = pathPattern
				a.Priority = priority
				a.Enabled = enabled
				assignments[id] = a
				return nil
			}
			return service.ErrProxyACLNotFound
		},
		RemoveFromProxyFunc: func(proxyID, groupID int) error {
			for id, a := range assignments {
				if a.ProxyID == proxyID && a.ACLGroupID == groupID {
					delete(assignments, id)
					return nil
				}
			}
			return service.ErrProxyACLNotFound
		},
	}

	r := setupProxyACLTestRouter(mockService)

	// Step 1: Get ACL (empty)
	t.Run("Get empty ACL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies/1/acl", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data := response["data"].([]interface{})
		assert.Empty(t, data)
	})

	// Step 2: Assign ACL
	t.Run("Assign ACL", func(t *testing.T) {
		body := `{"acl_group_id": 1, "path_pattern": "/*", "priority": 0}`
		req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	// Step 3: Get ACL (with assignment)
	t.Run("Get ACL with assignment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies/1/acl", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data := response["data"].([]interface{})
		assert.Len(t, data, 1)
	})

	// Step 4: Update assignment
	t.Run("Update assignment", func(t *testing.T) {
		body := `{"path_pattern": "/api/*", "priority": 10, "enabled": true}`
		req := httptest.NewRequest(http.MethodPut, "/api/proxies/1/acl/1", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify update
		assert.Equal(t, "/api/*", assignments[1].PathPattern)
		assert.Equal(t, 10, assignments[1].Priority)
	})

	// Step 5: Remove ACL
	t.Run("Remove ACL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1/acl/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Step 6: Verify removal
	t.Run("Verify removal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies/1/acl", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data := response["data"].([]interface{})
		assert.Empty(t, data)
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestProxyACLHandler_MultipleAssignments(t *testing.T) {
	mockService := &MockACLService{
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Priority: 100, Enabled: true},
				{ID: 2, ProxyID: proxyID, ACLGroupID: 2, PathPattern: "/api/*", Priority: 50, Enabled: true},
				{ID: 3, ProxyID: proxyID, ACLGroupID: 3, PathPattern: "/admin/*", Priority: 10, Enabled: false},
			}, nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1/acl", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	assert.Len(t, data, 3)
}

func TestProxyACLHandler_LargeProxyID(t *testing.T) {
	mockService := &MockACLService{
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			assert.Equal(t, 999999999, proxyID)
			return []models.ProxyACLAssignment{}, nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/999999999/acl", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProxyACLHandler_SpecialCharactersInPathPattern(t *testing.T) {
	var capturedPattern string
	mockService := &MockACLService{
		AssignToProxyFunc: func(proxyID, groupID int, pathPattern string, priority int) error {
			capturedPattern = pathPattern
			return nil
		},
		GetProxyACLFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil
		},
	}

	r := setupProxyACLTestRouter(mockService)
	body := `{"acl_group_id": 1, "path_pattern": "/api/v1/users/*/profile", "priority": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/acl", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/api/v1/users/*/profile", capturedPattern)
}
