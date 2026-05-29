package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/sla"
)

type slaRepo struct {
	db *sqlx.DB
}

func NewSLARepo(db *sqlx.DB) sla.Repository {
	return &slaRepo{db: db}
}

func (r *slaRepo) Create(ctx context.Context, p *sla.Policy) error {
	q := `INSERT INTO sla_policies
	        (id, org_id, name, category_id, response_hours, resolution_hours,
	         resolution_value, resolution_unit, conditions)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)`
	_, err := r.db.ExecContext(ctx, q,
		p.ID, p.OrgID, p.Name, p.CategoryID, p.ResponseHours, p.ResolutionHoursComputed(),
		p.ResolutionValue, p.ResolutionUnit, string(p.Conditions))
	return err
}

func (r *slaRepo) FindByID(ctx context.Context, id, orgID string) (*sla.Policy, error) {
	var p sla.Policy
	err := r.db.GetContext(ctx, &p,
		`SELECT * FROM sla_policies WHERE id = $1 AND org_id = $2`, id, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

func (r *slaRepo) FindByCategoryID(ctx context.Context, categoryID, orgID string) (*sla.Policy, error) {
	var p sla.Policy
	err := r.db.GetContext(ctx, &p,
		`SELECT * FROM sla_policies WHERE category_id = $1 AND org_id = $2 AND is_active = TRUE`,
		categoryID, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

func (r *slaRepo) ListByOrg(ctx context.Context, orgID string) ([]*sla.Policy, error) {
	policies := make([]*sla.Policy, 0)
	err := r.db.SelectContext(ctx, &policies,
		`SELECT * FROM sla_policies WHERE org_id = $1 ORDER BY name ASC`, orgID)
	return policies, err
}

func (r *slaRepo) ListWithCategoryStats(ctx context.Context, orgID string) ([]*sla.CategorySLAStat, error) {
	q := `
		SELECT
			c.id   AS category_id,
			c.name AS category_name,
			sp.id  AS sla_id,
			sp.resolution_value,
			sp.resolution_unit,
			sp.resolution_hours,
			AVG(EXTRACT(EPOCH FROM (t.resolved_at - t.created_at)) / 3600.0) AS avg_resolution_hours
		FROM categories c
		LEFT JOIN sla_policies sp
			ON sp.category_id = c.id AND sp.org_id = c.org_id AND sp.is_active = TRUE
		LEFT JOIN tickets t
			ON t.category_id = c.id AND t.org_id = c.org_id AND t.resolved_at IS NOT NULL
		WHERE c.org_id = $1
		GROUP BY c.id, c.name, sp.id, sp.resolution_value, sp.resolution_unit, sp.resolution_hours
		ORDER BY c.name ASC`

	stats := make([]*sla.CategorySLAStat, 0)
	err := r.db.SelectContext(ctx, &stats, q, orgID)
	return stats, err
}

func (r *slaRepo) Update(ctx context.Context, id, orgID string, input sla.CreateInput) (*sla.Policy, error) {
	// Compute canonical hours before saving.
	tmp := &sla.Policy{ResolutionValue: input.ResolutionValue, ResolutionUnit: input.ResolutionUnit}
	resolutionHours := tmp.ResolutionHoursComputed()

	q := `UPDATE sla_policies SET
	          name             = COALESCE($3, name),
	          category_id      = COALESCE($4, category_id),
	          response_hours   = COALESCE($5, response_hours),
	          resolution_value = COALESCE($6, resolution_value),
	          resolution_unit  = COALESCE($7, resolution_unit),
	          resolution_hours = $8,
	          updated_at       = NOW()
	      WHERE id = $1 AND org_id = $2
	      RETURNING *`

	var nameArg *string
	if input.Name != "" {
		nameArg = &input.Name
	}
	var responseArg *int
	if input.ResponseHours > 0 {
		responseArg = &input.ResponseHours
	}
	var valArg *int
	if input.ResolutionValue > 0 {
		valArg = &input.ResolutionValue
	}
	var unitArg *string
	if input.ResolutionUnit != "" {
		unitArg = &input.ResolutionUnit
	}

	var p sla.Policy
	err := r.db.GetContext(ctx, &p, q, id, orgID,
		nameArg, input.CategoryID, responseArg, valArg, unitArg, resolutionHours)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

func (r *slaRepo) Delete(ctx context.Context, id, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sla_policies WHERE id = $1 AND org_id = $2`, id, orgID)
	return err
}
