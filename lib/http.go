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

	AddRoute(operation *Operation) gin.IRoutes
	Serve() error
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

// Response represents a response structure for an HTTP operation
type Response struct {
	Status int
	Ref    string
	Type   string
	Errors []int
}

// Operation represents a given API action that can be used to register endpoints & spawn documentation
type Operation struct {
	Name        string
	Description string
	Tag         string
	Method      string
	Path        string
	Handler     gin.HandlerFunc `json:"-"`
	Parameters  []string
	Query       []string
	RequestBody string
	Response    *Response
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
	Log.Debugw("Loading HTML templates", "templates_dir", dir)
	h.LoadHTMLGlob(dir)
}

// Register an HTTP operation with the Gin router
func (h *zHTTP) AddRoute(operation *Operation) gin.IRoutes {
	Log.Debugw("Attaching HTTP route", "operation", operation)

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
		Log.Errorw("Unsupported HTTP method", operation.Method)
		return nil
	}

	return route
}

// Serve the HTTP router on the configured port
func (h *zHTTP) Serve() error {
	Log.Infow("Starting HTTP server", "port", config.Port())
	return h.Run(":" + config.Port())
}
