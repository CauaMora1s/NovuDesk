package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	authsvc "github.com/novudesk/novudesk/internal/application/auth"
	ticketsvc "github.com/novudesk/novudesk/internal/application/ticket"
	teamdomain "github.com/novudesk/novudesk/internal/domain/team"
	"github.com/novudesk/novudesk/internal/domain/ticket"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
	"github.com/novudesk/novudesk/pkg/pagination"
	"github.com/novudesk/novudesk/pkg/validator"
)

type TicketHandler struct {
	svc       *ticketsvc.Service
	teamsRepo teamdomain.Repository
}

func NewTicketHandler(svc *ticketsvc.Service, teamsRepo teamdomain.Repository) *TicketHandler {
	return &TicketHandler{svc: svc, teamsRepo: teamsRepo}
}

// isPrivileged performs a real-time DB check to avoid stale JWT team_ids.
func (h *TicketHandler) isPrivileged(ctx context.Context, claims *authsvc.Claims) bool {
	if claims.RoleName == "owner" || claims.RoleName == "admin" {
		return true
	}
	teamIDs, err := h.teamsRepo.ListTeamIDsByUser(ctx, claims.UserID, claims.OrgID)
	if err != nil {
		return false
	}
	return len(teamIDs) > 0
}

type createTicketRequest struct {
	Title        string          `json:"title"         validate:"required,min=3,max=255"`
	Description  *string         `json:"description"`
	Priority     ticket.Priority `json:"priority"      validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID   *string         `json:"assignee_id"`
	TeamID       *string         `json:"team_id"`
	CategoryID   *string         `json:"category_id"`
	SLAPolicyID  *string         `json:"sla_policy_id"`
	Tags         []string        `json:"tags"`
	CustomFields json.RawMessage `json:"custom_fields"`
}

func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}

	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = ticket.PriorityNormal
	}

	t, err := h.svc.Create(r.Context(), ticket.CreateInput{
		OrgID:        claims.OrgID,
		Title:        req.Title,
		Description:  req.Description,
		Priority:     priority,
		AssigneeID:   req.AssigneeID,
		TeamID:       req.TeamID,
		CategoryID:   req.CategoryID,
		SLAPolicyID:  req.SLAPolicyID,
		Tags:         req.Tags,
		CustomFields: req.CustomFields,
		CreatedBy:    claims.UserID,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Created(w, t)
}

func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	t, err := h.svc.Get(r.Context(), id, claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	// Non-privileged users (no team, not admin/owner) may only view their own tickets.
	isPrivileged := h.isPrivileged(r.Context(), claims)
	if !isPrivileged {
		if t.RequesterID == nil || *t.RequesterID != claims.UserID {
			respond.Error(w, apperrors.NotFound(apperrors.CodeTicketNotFound, "ticket not found"))
			return
		}
	}

	respond.Ok(w, t)
}

func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	pg := pagination.ParseRequest(r)
	q := r.URL.Query()

	filter := ticket.Filter{
		Query: q.Get("q"),
	}

	if s := q.Get("status"); s != "" {
		filter.Status = []ticket.Status{ticket.Status(s)}
	}
	if p := q.Get("priority"); p != "" {
		filter.Priority = []ticket.Priority{ticket.Priority(p)}
	}
	if a := q.Get("assignee_id"); a != "" {
		filter.AssigneeID = &a
	}
	if t := q.Get("team_id"); t != "" {
		filter.TeamID = &t
	}
	if cat := q.Get("category_id"); cat != "" {
		filter.CategoryID = &cat
	}

	// Users with no teams can only see their own tickets.
	isAgent := h.isPrivileged(r.Context(), claims)
	if !isAgent {
		uid := claims.UserID
		filter.RequesterID = &uid
	}

	tickets, total, err := h.svc.List(r.Context(), claims.OrgID, filter, pg.PerPage, pg.Offset())
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, tickets, pagination.Meta{
		Total:   total,
		PerPage: pg.PerPage,
	})
}

type updateTicketRequest struct {
	Title        *string          `json:"title"       validate:"omitempty,min=3,max=255"`
	Description  *string          `json:"description"`
	Status       *ticket.Status   `json:"status"      validate:"omitempty,oneof=open pending on_hold resolved closed"`
	Priority     *ticket.Priority `json:"priority"    validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID   *string          `json:"assignee_id"`
	TeamID       *string          `json:"team_id"`
	CategoryID   *string          `json:"category_id"`
	Tags         []string         `json:"tags"`
	CustomFields json.RawMessage  `json:"custom_fields"`
}

func (h *TicketHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req updateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}

	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	// Non-privileged users may only update their own tickets and cannot change status or assignee.
	isPrivileged := h.isPrivileged(r.Context(), claims)
	if !isPrivileged {
		if req.Status != nil || req.AssigneeID != nil {
			respond.Forbidden(w, "only team members can change status or assignee")
			return
		}
		existing, err := h.svc.Get(r.Context(), id, claims.OrgID)
		if err != nil || existing == nil || existing.RequesterID == nil || *existing.RequesterID != claims.UserID {
			respond.Forbidden(w, "insufficient permissions")
			return
		}
	}

	t, err := h.svc.Update(r.Context(), id, claims.OrgID, claims.UserID, ticket.UpdateInput{
		Title:        req.Title,
		Description:  req.Description,
		Status:       req.Status,
		Priority:     req.Priority,
		AssigneeID:   req.AssigneeID,
		TeamID:       req.TeamID,
		CategoryID:   req.CategoryID,
		Tags:         req.Tags,
		CustomFields: req.CustomFields,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, t)
}

func (h *TicketHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id, claims.OrgID, claims.UserID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}
