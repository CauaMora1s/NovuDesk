package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	rolesvc "github.com/novudesk/novudesk/internal/application/role"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	"github.com/novudesk/novudesk/pkg/validator"
)

type RoleHandler struct {
	roles *rolesvc.Service
}

func NewRoleHandler(roles *rolesvc.Service) *RoleHandler {
	return &RoleHandler{roles: roles}
}

// ListPermissions returns all available system permissions.
func (h *RoleHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.roles.ListAllPermissions(r.Context())
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, perms)
}

// List returns all roles (system + org) with their permissions.
func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	roles, err := h.roles.ListForOrg(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, roles)
}

// Get returns a single role with its permissions.
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	roleID := chi.URLParam(r, "id")

	role, err := h.roles.GetWithPermissions(r.Context(), roleID, claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, role)
}

type roleRequest struct {
	Name           string   `json:"name"            validate:"required,min=2,max=100"`
	PermissionKeys []string `json:"permission_keys" validate:"required"`
}

// Create creates a new custom role for the organization.
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req roleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	role, err := h.roles.Create(r.Context(), claims.OrgID, req.Name, req.PermissionKeys)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Created(w, role)
}

// Update updates the name and permissions of an existing custom org role.
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	roleID := chi.URLParam(r, "id")

	var req roleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	role, err := h.roles.Update(r.Context(), roleID, claims.OrgID, req.Name, req.PermissionKeys)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, role)
}

// Delete removes a custom org role.
func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	roleID := chi.URLParam(r, "id")

	if err := h.roles.Delete(r.Context(), roleID, claims.OrgID); err != nil {
		respond.Error(w, err)
		return
	}
	respond.NoContent(w)
}
