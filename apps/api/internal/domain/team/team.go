package team

import (
	"context"
	"time"
)

type Team struct {
	ID          string    `db:"id"          json:"id"`
	OrgID       string    `db:"org_id"      json:"org_id"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"  json:"updated_at"`
}

type Member struct {
	TeamID    string    `db:"team_id"   json:"team_id"`
	UserID    string    `db:"user_id"   json:"user_id"`
	FullName  string    `db:"full_name" json:"full_name"`
	Email     string    `db:"email"     json:"email"`
	AvatarURL *string   `db:"avatar_url" json:"avatar_url"`
	AddedAt   time.Time `db:"added_at"  json:"added_at"`
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
	Create(ctx context.Context, team *Team) error
	FindByID(ctx context.Context, id, orgID string) (*Team, error)
	ListByOrg(ctx context.Context, orgID string) ([]*Team, error)
	Update(ctx context.Context, id, orgID string, input UpdateInput) (*Team, error)
	Delete(ctx context.Context, id, orgID string) error

	AddMember(ctx context.Context, teamID, userID string) error
	RemoveMember(ctx context.Context, teamID, userID string) error
	ListMembers(ctx context.Context, teamID, orgID string) ([]*Member, error)
}
