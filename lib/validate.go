package zbz

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// ValidationRule represents a single validation rule with its parameters
type ValidationRule struct {
	Name   string
	Params []string
}

// ValidationError represents a validation failure for a specific field
type ValidationError struct {
	Field   string `json:"field"`
	Value   any    `json:"value,omitempty"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}

// ValidationErrors represents multiple validation failures
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// ParsedValidationRules represents validation rules in a structured format
// Used to pass parsed rules to Database and Docs for constraint generation
type ParsedValidationRules struct {
	Rules     []ValidationRule `json:"rules"`
	FieldType string          `json:"fieldType"`
	FieldName string          `json:"fieldName"`
}

func (ve ValidationErrors) Error() string {
	messages := make([]string, len(ve.Errors))
	for i, err := range ve.Errors {
		messages[i] = fmt.Sprintf("%s: %s", err.Field, err.Message)
	}
	return strings.Join(messages, "; ")
}

// Validate is an interface that defines methods for validating values
type Validate interface {
	// Core validation methods
	IsValidID(v any) error
	IsValid(v any) error

	// Error extraction and formatting
	ExtractErrors(err error) map[string]string

	// Rule parsing (used by other services to understand validation tags)
	ParseValidationRules(validate string) []ValidationRule
}

// zValidate implements the Validate interface using go-playground/validator
type zValidate struct {
	validator *validator.Validate
}

// IsValidID checks if the provided value is a valid UUID
func (v *zValidate) IsValidID(value any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	err := v.validator.Var(str, "uuid")
	if err != nil {
		return v.convertValidationErrors(err)
	}
	return nil
}

// IsValid checks if the provided struct is valid according to its validation tags
func (v *zValidate) IsValid(value any) error {
	err := v.validator.Struct(value)
	if err != nil {
		return v.convertValidationErrors(err)
	}
	return nil
}

// ExtractErrors converts validation errors into a map of field names to error messages
func (v *zValidate) ExtractErrors(err error) map[string]string {
	if err == nil {
		return nil
	}

	// Handle our custom ValidationErrors type
	if validationErrors, ok := err.(ValidationErrors); ok {
		result := make(map[string]string)
		for _, e := range validationErrors.Errors {
			result[e.Field] = e.Message
		}
		return result
	}

	// Handle validator.ValidationErrors from go-playground/validator
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		result := make(map[string]string)
		for _, e := range validationErrors {
			fieldName := v.getFieldName(e)
			result[fieldName] = v.getErrorMessage(e)
		}
		return result
	}

	// Handle other error types
	return map[string]string{"_error": err.Error()}
}

// Note: Global validator removed - each Core now has its own validator instance

// NewValidate creates a new Validate instance using go-playground/validator
func NewValidate() Validate {
	v := validator.New()
	
	// Use JSON tags for field names
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	return &zValidate{
		validator: v,
	}
}

// convertValidationErrors converts go-playground/validator errors to our format
func (v *zValidate) convertValidationErrors(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errors []ValidationError
		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   v.getFieldName(e),
				Value:   e.Value(),
				Rule:    e.Tag(),
				Message: v.getErrorMessage(e),
				Param:   e.Param(),
			})
		}
		
		Log.Debug("Validation failed",
			zap.Int("error_count", len(errors)),
			zap.Any("errors", errors))
		
		return ValidationErrors{Errors: errors}
	}
	return err
}

// getFieldName extracts the field name from validation error
func (v *zValidate) getFieldName(err validator.FieldError) string {
	// Get JSON field name if available
	field := err.Field()
	
	// Convert from PascalCase to snake_case for consistency
	if field != "" && field[0] >= 'A' && field[0] <= 'Z' {
		return toSnakeCase(field)
	}
	
	return field
}

// getErrorMessage generates user-friendly error messages
func (v *zValidate) getErrorMessage(err validator.FieldError) string {
	param := err.Param()
	
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "uuid", "uuid4":
		return "Invalid UUID format"
	case "url":
		return "Invalid URL format"
	case "min":
		if err.Type().Kind() == reflect.String {
			return fmt.Sprintf("Value must be at least %s characters long", param)
		}
		return fmt.Sprintf("Value must be at least %s", param)
	case "max":
		if err.Type().Kind() == reflect.String {
			return fmt.Sprintf("Value must be at most %s characters long", param)
		}
		return fmt.Sprintf("Value must be at most %s", param)
	case "len":
		return fmt.Sprintf("Value must be exactly %s characters long", param)
	case "gt":
		return fmt.Sprintf("Value must be greater than %s", param)
	case "gte":
		return fmt.Sprintf("Value must be greater than or equal to %s", param)
	case "lt":
		return fmt.Sprintf("Value must be less than %s", param)
	case "lte":
		return fmt.Sprintf("Value must be less than or equal to %s", param)
	case "oneof":
		return fmt.Sprintf("Value must be one of: %s", param)
	case "alpha":
		return "Value must contain only alphabetic characters"
	case "alphanum":
		return "Value must contain only alphanumeric characters"
	case "numeric":
		return "Value must contain only numeric characters"
	default:
		return fmt.Sprintf("Failed %s validation", err.Tag())
	}
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// ParseValidationRules parses validation rules from a struct tag
func (v *zValidate) ParseValidationRules(validate string) []ValidationRule {
	if validate == "" {
		return nil
	}

	var rules []ValidationRule
	parts := strings.Split(validate, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "=") {
			// Rule with parameters: min=5, oneof=red blue green
			keyValue := strings.SplitN(part, "=", 2)
			name := strings.TrimSpace(keyValue[0])
			paramStr := strings.TrimSpace(keyValue[1])

			var params []string
			if paramStr != "" {
				// Handle space-separated params for oneof, or single param for others
				if name == "oneof" {
					params = strings.Fields(paramStr)
				} else {
					params = []string{paramStr}
				}
			}

			rules = append(rules, ValidationRule{
				Name:   name,
				Params: params,
			})
		} else {
			// Simple rule: required, email, uuid
			rules = append(rules, ValidationRule{
				Name:   part,
				Params: nil,
			})
		}
	}

	return rules
}

// Note: Global validator initialization removed - validators are created per-Core