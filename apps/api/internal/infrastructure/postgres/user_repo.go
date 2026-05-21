package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/user"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) user.Repository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *user.User) error {
	q := `INSERT INTO users (id, email, password_hash, full_name, avatar_url, locale)
	      VALUES (:id, :email, :password_hash, :full_name, :avatar_url, :locale)`
	_, err := r.db.NamedExecContext(ctx, q, u)
	return err
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = $1`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (r *userRepo) Update(ctx context.Context, id string, input user.UpdateInput) (*user.User, error) {
	q := `UPDATE users SET
	          full_name  = COALESCE($2, full_name),
	          avatar_url = COALESCE($3, avatar_url),
	          locale     = COALESCE($4, locale),
	          updated_at = NOW()
	      WHERE id = $1 RETURNING *`
	var u user.User
	err := r.db.GetContext(ctx, &u, q, id, input.FullName, input.AvatarURL, input.Locale)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (r *userRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(1) FROM users WHERE email = $1`, email)
	return count > 0, err
}

func (r *userRepo) AddToOrg(ctx context.Context, userID, orgID, roleID string) error {
	q := `INSERT INTO organization_members (org_id, user_id, role_id)
	      VALUES ($1, $2, $3) ON CONFLICT (org_id, user_id) DO UPDATE SET role_id = $3, is_active = TRUE`
	_, err := r.db.ExecContext(ctx, q, orgID, userID, roleID)
	return err
}

func (r *userRepo) GetMember(ctx context.Context, userID, orgID string) (*user.Member, error) {
	q := `SELECT u.*, om.org_id, om.role_id, r.name AS role_name,
	             om.is_active AS member_is_active, om.joined_at
	      FROM users u
	      JOIN organization_members om ON om.user_id = u.id
	      JOIN roles r ON r.id = om.role_id
	      WHERE u.id = $1 AND om.org_id = $2`
	var m user.Member
	err := r.db.GetContext(ctx, &m, q, userID, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *userRepo) ListMembers(ctx context.Context, orgID string, limit, offset int) ([]*user.Member, int64, error) {
	q := `SELECT u.*, om.org_id, om.role_id, r.name AS role_name,
	             om.is_active AS member_is_active, om.joined_at
	      FROM users u
	      JOIN organization_members om ON om.user_id = u.id
	      JOIN roles r ON r.id = om.role_id
	      WHERE om.org_id = $1
	      ORDER BY om.joined_at DESC
	      LIMIT $2 OFFSET $3`

	members := make([]*user.Member, 0)
	if err := r.db.SelectContext(ctx, &members, q, orgID, limit, offset); err != nil {
		return nil, 0, err
	}

	var total int64
	r.db.GetContext(ctx, &total, `SELECT COUNT(1) FROM organization_members WHERE org_id = $1`, orgID)

	return members, total, nil
}

func (r *userRepo) UpdateMemberRole(ctx context.Context, userID, orgID, roleID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE organization_members SET role_id = $3 WHERE user_id = $1 AND org_id = $2`,
		userID, orgID, roleID)
	return err
}

func (r *userRepo) DeactivateMember(ctx context.Context, userID, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE organization_members SET is_active = FALSE WHERE user_id = $1 AND org_id = $2`,
		userID, orgID)
	return err
}

func (r *userRepo) ReactivateMember(ctx context.Context, userID, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE organization_members SET is_active = TRUE WHERE user_id = $1 AND org_id = $2`,
		userID, orgID)
	return err
}

func (r *userRepo) GetMemberID(ctx context.Context, userID, orgID string) (string, error) {
	var id string
	err := r.db.GetContext(ctx, &id,
		`SELECT id FROM organization_members WHERE user_id = $1 AND org_id = $2`,
		userID, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (r *userRepo) UpdateMemberProfile(ctx context.Context, userID, orgID, fullName, email string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET full_name = $2, email = $3, updated_at = NOW()
		 WHERE id = $1 AND EXISTS (
		     SELECT 1 FROM organization_members WHERE user_id = $1 AND org_id = $4
		 )`,
		userID, fullName, email, orgID)
	return err
}

func (r *userRepo) UpdateMemberPassword(ctx context.Context, userID, newPasswordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		userID, newPasswordHash)
	return err
}

func (r *userRepo) GetMemberPermissionOverrides(ctx context.Context, memberID string) ([]user.PermissionOverride, error) {
	type row struct {
		Key       string `db:"key"`
		IsGranted bool   `db:"is_granted"`
	}
	var rows []row
	err := r.db.SelectContext(ctx, &rows, `
		SELECT p.key, o.is_granted
		FROM member_permission_overrides o
		JOIN permissions p ON p.id = o.permission_id
		WHERE o.member_id = $1
		ORDER BY p.key`, memberID)
	if err != nil {
		return nil, err
	}
	overrides := make([]user.PermissionOverride, len(rows))
	for i, row := range rows {
		overrides[i] = user.PermissionOverride{PermissionKey: row.Key, IsGranted: row.IsGranted}
	}
	return overrides, nil
}

func (r *userRepo) SetMemberPermissionOverrides(ctx context.Context, memberID string, overrides []user.PermissionOverride) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM member_permission_overrides WHERE member_id = $1`, memberID); err != nil {
		return err
	}

	for _, o := range overrides {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO member_permission_overrides (member_id, permission_id, is_granted)
			SELECT $1, id, $3 FROM permissions WHERE key = $2
			ON CONFLICT (member_id, permission_id) DO UPDATE SET is_granted = $3`,
			memberID, o.PermissionKey, o.IsGranted)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
