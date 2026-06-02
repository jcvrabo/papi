package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rabobank/papi/internal/api/v1/middleware"
	"github.com/rabobank/papi/internal/domain/group"
	"github.com/rabobank/papi/internal/platform"
)

type GroupHandler struct {
	service   *group.Service
	roleStore platform.RoleStore
}

func NewGroupHandler(service *group.Service, roleStore platform.RoleStore) *GroupHandler {
	return &GroupHandler{service: service, roleStore: roleStore}
}

type GroupResponse struct {
	ID                    string               `json:"id"`
	Name                  string               `json:"name"`
	IsDefault             bool                 `json:"is_default"`
	OrchestrationStrategy string               `json:"orchestration_strategy"`
	CanaryEnvironmentID   string               `json:"canary_environment_id,omitempty"`
	Environments          []EnvironmentSummary `json:"environments"`
	CreatedAt             time.Time            `json:"created_at"`
}

type EnvironmentSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	HealthStatus string `json:"health_status"`
}

type GroupListResponse struct {
	Items []GroupResponse `json:"items"`
}

type CreateGroupRequest struct {
	Name                  string                   `json:"name"`
	IsDefault             bool                     `json:"is_default"`
	OrchestrationStrategy string                   `json:"orchestration_strategy"`
	CanaryEnvironmentID   string                   `json:"canary_environment_id,omitempty"`
	Environments          []CreateEnvironmentInput `json:"environments"`
}

type CreateEnvironmentInput struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	ConnectionConfig map[string]interface{} `json:"connection_config"`
}

type UpdateGroupRequest struct {
	Name                  string `json:"name,omitempty"`
	IsDefault             *bool  `json:"is_default,omitempty"`
	OrchestrationStrategy string `json:"orchestration_strategy,omitempty"`
	CanaryEnvironmentID   string `json:"canary_environment_id,omitempty"`
}

func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == "" {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	groups, err := h.service.List(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list groups")
		return
	}

	resp := GroupListResponse{Items: make([]GroupResponse, len(groups))}
	for i, g := range groups {
		resp.Items[i] = toGroupResponse(g)
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == "" {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}
	hasRole, err := h.roleStore.HasRole(r.Context(), user, platform.RoleAdmin)
	if err != nil || !hasRole {
		WriteError(w, http.StatusForbidden, "forbidden", "Admin role required")
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	var details []ValidationDetail
	if req.Name == "" {
		details = append(details, ValidationDetail{Field: "name", Message: "name is required"})
	}
	if len(req.Environments) == 0 {
		details = append(details, ValidationDetail{Field: "environments", Message: "at least one environment required"})
	}
	if len(details) > 0 {
		WriteValidationError(w, "Validation failed", details)
		return
	}

	g := &group.Group{
		Name:                  req.Name,
		IsDefault:             req.IsDefault,
		OrchestrationStrategy: group.OrchestrationStrategy(req.OrchestrationStrategy),
		CanaryEnvironmentID:   req.CanaryEnvironmentID,
	}
	if g.OrchestrationStrategy == "" {
		g.OrchestrationStrategy = group.StrategyReplicate
	}

	if err := h.service.Create(r.Context(), g); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create group")
		return
	}

	WriteJSON(w, http.StatusCreated, toGroupResponse(*g))
}

func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == "" {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}
	hasRole, _ := h.roleStore.HasRole(r.Context(), user, platform.RoleAdmin)
	if !hasRole {
		WriteError(w, http.StatusForbidden, "forbidden", "Admin role required")
		return
	}

	groupID := chi.URLParam(r, "groupId")
	existing, err := h.service.GetByID(r.Context(), groupID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "Group not found")
		return
	}

	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.IsDefault != nil {
		existing.IsDefault = *req.IsDefault
	}
	if req.OrchestrationStrategy != "" {
		existing.OrchestrationStrategy = group.OrchestrationStrategy(req.OrchestrationStrategy)
	}

	if err := h.service.Update(r.Context(), existing); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update group")
		return
	}

	WriteJSON(w, http.StatusOK, toGroupResponse(*existing))
}

func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == "" {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}
	hasRole, _ := h.roleStore.HasRole(r.Context(), user, platform.RoleAdmin)
	if !hasRole {
		WriteError(w, http.StatusForbidden, "forbidden", "Admin role required")
		return
	}

	groupID := chi.URLParam(r, "groupId")
	if err := h.service.Delete(r.Context(), groupID); err != nil {
		if err == group.ErrHasNamespaces {
			WriteError(w, http.StatusConflict, "conflict", "Group has active namespace assignments")
			return
		}
		WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toGroupResponse(g group.Group) GroupResponse {
	envs := make([]EnvironmentSummary, len(g.Environments))
	for i, e := range g.Environments {
		envs[i] = EnvironmentSummary{
			ID:           e.ID,
			Name:         e.Name,
			Type:         e.Type,
			HealthStatus: e.HealthStatus,
		}
	}
	return GroupResponse{
		ID:                    g.ID,
		Name:                  g.Name,
		IsDefault:             g.IsDefault,
		OrchestrationStrategy: string(g.OrchestrationStrategy),
		CanaryEnvironmentID:   g.CanaryEnvironmentID,
		Environments:          envs,
		CreatedAt:             g.CreatedAt,
	}
}
