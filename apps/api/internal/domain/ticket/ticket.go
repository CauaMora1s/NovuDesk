package ticket

import (
	"context"
	"encoding/json"
	"time"
)

type Status string
type Priority string

const (
	StatusOpen     Status = "open"
	StatusPending  Status = "pending"
	StatusOnHold   Status = "on_hold"
	StatusResolved Status = "resolved"
	StatusClosed   Status = "closed"
)

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

type Ticket struct {
	ID                  string          `db:"id"                   json:"id"`
	OrgID               string          `db:"org_id"               json:"org_id"`
	Number              int64           `db:"number"               json:"number"`
	Title               string          `db:"title"                json:"title"`
	Description         *string         `db:"description"          json:"description"`
	Status              Status          `db:"status"               json:"status"`
	Priority            Priority        `db:"priority"             json:"priority"`
	AssigneeID          *string         `db:"assignee_id"          json:"assignee_id"`
	AssigneeName        *string         `db:"assignee_name"        json:"assignee_name,omitempty"`
	TeamID              *string         `db:"team_id"              json:"team_id"`
	TeamName            *string         `db:"team_name"            json:"team_name,omitempty"`
	RequesterID         *string         `db:"requester_id"         json:"requester_id"`
	RequesterName       *string         `db:"requester_name"       json:"requester_name,omitempty"`
	CategoryID          *string         `db:"category_id"          json:"category_id"`
	CategoryName        *string         `db:"category_name"        json:"category_name,omitempty"`
	SLAPolicyID         *string         `db:"sla_policy_id"        json:"sla_policy_id"`
	SLAResponseDueAt    *time.Time      `db:"sla_response_due_at"  json:"sla_response_due_at"`
	SLAResolutionDueAt  *time.Time      `db:"sla_resolution_due_at" json:"sla_resolution_due_at"`
	SLABreached         bool            `db:"sla_breached"         json:"sla_breached"`
	Tags                []string        `db:"tags"                 json:"tags"`
	CustomFields        json.RawMessage `db:"custom_fields"        json:"custom_fields"`
	CreatedAt           time.Time       `db:"created_at"           json:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"           json:"updated_at"`
	ResolvedAt          *time.Time      `db:"resolved_at"          json:"resolved_at"`
	ClosedAt            *time.Time      `db:"closed_at"            json:"closed_at"`
}

type CreateInput struct {
	OrgID        string
	Title        string
	Description  *string
	Priority     Priority
	AssigneeID   *string
	TeamID       *string
	CategoryID   *string
	RequesterID  *string
	SLAPolicyID  *string
	Tags         []string
	CustomFields json.RawMessage
	CreatedBy    string
}

type UpdateInput struct {
	Title        *string
	Description  *string
	Status       *Status
	Priority     *Priority
	AssigneeID   *string
	TeamID       *string
	CategoryID   *string
	SLAPolicyID  *string
	Tags         []string
	CustomFields json.RawMessage
}

type Filter struct {
	Status      []Status
	Priority    []Priority
	AssigneeID  *string
	TeamID      *string
	RequesterID *string
	CategoryID  *string
	Tags        []string
	Query       string
	SLABreached *bool
}

type Repository interface {
	Create(ctx context.Context, ticket *Ticket) error
	FindByID(ctx context.Context, id, orgID string) (*Ticket, error)
	FindByNumber(ctx context.Context, number int64, orgID string) (*Ticket, error)
	List(ctx context.Context, orgID string, filter Filter, limit, offset int) ([]*Ticket, int64, error)
	Update(ctx context.Context, id, orgID string, input UpdateInput) (*Ticket, error)
	Delete(ctx context.Context, id, orgID string) error
	NextNumber(ctx context.Context, orgID string) (int64, error)
	UpdateSLABreach(ctx context.Context, orgID string) (int, error)
}
