package contract

import (
	"context"

	"github.com/rabobank/papi/internal/api/v1/handlers"
	"github.com/rabobank/papi/internal/api/v1/middleware"
)

type mockNamespaceService struct {
	namespaces []handlers.NamespaceSummary
	total      int
	err        error
}

func (m *mockNamespaceService) ListForUser(ctx context.Context, user string, page, pageSize int) ([]handlers.NamespaceSummary, int, error) {
	return m.namespaces, m.total, m.err
}

func setUserContext(ctx context.Context, user string) context.Context {
	return middleware.SetUserInContext(ctx, user)
}
