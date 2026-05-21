package user

import (
	"context"
	"time"
)

type User struct {
	ID           string    `db:"id"            json:"id"`
	Email        string    `db:"email"         json:"email"`
	PasswordHash *string   `db:"password_hash" json:"-"`
	FullName     string    `db:"full_name"     json:"full_name"`
	AvatarURL    *string   `db:"avatar_url"    json:"avatar_url"`
	Locale       string    `db:"locale"        json:"locale"`
	IsActive     bool      `db:"is_active"     json:"is_active"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"    json:"updated_at"`
}

type CreateInput struct {
	Email        string
	PasswordHash string
	FullName     string
	Locale       string
}

type UpdateInput struct {
	FullName  *string
	AvatarURL *string
	Locale    *string
}

// Member represents a user within an organization context.
type Member struct {
	User
	OrgID     string    `db:"org_id"           json:"org_id"`
	RoleID    string    `db:"role_id"          json:"role_id"`
	RoleName  string    `db:"role_name"        json:"role_name"`
	IsActive  bool      `db:"member_is_active" json:"is_active"`
	JoinedAt  time.Time `db:"joined_at"        json:"joined_at"`
}

// PermissionOverride represents a per-member permission grant or denial.
type PermissionOverride struct {
	PermissionKey string `db:"permission_key" json:"permission_key"`
	IsGranted     bool   `db:"is_granted"     json:"is_granted"`
}

// EffectivePermissions holds the role-based keys with overrides already applied.
type EffectivePermissions struct {
	RolePermissions []string             `json:"role_permissions"`
	Overrides       []PermissionOverride `json:"overrides"`
	Effective       []string             `json:"effective"`
}

type Repository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id string, input UpdateInput) (*User, error)
	EmailExists(ctx context.Context, email string) (bool, error)

	// Membership operations
	AddToOrg(ctx context.Context, userID, orgID, roleID string) error
	GetMember(ctx context.Context, userID, orgID string) (*Member, error)
	GetMemberID(ctx context.Context, userID, orgID string) (string, error)
	ListMembers(ctx context.Context, orgID string, limit, offset int) ([]*Member, int64, error)
	UpdateMemberRole(ctx context.Context, userID, orgID, roleID string) error
	UpdateMemberProfile(ctx context.Context, userID, orgID, fullName, email string) error
	UpdateMemberPassword(ctx context.Context, userID, newPasswordHash string) error
	DeactivateMember(ctx context.Context, userID, orgID string) error
	ReactivateMember(ctx context.Context, userID, orgID string) error

	// Permission overrides
	GetMemberPermissionOverrides(ctx context.Context, memberID string) ([]PermissionOverride, error)
	SetMemberPermissionOverrides(ctx context.Context, memberID string, overrides []PermissionOverride) error
}
