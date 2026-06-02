package operation

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type Type string

const (
	TypeCreateNamespace Type = "create_namespace"
	TypeDeleteNamespace Type = "delete_namespace"
)

type Operation struct {
	ID                   string
	NamespaceID          string
	RuntimeEnvironmentID string
	OperationType        Type
	Status               Status
	IdempotencyKey       string
	Payload              map[string]interface{}
	Attempts             int
	LastAttemptedAt      *time.Time
	ErrorMessage         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
