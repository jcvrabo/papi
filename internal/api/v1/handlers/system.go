package handlers

import (
	"context"
	"net/http"
)

type HealthStatus struct {
	Status       string              `json:"status"`
	Environments []EnvironmentHealth `json:"environments,omitempty"`
}

type EnvironmentHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type InfoResponse struct {
	Version    string   `json:"version"`
	APIVersion string   `json:"api_version"`
	Auth       AuthInfo `json:"auth"`
}

type AuthInfo struct {
	OIDCIssuer    string `json:"oidc_issuer"`
	TokenEndpoint string `json:"token_endpoint"`
}

type HealthService interface {
	GetHealth(ctx context.Context) (*HealthStatus, error)
}

type SystemHandler struct {
	healthService HealthService
	version       string
	oidcIssuer    string
}

func NewSystemHandler(healthService HealthService, version, oidcIssuer string) *SystemHandler {
	return &SystemHandler{healthService: healthService, version: version, oidcIssuer: oidcIssuer}
}

func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	health, err := h.healthService.GetHealth(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusServiceUnavailable, HealthStatus{Status: "unhealthy"})
		return
	}
	status := http.StatusOK
	if health.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}
	WriteJSON(w, status, health)
}

func (h *SystemHandler) Info(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, InfoResponse{
		Version:    h.version,
		APIVersion: "v1",
		Auth: AuthInfo{
			OIDCIssuer:    h.oidcIssuer,
			TokenEndpoint: h.oidcIssuer + "/oauth/token",
		},
	})
}
