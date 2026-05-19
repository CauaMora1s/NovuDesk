package sla

import (
	"context"
	"encoding/json"
	"time"
)

type Policy struct {
	ID              string          `db:"id"`
	OrgID           string          `db:"org_id"`
	Name            string          `db:"name"`
	ResponseHours   int             `db:"response_hours"`
	ResolutionHours int             `db:"resolution_hours"`
	Conditions      json.RawMessage `db:"conditions"`
	IsActive        bool            `db:"is_active"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

type CreateInput struct {
	OrgID           string
	Name            string
	ResponseHours   int
	ResolutionHours int
	Conditions      json.RawMessage
}

// CalculateDueDates computes response and resolution deadlines from a reference time.
func (p *Policy) CalculateDueDates(from time.Time) (responseDue, resolutionDue time.Time) {
	responseDue = from.Add(time.Duration(p.ResponseHours) * time.Hour)
	resolutionDue = from.Add(time.Duration(p.ResolutionHours) * time.Hour)
	return
}

type Repository interface {
	Create(ctx context.Context, policy *Policy) error
	FindByID(ctx context.Context, id, orgID string) (*Policy, error)
	ListByOrg(ctx context.Context, orgID string) ([]*Policy, error)
	Update(ctx context.Context, id, orgID string, input CreateInput) (*Policy, error)
	Delete(ctx context.Context, id, orgID string) error
}
