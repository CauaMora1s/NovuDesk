package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	billingsvc "github.com/novudesk/novudesk/internal/application/billing"
	orgsvc "github.com/novudesk/novudesk/internal/application/organization"
	domainbilling "github.com/novudesk/novudesk/internal/domain/billing"
	"github.com/novudesk/novudesk/internal/domain/organization"
	"github.com/novudesk/novudesk/internal/domain/plan"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	"github.com/novudesk/novudesk/pkg/validator"
)

type OrganizationHandler struct {
	orgs    *orgsvc.Service
	billing *billingsvc.Service
}

func NewOrganizationHandler(orgs *orgsvc.Service, billing *billingsvc.Service) *OrganizationHandler {
	return &OrganizationHandler{orgs: orgs, billing: billing}
}

// overviewResponse is the full payload for the organization settings page.
type overviewResponse struct {
	Organization   *organization.Organization `json:"organization"`
	Plan           plan.Plan                  `json:"plan"`
	Usage          *organization.Usage        `json:"usage"`
	PendingSession *domainbilling.PaymentSession `json:"pending_session"`
}

// Get returns the organization overview: info, current plan, usage, pending session.
func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	overview, err := h.orgs.GetOverview(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	pending, err := h.billing.ActivePending(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, overviewResponse{
		Organization:   overview.Organization,
		Plan:           overview.Plan,
		Usage:          overview.Usage,
		PendingSession: pending,
	})
}

type updateOrgRequest struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
}

// Update renames the organization.
func (h *OrganizationHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	org, err := h.orgs.Rename(r.Context(), claims.OrgID, req.Name)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, org)
}

// ListPlans returns the available plan catalog.
func (h *OrganizationHandler) ListPlans(w http.ResponseWriter, _ *http.Request) {
	respond.Ok(w, plan.Catalog())
}

type initiatePlanRequest struct {
	ToTier string `json:"to_tier" validate:"required"`
}

// InitiatePlanChange creates a pending payment session for a plan change.
func (h *OrganizationHandler) InitiatePlanChange(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req initiatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	session, err := h.billing.InitiateChange(r.Context(), claims.OrgID, claims.UserID, req.ToTier)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Created(w, session)
}

// ConfirmPlanChange completes a pending session and applies the plan change.
func (h *OrganizationHandler) ConfirmPlanChange(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	sessionID := chi.URLParam(r, "id")

	session, err := h.billing.Confirm(r.Context(), claims.OrgID, sessionID)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, session)
}

// CancelPlanChange cancels a pending session without changing the plan.
func (h *OrganizationHandler) CancelPlanChange(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	sessionID := chi.URLParam(r, "id")

	if err := h.billing.Cancel(r.Context(), claims.OrgID, sessionID); err != nil {
		respond.Error(w, err)
		return
	}
	respond.NoContent(w)
}

// ListSessions returns the organization's billing history.
func (h *OrganizationHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	sessions, err := h.billing.History(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, sessions)
}
