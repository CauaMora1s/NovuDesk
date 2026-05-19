package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	catsvc "github.com/novudesk/novudesk/internal/application/category"
	"github.com/novudesk/novudesk/internal/domain/category"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	"github.com/novudesk/novudesk/pkg/validator"
)

type CategoryHandler struct {
	svc *catsvc.Service
}

func NewCategoryHandler(svc *catsvc.Service) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	cats, err := h.svc.List(r.Context(), claims.OrgID)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, cats)
}

type categoryRequest struct {
	Name        string  `json:"name"        validate:"required,min=1,max=100"`
	Description *string `json:"description"`
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	c, err := h.svc.Create(r.Context(), claims.OrgID, req.Name, req.Description)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Created(w, c)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	c, err := h.svc.Update(r.Context(), id, claims.OrgID, category.UpdateInput{
		Name:        &req.Name,
		Description: req.Description,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, c)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id, claims.OrgID); err != nil {
		respond.Error(w, err)
		return
	}

	respond.NoContent(w)
}
