package category

import (
	"context"
	"time"
)

type Category struct {
	ID          string    `db:"id"          json:"id"`
	OrgID       string    `db:"org_id"      json:"org_id"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"  json:"updated_at"`
}

type CreateInput struct {
	OrgID       string
	Name        string
	Description *string
}

type UpdateInput struct {
	Name        *string
	Description *string
}

type Repository interface {
	Create(ctx context.Context, c *Category) error
	FindByID(ctx context.Context, id, orgID string) (*Category, error)
	ListByOrg(ctx context.Context, orgID string) ([]*Category, error)
	Update(ctx context.Context, id, orgID string, input UpdateInput) (*Category, error)
	Delete(ctx context.Context, id, orgID string) error
	ListByTeam(ctx context.Context, teamID string) ([]*Category, error)
	AddToTeam(ctx context.Context, teamID, categoryID string) error
	RemoveFromTeam(ctx context.Context, teamID, categoryID string) error
}
