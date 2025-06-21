package zbz

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Engine is the main application engine that provides methods to register models, attach HTTP operations, and inject core resources.
type Engine interface {
	Register(models ...*Meta)
	Attach(ops ...*Operation)
	Inject(cores ...Core)
	Prime()
	Start()
}

// zEngine is the main application engine that holds the router, database connection, and authenticator.
type zEngine struct {
	auth     Auth
	database Database
	docs     Docs
	health   Health
	http     HTTP
}

// NewEngine initializes a new Engine instance with the necessary components.
func NewEngine() Engine {
	engine := &zEngine{
		auth:     NewAuth(),
		database: NewDatabase(),
		docs:     NewDocs(),
		health:   NewHealth(),
		http:     NewHTTP(),
	}
	engine.Prime()
	return engine
}

// Register data models with the engine's documentation service.
func (e *zEngine) Register(models ...*Meta) {
	Log.Debugw("Registering database models", "models_count", len(models))
	for _, model := range models {
		e.docs.AddSchema(model)
	}
}

// Attach HTTP operations to the router & documentation service.
func (e *zEngine) Attach(ops ...*Operation) {
	Log.Debugw("Attaching HTTP operations", "operations_count", len(ops))
	for _, op := range ops {
		e.docs.AddPath(op)
		e.http.AddRoute(op)
	}
}

// Compose a database query and prepare it for execution.
func (e *zEngine) Compose(contracts ...*MacroContract) {
	Log.Debugw("Composing query statements", "contracts_count", len(contracts))
	for _, contract := range contracts {
		err := e.database.Prepare(contract)
		if err != nil {
			Log.Fatalw("Failed to prepare statement", "contract", contract, "error", err)
		}
	}
}

// Execute a database query via a contract and return the result.
func (e *zEngine) Execute(contract *MacroContract, params map[string]any) (*sqlx.Rows, error) {
	Log.Debugw("Executing query contract with params", "contract", contract, "params", params)

	e.database.Prepare(contract)
	defer e.database.Dismiss(contract)

	result, err := e.database.Execute(contract, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Inject a core resource to the engine, creating default CRUD endpoints & documentation.
func (e *zEngine) Inject(cores ...Core) {
	Log.Debugw("Injecting data cores", "cores_count", len(cores))
	for _, core := range cores {
		meta := core.Meta()
		e.Register(meta)

		_, err := e.Execute(core.Table(), nil)
		if err != nil {
			Log.Fatalw("Failed to create table", "meta", meta, "error", err)
		}
		e.Compose(
			core.Contracts()...,
		)
		e.Attach(
			core.Operations()...,
		)

	}
}

// Prime the engine by setting up middleware & default endpoints
func (e *zEngine) Prime() {
	Log.Debugw("Priming the engine")

	// Bind services to the HTTP router
	e.http.Use(func(c *gin.Context) {
		c.Set("db", e.database)
		c.Next()
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

	// Add auth endpoints w/o documentation
	e.http.GET("/auth/login", e.auth.LoginHandler)
	e.http.POST("/auth/callback", e.auth.CallbackHandler)
	e.http.GET("/auth/logout", e.auth.LogoutHandler)

	// Add health check endpoint w/o documentation
	e.http.GET("/health", e.health.HealthCheckHandler)
}

// Start the engine by running an HTTP server
func (e *zEngine) Start() {
	Log.Debugw("Starting the engine")
	e.http.Serve()
}
