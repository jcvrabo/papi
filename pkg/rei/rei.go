package rei

import "context"

type RuntimeEnvironment interface {
	CreateNamespace(ctx context.Context, req CreateNamespaceRequest) (CreateNamespaceResult, error)
	DeleteNamespace(ctx context.Context, req DeleteNamespaceRequest) error
	HealthCheck(ctx context.Context) (HealthStatus, error)
}

type CreateNamespaceRequest struct {
	NamespaceID    string
	CompositeName  string
	NameComponents map[string]string
	Metadata       map[string]string
}

type CreateNamespaceResult struct {
	ExtensionData map[string]string
}

type DeleteNamespaceRequest struct {
	NamespaceID   string
	CompositeName string
	ExtensionData map[string]string
}

type HealthStatus struct {
	Status  Status
	Message string
}

type Status string

const (
	StatusHealthy     Status = "healthy"
	StatusDegraded    Status = "degraded"
	StatusUnavailable Status = "unavailable"
)

type Factory func(connectionConfig map[string]interface{}) (RuntimeEnvironment, error)

var Registry = map[string]Factory{}
