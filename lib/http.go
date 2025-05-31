package zbz

import (
	"github.com/gin-gonic/gin"
)

// HTTP is an interface for the HTTP server
type HTTP interface {
	GET(path string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(path string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(path string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(path string, handlers ...gin.HandlerFunc) gin.IRoutes

	Serve() error
}

// ZbzHTTP is responsible for setting up the HTTP router
type ZbzHTTP struct {
	*gin.Engine
	config Config
	log    Logger
}

// NewHTTP creates a new HTTP instance
func NewHTTP(l Logger, c Config) HTTP {
	gin.SetMode(gin.ReleaseMode) // disable GIN debug logs
	e := gin.New()

	e.Use(gin.Recovery()) // recover from panics and log them
	e.Use(l.Middleware)   // logs requests and responses

	e.LoadHTMLGlob("lib/templates/*") // load HTML templates

	return &ZbzHTTP{
		Engine: e,
		config: c,
		log:    l,
	}
}

// Serve starts the HTTP router on the configured port
func (h *ZbzHTTP) Serve() error {
	return h.Run(":" + h.config.Port())
}
