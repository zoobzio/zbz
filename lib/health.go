package zbz

import (
	"github.com/gin-gonic/gin"
)

// Health is an interface for the health check functionality
type Health interface {
	HealthCheckHandler(c *gin.Context)
}

// ZbzHealth implements the Health interface
type ZbzHealth struct {
	config Config
	log    Logger
}

// NewHealth creates a new Health instance
func NewHealth(l Logger, c Config) Health {
	return &ZbzHealth{
		config: c,
		log:    l,
	}
}

// HealthCheckHandler handles the health check endpoint
func (h *ZbzHealth) HealthCheckHandler(c *gin.Context) {
	h.log.Info("Checking service integrity...")
	c.JSON(200, gin.H{
		"status":  "healthy",
		"message": "The service is running smoothly.",
	})
}
