package zbz

import (
	"github.com/gin-gonic/gin"
)

// HTTP is an interface for the HTTP server
type HTTP interface {
	Register(operation *HTTPOperation) gin.IRoutes
	Serve() error
}

// HTTPResponse represents a response structure for an HTTP operation
type HTTPResponse struct {
	Status int
	Ref    string
	List   bool
	Errors []int
}

// HTTPOperation represents a given API action that can be used to register endpoints & spawn documentation
type HTTPOperation struct {
	Name        string
	Summary     string
	Description string
	Tag         string
	Method      string
	Path        string
	Handler     gin.HandlerFunc
	Auth        bool
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

// Register an HTTP operation with the Gin router
func (h *ZbzHTTP) Register(operation *HTTPOperation) gin.IRoutes {
	h.log.Infof("[%s] %s", operation.Method, operation.Path)
	switch operation.Method {
	case "GET":
		return h.GET(operation.Path, operation.Handler)
	case "POST":
		return h.POST(operation.Path, operation.Handler)
	case "PUT":
		return h.PUT(operation.Path, operation.Handler)
	case "DELETE":
		return h.DELETE(operation.Path, operation.Handler)
	default:
		h.log.Errorf("Unsupported HTTP method: %s", operation.Method)
		return nil
	}
}

// Serve the HTTP router on the configured port
func (h *ZbzHTTP) Serve() error {
	return h.Run(":" + h.config.Port())
}
