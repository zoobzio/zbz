package http

// HTTPDriver defines the interface that HTTP adapters must implement
// This is what user-initialized drivers (Gin, FastHTTP, etc.) implement
type HTTPDriver interface {
	// Route registration
	AddRoute(method, path string, handler func(RequestContext)) error
	
	// Middleware management
	AddMiddleware(middleware func(RequestContext, func())) error
	
	// Server lifecycle
	Start(address string) error
	Stop() error
	
	// Driver metadata
	DriverName() string
	DriverVersion() string
}