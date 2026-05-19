package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/team"
)

type teamRepo struct {
	db *sqlx.DB
}

func NewTeamRepo(db *sqlx.DB) team.Repository {
	return &teamRepo{db: db}
}

func (r *teamRepo) Create(ctx context.Context, t *team.Team) error {
	q := `INSERT INTO teams (id, org_id, name, description)
	      VALUES (:id, :org_id, :name, :description)`
	_, err := r.db.NamedExecContext(ctx, q, t)
	return err
}

func (r *teamRepo) FindByID(ctx context.Context, id, orgID string) (*team.Team, error) {
	var t team.Team
	err := r.db.GetContext(ctx, &t,
		`SELECT * FROM teams WHERE id = $1 AND org_id = $2`, id, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (r *teamRepo) ListByOrg(ctx context.Context, orgID string) ([]*team.Team, error) {
	teams := make([]*team.Team, 0)
	err := r.db.SelectContext(ctx, &teams,
		`SELECT * FROM teams WHERE org_id = $1 ORDER BY name ASC`, orgID)
	return teams, err
}

func (r *teamRepo) Update(ctx context.Context, id, orgID string, input team.UpdateInput) (*team.Team, error) {
	q := `UPDATE teams SET
	          name        = COALESCE($3, name),
	          description = COALESCE($4, description),
	          updated_at  = NOW()
	      WHERE id = $1 AND org_id = $2
	      RETURNING *`
	var t team.Team
	err := r.db.GetContext(ctx, &t, q, id, orgID, input.Name, input.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (r *teamRepo) Delete(ctx context.Context, id, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM teams WHERE id = $1 AND org_id = $2`, id, orgID)
	return err
}

func (r *teamRepo) AddMember(ctx context.Context, teamID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		teamID, userID)
	return err
}

func (r *teamRepo) RemoveMember(ctx context.Context, teamID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	return err
}

func (r *teamRepo) ListMembers(ctx context.Context, teamID, orgID string) ([]*team.Member, error) {
	q := `SELECT tm.team_id, tm.user_id, u.full_name, u.email, u.avatar_url, tm.added_at
	      FROM team_members tm
	      JOIN users u ON u.id = tm.user_id
	      JOIN organization_members om ON om.user_id = tm.user_id AND om.org_id = $2
	      WHERE tm.team_id = $1
	      ORDER BY u.full_name ASC`
	members := make([]*team.Member, 0)
	err := r.db.SelectContext(ctx, &members, q, teamID, orgID)
	return members, err
}

// ListByUser returns all team IDs for a given user in an org.
func (r *teamRepo) ListTeamIDsByUser(ctx context.Context, userID, orgID string) ([]string, error) {
	q := `SELECT tm.team_id FROM team_members tm
	      JOIN teams t ON t.id = tm.team_id
	      WHERE tm.user_id = $1 AND t.org_id = $2`
	ids := make([]string, 0)
	err := r.db.SelectContext(ctx, &ids, q, userID, orgID)
	return ids, err
}
