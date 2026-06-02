package platform

import "context"

type Role string

const (
	RoleAdmin            Role = "admin"
	RoleNamespaceCreator Role = "namespace_creator"
)

type RoleStore interface {
	HasRole(ctx context.Context, userIdentifier string, role Role) (bool, error)
	GetRoles(ctx context.Context, userIdentifier string) ([]Role, error)
}

func RequireRole(store RoleStore, role Role, userIdentifier string, ctx context.Context) (bool, error) {
	return store.HasRole(ctx, userIdentifier, role)
}
