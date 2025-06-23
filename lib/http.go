package zbz

import (
	"fmt"
	"zbz/lib/http"
	"zbz/shared/logger"
)

// RequestContext is aliased from http package
type RequestContext = http.RequestContext

// HTTPDriver is aliased from http package
type HTTPDriver = http.HTTPDriver

// HTTP is the ZBZ service that handles business logic using a driver
type HTTP interface {
	// High-level operations that understand ZBZ concepts
	RegisterHandler(contract *HandlerContract) error
	RegisterMiddleware(middleware func(RequestContext, func())) error
	
	// Error handling system integration
	SetErrorManager(manager ErrorManager)
	GetErrorManager() ErrorManager
	
	// Auth middleware integration
	SetAuth(auth Auth)
	
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
	logger.Info("Creating HTTP service with internal docs", 
		logger.String("name", name))
	
	return &zHTTP{
		driver:              driver,
		contractName:        name,
		contractDescription: description,
		errorManager:        NewErrorManager(),
		docs:                NewDocs(), // Internal docs service
		middlewares:         make([]func(RequestContext, func()), 0),
	}
}

// RegisterHandler converts a HandlerContract to driver-specific routing
func (h *zHTTP) RegisterHandler(contract *HandlerContract) error {
	// Create a wrapped handler that includes ZBZ logic
	wrappedHandler := h.wrapHandler(contract)
	
	// Register with the driver
	err := h.driver.AddRoute(contract.Method, contract.Path, wrappedHandler)
	if err != nil {
		logger.Error("Failed to register handler", 
			logger.String("path", contract.Path),
			logger.String("method", contract.Method),
			logger.Err(err))
		return err
	}
	
	logger.Debug("Registered HTTP handler", 
		logger.String("name", contract.Name),
		logger.String("method", contract.Method),
		logger.String("path", contract.Path),
		logger.Bool("auth", contract.Auth))
	
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
		logger.Fatal("No HTTP driver configured - cannot start server")
		return fmt.Errorf("no HTTP driver configured")
	}
	
	logger.Info("Starting HTTP server", 
		logger.String("address", address),
		logger.String("driver", h.driver.DriverName()))
	
	return h.driver.Start(address)
}

// Shutdown stops the HTTP server
func (h *zHTTP) Shutdown() error {
	logger.Info("Shutting down HTTP server")
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