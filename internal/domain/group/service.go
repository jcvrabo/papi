package group

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("group not found")
	ErrHasNamespaces = errors.New("group has active namespace assignments")
)

type Store interface {
	List(ctx context.Context) ([]Group, error)
	Create(ctx context.Context, g *Group) error
	Update(ctx context.Context, g *Group) error
	Delete(ctx context.Context, id string) error
	HasNamespaces(ctx context.Context, groupID string) (bool, error)
	GetByID(ctx context.Context, id string) (*Group, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context) ([]Group, error) {
	return s.store.List(ctx)
}

func (s *Service) Create(ctx context.Context, g *Group) error {
	if g.OrchestrationStrategy == "" {
		g.OrchestrationStrategy = StrategyReplicate
	}
	return s.store.Create(ctx, g)
}

func (s *Service) Update(ctx context.Context, g *Group) error {
	return s.store.Update(ctx, g)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	has, err := s.store.HasNamespaces(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return ErrHasNamespaces
	}
	return s.store.Delete(ctx, id)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Group, error) {
	return s.store.GetByID(ctx, id)
}
