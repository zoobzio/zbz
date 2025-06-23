package zbz

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"zbz/shared/logger"
)

// Engine is the main application engine that provides methods to register models, attach HTTP operations, and inject core resources.
type Engine interface {
	Register(models ...*Meta)
	Attach(contracts ...*HandlerContract)
	Inject(contracts ...ContractInjector)
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
		logger.Log.Fatal("Engine not initialized - call NewEngine() first")
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
	logger.Log.Debug("HTTP provider registered with engine")
}

// SetDatabase sets the database provider (supports multiple instances)
func (e *zEngine) SetDatabase(provider DatabaseProvider) {
	e.databaseProvider = provider
	logger.Log.Debug("Database provider registered with engine")
}

// SetAuth sets the Auth provider (singleton service)
func (e *zEngine) SetAuth(provider AuthProvider) {
	e.authProvider = provider
	logger.Log.Debug("Auth provider registered with engine")
}


// SetCore sets the Core provider (supports multiple instances)
func (e *zEngine) SetCore(provider CoreProvider) {
	e.coreProvider = provider
	logger.Log.Debug("Core provider registered with engine")
}

// GetDatabaseSchema returns the schema for a specific database (deprecated in provider architecture)
func (e *zEngine) GetDatabaseSchema(key string) Schema {
	logger.Log.Warn("GetDatabaseSchema is deprecated in provider architecture")
	return nil
}

// RegisterDatabase registers a database instance (deprecated in provider architecture)
func (e *zEngine) RegisterDatabase(key string, database Database) {
	logger.Log.Warn("RegisterDatabase is deprecated in provider architecture - use providers instead")
}

// GetDatabase retrieves a database instance (deprecated in provider architecture)
func (e *zEngine) GetDatabase(key string) Database {
	logger.Log.Warn("GetDatabase is deprecated in provider architecture - use providers instead")
	return nil
}



// Register data models with the engine's documentation service.
func (e *zEngine) Register(models ...*Meta) {
	logger.Log.Debug("Registering database models", logger.Int("models_count", len(models)))
	
	// Note: Model registration with docs now handled internally by HTTP service
	
	// Note: Database schema registration now happens through contracts during injection
	logger.Log.Debug("Model registration complete - database schemas managed through contracts")
}

// Attach HTTP handler contracts to the router & documentation service.
func (e *zEngine) Attach(contracts ...*HandlerContract) {
	logger.Log.Debug("Attaching HTTP handler contracts", logger.Int("contracts_count", len(contracts)))
	
	// Resolve HTTP service
	if e.httpProvider == nil {
		logger.Log.Warn("No HTTP provider configured - cannot attach handlers")
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
	logger.Log.Warn("Compose is deprecated in provider architecture - use database contracts directly")
}

// Execute a database query via a contract (deprecated in provider architecture)
func (e *zEngine) Execute(databaseKey string, contract *MacroContract, params map[string]any) (*sqlx.Rows, error) {
	logger.Log.Warn("Execute is deprecated in provider architecture - use database contracts directly")
	return nil, fmt.Errorf("execute is deprecated in provider architecture")
}


// Inject contracts into the engine - each contract handles its own complete setup
func (e *zEngine) Inject(contracts ...ContractInjector) {
	logger.Log.Debug("Injecting contracts", logger.Int("contract_count", len(contracts)))
	for _, contract := range contracts {
		// Each contract is self-contained and handles its own injection
		contract.Inject()
	}
}

// Prime the engine by setting up middleware & default endpoints
func (e *zEngine) Prime() {
	logger.Log.Debug("Priming the engine")

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
		// Note: Schema endpoints are now handled through database contracts during injection
		
		// Add auth endpoints if auth service is available
		if authService != nil {
			httpService.RegisterHandler(authService.LoginContract())
			httpService.RegisterHandler(authService.CallbackContract())
			httpService.RegisterHandler(authService.LogoutContract())
		}

		// Add health check endpoint
		httpService.RegisterHandler(e.health.HealthCheckContract())
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
	logger.Log.Warn("GetDocs is deprecated - docs are now internal to HTTP service")
	return nil
}

// RegisterCore registers a core (deprecated in provider architecture)
func (e *zEngine) RegisterCore(name string, core Core) {
	logger.Log.Warn("RegisterCore is deprecated in provider architecture - cores managed through contracts")
}

// Start the engine by running an HTTP server
func (e *zEngine) Start(address string) {
	logger.Log.Debug("Starting the engine", logger.String("address", address))
	
	if e.httpProvider == nil {
		logger.Log.Fatal("No HTTP provider configured - call SetHTTP() first")
	}
	
	httpService := e.httpProvider.HTTP()
	httpService.Serve(address)
}
