package zbz

import (
	"regexp"

	"github.com/gin-gonic/gin"
	p "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
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

	// Error management
	SetErrorResponse(statusCode int, error *Error)
	GetErrorResponse(statusCode int) *Error
	GetErrorManager() ErrorManager

	// Auth integration
	SetAuth(auth Auth)
	GetAuth() Auth
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
	Scope       string // Auth0 permission scope (e.g., "read:users")
}

// zHTTP is responsible for setting up the HTTP router
type zHTTP struct {
	*gin.Engine
	errorManager ErrorManager
	auth         Auth // Auth interface for middleware access
}

// NewHTTP creates a new HTTP instance
func NewHTTP() HTTP {
	gin.SetMode(gin.ReleaseMode) // disable GIN debug logs
	engine := gin.New()

	prometheus := p.NewPrometheus("gin")
	prometheus.Use(engine)

	errorManager := NewErrorManager()

	engine.Use(gin.Recovery())
	engine.Use(LogMiddleware)
	engine.Use(otelgin.Middleware("zbz"))
	engine.Use(AutoErrorMiddleware(errorManager))

	http := &zHTTP{
		Engine:       engine,
		errorManager: errorManager,
	}

	http.LoadTemplates("lib/templates/*")

	return http
}

// LoadTemplates loads HTML templates from the specified directory
func (h *zHTTP) LoadTemplates(dir string) {
	Log.Debug("Loading HTML templates", zap.String("templates_dir", dir))
	h.LoadHTMLGlob(dir)
}

// Register an HTTP operation with the Gin router
func (h *zHTTP) AddRoute(operation *Operation) gin.IRoutes {
	Log.Debug("Attaching HTTP route", zap.Any("operation", operation))

	path := operation.Path
	re := regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)
	path = re.ReplaceAllString(path, `:$1`)

	// Build middleware chain
	handlers := []gin.HandlerFunc{}

	// Add auth middleware if required
	if operation.Auth {
		if auth := h.getAuthMiddleware(); auth != nil {
			handlers = append(handlers, auth)
		} else {
			Log.Warn("Auth required but no auth middleware available", zap.String("path", path))
		}
	}

	// Add scope middleware if specified
	if operation.Scope != "" {
		if scopeMiddleware := h.getScopeMiddleware(operation.Scope); scopeMiddleware != nil {
			handlers = append(handlers, scopeMiddleware)
		} else {
			Log.Warn("Scope specified but no scope middleware available",
				zap.String("path", path),
				zap.String("scope", operation.Scope))
		}
	}

	// Add the main handler
	handlers = append(handlers, operation.Handler)

	var route gin.IRoutes
	switch operation.Method {
	case "GET":
		route = h.GET(path, handlers...)
	case "POST":
		route = h.POST(path, handlers...)
	case "PUT":
		route = h.PUT(path, handlers...)
	case "DELETE":
		route = h.DELETE(path, handlers...)
	default:
		Log.Error("Unsupported HTTP method", zap.String("method", operation.Method))
		return nil
	}

	return route
}

// Serve the HTTP router on the configured port
func (h *zHTTP) Serve() error {
	Log.Info("Starting HTTP server", zap.String("port", config.Port()))
	return h.Run(":" + config.Port())
}

// SetErrorResponse sets a custom error response for a given status code
func (h *zHTTP) SetErrorResponse(statusCode int, error *Error) {
	h.errorManager.SetError(statusCode, error)
}

// GetErrorResponse retrieves the error response for a given status code
func (h *zHTTP) GetErrorResponse(statusCode int) *Error {
	return h.errorManager.GetError(statusCode)
}

// GetErrorManager returns the error manager instance
func (h *zHTTP) GetErrorManager() ErrorManager {
	return h.errorManager
}

// SetAuth sets the Auth interface for middleware access
func (h *zHTTP) SetAuth(auth Auth) {
	h.auth = auth
}

// GetAuth returns the Auth interface
func (h *zHTTP) GetAuth() Auth {
	return h.auth
}

// getAuthMiddleware returns the authentication middleware if available
func (h *zHTTP) getAuthMiddleware() gin.HandlerFunc {
	if h.auth != nil {
		return h.auth.TokenMiddleware()
	}
	return nil
}

// getScopeMiddleware returns scope-specific middleware if available
func (h *zHTTP) getScopeMiddleware(scope string) gin.HandlerFunc {
	if h.auth != nil {
		return h.auth.ScopeMiddleware(scope)
	}
	return nil
}

// AutoErrorMiddleware automatically handles error responses based on status codes
func AutoErrorMiddleware(errorManager ErrorManager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next() // Let handler run first

		status := ctx.Writer.Status()
		if status >= 400 && status <= 599 { // Error status codes
			if ctx.Writer.Size() == 0 { // No body written yet
				// Get error details if available
				var errorDetails map[string]any
				if details, exists := ctx.Get("error_details"); exists {
					if detailsMap, ok := details.(map[string]string); ok {
						errorDetails = make(map[string]any)
						for k, v := range detailsMap {
							errorDetails[k] = v
						}
					} else if detailsAny, ok := details.(map[string]any); ok {
						errorDetails = detailsAny
					}
				}

				// Check for custom message in context
				if customMessage, exists := ctx.Get("error_message"); exists {
					if error := errorManager.GetError(status); error != nil {
						response := error.Response
						if msgStr, ok := customMessage.(string); ok && msgStr != "" {
							response.Message = msgStr
						}
						if errorDetails != nil {
							response.Details = errorDetails
						}
						ctx.JSON(status, response)
					}
				} else if error := errorManager.GetError(status); error != nil {
					// Use default error response
					response := error.Response
					if errorDetails != nil {
						response.Details = errorDetails
					}
					ctx.JSON(status, response)
				}
			}
		}
	}
}
