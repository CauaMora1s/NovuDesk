package audit

import (
	"context"
	"encoding/json"
	"time"
)

type ActorType string

const (
	ActorUser       ActorType = "user"
	ActorSystem     ActorType = "system"
	ActorAutomation ActorType = "automation"
)

type Log struct {
	ID           string          `db:"id"`
	OrgID        string          `db:"org_id"`
	ActorID      *string         `db:"actor_id"`
	ActorType    ActorType       `db:"actor_type"`
	ResourceType string          `db:"resource_type"`
	ResourceID   string          `db:"resource_id"`
	Action       string          `db:"action"`
	Before       json.RawMessage `db:"before"`
	After        json.RawMessage `db:"after"`
	Metadata     json.RawMessage `db:"metadata"`
	CreatedAt    time.Time       `db:"created_at"`
}

type CreateInput struct {
	OrgID        string
	ActorID      *string
	ActorType    ActorType
	ResourceType string
	ResourceID   string
	Action       string
	Before       any
	After        any
	Metadata     any
}

type Filter struct {
	ResourceType *string
	ResourceID   *string
	ActorID      *string
	From         *time.Time
	To           *time.Time
}

type Repository interface {
	Create(ctx context.Context, entry *Log) error
	ListByOrg(ctx context.Context, orgID string, filter Filter, limit, offset int) ([]*Log, int64, error)
	ListByResource(ctx context.Context, orgID, resourceType, resourceID string, limit, offset int) ([]*Log, int64, error)
}
