package zbz

import (
	"regexp"

	"github.com/gin-gonic/gin"
	p "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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

// HTTPPathParameter represents a path parameter in an HTTP request
type HTTPPathParameter struct {
	Name        string
	Description string
	Required    bool
}

// HTTPHeaderParameter represents a header parameter in an HTTP request
type HTTPQueryParameter struct {
	Name        string
	Description string
	Required    bool
}

// HTTPQueryParameter represents a query parameter in an HTTP request
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

// zHTTP is responsible for setting up the HTTP router
type zHTTP struct {
	*gin.Engine
}

// NewHTTP creates a new HTTP instance
func NewHTTP() HTTP {
	gin.SetMode(gin.ReleaseMode) // disable GIN debug logs
	engine := gin.New()

	prometheus := p.NewPrometheus("gin")
	prometheus.Use(engine)

	engine.Use(gin.Recovery())
	engine.Use(LogMiddleware)
	engine.Use(otelgin.Middleware("zbz"))

	http := &zHTTP{
		Engine: engine,
	}

	http.LoadTemplates("lib/templates/*")

	return http
}

// LoadTemplates loads HTML templates from the specified directory
func (h *zHTTP) LoadTemplates(dir string) {
	Log.Debugf("Loading templates from %s", dir)
	h.LoadHTMLGlob(dir)
}

// Register an HTTP operation with the Gin router
func (h *zHTTP) AddRoute(operation *HTTPOperation) gin.IRoutes {
	Log.Debugf("[%s] %s", operation.Method, operation.Path)

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
		Log.Errorf("Unsupported HTTP method: %s", operation.Method)
		return nil
	}

	return route
}

// Serve the HTTP router on the configured port
func (h *zHTTP) Serve() error {
	Log.Infof("Starting HTTP server on port %s", config.Port())
	return h.Run(":" + config.Port())
}
