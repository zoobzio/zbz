package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"zbz/shared/logger"
)

// GinDriver implements HTTPDriver using the Gin framework
type GinDriver struct {
	engine  *gin.Engine
	server  *http.Server
	address string
}

// NewGinDriver creates a new Gin-based HTTP driver
func NewGinDriver(devMode bool) *GinDriver {
	// Always use release mode by default to avoid debug spam
	// Gin debug logs are not useful for application debugging
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// Add ZBZ HTTP tracing and logging middleware
	engine.Use(func(c *gin.Context) {
		// Extract tracing propagation from headers
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		
		// Create a new span for this HTTP request
		tracer := otel.Tracer("zbz-http")
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.Request.URL.Path,
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.user_agent", c.GetHeader("User-Agent")),
				attribute.String("http.remote_addr", c.ClientIP()),
			),
		)
		defer span.End()
		
		// Update the request context with the span
		c.Request = c.Request.WithContext(ctx)
		
		start := time.Now()
		traceID := span.SpanContext().TraceID().String()
		
		// Log incoming request
		logger.Log.Info("HTTP Request",
			logger.String("method", c.Request.Method),
			logger.String("path", c.Request.URL.Path),
			logger.String("remote_addr", c.ClientIP()),
			logger.String("user_agent", c.GetHeader("User-Agent")),
			logger.String("trace_id", traceID),
			logger.Any("query_params", c.Request.URL.Query()),
			logger.Any("headers", c.Request.Header))

		// Process request
		c.Next()

		// Add response attributes to span
		status := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", status),
			attribute.Int("http.response_size", c.Writer.Size()),
		)
		
		// Set span status based on HTTP status
		if status >= 400 {
			span.RecordError(fmt.Errorf("HTTP %d", status))
		}

		// Log response
		duration := time.Since(start)
		logger.Log.Info("HTTP Response",
			logger.String("method", c.Request.Method),
			logger.String("path", c.Request.URL.Path),
			logger.Int("status", status),
			logger.String("trace_id", traceID),
			logger.Duration("duration", duration),
			logger.Int("response_size", c.Writer.Size()))
	})

	// Add recovery middleware for panics
	engine.Use(gin.Recovery())

	return &GinDriver{
		engine: engine,
	}
}

// AddRoute adds a route to the Gin engine
func (g *GinDriver) AddRoute(method, path string, handler func(RequestContext)) error {
	// Convert our RequestContext handler to Gin handler
	ginHandler := func(c *gin.Context) {
		// Create RequestContext wrapper around Gin context
		ctx := NewGinRequestContext(c)
		handler(ctx)
	}

	switch method {
	case "GET":
		g.engine.GET(path, ginHandler)
	case "POST":
		g.engine.POST(path, ginHandler)
	case "PUT":
		g.engine.PUT(path, ginHandler)
	case "DELETE":
		g.engine.DELETE(path, ginHandler)
	case "PATCH":
		g.engine.PATCH(path, ginHandler)
	case "OPTIONS":
		g.engine.OPTIONS(path, ginHandler)
	case "HEAD":
		g.engine.HEAD(path, ginHandler)
	default:
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}

	logger.Debug("Added route", 
		logger.String("method", method), 
		logger.String("path", path))
	
	return nil
}

// AddMiddleware adds middleware to the Gin engine
func (g *GinDriver) AddMiddleware(middleware func(RequestContext, func())) error {
	ginMiddleware := gin.HandlerFunc(func(c *gin.Context) {
		ctx := NewGinRequestContext(c)
		middleware(ctx, func() {
			c.Next()
		})
	})

	g.engine.Use(ginMiddleware)
	logger.Debug("Added middleware")
	
	return nil
}

// Start starts the HTTP server
func (g *GinDriver) Start(address string) error {
	g.address = address
	g.server = &http.Server{
		Addr:    address,
		Handler: g.engine,
	}

	logger.Info("Starting Gin HTTP server", logger.String("address", address))
	
	// Start server and block until it stops
	if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Failed to start HTTP server", logger.Err(err))
		return err
	}

	return nil
}

// Stop stops the HTTP server
func (g *GinDriver) Stop() error {
	if g.server == nil {
		return nil
	}

	logger.Info("Stopping Gin HTTP server")
	return g.server.Close()
}

// DriverName returns the driver name
func (g *GinDriver) DriverName() string {
	return "gin"
}

// DriverVersion returns the driver version
func (g *GinDriver) DriverVersion() string {
	return "1.9.1" // Gin version
}

// GinRequestContext implements RequestContext for Gin
type GinRequestContext struct {
	ctx *gin.Context
}

// NewGinRequestContext creates a new Gin request context wrapper
func NewGinRequestContext(ctx *gin.Context) RequestContext {
	return &GinRequestContext{ctx: ctx}
}

// Method returns the HTTP method
func (g *GinRequestContext) Method() string {
	return g.ctx.Request.Method
}

// Path returns the request path
func (g *GinRequestContext) Path() string {
	return g.ctx.Request.URL.Path
}

// PathParam returns a path parameter
func (g *GinRequestContext) PathParam(name string) string {
	return g.ctx.Param(name)
}

// QueryParam returns a query parameter
func (g *GinRequestContext) QueryParam(name string) string {
	return g.ctx.Query(name)
}

// Header returns a request header
func (g *GinRequestContext) Header(name string) string {
	return g.ctx.GetHeader(name)
}

// BodyBytes returns the request body as bytes
func (g *GinRequestContext) BodyBytes() ([]byte, error) {
	return io.ReadAll(g.ctx.Request.Body)
}

// Cookie returns a cookie value
func (g *GinRequestContext) Cookie(name string) (string, error) {
	return g.ctx.Cookie(name)
}

// SetCookie sets a cookie
func (g *GinRequestContext) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	g.ctx.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

// Status sets the response status code
func (g *GinRequestContext) Status(code int) {
	g.ctx.Status(code)
}

// SetHeader sets a response header
func (g *GinRequestContext) SetHeader(name, value string) {
	g.ctx.Header(name, value)
}

// JSON sends a JSON response
func (g *GinRequestContext) JSON(data any) error {
	g.ctx.JSON(g.ctx.Writer.Status(), data)
	return nil
}

// Data sends raw data with content type
func (g *GinRequestContext) Data(contentType string, data []byte) error {
	g.ctx.Data(g.ctx.Writer.Status(), contentType, data)
	return nil
}

// HTML renders an HTML template
func (g *GinRequestContext) HTML(name string, data any) error {
	g.ctx.HTML(g.ctx.Writer.Status(), name, data)
	return nil
}

// Redirect performs a redirect
func (g *GinRequestContext) Redirect(code int, url string) {
	g.ctx.Redirect(code, url)
}

// Set stores a value in the context
func (g *GinRequestContext) Set(key string, value any) {
	g.ctx.Set(key, value)
}

// Get retrieves a value from the context
func (g *GinRequestContext) Get(key string) (any, bool) {
	return g.ctx.Get(key)
}

// Context returns the underlying context.Context
func (g *GinRequestContext) Context() context.Context {
	return g.ctx.Request.Context()
}

// Unwrap returns the underlying Gin context
func (g *GinRequestContext) Unwrap() any {
	return g.ctx
}