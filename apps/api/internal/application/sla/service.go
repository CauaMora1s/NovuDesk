package sla

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/sla"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type Service struct {
	repo sla.Repository
}

func NewService(repo sla.Repository) *Service {
	return &Service{repo: repo}
}

// ListWithCategoryStats returns all org categories with their SLA and avg resolution time.
func (s *Service) ListWithCategoryStats(ctx context.Context, orgID string) ([]*sla.CategorySLAStat, error) {
	stats, err := s.repo.ListWithCategoryStats(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return stats, nil
}

// UpsertForCategory creates or updates the SLA policy for a given category.
func (s *Service) UpsertForCategory(ctx context.Context, orgID, categoryID string, input sla.CreateInput) (*sla.Policy, error) {
	existing, err := s.repo.FindByCategoryID(ctx, categoryID, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	if existing != nil {
		input.CategoryID = &categoryID
		p, err := s.repo.Update(ctx, existing.ID, orgID, input)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		return p, nil
	}

	catID := categoryID
	p := &sla.Policy{
		ID:              uuid.NewString(),
		OrgID:           orgID,
		Name:            input.Name,
		CategoryID:      &catID,
		ResponseHours:   input.ResponseHours,
		ResolutionValue: input.ResolutionValue,
		ResolutionUnit:  input.ResolutionUnit,
		Conditions:      json.RawMessage("{}"),
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if p.Name == "" {
		p.Name = "SLA"
	}
	if p.ResolutionUnit == "" {
		p.ResolutionUnit = "hours"
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, apperrors.Internal(err)
	}
	return p, nil
}

// Delete removes a SLA policy.
func (s *Service) Delete(ctx context.Context, id, orgID string) error {
	existing, err := s.repo.FindByID(ctx, id, orgID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if existing == nil {
		return apperrors.NotFound(apperrors.CodeSLANotFound, "SLA policy not found")
	}
	if err := s.repo.Delete(ctx, id, orgID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}
