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
