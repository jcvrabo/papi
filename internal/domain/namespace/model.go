package namespace

import "time"

type Status string

const (
	StatusActive       Status = "active"
	StatusProvisioning Status = "provisioning"
	StatusFailed       Status = "failed"
)

type MemberType string

const (
	MemberTypeUser          MemberType = "user"
	MemberTypeIdentityGroup MemberType = "identity_group"
)

type MemberRole string

const (
	RoleRead      MemberRole = "read"
	RoleWrite     MemberRole = "write"
	RoleReadWrite MemberRole = "read_write"
)

type Namespace struct {
	ID                        string
	CompositeName             string
	NameComponents            map[string]string
	Metadata                  map[string]string
	RuntimeEnvironmentGroupID string
	Status                    Status
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type Member struct {
	ID               string
	NamespaceID      string
	MemberType       MemberType
	MemberIdentifier string
	Role             MemberRole
	CreatedAt        time.Time
}
