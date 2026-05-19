package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	teamsvc "github.com/novudesk/novudesk/internal/application/team"
	usersvc "github.com/novudesk/novudesk/internal/application/user"
	"github.com/novudesk/novudesk/internal/domain/role"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	"github.com/novudesk/novudesk/pkg/pagination"
	"github.com/novudesk/novudesk/pkg/validator"
)

type MemberHandler struct {
	users *usersvc.Service
	teams *teamsvc.Service
	roles role.Repository
}

func NewMemberHandler(users *usersvc.Service, teams *teamsvc.Service, roles role.Repository) *MemberHandler {
	return &MemberHandler{users: users, teams: teams, roles: roles}
}

// List returns all members of the organization.
func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	pg := pagination.ParseRequest(r)

	members, total, err := h.users.ListMembers(r.Context(), claims.OrgID, pg.PerPage, pg.Offset())
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, members, pagination.Meta{Total: total, PerPage: pg.PerPage})
}

type createMemberRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,min=8"`
	RoleID   string `json:"role_id"   validate:"required"`
	TeamID   string `json:"team_id"`
}

// Create creates a new user and adds them to the organization.
func (h *MemberHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req createMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	u, err := h.users.Register(r.Context(), usersvc.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Locale:   "pt",
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	if err := h.users.AcceptInvite(r.Context(), u.ID, claims.OrgID, req.RoleID); err != nil {
		respond.Error(w, err)
		return
	}

	if req.TeamID != "" {
		if err := h.teams.AddMember(r.Context(), req.TeamID, u.ID); err != nil {
			respond.Error(w, err)
			return
		}
	}

	member, err := h.users.GetMember(r.Context(), u.ID, claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Created(w, member)
}

type updateMemberRequest struct {
	RoleID string `json:"role_id" validate:"required"`
}

// UpdateRole changes a member's role within the organization.
func (h *MemberHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	userID := chi.URLParam(r, "id")

	var req updateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	if err := h.users.UpdateMemberRole(r.Context(), userID, claims.OrgID, req.RoleID); err != nil {
		respond.Error(w, err)
		return
	}

	member, err := h.users.GetMember(r.Context(), userID, claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, member)
}

// Deactivate disables a member's access to the organization.
func (h *MemberHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	userID := chi.URLParam(r, "id")

	if err := h.users.Deactivate(r.Context(), userID, claims.OrgID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}

// ListRoles returns all roles available for the org (used by member creation forms).
func (h *MemberHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	roles, err := h.roles.ListByOrg(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, roles)
}
