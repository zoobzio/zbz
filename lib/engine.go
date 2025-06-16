package zbz

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Engine is the main application engine that provides methods to register models, attach HTTP operations, and inject core resources.
type Engine interface {
	Register(models ...*Meta)
	Attach(ops ...*HTTPOperation)
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
	Log.Debugf("Registering %d models", len(models))
	for _, model := range models {
		e.docs.AddSchema(model)
	}
}

// Attach HTTP operations to the router & documentation service.
func (e *zEngine) Attach(ops ...*HTTPOperation) {
	Log.Debugf("Attaching %d HTTP operations", len(ops))
	for _, op := range ops {
		e.docs.AddPath(op)
		e.http.AddRoute(op)
	}
}

// Compose a database query and prepare it for execution.
func (e *zEngine) Compose(contracts ...*MacroContract) {
	Log.Debugf("Composing %d query statements", len(contracts))
	for _, contract := range contracts {
		Log.Debugf("Composing statement: %s", contract.Name)
		err := e.database.Prepare(contract)
		if err != nil {
			Log.Fatalf("Failed to prepare statement %s - %v", contract.Name, err)
		}

	}
}

// Execute a database query via a contract and return the result.
func (e *zEngine) Execute(contract *MacroContract, params map[string]any) (*sqlx.Rows, error) {
	Log.Debugf("Executing contract: %s with params: %v", contract.Name, params)
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
	Log.Debugf("Injecting %d cores", len(cores))
	for _, core := range cores {
		meta := core.Meta()
		e.Register(meta)

		_, err := e.Execute(core.Table(), nil)
		if err != nil {
			Log.Fatalf("Failed to create table for %s: %v", meta.Name, err)
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
	Log.Debug("Priming the engine")

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
	e.http.POST("/auth/callback", e.auth.CallbackHandler)

	// Attach default endpoints
	e.Attach(
		// Attach the authentication endpoints w/ documentation
		&HTTPOperation{
			Name:        "Login",
			Description: "Endpoint for user login. Redirects to the authentication provider.",
			Method:      "GET",
			Path:        "/auth/login",
			Tag:         "Auth",
			Handler:     e.auth.LoginHandler,
			Response: &HTTPResponse{
				Status: http.StatusTemporaryRedirect,
			},
			Auth: false,
		},
		&HTTPOperation{
			Name:        "Logout",
			Description: "Endpoint for user logout. Clears the session and redirects to the home page.",
			Method:      "GET",
			Path:        "/auth/logout",
			Tag:         "Auth",
			Handler:     e.auth.LogoutHandler,
			Auth:        false,
		},

		// Attach the health check endpoint w/ documentation
		&HTTPOperation{
			Name:        "Health Check",
			Description: "Endpoint to check the health of the application. Returns a 200 OK status if the application is running.",
			Method:      "GET",
			Path:        "/health",
			Tag:         "Health",
			Handler:     e.health.HealthCheckHandler,
			Auth:        false,
		},
	)
}

// Start the engine by running an HTTP server
func (e *zEngine) Start() {
	Log.Debug("Starting the engine")
	e.http.Serve()
}
