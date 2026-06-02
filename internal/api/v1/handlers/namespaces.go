package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rabobank/papi/internal/api/v1/middleware"
	"github.com/rabobank/papi/internal/domain/namespace"
	"github.com/rabobank/papi/internal/platform"
)

type NamespaceSummary struct {
	ID             string            `json:"id"`
	CompositeName  string            `json:"composite_name"`
	NameComponents map[string]string `json:"name_components,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Status         string            `json:"status"`
	GroupID        string            `json:"group_id"`
	CreatedAt      time.Time         `json:"created_at"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type NamespaceListResponse struct {
	Items      []NamespaceSummary `json:"items"`
	Pagination Pagination         `json:"pagination"`
}

type CreateNamespaceRequest struct {
	NameComponents map[string]string `json:"name_components"`
	Members        []MemberInput     `json:"members"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	GroupID        string            `json:"group_id,omitempty"`
}

type MemberInput struct {
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	Role       string `json:"role"`
}

type MemberOutput struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	Role       string `json:"role"`
}

type NamespaceResponse struct {
	ID             string            `json:"id"`
	CompositeName  string            `json:"composite_name"`
	NameComponents map[string]string `json:"name_components"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Status         string            `json:"status"`
	Members        []MemberOutput    `json:"members"`
	GroupID        string            `json:"group_id"`
	CreatedAt      time.Time         `json:"created_at"`
}

type NamespaceListService interface {
	ListForUser(ctx context.Context, user string, page, pageSize int) ([]NamespaceSummary, int, error)
}

type NamespaceCreateService interface {
	CreateNamespace(ctx context.Context, input namespace.CreateNamespaceInput) (*namespace.CreateNamespaceOutput, []namespace.ValidationError, error)
}

type NamespaceHandler struct {
	listService   NamespaceListService
	createService NamespaceCreateService
	roleStore     platform.RoleStore
}

func NewNamespaceHandler(listService NamespaceListService, createService NamespaceCreateService, roleStore platform.RoleStore) *NamespaceHandler {
	return &NamespaceHandler{listService: listService, createService: createService, roleStore: roleStore}
}

func (h *NamespaceHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == "" {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "User context not found")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	items, total, err := h.listService.ListForUser(r.Context(), user, page, pageSize)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list namespaces")
		return
	}

	if items == nil {
		items = []NamespaceSummary{}
	}

	WriteJSON(w, http.StatusOK, NamespaceListResponse{
		Items: items,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

func (h *NamespaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == "" {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "User context not found")
		return
	}

	// Check namespace-creator role
	hasRole, err := h.roleStore.HasRole(r.Context(), user, platform.RoleNamespaceCreator)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to check permissions")
		return
	}
	if !hasRole {
		// Also check admin
		hasRole, err = h.roleStore.HasRole(r.Context(), user, platform.RoleAdmin)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to check permissions")
			return
		}
		if !hasRole {
			WriteError(w, http.StatusForbidden, "forbidden", "Namespace creator role required")
			return
		}
	}

	var req CreateNamespaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// Validate required fields
	var details []ValidationDetail
	if len(req.NameComponents) == 0 {
		details = append(details, ValidationDetail{Field: "name_components", Message: "must not be empty"})
	}
	if len(req.Members) == 0 {
		details = append(details, ValidationDetail{Field: "members", Message: "at least one member is required"})
	}
	if len(details) > 0 {
		WriteValidationError(w, "Validation failed", details)
		return
	}

	// Convert to domain input
	var members []namespace.MemberInput
	for _, m := range req.Members {
		members = append(members, namespace.MemberInput{
			Identifier: m.Identifier,
			Type:       namespace.MemberType(m.Type),
			Role:       namespace.MemberRole(m.Role),
		})
	}

	input := namespace.CreateNamespaceInput{
		NameComponents: req.NameComponents,
		Members:        members,
		Metadata:       req.Metadata,
		GroupID:        req.GroupID,
	}

	output, validationErrors, err := h.createService.CreateNamespace(r.Context(), input)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create namespace")
		return
	}
	if len(validationErrors) > 0 {
		var details []ValidationDetail
		for _, ve := range validationErrors {
			details = append(details, ValidationDetail{Field: ve.Field, Message: ve.Message})
		}
		WriteValidationError(w, "Validation failed", details)
		return
	}

	var memberOutputs []MemberOutput
	for _, m := range output.Members {
		memberOutputs = append(memberOutputs, MemberOutput{
			ID:         m.ID,
			Identifier: m.MemberIdentifier,
			Type:       string(m.MemberType),
			Role:       string(m.Role),
		})
	}

	resp := NamespaceResponse{
		ID:             output.Namespace.ID,
		CompositeName:  output.Namespace.CompositeName,
		NameComponents: output.Namespace.NameComponents,
		Metadata:       output.Namespace.Metadata,
		Status:         string(output.Namespace.Status),
		Members:        memberOutputs,
		GroupID:        output.Namespace.RuntimeEnvironmentGroupID,
		CreatedAt:      output.Namespace.CreatedAt,
	}

	WriteJSON(w, http.StatusAccepted, resp)
}

func (h *NamespaceHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == "" {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "User context not found")
		return
	}

	// TODO: implement namespace status lookup
	WriteError(w, http.StatusNotImplemented, "not_implemented", "Not yet implemented")
}
