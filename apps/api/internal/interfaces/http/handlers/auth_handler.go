package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	authsvc "github.com/novudesk/novudesk/internal/application/auth"
	usersvc "github.com/novudesk/novudesk/internal/application/user"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
	"github.com/novudesk/novudesk/pkg/validator"
)

type AuthHandler struct {
	authSvc *authsvc.Service
	userSvc *usersvc.Service
}

func NewAuthHandler(authSvc *authsvc.Service, userSvc *usersvc.Service) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userSvc: userSvc}
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
	OrgSlug  string `json:"org_slug" validate:"required"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	// The org_slug is resolved to an org_id by the auth service.
	accessToken, refreshToken, err := h.authSvc.Login(r.Context(), req.Email, req.Password, req.OrgSlug)
	if err != nil {
		respond.Error(w, err)
		return
	}

	// Refresh token delivered as HTTP-only cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	respond.Ok(w, loginResponse{
		AccessToken: accessToken,
		ExpiresIn:   15 * 60,
	})
}

type registerRequest struct {
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=2"`
	Locale   string `json:"locale"    validate:"omitempty,oneof=pt en"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, err)
		return
	}
	if errs := validator.Validate(req); errs != nil {
		respond.ValidationError(w, errs)
		return
	}

	u, err := h.userSvc.Register(r.Context(), usersvc.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Locale:   req.Locale,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Created(w, map[string]string{
		"id":    u.ID,
		"email": u.Email,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		MaxAge:   -1,
	})
	respond.NoContent(w)
}
