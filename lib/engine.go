package zbz

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Engine is the main application engine that provides methods to register models, attach HTTP operations, and inject core resources.
type Engine interface {
	Register(models ...*Meta)
	Attach(ops ...*Operation)
	Inject(contracts ...*CoreContract)
	Prime()
	Start()
	GetUserCore() Core // Access to built-in user core
}

// zEngine is the main application engine that holds the router, database connection, and authenticator.
type zEngine struct {
	auth     Auth
	database Database
	docs     Docs
	health   Health
	http     HTTP
	schema   Schema
	userCore Core // Built-in user management core
}

// NewEngine initializes a new Engine instance with the necessary components.
func NewEngine() Engine {
	// Initialize Redis client for auth caching
	redisClient := NewRedisClient()

	// Create HTTP instance
	http := NewHTTP()

	// Create Auth with Redis integration
	auth := NewAuth(redisClient)

	// Create user core with limited operations (read, update only)
	userCore := NewCore[User]("Built-in user management for authentication")

	engine := &zEngine{
		auth:     auth,
		database: NewDatabase(),
		docs:     NewDocs(),
		health:   NewHealth(),
		http:     http,
		schema:   NewSchema(),
		userCore: userCore,
	}

	// Link auth to HTTP for middleware access
	http.SetAuth(auth)

	// Give auth access to user core
	auth.SetUserCoreGetter(engine.GetUserCore)

	engine.Prime()
	return engine
}

// Register data models with the engine's documentation service.
func (e *zEngine) Register(models ...*Meta) {
	Log.Debug("Registering database models", zap.Int("models_count", len(models)))
	for _, model := range models {
		e.docs.AddSchema(model)
		e.schema.AddMeta(model)
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

// Compose a database query and prepare it for execution.
func (e *zEngine) Compose(contracts ...*MacroContract) {
	Log.Debug("Composing query statements", zap.Int("contracts_count", len(contracts)))
	for _, contract := range contracts {
		err := e.database.Prepare(contract)
		if err != nil {
			Log.Fatal("Failed to prepare statement", zap.Any("contract", contract), zap.Error(err))
		}
	}
}

// Execute a database query via a contract and return the result.
func (e *zEngine) Execute(contract *MacroContract, params map[string]any) (*sqlx.Rows, error) {
	Log.Debug("Executing query contract with params", zap.Any("contract", contract), zap.Any("params", params))

	e.database.Prepare(contract)
	defer e.database.Dismiss(contract)

	result, err := e.database.Execute(contract, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Inject core resources to the engine using CoreContracts, creating CRUD endpoints & documentation.
func (e *zEngine) Inject(contracts ...*CoreContract) {
	Log.Debug("Injecting core contracts", zap.Int("contract_count", len(contracts)))
	for _, contract := range contracts {
		core := contract.Core
		meta := core.Meta()
		e.Register(meta)

		// Add OpenAPI tag with the contract's description
		e.docs.AddTag(meta.Name, contract.Description)

		_, err := e.Execute(core.Table(), nil)
		if err != nil {
			Log.Fatal("Failed to create table", zap.Any("meta", meta), zap.Error(err))
		}
		e.Compose(
			core.Contracts()...,
		)

		// Filter operations based on enabled handlers
		operations := core.Operations()
		if contract.Handlers != nil && len(contract.Handlers) > 0 {
			// Filter to only include specified handlers
			filteredOps := make([]*Operation, 0)
			for _, op := range operations {
				for _, enabledHandler := range contract.Handlers {
					if strings.Contains(op.Name, enabledHandler) {
						filteredOps = append(filteredOps, op)
						break
					}
				}
			}
			operations = filteredOps
		}

		e.Attach(operations...)
	}
}

// Prime the engine by setting up middleware & default endpoints
func (e *zEngine) Prime() {
	Log.Debug("Priming the engine")

	// Bind services to the HTTP router
	e.http.Use(func(c *gin.Context) {
		c.Set("db", e.database)
		c.Next()
	})

	// Set up built-in User core with limited operations
	e.Inject(&CoreContract{
		Core:        e.userCore,
		Description: "Built-in user management for authentication and profile operations",
		Handlers:    []string{"Get", "Update"}, // Only allow read and update operations
	})

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

	// Add documentation endpoints
	e.http.GET("/openapi", e.docs.SpecHandler)
	e.http.GET("/docs", e.docs.ScalarHandler)
	e.http.GET("/schema", e.schema.SchemaHandler)

	// Add auth endpoints w/o documentation
	e.http.GET("/auth/login", e.auth.LoginHandler)
	e.http.POST("/auth/callback", e.auth.CallbackHandler)
	e.http.GET("/auth/logout", e.auth.LogoutHandler)

	// Add health check endpoint w/o documentation
	e.http.GET("/health", e.health.HealthCheckHandler)
}

// GetUserCore returns the built-in user core
func (e *zEngine) GetUserCore() Core {
	return e.userCore
}

// Start the engine by running an HTTP server
func (e *zEngine) Start() {
	Log.Debug("Starting the engine")
	e.http.Serve()
}
