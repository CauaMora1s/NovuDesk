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
	CategoryID      *string         `db:"category_id"`
	ResponseHours   int             `db:"response_hours"`
	ResolutionHours int             `db:"resolution_hours"`
	ResolutionValue int             `db:"resolution_value"`
	ResolutionUnit  string          `db:"resolution_unit"` // "hours" | "days" | "weeks"
	Conditions      json.RawMessage `db:"conditions"`
	IsActive        bool            `db:"is_active"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

// ResolutionHoursComputed converts resolution_value + resolution_unit to hours.
func (p *Policy) ResolutionHoursComputed() int {
	switch p.ResolutionUnit {
	case "days":
		return p.ResolutionValue * 24
	case "weeks":
		return p.ResolutionValue * 24 * 7
	default: // "hours"
		return p.ResolutionValue
	}
}

// CalculateDueDates computes response and resolution deadlines from a reference time.
func (p *Policy) CalculateDueDates(from time.Time) (responseDue, resolutionDue time.Time) {
	responseDue = from.Add(time.Duration(p.ResponseHours) * time.Hour)
	resolutionDue = from.Add(time.Duration(p.ResolutionHoursComputed()) * time.Hour)
	return
}

type CreateInput struct {
	OrgID           string
	Name            string
	CategoryID      *string
	ResponseHours   int
	ResolutionValue int
	ResolutionUnit  string
	Conditions      json.RawMessage
}

// CategorySLAStat holds a category with its SLA policy (if any) and avg resolution time.
type CategorySLAStat struct {
	CategoryID          string   `db:"category_id"          json:"category_id"`
	CategoryName        string   `db:"category_name"        json:"category_name"`
	SLAID               *string  `db:"sla_id"               json:"sla_id"`
	ResolutionValue     *int     `db:"resolution_value"     json:"resolution_value"`
	ResolutionUnit      *string  `db:"resolution_unit"      json:"resolution_unit"`
	ResolutionHours     *int     `db:"resolution_hours"     json:"resolution_hours"`
	AvgResolutionHours  *float64 `db:"avg_resolution_hours" json:"avg_resolution_hours"`
}

type Repository interface {
	Create(ctx context.Context, policy *Policy) error
	FindByID(ctx context.Context, id, orgID string) (*Policy, error)
	FindByCategoryID(ctx context.Context, categoryID, orgID string) (*Policy, error)
	ListByOrg(ctx context.Context, orgID string) ([]*Policy, error)
	ListWithCategoryStats(ctx context.Context, orgID string) ([]*CategorySLAStat, error)
	Update(ctx context.Context, id, orgID string, input CreateInput) (*Policy, error)
	Delete(ctx context.Context, id, orgID string) error
}
