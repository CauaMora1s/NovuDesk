package category

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/category"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type Service struct {
	categories category.Repository
}

func NewService(categories category.Repository) *Service {
	return &Service{categories: categories}
}

func (s *Service) Create(ctx context.Context, orgID, name string, description *string) (*category.Category, error) {
	c := &category.Category{
		ID:          uuid.NewString(),
		OrgID:       orgID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.categories.Create(ctx, c); err != nil {
		return nil, apperrors.Internal(err)
	}
	return c, nil
}

func (s *Service) List(ctx context.Context, orgID string) ([]*category.Category, error) {
	cats, err := s.categories.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return cats, nil
}

func (s *Service) Get(ctx context.Context, id, orgID string) (*category.Category, error) {
	c, err := s.categories.FindByID(ctx, id, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if c == nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "category not found")
	}
	return c, nil
}

func (s *Service) Update(ctx context.Context, id, orgID string, input category.UpdateInput) (*category.Category, error) {
	c, err := s.categories.Update(ctx, id, orgID, input)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if c == nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "category not found")
	}
	return c, nil
}

func (s *Service) Delete(ctx context.Context, id, orgID string) error {
	if err := s.categories.Delete(ctx, id, orgID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) ListByTeam(ctx context.Context, teamID string) ([]*category.Category, error) {
	cats, err := s.categories.ListByTeam(ctx, teamID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return cats, nil
}

func (s *Service) AddToTeam(ctx context.Context, teamID, categoryID string) error {
	if err := s.categories.AddToTeam(ctx, teamID, categoryID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) RemoveFromTeam(ctx context.Context, teamID, categoryID string) error {
	if err := s.categories.RemoveFromTeam(ctx, teamID, categoryID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}
