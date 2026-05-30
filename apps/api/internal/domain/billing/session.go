package billing

import (
	"context"
	"time"
)

// SessionStatus is the lifecycle state of a plan-change payment session.
type SessionStatus string

const (
	StatusPending   SessionStatus = "pending"
	StatusCompleted SessionStatus = "completed"
	StatusCancelled SessionStatus = "cancelled"
	StatusFailed    SessionStatus = "failed"
	StatusExpired   SessionStatus = "expired"
)

// PaymentSession records a single attempt to change an organization's plan.
// The organization's active plan only changes when a session reaches
// StatusCompleted; while pending/cancelled/failed the previous plan stays active.
type PaymentSession struct {
	ID             string        `db:"id"              json:"id"`
	OrgID          string        `db:"org_id"          json:"org_id"`
	FromTier       *string       `db:"from_tier"       json:"from_tier"`
	ToTier         string        `db:"to_tier"         json:"to_tier"`
	Status         SessionStatus `db:"status"          json:"status"`
	AmountCents    int64         `db:"amount_cents"    json:"amount_cents"`
	ProrationCents int64         `db:"proration_cents" json:"proration_cents"`
	Currency       string        `db:"currency"        json:"currency"`
	Provider       string        `db:"provider"        json:"provider"`
	ProviderRef    *string       `db:"provider_ref"    json:"provider_ref"`
	CreatedBy      *string       `db:"created_by"      json:"created_by"`
	CreatedAt      time.Time     `db:"created_at"      json:"created_at"`
	CompletedAt    *time.Time    `db:"completed_at"    json:"completed_at"`
	ExpiresAt      *time.Time    `db:"expires_at"      json:"expires_at"`
}

// SessionRepository persists payment sessions.
type SessionRepository interface {
	Create(ctx context.Context, s *PaymentSession) error
	FindByID(ctx context.Context, id, orgID string) (*PaymentSession, error)
	UpdateStatus(ctx context.Context, id, orgID string, status SessionStatus, completedAt *time.Time) error
	ListByOrg(ctx context.Context, orgID string) ([]*PaymentSession, error)
	FindActivePending(ctx context.Context, orgID string) (*PaymentSession, error)
}
