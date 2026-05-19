package respond

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/novudesk/novudesk/pkg/errors"
	"github.com/novudesk/novudesk/pkg/validator"
)

type envelope struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

type errBody struct {
	Error errDetail `json:"error"`
}

type errDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// JSON writes a 200 OK JSON response.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Ok writes a 200 response with data + optional meta.
func Ok(w http.ResponseWriter, data any, meta ...any) {
	env := envelope{Data: data}
	if len(meta) > 0 {
		env.Meta = meta[0]
	}
	JSON(w, http.StatusOK, env)
}

// Created writes a 201 response.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, envelope{Data: data})
}

// NoContent writes a 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes a structured error response.
func Error(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if apperrors.As(err, &appErr) {
		JSON(w, appErr.HTTPStatus, errBody{
			Error: errDetail{Code: string(appErr.Code), Message: appErr.Message},
		})
		return
	}

	JSON(w, http.StatusInternalServerError, errBody{
		Error: errDetail{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"},
	})
}

// ValidationError writes a 422 response with field-level errors.
func ValidationError(w http.ResponseWriter, errs []validator.FieldError) {
	JSON(w, http.StatusUnprocessableEntity, errBody{
		Error: errDetail{
			Code:    "VALIDATION_ERROR",
			Message: "request validation failed",
			Details: errs,
		},
	})
}

// Unauthorized writes a 401 response.
func Unauthorized(w http.ResponseWriter, msg string) {
	JSON(w, http.StatusUnauthorized, errBody{
		Error: errDetail{Code: "UNAUTHORIZED", Message: msg},
	})
}

// Forbidden writes a 403 response.
func Forbidden(w http.ResponseWriter, msg string) {
	JSON(w, http.StatusForbidden, errBody{
		Error: errDetail{Code: "FORBIDDEN", Message: msg},
	})
}
