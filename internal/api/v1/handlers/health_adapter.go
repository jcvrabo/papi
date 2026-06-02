package handlers

import (
	"context"

	"github.com/rabobank/papi/internal/health"
)

type HealthServiceAdapter struct {
	service *health.Service
}

func NewHealthServiceAdapter(service *health.Service) *HealthServiceAdapter {
	return &HealthServiceAdapter{service: service}
}

func (a *HealthServiceAdapter) GetHealth(ctx context.Context) (*HealthStatus, error) {
	status, err := a.service.GetHealth(ctx)
	if err != nil {
		return nil, err
	}
	envs := make([]EnvironmentHealth, len(status.Environments))
	for i, e := range status.Environments {
		envs[i] = EnvironmentHealth{Name: e.Name, Status: e.Status}
	}
	return &HealthStatus{Status: status.Status, Environments: envs}, nil
}
