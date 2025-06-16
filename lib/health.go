package zbz

import (
	"github.com/gin-gonic/gin"
)

// Health is an interface for the health check functionality
type Health interface {
	HealthCheckHandler(c *gin.Context)
}

// zHealth implements the Health interface
type zHealth struct{}

// NewHealth creates a new Health instance
func NewHealth() Health {
	return &zHealth{}
}

// HealthCheckHandler handles the health check endpoint
func (h *zHealth) HealthCheckHandler(c *gin.Context) {
	Log.Info("Checking service integrity...")
	c.JSON(200, gin.H{
		"status":  "healthy",
		"message": "The service is running smoothly.",
	})
}
