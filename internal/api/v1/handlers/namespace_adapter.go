package handlers

import (
	"context"

	"github.com/rabobank/papi/internal/domain/namespace"
)

type NamespaceServiceAdapter struct {
	service *namespace.Service
}

func NewNamespaceServiceAdapter(service *namespace.Service) *NamespaceServiceAdapter {
	return &NamespaceServiceAdapter{service: service}
}

func (a *NamespaceServiceAdapter) ListForUser(ctx context.Context, user string, page, pageSize int) ([]NamespaceSummary, int, error) {
	namespaces, total, err := a.service.ListForUser(ctx, user, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	items := make([]NamespaceSummary, len(namespaces))
	for i, ns := range namespaces {
		items[i] = NamespaceSummary{
			ID:             ns.ID,
			CompositeName:  ns.CompositeName,
			NameComponents: ns.NameComponents,
			Metadata:       ns.Metadata,
			Status:         string(ns.Status),
			GroupID:        ns.RuntimeEnvironmentGroupID,
			CreatedAt:      ns.CreatedAt,
		}
	}
	return items, total, nil
}
