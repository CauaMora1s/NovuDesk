package comment

import (
	"context"
	"time"
)

type Comment struct {
	ID           string     `db:"id"            json:"id"`
	TicketID     string     `db:"ticket_id"     json:"ticket_id"`
	OrgID        string     `db:"org_id"        json:"org_id"`
	AuthorID     *string    `db:"author_id"     json:"author_id"`
	AuthorName   *string    `db:"author_name"   json:"author_name,omitempty"`
	AuthorAvatar *string    `db:"author_avatar" json:"author_avatar,omitempty"`
	Body         string     `db:"body"          json:"body"`
	IsInternal   bool       `db:"is_internal"   json:"is_internal"`
	CreatedAt    time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"    json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"    json:"deleted_at,omitempty"`
}

type CreateInput struct {
	TicketID   string
	OrgID      string
	AuthorID   string
	Body       string
	IsInternal bool
}

type UpdateInput struct {
	Body string
}

type Repository interface {
	Create(ctx context.Context, c *Comment) error
	FindByID(ctx context.Context, id, orgID string) (*Comment, error)
	ListByTicket(ctx context.Context, ticketID, orgID string, includeInternal bool) ([]*Comment, error)
	Update(ctx context.Context, id, orgID string, input UpdateInput) (*Comment, error)
	SoftDelete(ctx context.Context, id, orgID string) error
}
