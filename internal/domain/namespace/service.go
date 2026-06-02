package namespace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rabobank/papi/internal/domain/operation"
)

type Store interface {
	ListByUser(ctx context.Context, userIdentifier string, page, pageSize int) ([]Namespace, int, error)
	NameExists(ctx context.Context, compositeName string) (bool, error)
	Create(ctx context.Context, ns *Namespace, members []Member) error
}

type NameTemplateStore interface {
	GetActiveTemplate(ctx context.Context) (*NamingConfig, error)
}

type MetadataRuleStore interface {
	GetMandatoryRules(ctx context.Context) ([]MetadataRule, error)
}

type OperationStore interface {
	Create(ctx context.Context, op *operation.Operation) error
}

type GroupStore interface {
	GetDefaultGroup(ctx context.Context) (string, error)
	GetEnvironmentsByGroup(ctx context.Context, groupID string) ([]string, error)
}

type CreateNamespaceInput struct {
	NameComponents map[string]string
	Members        []MemberInput
	Metadata       map[string]string
	GroupID        string
}

type MemberInput struct {
	Identifier string
	Type       MemberType
	Role       MemberRole
}

type CreateNamespaceOutput struct {
	Namespace Namespace
	Members   []Member
}

type Service struct {
	store             Store
	nameTemplateStore NameTemplateStore
	metadataStore     MetadataRuleStore
	operationStore    OperationStore
	groupStore        GroupStore
}

type ServiceConfig struct {
	Store             Store
	NameTemplateStore NameTemplateStore
	MetadataStore     MetadataRuleStore
	OperationStore    OperationStore
	GroupStore        GroupStore
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func NewServiceWithStores(cfg ServiceConfig) *Service {
	return &Service{
		store:             cfg.Store,
		nameTemplateStore: cfg.NameTemplateStore,
		metadataStore:     cfg.MetadataStore,
		operationStore:    cfg.OperationStore,
		groupStore:        cfg.GroupStore,
	}
}

func (s *Service) ListForUser(ctx context.Context, userIdentifier string, page, pageSize int) ([]Namespace, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	return s.store.ListByUser(ctx, userIdentifier, page, pageSize)
}

func (s *Service) CreateNamespace(ctx context.Context, input CreateNamespaceInput) (*CreateNamespaceOutput, []ValidationError, error) {
	// 1. Get naming template and validate
	tmpl, err := s.nameTemplateStore.GetActiveTemplate(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get naming template: %w", err)
	}

	compositeName, validationErrors := ValidateNameComponents(tmpl, input.NameComponents)
	if len(validationErrors) > 0 {
		return nil, validationErrors, nil
	}

	// 2. Check name uniqueness
	exists, err := s.store.NameExists(ctx, compositeName)
	if err != nil {
		return nil, nil, fmt.Errorf("check name uniqueness: %w", err)
	}
	if exists {
		return nil, []ValidationError{{Field: "name", Message: "namespace name already exists"}}, nil
	}

	// 3. Validate metadata
	rules, err := s.metadataStore.GetMandatoryRules(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get metadata rules: %w", err)
	}
	metaErrors := ValidateMetadata(rules, input.Metadata)
	if len(metaErrors) > 0 {
		return nil, metaErrors, nil
	}

	// 4. Resolve group
	groupID := input.GroupID
	if groupID == "" {
		groupID, err = s.groupStore.GetDefaultGroup(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("get default group: %w", err)
		}
	}

	// 5. Get environments
	envIDs, err := s.groupStore.GetEnvironmentsByGroup(ctx, groupID)
	if err != nil {
		return nil, nil, fmt.Errorf("get environments: %w", err)
	}

	// 6. Persist namespace + members
	nsID := uuid.New().String()
	ns := &Namespace{
		ID:                        nsID,
		CompositeName:             compositeName,
		NameComponents:            input.NameComponents,
		Metadata:                  input.Metadata,
		RuntimeEnvironmentGroupID: groupID,
		Status:                    StatusProvisioning,
	}

	var members []Member
	for _, m := range input.Members {
		members = append(members, Member{
			ID:               uuid.New().String(),
			NamespaceID:      nsID,
			MemberType:       m.Type,
			MemberIdentifier: m.Identifier,
			Role:             m.Role,
		})
	}

	if err := s.store.Create(ctx, ns, members); err != nil {
		return nil, nil, fmt.Errorf("persist namespace: %w", err)
	}

	// 7. Queue operations for each environment
	for _, envID := range envIDs {
		op := &operation.Operation{
			ID:                   uuid.New().String(),
			NamespaceID:          nsID,
			RuntimeEnvironmentID: envID,
			OperationType:        operation.TypeCreateNamespace,
			Status:               operation.StatusPending,
			IdempotencyKey:       fmt.Sprintf("create-%s-%s", nsID, envID),
			Payload: map[string]interface{}{
				"composite_name":  compositeName,
				"name_components": input.NameComponents,
				"metadata":        input.Metadata,
			},
		}
		if err := s.operationStore.Create(ctx, op); err != nil {
			return nil, nil, fmt.Errorf("queue operation: %w", err)
		}
	}

	return &CreateNamespaceOutput{Namespace: *ns, Members: members}, nil, nil
}
