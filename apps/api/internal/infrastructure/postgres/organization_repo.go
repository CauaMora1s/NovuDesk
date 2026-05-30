package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/organization"
)

type orgRepo struct {
	db *sqlx.DB
}

func NewOrgRepo(db *sqlx.DB) organization.Repository {
	return &orgRepo{db: db}
}

func (r *orgRepo) Create(ctx context.Context, org *organization.Organization) error {
	q := `INSERT INTO organizations (id, name, slug, logo_url, plan_tier, settings)
	      VALUES (:id, :name, :slug, :logo_url, :plan_tier, :settings)`
	_, err := r.db.NamedExecContext(ctx, q, org)
	return err
}

func (r *orgRepo) FindByID(ctx context.Context, id string) (*organization.Organization, error) {
	var org organization.Organization
	err := r.db.GetContext(ctx, &org, `SELECT * FROM organizations WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &org, err
}

func (r *orgRepo) FindBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	var org organization.Organization
	err := r.db.GetContext(ctx, &org, `SELECT * FROM organizations WHERE slug = $1`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &org, err
}

func (r *orgRepo) Update(ctx context.Context, id string, input organization.UpdateInput) (*organization.Organization, error) {
	q := `UPDATE organizations SET
	          name       = COALESCE($2, name),
	          logo_url   = COALESCE($3, logo_url),
	          settings   = COALESCE($4, settings),
	          updated_at = NOW()
	      WHERE id = $1
	      RETURNING *`
	var org organization.Organization
	err := r.db.GetContext(ctx, &org, q, id, input.Name, input.LogoURL, input.Settings)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &org, err
}

func (r *orgRepo) UpdatePlan(ctx context.Context, id string, input organization.PlanUpdate) (*organization.Organization, error) {
	q := `UPDATE organizations SET
	          plan_tier      = $2,
	          plan_renews_at = $3,
	          updated_at     = NOW()
	      WHERE id = $1
	      RETURNING *`
	var org organization.Organization
	err := r.db.GetContext(ctx, &org, q, id, input.PlanTier, input.RenewsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &org, err
}

func (r *orgRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(1) FROM organizations WHERE slug = $1`, slug)
	if err != nil {
		return false, fmt.Errorf("slug exists: %w", err)
	}
	return count > 0, nil
}
