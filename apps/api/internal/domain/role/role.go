package role

import (
	"context"
	"time"
)

type Role struct {
	ID           string    `db:"id"             json:"id"`
	OrgID        *string   `db:"org_id"         json:"org_id"`
	Name         string    `db:"name"           json:"name"`
	IsSystemRole bool      `db:"is_system_role" json:"is_system_role"`
	CreatedAt    time.Time `db:"created_at"     json:"created_at"`
}

type Permission struct {
	ID          string    `db:"id"          json:"id"`
	Key         string    `db:"key"         json:"key"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

// System role names — immutable.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleAgent  = "agent"
	RoleViewer = "viewer"
)

type CreateInput struct {
	OrgID       string
	Name        string
	Permissions []string
}

type Repository interface {
	Create(ctx context.Context, role *Role) error
	FindByID(ctx context.Context, id string) (*Role, error)
	FindByName(ctx context.Context, orgID, name string) (*Role, error)
	FindSystemRole(ctx context.Context, name string) (*Role, error)
	ListByOrg(ctx context.Context, orgID string) ([]*Role, error)
	Delete(ctx context.Context, id, orgID string) error

	// Permission assignment
	SetPermissions(ctx context.Context, roleID string, permissionKeys []string) error
	GetPermissions(ctx context.Context, roleID string) ([]*Permission, error)
	GetPermissionKeys(ctx context.Context, roleID string) ([]string, error)
	ListAllPermissions(ctx context.Context) ([]*Permission, error)
}
