package role_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	rolesvc "github.com/novudesk/novudesk/internal/application/role"
	"github.com/novudesk/novudesk/internal/domain/role"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

// --- fake repository ---

type fakeRoleRepo struct {
	roles       []*role.Role
	permissions map[string][]*role.Permission
	allPerms    []*role.Permission
	err         error
}

func (f *fakeRoleRepo) Create(_ context.Context, r *role.Role) error {
	if f.err != nil {
		return f.err
	}
	f.roles = append(f.roles, r)
	return nil
}

func (f *fakeRoleRepo) FindByID(_ context.Context, id string) (*role.Role, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, r := range f.roles {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func (f *fakeRoleRepo) FindByName(_ context.Context, orgID, name string) (*role.Role, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, r := range f.roles {
		if r.Name == name && r.OrgID != nil && *r.OrgID == orgID {
			return r, nil
		}
	}
	return nil, nil
}

func (f *fakeRoleRepo) FindSystemRole(_ context.Context, name string) (*role.Role, error) {
	for _, r := range f.roles {
		if r.IsSystemRole && r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

func (f *fakeRoleRepo) ListByOrg(_ context.Context, orgID string) ([]*role.Role, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*role.Role
	for _, r := range f.roles {
		if r.IsSystemRole || (r.OrgID != nil && *r.OrgID == orgID) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRoleRepo) Update(_ context.Context, r *role.Role) error {
	if f.err != nil {
		return f.err
	}
	for i, existing := range f.roles {
		if existing.ID == r.ID {
			f.roles[i] = r
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakeRoleRepo) Delete(_ context.Context, id, _ string) error {
	if f.err != nil {
		return f.err
	}
	for i, r := range f.roles {
		if r.ID == id {
			f.roles = append(f.roles[:i], f.roles[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeRoleRepo) SetPermissions(_ context.Context, roleID string, keys []string) error {
	if f.permissions == nil {
		f.permissions = map[string][]*role.Permission{}
	}
	perms := make([]*role.Permission, len(keys))
	for i, k := range keys {
		perms[i] = &role.Permission{ID: "perm-" + k, Key: k, CreatedAt: time.Now()}
	}
	f.permissions[roleID] = perms
	return nil
}

func (f *fakeRoleRepo) GetPermissions(_ context.Context, roleID string) ([]*role.Permission, error) {
	if f.permissions == nil {
		return nil, nil
	}
	return f.permissions[roleID], nil
}

func (f *fakeRoleRepo) GetPermissionKeys(_ context.Context, roleID string) ([]string, error) {
	perms := f.permissions[roleID]
	keys := make([]string, len(perms))
	for i, p := range perms {
		keys[i] = p.Key
	}
	return keys, nil
}

func (f *fakeRoleRepo) ListAllPermissions(_ context.Context) ([]*role.Permission, error) {
	return f.allPerms, nil
}

// --- helpers ---

func ptr(s string) *string { return &s }

func newSystemRole(id, name string) *role.Role {
	return &role.Role{ID: id, Name: name, IsSystemRole: true, CreatedAt: time.Now()}
}

func newOrgRole(id, name, org string) *role.Role {
	return &role.Role{ID: id, OrgID: ptr(org), Name: name, IsSystemRole: false, CreatedAt: time.Now()}
}

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	var e *apperrors.AppError
	if !apperrors.As(err, &e) {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	return e.HTTPStatus
}

const testOrg = "org-abc"

// --- tests ---

func TestRoleService_Create_WhenNameExists_ReturnsConflict(t *testing.T) {
	repo := &fakeRoleRepo{
		roles: []*role.Role{newOrgRole("r1", "Support", testOrg)},
	}
	svc := rolesvc.NewService(repo)

	_, err := svc.Create(context.Background(), testOrg, "Support", []string{"tickets:read"})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if appErrStatus(t, err) != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", appErrStatus(t, err))
	}
}

func TestRoleService_Create_Success_ReturnsRoleWithPermissions(t *testing.T) {
	repo := &fakeRoleRepo{}
	svc := rolesvc.NewService(repo)

	rwp, err := svc.Create(context.Background(), testOrg, "Billing", []string{"tickets:read"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rwp.Name != "Billing" {
		t.Errorf("expected name Billing, got %s", rwp.Name)
	}
	if len(rwp.Permissions) != 1 || rwp.Permissions[0].Key != "tickets:read" {
		t.Errorf("unexpected permissions: %v", rwp.Permissions)
	}
}

func TestRoleService_Update_WhenSystemRole_ReturnsForbidden(t *testing.T) {
	repo := &fakeRoleRepo{
		roles: []*role.Role{newSystemRole("sys-1", role.RoleAdmin)},
	}
	svc := rolesvc.NewService(repo)

	_, err := svc.Update(context.Background(), "sys-1", testOrg, "NewName", nil)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if appErrStatus(t, err) != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", appErrStatus(t, err))
	}
}

func TestRoleService_Update_WhenOtherOrg_ReturnsForbidden(t *testing.T) {
	repo := &fakeRoleRepo{
		roles: []*role.Role{newOrgRole("r1", "Custom", "org-other")},
	}
	svc := rolesvc.NewService(repo)

	_, err := svc.Update(context.Background(), "r1", testOrg, "Renamed", nil)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if appErrStatus(t, err) != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", appErrStatus(t, err))
	}
}

func TestRoleService_Delete_WhenSystemRole_ReturnsForbidden(t *testing.T) {
	repo := &fakeRoleRepo{
		roles: []*role.Role{newSystemRole("sys-1", role.RoleOwner)},
	}
	svc := rolesvc.NewService(repo)

	err := svc.Delete(context.Background(), "sys-1", testOrg)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if appErrStatus(t, err) != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", appErrStatus(t, err))
	}
}

func TestRoleService_Delete_WhenNotFound_ReturnsNotFound(t *testing.T) {
	repo := &fakeRoleRepo{}
	svc := rolesvc.NewService(repo)

	err := svc.Delete(context.Background(), "nonexistent", testOrg)
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
	if !apperrors.IsNotFound(err) {
		t.Errorf("expected 404 NotFound, got status %d", appErrStatus(t, err))
	}
}

func TestRoleService_GetWithPermissions_WhenOtherOrg_ReturnsForbidden(t *testing.T) {
	repo := &fakeRoleRepo{
		roles: []*role.Role{newOrgRole("r1", "Custom", "org-other")},
	}
	svc := rolesvc.NewService(repo)

	_, err := svc.GetWithPermissions(context.Background(), "r1", testOrg)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if appErrStatus(t, err) != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", appErrStatus(t, err))
	}
}
