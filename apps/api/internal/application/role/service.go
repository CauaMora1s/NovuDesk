package role

import (
	"context"

	"github.com/google/uuid"

	"github.com/novudesk/novudesk/internal/domain/role"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type Service struct {
	roles role.Repository
}

func NewService(roles role.Repository) *Service {
	return &Service{roles: roles}
}

func (s *Service) ListForOrg(ctx context.Context, orgID string) ([]*role.RoleWithPermissions, error) {
	roles, err := s.roles.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	result := make([]*role.RoleWithPermissions, 0, len(roles))
	for _, r := range roles {
		perms, err := s.roles.GetPermissions(ctx, r.ID)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		result = append(result, &role.RoleWithPermissions{Role: *r, Permissions: perms})
	}
	return result, nil
}

func (s *Service) ListAllPermissions(ctx context.Context) ([]*role.Permission, error) {
	perms, err := s.roles.ListAllPermissions(ctx)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return perms, nil
}

func (s *Service) GetWithPermissions(ctx context.Context, roleID, orgID string) (*role.RoleWithPermissions, error) {
	r, err := s.roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if r == nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "role not found")
	}
	// Org roles must belong to the org; system roles are accessible to all.
	if !r.IsSystemRole && (r.OrgID == nil || *r.OrgID != orgID) {
		return nil, apperrors.Forbidden("role does not belong to this organization")
	}

	perms, err := s.roles.GetPermissions(ctx, r.ID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return &role.RoleWithPermissions{Role: *r, Permissions: perms}, nil
}

func (s *Service) Create(ctx context.Context, orgID, name string, permKeys []string) (*role.RoleWithPermissions, error) {
	existing, err := s.roles.FindByName(ctx, orgID, name)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if existing != nil {
		return nil, apperrors.Conflict(apperrors.CodeConflict, "a role with this name already exists")
	}

	r := &role.Role{
		ID:           uuid.NewString(),
		OrgID:        &orgID,
		Name:         name,
		IsSystemRole: false,
	}
	if err := s.roles.Create(ctx, r); err != nil {
		return nil, apperrors.Internal(err)
	}

	if err := s.roles.SetPermissions(ctx, r.ID, permKeys); err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.GetWithPermissions(ctx, r.ID, orgID)
}

func (s *Service) Update(ctx context.Context, roleID, orgID, name string, permKeys []string) (*role.RoleWithPermissions, error) {
	r, err := s.roles.FindByID(ctx, roleID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if r == nil {
		return nil, apperrors.NotFound(apperrors.CodeNotFound, "role not found")
	}
	if r.IsSystemRole {
		return nil, apperrors.Forbidden("system roles cannot be modified")
	}
	if r.OrgID == nil || *r.OrgID != orgID {
		return nil, apperrors.Forbidden("role does not belong to this organization")
	}

	// Check name uniqueness when changing the name.
	if r.Name != name {
		dup, err := s.roles.FindByName(ctx, orgID, name)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		if dup != nil {
			return nil, apperrors.Conflict(apperrors.CodeConflict, "a role with this name already exists")
		}
	}

	r.Name = name
	if err := s.roles.Update(ctx, r); err != nil {
		return nil, apperrors.Internal(err)
	}

	if err := s.roles.SetPermissions(ctx, r.ID, permKeys); err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.GetWithPermissions(ctx, r.ID, orgID)
}

func (s *Service) Delete(ctx context.Context, roleID, orgID string) error {
	r, err := s.roles.FindByID(ctx, roleID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if r == nil {
		return apperrors.NotFound(apperrors.CodeNotFound, "role not found")
	}
	if r.IsSystemRole {
		return apperrors.Forbidden("system roles cannot be deleted")
	}
	if r.OrgID == nil || *r.OrgID != orgID {
		return apperrors.Forbidden("role does not belong to this organization")
	}

	if err := s.roles.Delete(ctx, roleID, orgID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}
