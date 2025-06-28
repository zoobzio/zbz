package cereal

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Validator interface for pluggable validation
type Validator interface {
	Validate(s interface{}) error
}

// DefaultValidator using go-playground/validator
type DefaultValidator struct {
	validate *validator.Validate
}

// NewDefaultValidator creates a new validator instance
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		validate: validator.New(),
	}
}

// Validate validates a struct using struct tags
func (v *DefaultValidator) Validate(s interface{}) error {
	return v.validate.Struct(s)
}

// Global validator instance
var globalValidator Validator = NewDefaultValidator()

// SetValidator allows setting a custom validator implementation
func SetValidator(v Validator) {
	globalValidator = v
}

// Validate validates a struct using the global validator
func Validate(s interface{}) error {
	if globalValidator == nil {
		return nil // No validation if no validator set
	}

	// Ensure built-in validators are registered
	ensureBuiltinValidators()

	// Only validate structs or pointers to structs
	rv := reflect.ValueOf(s)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	
	if rv.Kind() != reflect.Struct {
		return nil // Skip non-struct types
	}

	// Run struct tag validation (custom validators handle their own business logic)
	return globalValidator.Validate(s)
}

// ValidationError wraps validation errors with context
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s' with tag '%s': %s", e.Field, e.Tag, e.Message)
}

// FormatValidationErrors converts validator errors to our format
func FormatValidationErrors(err error) []ValidationError {
	var errors []ValidationError
	
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			errors = append(errors, ValidationError{
				Field:   e.Field(),
				Value:   fmt.Sprintf("%v", e.Value()),
				Tag:     e.Tag(),
				Message: getErrorMessage(e),
			})
		}
	}
	
	return errors
}

// getErrorMessage provides human-readable error messages
func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters long", e.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters long", e.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", e.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", e.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", e.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", e.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", e.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", e.Param())
	case "url":
		return "must be a valid URL"
	case "uri":
		return "must be a valid URI"
	case "alpha":
		return "must contain only alphabetic characters"
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "numeric":
		return "must be a valid number"
	case "uuid":
		return "must be a valid UUID"
	case "json":
		return "must be valid JSON"
	default:
		return fmt.Sprintf("validation failed for tag '%s'", e.Tag())
	}
}