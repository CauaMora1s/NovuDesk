package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	rolesvc "github.com/novudesk/novudesk/internal/application/role"
	"github.com/novudesk/novudesk/internal/domain/role"
	"github.com/novudesk/novudesk/internal/interfaces/http/handlers"
)

// --- fake role repository ---

type fakeRoleRepository struct {
	roles       []*role.Role
	permissions map[string][]*role.Permission
	err         error
}

func (f *fakeRoleRepository) Create(_ context.Context, r *role.Role) error {
	if f.err != nil {
		return f.err
	}
	f.roles = append(f.roles, r)
	return nil
}

func (f *fakeRoleRepository) FindByID(_ context.Context, id string) (*role.Role, error) {
	for _, r := range f.roles {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func (f *fakeRoleRepository) FindByName(_ context.Context, orgID, name string) (*role.Role, error) {
	for _, r := range f.roles {
		if r.Name == name && r.OrgID != nil && *r.OrgID == orgID {
			return r, nil
		}
	}
	return nil, nil
}

func (f *fakeRoleRepository) FindSystemRole(_ context.Context, name string) (*role.Role, error) {
	for _, r := range f.roles {
		if r.IsSystemRole && r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

func (f *fakeRoleRepository) ListByOrg(_ context.Context, orgID string) ([]*role.Role, error) {
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

func (f *fakeRoleRepository) Update(_ context.Context, r *role.Role) error {
	for i, existing := range f.roles {
		if existing.ID == r.ID {
			f.roles[i] = r
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakeRoleRepository) Delete(_ context.Context, id, _ string) error {
	for i, r := range f.roles {
		if r.ID == id {
			f.roles = append(f.roles[:i], f.roles[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeRoleRepository) SetPermissions(_ context.Context, roleID string, keys []string) error {
	if f.permissions == nil {
		f.permissions = map[string][]*role.Permission{}
	}
	perms := make([]*role.Permission, len(keys))
	for i, k := range keys {
		perms[i] = &role.Permission{ID: "p-" + k, Key: k, CreatedAt: time.Now()}
	}
	f.permissions[roleID] = perms
	return nil
}

func (f *fakeRoleRepository) GetPermissions(_ context.Context, roleID string) ([]*role.Permission, error) {
	if f.permissions == nil {
		return nil, nil
	}
	return f.permissions[roleID], nil
}

func (f *fakeRoleRepository) GetPermissionKeys(_ context.Context, roleID string) ([]string, error) {
	perms := f.permissions[roleID]
	keys := make([]string, len(perms))
	for i, p := range perms {
		keys[i] = p.Key
	}
	return keys, nil
}

func (f *fakeRoleRepository) ListAllPermissions(_ context.Context) ([]*role.Permission, error) {
	return nil, nil
}

// --- helpers ---

func rolePtr(s string) *string { return &s }

func newRoleHandler(repo role.Repository) *handlers.RoleHandler {
	return handlers.NewRoleHandler(rolesvc.NewService(repo))
}

// --- tests ---

func TestRoleHandler_List_ReturnsRolesForOrg(t *testing.T) {
	repo := &fakeRoleRepository{
		roles: []*role.Role{
			{ID: "r1", OrgID: rolePtr(testOrgID), Name: "Support", IsSystemRole: false, CreatedAt: time.Now()},
		},
	}
	h := newRoleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	env := decodeEnvelope(t, rec.Body)
	data, ok := env["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", env["data"])
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 role, got %d", len(data))
	}
}

func TestRoleHandler_Get_WhenNotFound_Returns404(t *testing.T) {
	repo := &fakeRoleRepository{}
	h := newRoleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/missing", nil)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestRoleHandler_Create_ValidInput_Returns201(t *testing.T) {
	repo := &fakeRoleRepository{}
	h := newRoleHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"name":            "Billing",
		"permission_keys": []string{"tickets:read"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestRoleHandler_Create_DuplicateName_Returns409(t *testing.T) {
	repo := &fakeRoleRepository{
		roles: []*role.Role{
			{ID: "r1", OrgID: rolePtr(testOrgID), Name: "Billing", IsSystemRole: false, CreatedAt: time.Now()},
		},
	}
	h := newRoleHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"name":            "Billing",
		"permission_keys": []string{"tickets:read"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())

	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestRoleHandler_Delete_SystemRole_Returns403(t *testing.T) {
	repo := &fakeRoleRepository{
		roles: []*role.Role{
			{ID: "sys-1", Name: role.RoleAdmin, IsSystemRole: true, CreatedAt: time.Now()},
		},
	}
	h := newRoleHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/sys-1", nil)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "sys-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRoleHandler_Delete_ValidRole_Returns204(t *testing.T) {
	repo := &fakeRoleRepository{
		roles: []*role.Role{
			{ID: "r1", OrgID: rolePtr(testOrgID), Name: "Custom", IsSystemRole: false, CreatedAt: time.Now()},
		},
	}
	h := newRoleHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/r1", nil)
	req = injectClaims(req, fakeClaims())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "r1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", rec.Code, rec.Body.String())
	}
}
