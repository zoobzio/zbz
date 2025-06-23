package zbz

import "github.com/gin-gonic/gin"

// Context provides a minimal framework-agnostic interface for ZBZ internal handlers
// Defines only the methods ZBZ internals actually need
type Context interface {
	// Request data
	Param(key string) string           // Path parameters (e.g., /user/{id})
	GetRawData() ([]byte, error)       // Request body
	
	// Response handling  
	Status(code int)                   // HTTP status code
	
	// Framework state (for middleware communication)
	MustGet(key string) any            // Get middleware-set values (like database)
	Set(key string, value any)         // Set error state for auto-error middleware
}

// zContext embeds gin.Context to satisfy our Context interface
// gin.Context's methods are promoted automatically
type zContext struct {
	*gin.Context
}