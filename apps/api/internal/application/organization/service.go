package organization

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/organization"
	"github.com/novudesk/novudesk/internal/domain/role"
	"github.com/novudesk/novudesk/internal/domain/user"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type Service struct {
	orgs  organization.Repository
	users user.Repository
	roles role.Repository
}

func NewService(orgs organization.Repository, users user.Repository, roles role.Repository) *Service {
	return &Service{orgs: orgs, users: users, roles: roles}
}

// Create registers a new organization and seeds the owner member.
func (s *Service) Create(ctx context.Context, input organization.CreateInput) (*organization.Organization, error) {
	slug := sanitizeSlug(input.Slug)

	exists, err := s.orgs.SlugExists(ctx, slug)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if exists {
		return nil, apperrors.Conflict(apperrors.CodeSlugTaken, "organization slug is already taken")
	}

	org := &organization.Organization{
		ID:       uuid.NewString(),
		Name:     input.Name,
		Slug:     slug,
		PlanTier: "free",
	}

	if err := s.orgs.Create(ctx, org); err != nil {
		return nil, apperrors.Internal(err)
	}

	// Find system owner role and assign to the creator.
	ownerRole, err := s.roles.FindSystemRole(ctx, role.RoleOwner)
	if err != nil || ownerRole == nil {
		return nil, apperrors.Internal(err)
	}

	if err := s.users.AddToOrg(ctx, input.OwnerID, org.ID, ownerRole.ID); err != nil {
		return nil, apperrors.Internal(err)
	}

	return org, nil
}

func (s *Service) Get(ctx context.Context, id string) (*organization.Organization, error) {
	org, err := s.orgs.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if org == nil {
		return nil, apperrors.NotFound(apperrors.CodeOrgNotFound, "organization not found")
	}
	return org, nil
}

func (s *Service) Update(ctx context.Context, id string, input organization.UpdateInput) (*organization.Organization, error) {
	org, err := s.orgs.Update(ctx, id, input)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if org == nil {
		return nil, apperrors.NotFound(apperrors.CodeOrgNotFound, "organization not found")
	}
	return org, nil
}

func sanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
