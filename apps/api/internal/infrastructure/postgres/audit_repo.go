package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/audit"
)

type auditRepo struct {
	db *sqlx.DB
}

func NewAuditRepo(db *sqlx.DB) audit.Repository {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(ctx context.Context, entry *audit.Log) error {
	q := `INSERT INTO audit_logs
	        (id, org_id, actor_id, actor_type, resource_type, resource_id, action, before, after, metadata)
	      VALUES
	        (:id, :org_id, :actor_id, :actor_type, :resource_type, :resource_id, :action, :before, :after, :metadata)`
	_, err := r.db.NamedExecContext(ctx, q, entry)
	return err
}

func (r *auditRepo) ListByOrg(ctx context.Context, orgID string, f audit.Filter, limit, offset int) ([]*audit.Log, int64, error) {
	q := `SELECT * FROM audit_logs WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	var logs []*audit.Log
	err := r.db.SelectContext(ctx, &logs, q, orgID, limit, offset)

	var total int64
	r.db.GetContext(ctx, &total, `SELECT COUNT(1) FROM audit_logs WHERE org_id = $1`, orgID)

	return logs, total, err
}

func (r *auditRepo) ListByResource(ctx context.Context, orgID, resourceType, resourceID string, limit, offset int) ([]*audit.Log, int64, error) {
	q := `SELECT * FROM audit_logs
	      WHERE org_id = $1 AND resource_type = $2 AND resource_id = $3
	      ORDER BY created_at DESC LIMIT $4 OFFSET $5`
	var logs []*audit.Log
	err := r.db.SelectContext(ctx, &logs, q, orgID, resourceType, resourceID, limit, offset)

	var total int64
	r.db.GetContext(ctx, &total,
		`SELECT COUNT(1) FROM audit_logs WHERE org_id = $1 AND resource_type = $2 AND resource_id = $3`,
		orgID, resourceType, resourceID)

	return logs, total, err
}

