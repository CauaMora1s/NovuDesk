package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	authsvc "github.com/novudesk/novudesk/internal/application/auth"
	"github.com/novudesk/novudesk/internal/domain/role"
	"github.com/novudesk/novudesk/internal/domain/user"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

type RegisterInput struct {
	Email    string
	Password string
	FullName string
	Locale   string
}

type Service struct {
	users user.Repository
	roles role.Repository
}

func NewService(users user.Repository, roles role.Repository) *Service {
	return &Service{users: users, roles: roles}
}

// Register creates a new user account (no org assigned yet).
func (s *Service) Register(ctx context.Context, input RegisterInput) (*user.User, error) {
	exists, err := s.users.EmailExists(ctx, input.Email)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if exists {
		return nil, apperrors.Conflict(apperrors.CodeEmailTaken, "email address is already registered")
	}

	hash, err := authsvc.HashPassword(input.Password)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	locale := input.Locale
	if locale == "" {
		locale = "pt"
	}

	u := &user.User{
		ID:           uuid.NewString(),
		Email:        input.Email,
		PasswordHash: &hash,
		FullName:     input.FullName,
		Locale:       locale,
		IsActive:     true,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, apperrors.Internal(err)
	}

	return u, nil
}

// GetProfile returns a user's profile.
func (s *Service) GetProfile(ctx context.Context, userID string) (*user.User, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if u == nil {
		return nil, apperrors.NotFound(apperrors.CodeUserNotFound, "user not found")
	}
	return u, nil
}

// AcceptInvite links the user to an org with the pre-assigned role.
func (s *Service) AcceptInvite(ctx context.Context, userID, orgID, roleID string) error {
	if err := s.users.AddToOrg(ctx, userID, orgID, roleID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// ListMembers returns paginated organization members.
func (s *Service) ListMembers(ctx context.Context, orgID string, limit, offset int) ([]*user.Member, int64, error) {
	members, total, err := s.users.ListMembers(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	return members, total, nil
}

// GetMember returns a single org member.
func (s *Service) GetMember(ctx context.Context, userID, orgID string) (*user.Member, error) {
	m, err := s.users.GetMember(ctx, userID, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if m == nil {
		return nil, apperrors.NotFound(apperrors.CodeUserNotFound, "member not found")
	}
	return m, nil
}

// UpdateMemberRole changes a member's role within an org.
func (s *Service) UpdateMemberRole(ctx context.Context, userID, orgID, roleID string) error {
	if err := s.users.UpdateMemberRole(ctx, userID, orgID, roleID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// Deactivate disables a member's access to an org.
func (s *Service) Deactivate(ctx context.Context, userID, orgID string) error {
	if err := s.users.DeactivateMember(ctx, userID, orgID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// Reactivate re-enables a member's access to an org.
func (s *Service) Reactivate(ctx context.Context, userID, orgID string) error {
	if err := s.users.ReactivateMember(ctx, userID, orgID); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// UpdateProfile updates a member's full name and email.
func (s *Service) UpdateProfile(ctx context.Context, userID, orgID, fullName, email string) error {
	// Verify the member belongs to the org.
	m, err := s.users.GetMember(ctx, userID, orgID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if m == nil {
		return apperrors.NotFound(apperrors.CodeUserNotFound, "member not found")
	}

	// Ensure the new email is not already taken by another user.
	if m.Email != email {
		exists, err := s.users.EmailExists(ctx, email)
		if err != nil {
			return apperrors.Internal(err)
		}
		if exists {
			return apperrors.Conflict(apperrors.CodeEmailTaken, "email address is already registered")
		}
	}

	if err := s.users.UpdateMemberProfile(ctx, userID, orgID, fullName, email); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// UpdatePassword sets a new password for a member.
func (s *Service) UpdatePassword(ctx context.Context, userID, orgID, newPassword string) error {
	m, err := s.users.GetMember(ctx, userID, orgID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if m == nil {
		return apperrors.NotFound(apperrors.CodeUserNotFound, "member not found")
	}

	hash, err := authsvc.HashPassword(newPassword)
	if err != nil {
		return apperrors.Internal(err)
	}

	if err := s.users.UpdateMemberPassword(ctx, userID, hash); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// GetEffectivePermissions returns the role permissions combined with per-member overrides.
func (s *Service) GetEffectivePermissions(ctx context.Context, userID, orgID string) (*user.EffectivePermissions, error) {
	m, err := s.users.GetMember(ctx, userID, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if m == nil {
		return nil, apperrors.NotFound(apperrors.CodeUserNotFound, "member not found")
	}

	rolePerms, err := s.roles.GetPermissionKeys(ctx, m.RoleID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	memberID, err := s.users.GetMemberID(ctx, userID, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	overrides, err := s.users.GetMemberPermissionOverrides(ctx, memberID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	effective := ApplyOverrides(rolePerms, overrides)
	return &user.EffectivePermissions{
		RolePermissions: rolePerms,
		Overrides:       overrides,
		Effective:       effective,
	}, nil
}

// SetPermissionOverrides replaces all per-member permission overrides.
func (s *Service) SetPermissionOverrides(ctx context.Context, userID, orgID string, overrides []user.PermissionOverride) error {
	m, err := s.users.GetMember(ctx, userID, orgID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if m == nil {
		return apperrors.NotFound(apperrors.CodeUserNotFound, "member not found")
	}

	memberID, err := s.users.GetMemberID(ctx, userID, orgID)
	if err != nil {
		return apperrors.Internal(err)
	}

	if err := s.users.SetMemberPermissionOverrides(ctx, memberID, overrides); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// InviteInput holds the data needed to create an invitation.
type InviteInput struct {
	OrgID     string
	Email     string
	RoleID    string
	InviterID string
}

// InviteToken is the raw (unhashed) token sent via email.
type InviteToken struct {
	Token     string
	ExpiresAt time.Time
}
