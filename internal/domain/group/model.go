package group

import "time"

type OrchestrationStrategy string

const (
	StrategyReplicate OrchestrationStrategy = "replicate"
	StrategyCanary    OrchestrationStrategy = "canary"
)

type Group struct {
	ID                    string
	Name                  string
	IsDefault             bool
	OrchestrationStrategy OrchestrationStrategy
	CanaryEnvironmentID   string
	Environments          []Environment
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Environment struct {
	ID               string
	GroupID          string
	Name             string
	Type             string
	ConnectionConfig map[string]interface{}
	HealthStatus     string
	CreatedAt        time.Time
}
