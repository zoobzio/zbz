package zbz

import (
	"fmt"
	"zbz/api/http"
	"zbz/zlog"
)

// RequestContext is aliased from http package
type RequestContext = http.RequestContext

// HTTPDriver is aliased from http package
type HTTPDriver = http.HTTPDriver

// HTTP is the ZBZ service that handles business logic using a driver
type HTTP interface {
	// High-level operations that understand ZBZ concepts
	RegisterHandler(contract *HandlerContract) error
	RegisterSilentRoute(method, path string, handler func(RequestContext)) error
	RegisterMiddleware(middleware func(RequestContext, func())) error
	RegisterModel(meta *Meta) error
	
	// Error handling system integration
	SetErrorManager(manager ErrorManager)
	GetErrorManager() ErrorManager
	
	// Auth middleware integration
	SetAuth(auth Auth)
	
	// Service resolution
	GetDocs() Docs
	
	// Server management
	Serve(address string) error
	Shutdown() error
	
	// Service metadata
	ContractName() string
	ContractDescription() string
}

// zHTTP implements HTTP service using an HTTPDriver
type zHTTP struct {
	driver              HTTPDriver
	contractName        string
	contractDescription string
	errorManager        ErrorManager
	auth                Auth
	docs                Docs
	middlewares         []func(RequestContext, func())
}

// NewHTTP creates an HTTP service with the provided driver and internal docs
func NewHTTP(driver HTTPDriver, name, description string) HTTP {
	zlog.Zlog.Info("Creating HTTP service with internal docs", 
		zlog.String("name", name))
	
	docs := NewDocs()
	
	http := &zHTTP{
		driver:              driver,
		contractName:        name,
		contractDescription: description,
		errorManager:        NewErrorManager(),
		docs:                docs,
		middlewares:         make([]func(RequestContext, func()), 0),
	}
	
	// Automatically register docs endpoints (internal, not documented in OpenAPI)
	if err := http.registerDocsEndpoints(); err != nil {
		zlog.Zlog.Warn("Failed to register docs endpoints", zlog.Err(err))
	}
	
	return http
}

// RegisterHandler converts a HandlerContract to driver-specific routing
func (h *zHTTP) RegisterHandler(contract *HandlerContract) error {
	// Create a wrapped handler that includes ZBZ logic
	wrappedHandler := h.wrapHandler(contract)
	
	// Register with the driver
	err := h.driver.AddRoute(contract.Method, contract.Path, wrappedHandler)
	if err != nil {
		zlog.Zlog.Error("Failed to register handler", 
			zlog.String("path", contract.Path),
			zlog.String("method", contract.Method),
			zlog.Err(err))
		return err
	}
	
	// Add handler to docs service for OpenAPI generation
	h.docs.AddPath(contract, h.errorManager)
	
	zlog.Zlog.Debug("Registered HTTP handler", 
		zlog.String("name", contract.Name),
		zlog.String("method", contract.Method),
		zlog.String("path", contract.Path),
		zlog.Bool("auth", contract.Auth))
	
	return nil
}

// wrapHandler creates a wrapped handler that includes ZBZ middleware logic
func (h *zHTTP) wrapHandler(contract *HandlerContract) func(RequestContext) {
	return func(ctx RequestContext) {
		// Create middleware chain
		middlewares := make([]func(RequestContext, func()), 0)
		
		// Add global middlewares
		middlewares = append(middlewares, h.middlewares...)
		
		// Add auth middleware if required
		if contract.Auth && h.auth != nil {
			middlewares = append(middlewares, h.auth.Middleware())
		} else if contract.Auth && h.auth == nil {
			zlog.Zlog.Fatal("Handler requires authentication but no auth service configured", 
				zlog.String("path", contract.Path),
				zlog.String("method", contract.Method),
				zlog.String("help", "Either configure auth service or set Auth: false on handler"))
		}
		
		// Add scope middleware if specified
		if contract.Scope != "" && h.auth != nil {
			middlewares = append(middlewares, h.auth.ScopeMiddleware(contract.Scope))
		}
		
		// Add error handling middleware
		middlewares = append(middlewares, h.errorMiddleware)
		
		// Execute middleware chain
		h.executeMiddlewareChain(ctx, middlewares, contract.Handler)
	}
}

// executeMiddlewareChain executes middleware in order
func (h *zHTTP) executeMiddlewareChain(ctx RequestContext, middlewares []func(RequestContext, func()), handler func(RequestContext)) {
	if len(middlewares) == 0 {
		handler(ctx)
		return
	}
	
	// Take first middleware and create next function
	middleware := middlewares[0]
	remainingMiddlewares := middlewares[1:]
	
	next := func() {
		h.executeMiddlewareChain(ctx, remainingMiddlewares, handler)
	}
	
	middleware(ctx, next)
}

// errorMiddleware handles automatic error responses
func (h *zHTTP) errorMiddleware(ctx RequestContext, next func()) {
	next() // Execute handler first
	
	// Check if an error status was set but no response body written
	// This would be detected by checking response size, but that's driver-specific
	// For now, we'll rely on the handler to properly set error responses
}

// RegisterMiddleware adds a global middleware
func (h *zHTTP) RegisterMiddleware(middleware func(RequestContext, func())) error {
	h.middlewares = append(h.middlewares, middleware)
	
	// Also register with driver for driver-level middleware
	return h.driver.AddMiddleware(middleware)
}

// SetErrorManager sets the error manager
func (h *zHTTP) SetErrorManager(manager ErrorManager) {
	h.errorManager = manager
}

// GetErrorManager returns the error manager
func (h *zHTTP) GetErrorManager() ErrorManager {
	return h.errorManager
}

// SetAuth sets the auth service for middleware
func (h *zHTTP) SetAuth(auth Auth) {
	h.auth = auth
}

// Serve starts the HTTP server
func (h *zHTTP) Serve(address string) error {
	if h.driver == nil {
		zlog.Zlog.Fatal("No HTTP driver configured - cannot start server")
		return fmt.Errorf("no HTTP driver configured")
	}
	
	zlog.Zlog.Info("Starting HTTP server", 
		zlog.String("address", address),
		zlog.String("driver", h.driver.DriverName()))
	
	return h.driver.Start(address)
}

// Shutdown stops the HTTP server
func (h *zHTTP) Shutdown() error {
	zlog.Zlog.Info("Shutting down HTTP server")
	return h.driver.Stop()
}

// ContractName returns the service name
func (h *zHTTP) ContractName() string {
	return h.contractName
}

// ContractDescription returns the service description
func (h *zHTTP) ContractDescription() string {
	return h.contractDescription
}

// RegisterModel adds a model schema to the internal docs service
func (h *zHTTP) RegisterModel(meta *Meta) error {
	h.docs.AddSchema(meta)
	
	zlog.Zlog.Debug("Registered model schema with HTTP docs service", 
		zlog.String("model", meta.Name),
		zlog.Any("meta", meta))
	
	return nil
}

// GetDocs returns the internal docs service
func (h *zHTTP) GetDocs() Docs {
	return h.docs
}

// RegisterSilentRoute registers a route without adding it to documentation
func (h *zHTTP) RegisterSilentRoute(method, path string, handler func(RequestContext)) error {
	// Create a wrapped handler that includes ZBZ logic but bypass docs
	wrappedHandler := func(ctx RequestContext) {
		// Apply error handling middleware logic
		if h.errorManager != nil {
			defer func() {
				if errorMessage, exists := ctx.Get("error_message"); exists {
					if msg, ok := errorMessage.(string); ok && msg != "" {
						// Custom error message set by handler
						statusCode, _ := ctx.Get("status_code")
						ctx.JSON(map[string]any{
							"error":   msg,
							"status":  "error",
							"code":    statusCode,
						})
						return
					}
				}
			}()
		}
		
		// Call the actual handler
		handler(ctx)
	}
	
	// Register directly with driver (no docs)
	err := h.driver.AddRoute(method, path, wrappedHandler)
	if err != nil {
		zlog.Zlog.Error("Failed to register silent route", 
			zlog.String("path", path),
			zlog.String("method", method),
			zlog.Err(err))
		return err
	}
	
	zlog.Zlog.Debug("Registered silent route", 
		zlog.String("method", method),
		zlog.String("path", path))
	
	return nil
}

// registerDocsEndpoints automatically registers the internal docs endpoints with auth
func (h *zHTTP) registerDocsEndpoints() error {
	// Create auth-protected handlers for docs endpoints
	openAPIHandler := func(ctx RequestContext) {
		// Apply auth middleware if available
		if h.auth != nil {
			authMiddleware := h.auth.EnsureAuthMiddleware()
			authMiddleware(ctx, func() {
				h.docs.SpecHandler(ctx)
			})
		} else {
			// No auth configured - serve docs directly
			h.docs.SpecHandler(ctx)
		}
	}
	
	docsHandler := func(ctx RequestContext) {
		// Apply auth middleware if available
		if h.auth != nil {
			authMiddleware := h.auth.EnsureAuthMiddleware()
			authMiddleware(ctx, func() {
				h.docs.ScalarHandler(ctx)
			})
		} else {
			// No auth configured - serve docs directly
			h.docs.ScalarHandler(ctx)
		}
	}
	
	// Register /openapi endpoint (OpenAPI spec in YAML format)
	if err := h.driver.AddRoute("GET", "/openapi", openAPIHandler); err != nil {
		return fmt.Errorf("failed to register /openapi endpoint: %w", err)
	}
	
	// Register /docs endpoint (Scalar documentation UI)
	if err := h.driver.AddRoute("GET", "/docs", docsHandler); err != nil {
		return fmt.Errorf("failed to register /docs endpoint: %w", err)
	}
	
	authStatus := "disabled"
	if h.auth != nil {
		authStatus = "enabled"
	}
	
	zlog.Zlog.Debug("Registered internal docs endpoints",
		zlog.String("openapi_endpoint", "/openapi"),
		zlog.String("docs_endpoint", "/docs"),
		zlog.String("auth_protection", authStatus))
	
	return nil
}