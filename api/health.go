package zbz

import (
	"net/http"
	"zbz/zlog"
)

// Health is an interface for the health check functionality
type Health interface {
	// Framework-agnostic health check handler
	HealthCheckHandler(ctx RequestContext)
	
	// Handler contract for engine to collect
	HealthCheckContract() *HandlerContract
}

// zHealth implements the Health interface
type zHealth struct{}

// NewHealth creates a new Health instance
func NewHealth() Health {
	return &zHealth{}
}

// HealthCheckHandler handles the health check endpoint
func (h *zHealth) HealthCheckHandler(ctx RequestContext) {
	zlog.Zlog.Info("Checking service integrity...")
	ctx.Status(http.StatusOK)
	ctx.JSON(map[string]any{
		"status":  "healthy",
		"message": "The service is running smoothly.",
	})
}

// HealthCheckContract returns the handler contract for health check endpoint
func (h *zHealth) HealthCheckContract() *HandlerContract {
	return &HandlerContract{
		Name:        "Health Check",
		Description: "System health status",
		Method:      "GET",
		Path:        "/health",
		Handler:     h.HealthCheckHandler,
		Auth:        false,
	}
}
