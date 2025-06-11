package zbz

import (
	"regexp"

	"github.com/gin-gonic/gin"
)

// HTTP is an interface for the HTTP server
type HTTP interface {
	Use(middlewares ...gin.HandlerFunc) gin.IRoutes

	GET(path string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(path string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(path string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(path string, handlers ...gin.HandlerFunc) gin.IRoutes

	AddRoute(operation *HTTPOperation) gin.IRoutes
	Serve() error
}

// HTTPResponse represents a response structure for an HTTP operation
type HTTPResponse struct {
	Status int
	Ref    string
	Type   string
	Errors []int
}

type HTTPPathParameter struct {
	Name        string
	Description string
	Required    bool
}

type HTTPQueryParameter struct {
	Name        string
	Description string
	Required    bool
}

type HTTPRequestBody struct {
	Description string
	Required    bool
}

// HTTPOperation represents a given API action that can be used to register endpoints & spawn documentation
type HTTPOperation struct {
	Name        string
	Description string
	Tag         string
	Method      string
	Path        string
	Handler     gin.HandlerFunc
	Parameters  []string
	Query       []string
	RequestBody string
	Response    *HTTPResponse
	Auth        bool
}

// ZbzHTTP is responsible for setting up the HTTP router
type ZbzHTTP struct {
	*gin.Engine

	auth   Auth
	config Config
	log    Logger
}

// NewHTTP creates a new HTTP instance
func NewHTTP(l Logger, c Config, a Auth) HTTP {
	gin.SetMode(gin.ReleaseMode) // disable GIN debug logs
	e := gin.New()

	e.Use(gin.Recovery())
	e.Use(l.Middleware)

	e.LoadHTMLGlob("lib/templates/*")

	return &ZbzHTTP{
		Engine: e,
		auth:   a,
		config: c,
		log:    l,
	}
}

// Register an HTTP operation with the Gin router
func (h *ZbzHTTP) AddRoute(operation *HTTPOperation) gin.IRoutes {
	h.log.Debugf("[%s] %s", operation.Method, operation.Path)

	path := operation.Path
	re := regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)
	path = re.ReplaceAllString(path, `:$1`)

	var route gin.IRoutes
	switch operation.Method {
	case "GET":
		route = h.GET(path, operation.Handler)
	case "POST":
		route = h.POST(path, operation.Handler)
	case "PUT":
		route = h.PUT(path, operation.Handler)
	case "DELETE":
		route = h.DELETE(path, operation.Handler)
	default:
		h.log.Errorf("Unsupported HTTP method: %s", operation.Method)
		return nil
	}

	return route
}

// Serve the HTTP router on the configured port
func (h *ZbzHTTP) Serve() error {
	h.log.Infof("Starting HTTP server on port %s", h.config.Port())
	return h.Run(":" + h.config.Port())
}
