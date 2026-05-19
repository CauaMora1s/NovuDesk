package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/category"
)

type categoryRepo struct {
	db *sqlx.DB
}

func NewCategoryRepo(db *sqlx.DB) category.Repository {
	return &categoryRepo{db: db}
}

func (r *categoryRepo) Create(ctx context.Context, c *category.Category) error {
	q := `INSERT INTO categories (id, org_id, name, description)
	      VALUES (:id, :org_id, :name, :description)`
	_, err := r.db.NamedExecContext(ctx, q, c)
	return err
}

func (r *categoryRepo) FindByID(ctx context.Context, id, orgID string) (*category.Category, error) {
	var c category.Category
	err := r.db.GetContext(ctx, &c,
		`SELECT * FROM categories WHERE id = $1 AND org_id = $2`, id, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *categoryRepo) ListByOrg(ctx context.Context, orgID string) ([]*category.Category, error) {
	cats := make([]*category.Category, 0)
	err := r.db.SelectContext(ctx, &cats,
		`SELECT * FROM categories WHERE org_id = $1 ORDER BY name ASC`, orgID)
	return cats, err
}

func (r *categoryRepo) Update(ctx context.Context, id, orgID string, input category.UpdateInput) (*category.Category, error) {
	q := `UPDATE categories SET
	          name        = COALESCE($3, name),
	          description = COALESCE($4, description),
	          updated_at  = NOW()
	      WHERE id = $1 AND org_id = $2
	      RETURNING *`
	var c category.Category
	err := r.db.GetContext(ctx, &c, q, id, orgID, input.Name, input.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (r *categoryRepo) Delete(ctx context.Context, id, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM categories WHERE id = $1 AND org_id = $2`, id, orgID)
	return err
}

func (r *categoryRepo) ListByTeam(ctx context.Context, teamID string) ([]*category.Category, error) {
	q := `SELECT c.* FROM categories c
	      JOIN team_categories tc ON tc.category_id = c.id
	      WHERE tc.team_id = $1
	      ORDER BY c.name ASC`
	cats := make([]*category.Category, 0)
	err := r.db.SelectContext(ctx, &cats, q, teamID)
	return cats, err
}

func (r *categoryRepo) AddToTeam(ctx context.Context, teamID, categoryID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO team_categories (team_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		teamID, categoryID)
	return err
}

func (r *categoryRepo) RemoveFromTeam(ctx context.Context, teamID, categoryID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM team_categories WHERE team_id = $1 AND category_id = $2`, teamID, categoryID)
	return err
}
