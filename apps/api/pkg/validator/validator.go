package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

var validate *validator.Validate

func init() {
	validate = validator.New()
	// Use JSON field names in error messages instead of Go struct names.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// Validate validates a struct and returns a slice of field errors.
// Returns nil if validation passes.
func Validate(v any) []FieldError {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}

	var errs validator.ValidationErrors
	if !isValidationErrors(err, &errs) {
		return []FieldError{{Field: "_", Message: err.Error()}}
	}

	out := make([]FieldError, 0, len(errs))
	for _, e := range errs {
		out = append(out, FieldError{
			Field:   e.Field(),
			Message: humanize(e),
		})
	}
	return out
}

func humanize(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", e.Param())
	case "uuid4":
		return "must be a valid UUID"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", e.Param())
	case "url":
		return "must be a valid URL"
	default:
		return fmt.Sprintf("failed validation: %s", e.Tag())
	}
}

func isValidationErrors(err error, target *validator.ValidationErrors) bool {
	if vErr, ok := err.(validator.ValidationErrors); ok {
		*target = vErr
		return true
	}
	return false
}
