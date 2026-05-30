package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/billing"
)

type paymentSessionRepo struct {
	db *sqlx.DB
}

func NewPaymentSessionRepo(db *sqlx.DB) billing.SessionRepository {
	return &paymentSessionRepo{db: db}
}

func (r *paymentSessionRepo) Create(ctx context.Context, s *billing.PaymentSession) error {
	const q = `
INSERT INTO payment_sessions
    (id, org_id, from_tier, to_tier, status, amount_cents, proration_cents, currency, provider, provider_ref, created_by, expires_at)
VALUES
    (:id, :org_id, :from_tier, :to_tier, :status, :amount_cents, :proration_cents, :currency, :provider, :provider_ref, :created_by, :expires_at)`
	_, err := r.db.NamedExecContext(ctx, q, s)
	return err
}

func (r *paymentSessionRepo) FindByID(ctx context.Context, id, orgID string) (*billing.PaymentSession, error) {
	var s billing.PaymentSession
	err := r.db.GetContext(ctx, &s, `SELECT * FROM payment_sessions WHERE id = $1 AND org_id = $2`, id, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *paymentSessionRepo) UpdateStatus(ctx context.Context, id, orgID string, status billing.SessionStatus, completedAt *time.Time) error {
	const q = `UPDATE payment_sessions SET status = $3, completed_at = $4 WHERE id = $1 AND org_id = $2`
	_, err := r.db.ExecContext(ctx, q, id, orgID, status, completedAt)
	return err
}

func (r *paymentSessionRepo) ListByOrg(ctx context.Context, orgID string) ([]*billing.PaymentSession, error) {
	var out []*billing.PaymentSession
	err := r.db.SelectContext(ctx, &out, `SELECT * FROM payment_sessions WHERE org_id = $1 ORDER BY created_at DESC`, orgID)
	return out, err
}

func (r *paymentSessionRepo) FindActivePending(ctx context.Context, orgID string) (*billing.PaymentSession, error) {
	var s billing.PaymentSession
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM payment_sessions WHERE org_id = $1 AND status = 'pending' ORDER BY created_at DESC LIMIT 1`, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}
