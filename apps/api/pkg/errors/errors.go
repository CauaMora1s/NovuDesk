package errors

import (
	"errors"
	"net/http"
)

// Code represents a machine-readable error code.
type Code string

const (
	CodeNotFound           Code = "NOT_FOUND"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeConflict           Code = "CONFLICT"
	CodeValidation         Code = "VALIDATION_ERROR"
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeBadRequest         Code = "BAD_REQUEST"
	CodeTooManyRequests    Code = "TOO_MANY_REQUESTS"
	CodeUnprocessable      Code = "UNPROCESSABLE_ENTITY"

	// Domain-specific codes
	CodeTicketNotFound     Code = "TICKET_NOT_FOUND"
	CodeUserNotFound       Code = "USER_NOT_FOUND"
	CodeOrgNotFound        Code = "ORGANIZATION_NOT_FOUND"
	CodeTeamNotFound       Code = "TEAM_NOT_FOUND"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	CodeTokenExpired       Code = "TOKEN_EXPIRED"
	CodeTokenInvalid       Code = "TOKEN_INVALID"
	CodeInviteExpired      Code = "INVITE_EXPIRED"
	CodeInviteNotFound     Code = "INVITE_NOT_FOUND"
	CodeEmailTaken         Code = "EMAIL_ALREADY_TAKEN"
	CodeSlugTaken          Code = "SLUG_ALREADY_TAKEN"
	CodeSLANotFound        Code = "SLA_POLICY_NOT_FOUND"
	CodeQuotaExceeded      Code = "QUOTA_EXCEEDED"
	CodeInvalidPlan        Code = "INVALID_PLAN"
	CodeSessionNotFound    Code = "PAYMENT_SESSION_NOT_FOUND"
)

// AppError is a typed application error that carries an HTTP status and error code.
type AppError struct {
	Code       Code
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError.
func New(code Code, message string, httpStatus int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus}
}

// Wrap wraps an underlying error with an AppError.
func Wrap(code Code, message string, httpStatus int, err error) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus, Err: err}
}

// Predefined constructors for common cases.

func NotFound(code Code, message string) *AppError {
	return New(code, message, http.StatusNotFound)
}

func Unauthorized(message string) *AppError {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

func Forbidden(message string) *AppError {
	return New(CodeForbidden, message, http.StatusForbidden)
}

func Conflict(code Code, message string) *AppError {
	return New(code, message, http.StatusConflict)
}

func BadRequest(message string) *AppError {
	return New(CodeBadRequest, message, http.StatusBadRequest)
}

func Validation(message string) *AppError {
	return New(CodeValidation, message, http.StatusUnprocessableEntity)
}

func Internal(err error) *AppError {
	return Wrap(CodeInternal, "an internal error occurred", http.StatusInternalServerError, err)
}

func TooManyRequests() *AppError {
	return New(CodeTooManyRequests, "rate limit exceeded, please slow down", http.StatusTooManyRequests)
}

// As checks whether an error is an AppError and writes it to target.
func As(err error, target **AppError) bool {
	return errors.As(err, target)
}

// IsNotFound returns true if the error is a 404 AppError.
func IsNotFound(err error) bool {
	var e *AppError
	return errors.As(err, &e) && e.HTTPStatus == http.StatusNotFound
}
