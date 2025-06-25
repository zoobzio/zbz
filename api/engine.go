package zbz

import (
	"fmt"
	"time"
	"github.com/jmoiron/sqlx"
	"zbz/zlog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Engine is the main application engine that provides methods to register models, attach HTTP operations, and inject core resources.
type Engine interface {
	Register(models ...*Meta)
	Attach(contracts ...*HandlerContract)
	Inject(providers ...CoreProvider)
	Prime()
	Start(address string)
	
	// Multi-database support
	RegisterDatabase(key string, database Database)
	GetDatabase(key string) Database
	
	// Service access for contracts (TODO: replace with service contracts)
	GetHTTP() HTTP
	GetDocs() Docs
	
	// Core registry for convention-over-configuration lookup
	RegisterCore(name string, core Core)
	
	// Provider management (pipeline configuration)
	SetHTTP(provider HTTPProvider)
	SetDatabase(provider DatabaseProvider) 
	SetAuth(provider AuthProvider)
	SetCore(provider CoreProvider)
	
}

// Global engine instance for contract access (TODO: replace with service contracts)
var globalEngine Engine

// GetEngine returns the global engine instance
func GetEngine() Engine {
	if globalEngine == nil {
		zlog.Zlog.Fatal("Engine not initialized - call NewEngine() first")
	}
	return globalEngine
}

// zEngine is the main application engine that resolves services through providers
type zEngine struct {
	httpProvider     HTTPProvider
	databaseProvider DatabaseProvider
	authProvider     AuthProvider
	coreProvider     CoreProvider
	health           Health // Health is always internal, not user-configurable
}

// NewEngine initializes a new Engine skeleton - services must be set by user via providers
func NewEngine() Engine {
	engine := &zEngine{
		health: NewHealth(), // Health is always internal, not user-configurable
	}

	// Set global engine reference for contract access
	globalEngine = engine
	
	return engine
}

// SetHTTP sets the HTTP provider (singleton service)
func (e *zEngine) SetHTTP(provider HTTPProvider) {
	e.httpProvider = provider
	zlog.Zlog.Debug("HTTP provider registered with engine")
}

// SetDatabase sets the database provider (supports multiple instances)
func (e *zEngine) SetDatabase(provider DatabaseProvider) {
	e.databaseProvider = provider
	zlog.Zlog.Debug("Database provider registered with engine")
}

// SetAuth sets the Auth provider (singleton service)
func (e *zEngine) SetAuth(provider AuthProvider) {
	e.authProvider = provider
	zlog.Zlog.Debug("Auth provider registered with engine")
}


// SetCore sets the Core provider (supports multiple instances)
func (e *zEngine) SetCore(provider CoreProvider) {
	e.coreProvider = provider
	zlog.Zlog.Debug("Core provider registered with engine")
}

// GetDatabaseSchema returns the schema for a specific database (deprecated in provider architecture)
func (e *zEngine) GetDatabaseSchema(key string) Schema {
	zlog.Zlog.Warn("GetDatabaseSchema is deprecated in provider architecture")
	return nil
}

// RegisterDatabase registers a database instance (deprecated in provider architecture)
func (e *zEngine) RegisterDatabase(key string, database Database) {
	zlog.Zlog.Warn("RegisterDatabase is deprecated in provider architecture - use providers instead")
}

// GetDatabase retrieves a database instance (deprecated in provider architecture)
func (e *zEngine) GetDatabase(key string) Database {
	zlog.Zlog.Warn("GetDatabase is deprecated in provider architecture - use providers instead")
	return nil
}



// Register data models with the engine's documentation service.
func (e *zEngine) Register(models ...*Meta) {
	zlog.Zlog.Debug("Registering database models", zlog.Int("models_count", len(models)))
	
	// Note: Model registration with docs now handled internally by HTTP service
	
	// Note: Database schema registration now happens through contracts during injection
	zlog.Zlog.Debug("Model registration complete - database schemas managed through contracts")
}

// Attach HTTP handler contracts to the router & documentation service.
func (e *zEngine) Attach(contracts ...*HandlerContract) {
	zlog.Zlog.Debug("Attaching HTTP handler contracts", zlog.Int("contracts_count", len(contracts)))
	
	// Resolve HTTP service
	if e.httpProvider == nil {
		zlog.Zlog.Warn("No HTTP provider configured - cannot attach handlers")
		return
	}
	httpService := e.httpProvider.HTTP()
	
	// Attach contracts - HTTP service handles docs internally
	for _, contract := range contracts {
		httpService.RegisterHandler(contract)
	}
}

// Compose a database query and prepare it for execution (deprecated in provider architecture)
func (e *zEngine) Compose(databaseKey string, contracts ...*MacroContract) {
	zlog.Zlog.Warn("Compose is deprecated in provider architecture - use database contracts directly")
}

// Execute a database query via a contract (deprecated in provider architecture)
func (e *zEngine) Execute(databaseKey string, contract *MacroContract, params map[string]any) (*sqlx.Rows, error) {
	zlog.Zlog.Warn("Execute is deprecated in provider architecture - use database contracts directly")
	return nil, fmt.Errorf("execute is deprecated in provider architecture")
}


// Inject core providers into the engine - orchestrates complete setup
func (e *zEngine) Inject(providers ...CoreProvider) {
	zlog.Zlog.Debug("Injecting core providers", zlog.Int("provider_count", len(providers)))
	
	for _, provider := range providers {
		// Resolve the core from the provider
		core := provider.Core()
		
		// Get core metadata
		name := core.ContractName()
		meta := core.Meta()
		
		zlog.Zlog.Debug("Processing core provider", 
			zlog.String("name", name),
			zlog.String("model", meta.Name))
		
		// We need to determine which database this core should use
		// For now, assume it's the default database from the DatabaseProvider
		if e.databaseProvider == nil {
			zlog.Zlog.Fatal("No database provider configured for core injection", 
				zlog.String("model", meta.Name))
		}
		
		database := e.databaseProvider.Database()
		
		// Set up the core with database
		core.setDatabase(database)
		
		// Register model metadata with the database schema
		database.GetSchema().AddMeta(meta)
		
		// Build and validate macro embeds
		embeds, err := BuildMacroEmbeds(database.GetSchema(), meta)
		if err != nil {
			zlog.Zlog.Fatal("Failed to build macro embeds for core", 
				zlog.String("model", meta.Name),
				zlog.Err(err))
		}
		
		// Store embeds in core and prepare statements
		core.setEmbeds(embeds)
		for _, macroContract := range core.MacroContracts() {
			err := database.Prepare(macroContract)
			if err != nil {
				zlog.Zlog.Fatal("Failed to prepare statement for core", 
					zlog.String("model", meta.Name),
					zlog.String("contract", macroContract.Name),
					zlog.Err(err))
			}
		}
		
		// Create table
		_, err = database.ExecuteOnce(core.Table(), map[string]any{})
		if err != nil {
			zlog.Zlog.Fatal("Failed to create table during injection", 
				zlog.String("model", meta.Name),
				zlog.Err(err))
		}
		
		// Register HTTP handlers
		handlerContracts := core.HandlerContracts()
		zlog.Zlog.Debug("Registering HTTP handlers for core", 
			zlog.String("name", name),
			zlog.Int("handler_count", len(handlerContracts)))
		
		e.Attach(handlerContracts...)
		
		// Add model schema to docs via HTTP service
		if e.httpProvider != nil {
			httpService := e.httpProvider.HTTP()
			if httpService != nil {
				err := httpService.RegisterModel(meta)
				if err != nil {
					zlog.Zlog.Warn("Failed to register model schema", 
						zlog.String("model", meta.Name),
						zlog.Err(err))
				}
				
				// Register API tag with resource description (already resolved)
				resourceDescription := core.ContractDescription()
				docs := httpService.GetDocs()
				docs.AddTag(meta.Name, resourceDescription)
				zlog.Zlog.Debug("Registered API tag for core", 
					zlog.String("name", meta.Name))
			}
		}
		
		zlog.Zlog.Info("Successfully injected core provider", 
			zlog.String("name", name),
			zlog.String("model", meta.Name))
	}
}

// createLoggingMiddleware creates the standard ZBZ logging and tracing middleware
func (e *zEngine) createLoggingMiddleware() func(RequestContext, func()) {
	return func(ctx RequestContext, next func()) {
		// Extract tracing propagation from headers
		otelCtx := otel.GetTextMapPropagator().Extract(ctx.Context(), propagation.HeaderCarrier(map[string][]string{
			"traceparent": {ctx.Header("traceparent")},
			"tracestate":  {ctx.Header("tracestate")},
		}))
		
		// Create a new span for this HTTP request
		tracer := otel.Tracer("zbz-http")
		otelCtx, span := tracer.Start(otelCtx, ctx.Method()+" "+ctx.Path(),
			trace.WithAttributes(
				attribute.String("http.method", ctx.Method()),
				attribute.String("http.url", ctx.Path()),
				attribute.String("http.user_agent", ctx.Header("User-Agent")),
			),
		)
		defer span.End()
		
		start := time.Now()
		traceID := span.SpanContext().TraceID().String()
		
		// Log incoming request
		zlog.Zlog.Info("HTTP Request",
			zlog.String("method", ctx.Method()),
			zlog.String("path", ctx.Path()),
			zlog.String("user_agent", ctx.Header("User-Agent")),
			zlog.String("trace_id", traceID))

		// Process request
		next()

		// Set span status based on response (we don't have direct access to status code)
		// The driver should handle response attributes
		
		// Log response
		duration := time.Since(start)
		zlog.Zlog.Info("HTTP Response",
			zlog.String("method", ctx.Method()),
			zlog.String("path", ctx.Path()),
			zlog.String("trace_id", traceID),
			zlog.Duration("duration", duration))
	}
}

// Prime the engine by setting up middleware & default endpoints
func (e *zEngine) Prime() {
	zlog.Zlog.Debug("Priming the engine")

	// Resolve services through providers
	var httpService HTTP
	var authService Auth
	
	if e.httpProvider != nil {
		httpService = e.httpProvider.HTTP()
	}
	if e.authProvider != nil {
		authService = e.authProvider.Auth()
	}

	// Note: Documentation setup is now handled internally by HTTP service
	
	// Add HTTP endpoints only if HTTP service is available
	if httpService != nil {
		// Update docs with app description from remark if available
		if appRemark, err := Remark.Get("app"); err == nil {
			docs := httpService.GetDocs()
			docs.SetDescription(appRemark)
			zlog.Zlog.Debug("Set app description from remark")
		}
		
		// Connect auth service to HTTP service for middleware
		if authService != nil {
			httpService.SetAuth(authService)
			zlog.Zlog.Debug("Connected auth service to HTTP service")
		}
		
		// Add standard ZBZ logging and tracing middleware to all HTTP services
		loggingMiddleware := e.createLoggingMiddleware()
		err := httpService.RegisterMiddleware(loggingMiddleware)
		if err != nil {
			zlog.Zlog.Warn("Failed to register logging middleware", zlog.Err(err))
		}
		
		// Note: Schema endpoints are now handled through database contracts during injection
		
		// Add auth endpoints silently if auth service is available (no docs)
		if authService != nil {
			err = httpService.RegisterSilentRoute("GET", "/auth/login", authService.LoginHandler)
			if err != nil {
				zlog.Zlog.Warn("Failed to register auth login endpoint", zlog.Err(err))
			}
			
			err = httpService.RegisterSilentRoute("GET", "/auth/callback", authService.CallbackHandler)
			if err != nil {
				zlog.Zlog.Warn("Failed to register auth callback endpoint", zlog.Err(err))
			}
			
			err = httpService.RegisterSilentRoute("GET", "/auth/logout", authService.LogoutHandler)
			if err != nil {
				zlog.Zlog.Warn("Failed to register auth logout endpoint", zlog.Err(err))
			}
			
			zlog.Zlog.Debug("Registered auth endpoints silently")
		}

		// Add health check endpoint (silent - no docs)
		err = httpService.RegisterSilentRoute("GET", "/health", e.health.HealthCheckHandler)
		if err != nil {
			zlog.Zlog.Warn("Failed to register health check endpoint", zlog.Err(err))
		}
	}
}



// GetHTTP returns the HTTP service instance (deprecated in provider architecture)
func (e *zEngine) GetHTTP() HTTP {
	if e.httpProvider != nil {
		return e.httpProvider.HTTP()
	}
	return nil
}

// GetDocs returns the Docs service instance (deprecated - docs now internal to HTTP)
func (e *zEngine) GetDocs() Docs {
	zlog.Zlog.Warn("GetDocs is deprecated - docs are now internal to HTTP service")
	return nil
}

// RegisterCore registers a core (deprecated in provider architecture)
func (e *zEngine) RegisterCore(name string, core Core) {
	zlog.Zlog.Warn("RegisterCore is deprecated in provider architecture - cores managed through contracts")
}


// Start the engine by running an HTTP server
func (e *zEngine) Start(address string) {
	zlog.Zlog.Debug("Starting the engine", zlog.String("address", address))
	
	if e.httpProvider == nil {
		zlog.Zlog.Fatal("No HTTP provider configured - call SetHTTP() first")
	}
	
	httpService := e.httpProvider.HTTP()
	httpService.Serve(address)
}
