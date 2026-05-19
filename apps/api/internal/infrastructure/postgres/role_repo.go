package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/novudesk/novudesk/internal/domain/role"
)

type roleRepo struct {
	db *sqlx.DB
}

func NewRoleRepo(db *sqlx.DB) role.Repository {
	return &roleRepo{db: db}
}

func (r *roleRepo) Create(ctx context.Context, ro *role.Role) error {
	q := `INSERT INTO roles (id, org_id, name, is_system_role) VALUES (:id, :org_id, :name, :is_system_role)`
	_, err := r.db.NamedExecContext(ctx, q, ro)
	return err
}

func (r *roleRepo) FindByID(ctx context.Context, id string) (*role.Role, error) {
	var ro role.Role
	err := r.db.GetContext(ctx, &ro, `SELECT * FROM roles WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &ro, err
}

func (r *roleRepo) FindByName(ctx context.Context, orgID, name string) (*role.Role, error) {
	var ro role.Role
	err := r.db.GetContext(ctx, &ro,
		`SELECT * FROM roles WHERE org_id = $1 AND name = $2 LIMIT 1`, orgID, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &ro, err
}

func (r *roleRepo) FindSystemRole(ctx context.Context, name string) (*role.Role, error) {
	var ro role.Role
	err := r.db.GetContext(ctx, &ro,
		`SELECT * FROM roles WHERE is_system_role = true AND name = $1 LIMIT 1`, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &ro, err
}

func (r *roleRepo) ListByOrg(ctx context.Context, orgID string) ([]*role.Role, error) {
	var roles []*role.Role
	err := r.db.SelectContext(ctx, &roles,
		`SELECT * FROM roles WHERE org_id = $1 OR is_system_role = true ORDER BY name`, orgID)
	return roles, err
}

func (r *roleRepo) Delete(ctx context.Context, id, orgID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM roles WHERE id = $1 AND org_id = $2 AND is_system_role = false`, id, orgID)
	return err
}

func (r *roleRepo) SetPermissions(ctx context.Context, roleID string, permissionKeys []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}

	for _, key := range permissionKeys {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT $1, id FROM permissions WHERE key = $2
			ON CONFLICT DO NOTHING`, roleID, key)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *roleRepo) GetPermissions(ctx context.Context, roleID string) ([]*role.Permission, error) {
	var perms []*role.Permission
	err := r.db.SelectContext(ctx, &perms, `
		SELECT p.* FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.key`, roleID)
	return perms, err
}

func (r *roleRepo) GetPermissionKeys(ctx context.Context, roleID string) ([]string, error) {
	var keys []string
	err := r.db.SelectContext(ctx, &keys, `
		SELECT p.key FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.key`, roleID)
	return keys, err
}

func (r *roleRepo) ListAllPermissions(ctx context.Context) ([]*role.Permission, error) {
	var perms []*role.Permission
	err := r.db.SelectContext(ctx, &perms, `SELECT * FROM permissions ORDER BY key`)
	return perms, err
}
