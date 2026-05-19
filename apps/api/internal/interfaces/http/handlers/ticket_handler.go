package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	ticketsvc "github.com/novudesk/novudesk/internal/application/ticket"
	"github.com/novudesk/novudesk/internal/domain/ticket"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	"github.com/novudesk/novudesk/pkg/pagination"
	"github.com/novudesk/novudesk/pkg/validator"
)

type TicketHandler struct {
	svc *ticketsvc.Service
}

func NewTicketHandler(svc *ticketsvc.Service) *TicketHandler {
	return &TicketHandler{svc: svc}
}

type createTicketRequest struct {
	Title        string          `json:"title"         validate:"required,min=3,max=255"`
	Description  *string         `json:"description"`
	Priority     ticket.Priority `json:"priority"      validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID   *string         `json:"assignee_id"`
	TeamID       *string         `json:"team_id"`
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
	Title        *string          `json:"title"      validate:"omitempty,min=3,max=255"`
	Description  *string          `json:"description"`
	Status       *ticket.Status   `json:"status"     validate:"omitempty,oneof=open pending on_hold resolved closed"`
	Priority     *ticket.Priority `json:"priority"   validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID   *string          `json:"assignee_id"`
	TeamID       *string          `json:"team_id"`
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

	t, err := h.svc.Update(r.Context(), id, claims.OrgID, claims.UserID, ticket.UpdateInput{
		Title:        req.Title,
		Description:  req.Description,
		Status:       req.Status,
		Priority:     req.Priority,
		AssigneeID:   req.AssigneeID,
		TeamID:       req.TeamID,
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
