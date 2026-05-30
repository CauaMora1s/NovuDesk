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

	// Billing state (1:1 with the organization).
	PlanRenewsAt       *time.Time `db:"plan_renews_at"       json:"plan_renews_at"`
	BillingStatus      string     `db:"billing_status"       json:"billing_status"`
	BillingProvider    string     `db:"billing_provider"     json:"billing_provider"`
	BillingCustomerRef *string    `db:"billing_customer_ref" json:"billing_customer_ref"`
	PaymentMethodBrand *string    `db:"payment_method_brand" json:"payment_method_brand"`
	PaymentMethodLast4 *string    `db:"payment_method_last4" json:"payment_method_last4"`
}

type CreateInput struct {
	Name    string
	Slug    string
	OwnerID string
}

type UpdateInput struct {
	Name     *string
	LogoURL  *string
	Settings json.RawMessage
}

// PlanUpdate carries the billing fields applied when a plan change completes.
type PlanUpdate struct {
	PlanTier  string
	RenewsAt  *time.Time
}

// Usage is a point-in-time snapshot of an organization's resource consumption.
type Usage struct {
	Members          int64 `db:"members"            json:"members"`
	TicketsThisMonth int64 `db:"tickets_this_month" json:"tickets_this_month"`
	StorageBytes     int64 `db:"storage_bytes"      json:"storage_bytes"`
	Teams            int64 `db:"teams"              json:"teams"`
	Categories       int64 `db:"categories"         json:"categories"`
	APIKeys          int64 `db:"api_keys"           json:"api_keys"`
}

type Repository interface {
	Create(ctx context.Context, org *Organization) error
	FindByID(ctx context.Context, id string) (*Organization, error)
	FindBySlug(ctx context.Context, slug string) (*Organization, error)
	Update(ctx context.Context, id string, input UpdateInput) (*Organization, error)
	UpdatePlan(ctx context.Context, id string, input PlanUpdate) (*Organization, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
}

// UsageRepository aggregates an organization's current resource usage.
type UsageRepository interface {
	Snapshot(ctx context.Context, orgID string) (*Usage, error)
}
