package zbz

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Engine is the main application engine that provides methods to register models, attach HTTP operations, and inject core resources.
type Engine interface {
	Register(models ...*Meta)
	Attach(ops ...*Operation)
	Inject(contracts ...ContractInjector)
	Prime()
	Start()
	
	// Multi-database support
	RegisterDatabase(key string, database Database)
	GetDatabase(key string) Database
	
	// Service access for contracts (TODO: replace with service contracts)
	GetHTTP() HTTP
	GetDocs() Docs
	
	// Core registry for convention-over-configuration lookup
	RegisterCore(name string, core Core)
	
}

// Global engine instance for contract access (TODO: replace with service contracts)
var globalEngine Engine

// GetEngine returns the global engine instance
func GetEngine() Engine {
	if globalEngine == nil {
		Log.Fatal("Engine not initialized - call NewEngine() first")
	}
	return globalEngine
}

// zEngine is the main application engine that holds the router, database connections, and authenticator.
type zEngine struct {
	auth      Auth
	databases map[string]Database // Support multiple databases
	cores     map[string]Core     // Registry of cores by contract key
	docs      Docs
	health    Health
	http      HTTP
}

// NewEngine initializes a new Engine instance with the necessary components.
func NewEngine() Engine {
	// Initialize Redis client for auth caching
	redisClient := NewRedisClient()

	// Create HTTP instance
	http := NewHTTP()

	// Create Auth with Redis integration
	auth := NewAuth(redisClient)

	engine := &zEngine{
		auth:      auth,
		databases: make(map[string]Database),
		cores:     make(map[string]Core),
		docs:      NewDocs(),
		health:    NewHealth(),
		http:      http,
	}
	
	// No default database creation - cores will create databases via contracts

	// Link auth to HTTP for middleware access
	http.SetAuth(auth)

	// Give auth access to user core via registry lookup (convention-over-configuration)
	// Any core named "User" will be used - framework default or user override
	auth.SetUserCoreGetter(func() Core {
		for _, core := range engine.cores {
			if core.Meta().Name == "User" {
				return core
			}
		}
		Log.Fatal("User core not found - ensure a contract with Name: 'User' is injected")
		return nil
	})

	// Set global engine reference for contract access
	globalEngine = engine
	
	engine.Prime()
	return engine
}

// GetDatabaseSchema returns the schema for a specific database
func (e *zEngine) GetDatabaseSchema(key string) Schema {
	db := e.GetDatabase(key)
	if db == nil {
		return nil
	}
	return db.GetSchema()
}

// RegisterDatabase registers a database instance with the given key
func (e *zEngine) RegisterDatabase(key string, database Database) {
	Log.Debug("Registering database", zap.String("key", key))
	e.databases[key] = database
}

// GetDatabase retrieves a database instance by key
func (e *zEngine) GetDatabase(key string) Database {
	db, exists := e.databases[key]
	if !exists {
		Log.Warn("Database not found, returning primary", zap.String("key", key))
		return e.databases["primary"] // Fallback to primary
	}
	return db
}



// Register data models with the engine's documentation service and all database schemas.
func (e *zEngine) Register(models ...*Meta) {
	Log.Debug("Registering database models", zap.Int("models_count", len(models)))
	for _, model := range models {
		e.docs.AddSchema(model)
		// Register model metadata with all database schemas
		for key, db := range e.databases {
			Log.Debug("Adding model to database schema", zap.String("model", model.Name), zap.String("database", key))
			db.GetSchema().AddMeta(model)
		}
	}
}

// Attach HTTP operations to the router & documentation service.
func (e *zEngine) Attach(ops ...*Operation) {
	Log.Debug("Attaching HTTP operations", zap.Int("operations_count", len(ops)))
	errorManager := e.http.GetErrorManager()
	for _, op := range ops {
		e.docs.AddPath(op, errorManager)
		e.http.AddRoute(op)
	}
}

// Compose a database query and prepare it for execution on a specific database.
func (e *zEngine) Compose(databaseKey string, contracts ...*MacroContract) {
	Log.Debug("Composing query statements", zap.String("database", databaseKey), zap.Int("contracts_count", len(contracts)))
	db := e.GetDatabase(databaseKey)
	for _, contract := range contracts {
		err := db.Prepare(contract)
		if err != nil {
			Log.Fatal("Failed to prepare statement", zap.String("database", databaseKey), zap.Any("contract", contract), zap.Error(err))
		}
	}
}

// Execute a database query via a contract and return the result on a specific database.
func (e *zEngine) Execute(databaseKey string, contract *MacroContract, params map[string]any) (*sqlx.Rows, error) {
	Log.Debug("Executing query contract with params", zap.String("database", databaseKey), zap.Any("contract", contract), zap.Any("params", params))

	db := e.GetDatabase(databaseKey)
	db.Prepare(contract)
	defer db.Dismiss(contract)

	result, err := db.Execute(contract, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}


// Inject contracts into the engine - each contract handles its own complete setup
func (e *zEngine) Inject(contracts ...ContractInjector) {
	Log.Debug("Injecting contracts", zap.Int("contract_count", len(contracts)))
	for _, contract := range contracts {
		// Each contract is self-contained and handles its own injection
		contract.Inject()
	}
}

// Prime the engine by setting up middleware & default endpoints
func (e *zEngine) Prime() {
	Log.Debug("Priming the engine")

	// Bind services to the HTTP router
	e.http.Use(func(c *gin.Context) {
		// Set primary database for backward compatibility
		c.Set("db", e.GetDatabase("primary"))
		// Set database registry for multi-database access
		c.Set("databases", e.databases)
		c.Next()
	})

	// Built-in user core will be injected via UserContract from main.go

	// Set up common inputs
	e.docs.AddParameter(&OpenAPIParameter{
		Name:        "id",
		In:          "path",
		Description: "A unique identifier for a given resource.",
		Required:    true,
		Schema: &OpenAPISchema{
			Type:    "string",
			Format:  "uuid",
			Example: "123e4567-e89b-12d3-a456-426614174000",
		},
	})

	// Add authenticated documentation endpoints
	docsOp := &Operation{
		Name:        "Get OpenAPI Specification",
		Description: "Get OpenAPI specification (full for admins, scoped for users)",
		Method:      "GET",
		Path:        "/openapi",
		Auth:        true,
		Handler:     e.docs.SpecHandler,
	}
	e.http.AddRoute(docsOp)
	
	scalarOp := &Operation{
		Name:        "View Interactive API Documentation", 
		Description: "View interactive API documentation (full for admins, scoped for users)",
		Method:      "GET",
		Path:        "/docs",
		Auth:        true,
		Handler:     e.docs.ScalarHandler,
	}
	e.http.AddRoute(scalarOp)
	
	// Schema endpoints for each database (unauthenticated for basic introspection)
	for key, db := range e.databases {
		path := fmt.Sprintf("/schema/%s", key)
		e.http.GET(path, db.GetSchema().SchemaHandler)
		Log.Debug("Added schema endpoint", zap.String("path", path), zap.String("database", key))
	}
	// Default schema endpoint points to primary database for backward compatibility
	if primary := e.GetDatabase("primary"); primary != nil {
		e.http.GET("/schema", primary.GetSchema().SchemaHandler)
	}

	// Add auth endpoints w/o documentation
	e.http.GET("/auth/login", e.auth.LoginHandler)
	e.http.POST("/auth/callback", e.auth.CallbackHandler)
	e.http.GET("/auth/logout", e.auth.LogoutHandler)

	// Add health check endpoint w/o documentation
	e.http.GET("/health", e.health.HealthCheckHandler)
}



// GetHTTP returns the HTTP service instance
func (e *zEngine) GetHTTP() HTTP {
	return e.http
}

// GetDocs returns the Docs service instance  
func (e *zEngine) GetDocs() Docs {
	return e.docs
}

// RegisterCore registers a core in the engine registry for lookup by name
func (e *zEngine) RegisterCore(name string, core Core) {
	e.cores[name] = core
	Log.Debug("Registered core in engine registry", zap.String("name", name))
}

// Start the engine by running an HTTP server
func (e *zEngine) Start() {
	Log.Debug("Starting the engine")
	e.http.Serve()
}
