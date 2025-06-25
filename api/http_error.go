package zbz

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
	"zbz/zlog"
)

// ErrorResponse represents a standardized error response structure
type ErrorResponse struct {
	Message string         `json:"message" example:"The request payload is invalid"`
	Code    string         `json:"code" example:"BAD_REQUEST"`
	Details map[string]any `json:"details,omitempty"`
}

// ValidationErrorResponse represents a validation error with detailed field errors
type ValidationErrorResponse struct {
	Message string                   `json:"message"`
	Code    string                   `json:"code"`
	Fields  map[string]string        `json:"fields"`          // Field-specific error messages
	Details []ValidationError        `json:"details"`         // Detailed validation errors
}

// HTTPError represents an HTTP error with its metadata for documentation
type HTTPError struct {
	Status      int
	Name        string
	Description string
	Response    ErrorResponse
}


// Predefined HTTP errors with standard responses
var (
	// 400 Bad Request
	ErrBadRequest = &HTTPError{
		Status:      http.StatusBadRequest,
		Name:        "BadRequest",
		Description: "The request cannot be processed due to invalid syntax or missing required fields",
		Response: ErrorResponse{
			Message: "The request payload is invalid",
			Code:    http.StatusText(http.StatusBadRequest),
		},
	}

	// 401 Unauthorized
	ErrUnauthorized = &HTTPError{
		Status:      http.StatusUnauthorized,
		Name:        "Unauthorized",
		Description: "Authentication is required and has failed or has not been provided",
		Response: ErrorResponse{
			Message: "Authentication required",
			Code:    http.StatusText(http.StatusUnauthorized),
		},
	}

	// 403 Forbidden
	ErrForbidden = &HTTPError{
		Status:      http.StatusForbidden,
		Name:        "Forbidden",
		Description: "The request is valid but the server is refusing to respond to it",
		Response: ErrorResponse{
			Message: "You don't have permission to access this resource",
			Code:    http.StatusText(http.StatusForbidden),
		},
	}

	// 404 Not Found
	ErrNotFound = &HTTPError{
		Status:      http.StatusNotFound,
		Name:        "NotFound",
		Description: "The requested resource could not be found",
		Response: ErrorResponse{
			Message: "The requested resource was not found",
			Code:    http.StatusText(http.StatusNotFound),
		},
	}

	// 409 Conflict
	ErrConflict = &HTTPError{
		Status:      http.StatusConflict,
		Name:        "Conflict",
		Description: "The request conflicts with the current state of the resource",
		Response: ErrorResponse{
			Message: "The request conflicts with the current state of the resource",
			Code:    http.StatusText(http.StatusConflict),
		},
	}

	// 422 Unprocessable Entity (for validation errors)
	ErrValidationFailed = &HTTPError{
		Status:      http.StatusUnprocessableEntity,
		Name:        "ValidationFailed",
		Description: "The request data failed validation rules",
		Response: ErrorResponse{
			Message: "The request data is invalid",
			Code:    http.StatusText(http.StatusUnprocessableEntity),
		},
	}
	
	// 422 Unprocessable Entity (generic)
	ErrUnprocessableEntity = &HTTPError{
		Status:      http.StatusUnprocessableEntity,
		Name:        "UnprocessableEntity",
		Description: "The request is well-formed but contains semantic errors",
		Response: ErrorResponse{
			Message: "The request contains invalid data",
			Code:    http.StatusText(http.StatusUnprocessableEntity),
		},
	}

	// 500 Internal Server Error
	ErrInternalServer = &HTTPError{
		Status:      http.StatusInternalServerError,
		Name:        "InternalServerError",
		Description: "An unexpected error occurred on the server",
		Response: ErrorResponse{
			Message: "An unexpected error occurred",
			Code:    http.StatusText(http.StatusInternalServerError),
		},
	}

	// 503 Service Unavailable
	ErrServiceUnavailable = &HTTPError{
		Status:      http.StatusServiceUnavailable,
		Name:        "ServiceUnavailable",
		Description: "The server is currently unable to handle the request",
		Response: ErrorResponse{
			Message: "The service is temporarily unavailable",
			Code:    http.StatusText(http.StatusServiceUnavailable),
		},
	}
)

// ErrorManager is an interface for managing HTTP error responses
type ErrorManager interface {
	SetError(statusCode int, error *HTTPError)
	GetError(statusCode int) *HTTPError
	HasError(statusCode int) bool
}

// zErrorManager implements the ErrorManager interface
type zErrorManager struct {
	errors map[int]*HTTPError
}

// NewErrorManager creates a new ErrorManager with default HTTP errors
func NewErrorManager() ErrorManager {
	return &zErrorManager{
		errors: map[int]*HTTPError{
			http.StatusBadRequest:          ErrBadRequest,
			http.StatusUnauthorized:        ErrUnauthorized,
			http.StatusForbidden:           ErrForbidden,
			http.StatusNotFound:            ErrNotFound,
			http.StatusConflict:            ErrConflict,
			http.StatusUnprocessableEntity: ErrValidationFailed, // Default to validation error for 422
			http.StatusInternalServerError: ErrInternalServer,
			http.StatusServiceUnavailable:  ErrServiceUnavailable,
		},
	}
}

// SetError sets a custom error response for a given status code
func (e *zErrorManager) SetError(statusCode int, error *HTTPError) {
	e.errors[statusCode] = error
}

// GetError retrieves the error response for a given status code
func (e *zErrorManager) GetError(statusCode int) *HTTPError {
	return e.errors[statusCode]
}

// HasError checks if an error response exists for a given status code
func (e *zErrorManager) HasError(statusCode int) bool {
	_, exists := e.errors[statusCode]
	return exists
}



// GetErrorSchema returns the OpenAPI schema for an error response using an ErrorManager
func GetErrorSchema(errorManager ErrorManager, statusCode int) *OpenAPISchema {
	err := errorManager.GetError(statusCode)
	if err == nil {
		return nil
	}

	schema := &OpenAPISchema{
		Type: "object",
		Properties: map[string]*OpenAPISchema{
			"message": {
				Type:        "string",
				Description: "A human-readable error message",
				Example:     err.Response.Message,
			},
			"code": {
				Type:        "string",
				Description: "A machine-readable error code",
				Example:     err.Response.Code,
			},
		},
		Required: []string{"message", "code"},
	}

	// Add details field for validation errors
	if statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity {
		schema.Properties["details"] = &OpenAPISchema{
			Type:        "object",
			Description: "Additional error details, such as field-specific validation errors",
			AdditionalProperties: &OpenAPISchema{
				Type: "string",
			},
		}
	}

	return schema
}

// RespondWithValidationError sends a detailed validation error response
func RespondWithValidationError(ctx *gin.Context, validationErrors ValidationErrors) {
	// Extract field errors for simplified access
	fieldErrors := make(map[string]string)
	for _, err := range validationErrors.Errors {
		fieldErrors[err.Field] = err.Message
	}
	
	response := ValidationErrorResponse{
		Message: "The request data contains validation errors",
		Code:    http.StatusText(http.StatusUnprocessableEntity),
		Fields:  fieldErrors,
		Details: validationErrors.Errors,
	}
	
	// Log validation failure with details
	zlog.Zlog.Info("Request validation failed",
		zlog.Any("validation_errors", validationErrors),
		zlog.Any("field_errors", fieldErrors))
	
	ctx.JSON(http.StatusUnprocessableEntity, response)
}

// RespondWithValidationFieldError sends a validation error response for specific fields
func RespondWithValidationFieldError(ctx *gin.Context, fieldErrors map[string]string) {
	// Convert field errors to ValidationError slice for consistency
	var validationErrors []ValidationError
	for field, message := range fieldErrors {
		validationErrors = append(validationErrors, ValidationError{
			Field:   field,
			Message: message,
			Rule:    "custom",
		})
	}
	
	response := ValidationErrorResponse{
		Message: "The request data contains validation errors",
		Code:    http.StatusText(http.StatusUnprocessableEntity),
		Fields:  fieldErrors,
		Details: validationErrors,
	}
	
	zlog.Zlog.Info("Request validation failed",
		zlog.Any("field_errors", fieldErrors))
	
	ctx.JSON(http.StatusUnprocessableEntity, response)
}