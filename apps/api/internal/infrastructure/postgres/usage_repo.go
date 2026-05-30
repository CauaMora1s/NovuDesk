package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/organization"
)

type usageRepo struct {
	db *sqlx.DB
}

func NewUsageRepo(db *sqlx.DB) organization.UsageRepository {
	return &usageRepo{db: db}
}

// Snapshot returns the organization's current resource consumption.
func (r *usageRepo) Snapshot(ctx context.Context, orgID string) (*organization.Usage, error) {
	const q = `
SELECT
    (SELECT COUNT(1) FROM organization_members WHERE org_id = $1 AND is_active)                                          AS members,
    (SELECT COUNT(1) FROM tickets WHERE org_id = $1 AND created_at >= date_trunc('month', NOW()))                        AS tickets_this_month,
    (SELECT COALESCE(SUM(size_bytes), 0) FROM ticket_attachments WHERE org_id = $1)                                      AS storage_bytes,
    (SELECT COUNT(1) FROM teams WHERE org_id = $1)                                                                       AS teams,
    (SELECT COUNT(1) FROM categories WHERE org_id = $1)                                                                  AS categories,
    (SELECT COUNT(1) FROM api_keys WHERE org_id = $1 AND revoked_at IS NULL)                                             AS api_keys`

	var u organization.Usage
	if err := r.db.GetContext(ctx, &u, q, orgID); err != nil {
		return nil, fmt.Errorf("usage snapshot: %w", err)
	}
	return &u, nil
}
