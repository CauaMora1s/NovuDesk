package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	slasvc "github.com/novudesk/novudesk/internal/application/sla"
	"github.com/novudesk/novudesk/internal/domain/sla"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	"github.com/novudesk/novudesk/pkg/validator"
)

type SLAHandler struct {
	svc *slasvc.Service
}

func NewSLAHandler(svc *slasvc.Service) *SLAHandler {
	return &SLAHandler{svc: svc}
}

func (h *SLAHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	stats, err := h.svc.ListWithCategoryStats(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, stats)
}

type upsertSLARequest struct {
	ResolutionValue int    `json:"resolution_value" validate:"required,min=1,max=9999"`
	ResolutionUnit  string `json:"resolution_unit"  validate:"required,oneof=hours days weeks"`
	ResponseHours   int    `json:"response_hours"   validate:"omitempty,min=0"`
}

func (h *SLAHandler) UpsertForCategory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	categoryID := chi.URLParam(r, "categoryId")

	var req upsertSLARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	p, err := h.svc.UpsertForCategory(r.Context(), claims.OrgID, categoryID, sla.CreateInput{
		ResolutionValue: req.ResolutionValue,
		ResolutionUnit:  req.ResolutionUnit,
		ResponseHours:   req.ResponseHours,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, p)
}

func (h *SLAHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id, claims.OrgID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}
