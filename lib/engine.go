package zbz

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Engine is the main application engine that holds the router, database connection, and authenticator.
type Engine struct {
	Auth     Auth
	Config   Config
	Database Database
	Docs     Docs
	Health   Health
	Http     HTTP
	Log      Logger
}

// NewEngine initializes a new Engine instance with the necessary components.
func NewEngine() *Engine {
	l := NewLogger()
	c := NewConfig(l)
	a := NewAuth(l, c)
	db := NewDatabase(l, c)
	d := NewDocs(l, c)
	h := NewHTTP(l, c, a)
	hp := NewHealth(l, c)
	e := &Engine{
		Auth:     a,
		Database: db,
		Docs:     d,
		Health:   hp,
		Http:     h,
		Log:      l,
	}
	e.Prime()
	return e
}

// Register data models with the engine's documentation service.
func (e *Engine) Register(models ...*Meta) {
	e.Log.Debugf("Registering %d models", len(models))
	for _, model := range models {
		e.Docs.AddSchema(model)
		// TODO consider adding a migration step
	}
}

// Attach HTTP operations to the router & documentation service.
func (e *Engine) Attach(ops ...*HTTPOperation) {
	e.Log.Debugf("Attaching %d HTTP operations", len(ops))
	for _, op := range ops {
		e.Docs.AddPath(op)
		e.Http.AddRoute(op)
	}
}

// Inject a core resource to the engine, creating default CRUD endpoints & documentation.
func (e *Engine) Inject(cores ...Core) {
	e.Log.Debugf("Injecting %d cores", len(cores))
	for _, core := range cores {
		meta := core.Meta()
		e.Register(meta)
		e.Attach(
			core.Operations()...,
		)
	}
}

// Prime the engine by setting up middleware & default endpoints
func (e *Engine) Prime() {
	e.Log.Debug("Priming the engine")

	// Bind services to the HTTP router
	e.Http.Use(func(c *gin.Context) {
		c.Set("db", e.Database)
		c.Set("log", e.Log)
		c.Next()
	})

	// Set up common inputs
	e.Docs.AddParameter(&OpenAPIParameter{
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
	e.Http.GET("/openapi", e.Docs.SpecHandler)
	e.Http.GET("/docs", e.Docs.ScalarHandler)

	// Add auth endpoints w/o documentation
	e.Http.POST("/auth/callback", e.Auth.CallbackHandler)

	// Attach default endpoints
	e.Attach(
		// Attach the authentication endpoints w/ documentation
		&HTTPOperation{
			Name:        "Login",
			Description: "Endpoint for user login. Redirects to the authentication provider.",
			Method:      "GET",
			Path:        "/auth/login",
			Tag:         "Auth",
			Handler:     e.Auth.LoginHandler,
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
			Handler:     e.Auth.LogoutHandler,
			Auth:        false,
		},

		// Attach the health check endpoint w/ documentation
		&HTTPOperation{
			Name:        "Health Check",
			Description: "Endpoint to check the health of the application. Returns a 200 OK status if the application is running.",
			Method:      "GET",
			Path:        "/health",
			Tag:         "Health",
			Handler:     e.Health.HealthCheckHandler,
			Auth:        false,
		},
	)
}

// Start the engine by running an HTTP server
func (e *Engine) Start() {
	e.Log.Debug("Starting the engine")
	e.Http.Serve()
}
