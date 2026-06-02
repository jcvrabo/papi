package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rabobank/papi/internal/api/v1/handlers"
)

func TestListNamespaces_Unauthorized(t *testing.T) {
	handler := handlers.NewNamespaceHandler(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListNamespaces_Success(t *testing.T) {
	mockService := &mockNamespaceService{
		namespaces: []handlers.NamespaceSummary{
			{ID: "test-id-1", CompositeName: "team1-proj1-dev", Status: "active"},
			{ID: "test-id-2", CompositeName: "team1-proj2-prod", Status: "provisioning"},
		},
		total: 2,
	}
	handler := handlers.NewNamespaceHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces?page=1&page_size=50", nil)
	req = req.WithContext(setUserContext(req.Context(), "testuser"))
	w := httptest.NewRecorder()

	handler.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp handlers.NamespaceListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, 1, resp.Pagination.Page)
	assert.Equal(t, 50, resp.Pagination.PageSize)
	assert.Equal(t, 2, resp.Pagination.Total)
}

func TestListNamespaces_EmptyList(t *testing.T) {
	mockService := &mockNamespaceService{namespaces: []handlers.NamespaceSummary{}, total: 0}
	handler := handlers.NewNamespaceHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces", nil)
	req = req.WithContext(setUserContext(req.Context(), "emptyuser"))
	w := httptest.NewRecorder()

	handler.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp handlers.NamespaceListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Pagination.Total)
}
