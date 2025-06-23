package zbz

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler provides HTTP handling for Core business logic
// Abstracts all HTTP concerns (status codes, error handling, serialization)
// while delegating business logic to Core
type Handler[T BaseModel] struct {
	core Core
}

// NewHandler creates a new Handler instance wrapping a Core
func NewHandler[T BaseModel](core Core) *Handler[T] {
	return &Handler[T]{
		core: core,
	}
}

// CreateHandler handles HTTP POST requests for creating new records
func (h *Handler[T]) CreateHandler(ctx *gin.Context) {
	result, err := h.core.Create(ctx)
	if err != nil {
		h.handleError(ctx, err)
		return
	}
	
	// Success - respond with created resource
	h.respondWithScopedJSON(ctx, http.StatusCreated, result)
}

// ReadHandler handles HTTP GET requests for retrieving records by ID
func (h *Handler[T]) ReadHandler(ctx *gin.Context) {
	result, err := h.core.Read(ctx)
	if err != nil {
		h.handleError(ctx, err)
		return
	}
	
	// Success - respond with resource
	h.respondWithScopedJSON(ctx, http.StatusOK, result)
}

// UpdateHandler handles HTTP PUT requests for updating records by ID
func (h *Handler[T]) UpdateHandler(ctx *gin.Context) {
	result, err := h.core.Update(ctx)
	if err != nil {
		h.handleError(ctx, err)
		return
	}
	
	// Success - respond with updated resource
	h.respondWithScopedJSON(ctx, http.StatusOK, result)
}

// DeleteHandler handles HTTP DELETE requests for removing records by ID
func (h *Handler[T]) DeleteHandler(ctx *gin.Context) {
	err := h.core.Delete(ctx)
	if err != nil {
		h.handleError(ctx, err)
		return
	}
	
	// Success - no content response
	ctx.Status(http.StatusNoContent)
}

// handleError centralizes error handling and HTTP response logic
func (h *Handler[T]) handleError(ctx *gin.Context, err error) {
	Log.Error("Handler operation failed", zap.Error(err))
	
	// Handle different error types
	switch {
	case isValidationError(err):
		validationErrors := extractValidationErrors(err)
		ctx.Set("error_details", validationErrors)
		ctx.Set("error_message", "Validation failed")
		ctx.Status(http.StatusBadRequest)
		
	case isNotFoundError(err):
		ctx.Set("error_message", "Resource not found")
		ctx.Status(http.StatusNotFound)
		
	case isConflictError(err):
		ctx.Set("error_message", "Resource already exists or conflicts with existing data")
		ctx.Status(http.StatusConflict)
		
	case isForbiddenError(err):
		ctx.Set("error_message", err.Error())
		ctx.Status(http.StatusForbidden)
		
	case isBadRequestError(err):
		ctx.Set("error_message", err.Error())
		ctx.Status(http.StatusBadRequest)
		
	default:
		// Generic server error
		ctx.Set("error_message", "An unexpected error occurred")
		ctx.Status(http.StatusInternalServerError)
	}
}

// Error type checking functions
// These can be expanded to check specific error types/interfaces

func isValidationError(err error) bool {
	_, ok := err.(ValidationErrors)
	return ok
}

func isNotFoundError(err error) bool {
	// Could check for specific "not found" error types
	return false // TODO: implement based on your error types
}

func isConflictError(err error) bool {
	// Check for duplicate key errors, etc.
	return false // TODO: implement based on your error types  
}

func isForbiddenError(err error) bool {
	// Check for permission/scope errors
	return false // TODO: implement based on your error types
}

func isBadRequestError(err error) bool {
	// Check for malformed request errors
	return false // TODO: implement based on your error types
}

func extractValidationErrors(err error) map[string]string {
	if validationErrors, ok := err.(ValidationErrors); ok {
		result := make(map[string]string)
		for _, e := range validationErrors.Errors {
			result[e.Field] = e.Message
		}
		return result
	}
	return map[string]string{"_error": err.Error()}
}

// respondWithScopedJSON sends a JSON response with field-level scoping
// Moved from cereal.go to belong to Handler (HTTP response concerns)
func (h *Handler[T]) respondWithScopedJSON(ctx *gin.Context, status int, data any) {
	jsonData, err := SerializeWithScopes(ctx, data, FormatJSON)
	if err != nil {
		Log.Error("Failed to serialize response with scopes", zap.Error(err))
		ctx.Set("error_message", "Failed to serialize response")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Data(status, "application/json", jsonData)
}