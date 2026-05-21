package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	catsvc "github.com/novudesk/novudesk/internal/application/category"
	teamsvc "github.com/novudesk/novudesk/internal/application/team"
	"github.com/novudesk/novudesk/internal/domain/team"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
	"github.com/novudesk/novudesk/pkg/validator"
)

type TeamHandler struct {
	teams      *teamsvc.Service
	categories *catsvc.Service
}

func NewTeamHandler(teams *teamsvc.Service, categories *catsvc.Service) *TeamHandler {
	return &TeamHandler{teams: teams, categories: categories}
}

func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	teams, err := h.teams.List(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	if claims.RoleName == "owner" || claims.RoleName == "admin" {
		respond.Ok(w, teams)
		return
	}

	// Non-admin: only return teams the user belongs to (per JWT claims).
	memberOf := make(map[string]bool, len(claims.TeamIDs))
	for _, id := range claims.TeamIDs {
		memberOf[id] = true
	}
	filtered := teams[:0]
	for _, t := range teams {
		if memberOf[t.ID] {
			filtered = append(filtered, t)
		}
	}
	respond.Ok(w, filtered)
}

type createTeamRequest struct {
	Name        string  `json:"name"        validate:"required,min=1,max=100"`
	Description *string `json:"description"`
}

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	t, err := h.teams.Create(r.Context(), claims.OrgID, req.Name, req.Description)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Created(w, t)
}

func (h *TeamHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	// Non-admin/owner may only access teams they belong to.
	if claims.RoleName != "owner" && claims.RoleName != "admin" {
		isMember := false
		for _, tid := range claims.TeamIDs {
			if tid == id {
				isMember = true
				break
			}
		}
		if !isMember {
			respond.Error(w, apperrors.NotFound(apperrors.Code("TEAM_NOT_FOUND"), "team not found"))
			return
		}
	}

	t, err := h.teams.Get(r.Context(), id, claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	members, _ := h.teams.ListMembers(r.Context(), id, claims.OrgID)
	cats, _ := h.categories.ListByTeam(r.Context(), id)

	respond.Ok(w, map[string]any{
		"team":       t,
		"members":    members,
		"categories": cats,
	})
}

func (h *TeamHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	t, err := h.teams.Update(r.Context(), id, claims.OrgID, team.UpdateInput{
		Name:        &req.Name,
		Description: req.Description,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, t)
}

func (h *TeamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.teams.Delete(r.Context(), id, claims.OrgID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}

type addMemberRequest struct {
	UserID string `json:"user_id" validate:"required"`
}

func (h *TeamHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	if err := h.teams.AddMember(r.Context(), id, req.UserID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}

func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	if err := h.teams.RemoveMember(r.Context(), id, userID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}

func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	// Non-admin/owner may only list members of teams they belong to.
	if claims.RoleName != "owner" && claims.RoleName != "admin" {
		isMember := false
		for _, tid := range claims.TeamIDs {
			if tid == id {
				isMember = true
				break
			}
		}
		if !isMember {
			respond.Forbidden(w, "access denied")
			return
		}
	}

	members, err := h.teams.ListMembers(r.Context(), id, claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, members)
}

type addCategoryRequest struct {
	CategoryID string `json:"category_id" validate:"required"`
}

func (h *TeamHandler) AddCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req addCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	if err := h.categories.AddToTeam(r.Context(), id, req.CategoryID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}

func (h *TeamHandler) RemoveCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	catID := chi.URLParam(r, "catId")

	if err := h.categories.RemoveFromTeam(r.Context(), id, catID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}

func (h *TeamHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	cats, err := h.categories.ListByTeam(r.Context(), id)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, cats)
}
