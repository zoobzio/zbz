package zbz

import (
	"github.com/gin-gonic/gin"
)

type Health interface {
	HealthCheckHandler(c *gin.Context)
}

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
