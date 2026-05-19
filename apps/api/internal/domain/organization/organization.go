package organization

import (
	"context"
	"encoding/json"
	"time"
)

type Organization struct {
	ID        string          `db:"id"         json:"id"`
	Name      string          `db:"name"       json:"name"`
	Slug      string          `db:"slug"       json:"slug"`
	LogoURL   *string         `db:"logo_url"   json:"logo_url"`
	PlanTier  string          `db:"plan_tier"  json:"plan_tier"`
	Settings  json.RawMessage `db:"settings"   json:"settings"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

type CreateInput struct {
	Name     string
	Slug     string
	OwnerID  string
}

type UpdateInput struct {
	Name     *string
	LogoURL  *string
	Settings json.RawMessage
}

type Repository interface {
	Create(ctx context.Context, org *Organization) error
	FindByID(ctx context.Context, id string) (*Organization, error)
	FindBySlug(ctx context.Context, slug string) (*Organization, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Organization, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
}
