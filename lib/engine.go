package zbz

import "net/http"

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
	return &Engine{
		Auth:     a,
		Database: db,
		Docs:     d,
		Health:   hp,
		Http:     h,
		Log:      l,
	}
}

// Inject registers HTTP operations with the router & documentation service.
func (e *Engine) Inject(ops ...*HTTPOperation) {
	e.Log.Debugf("Injecting %d operations into the engine", len(ops))
	for _, op := range ops {
		e.Docs.AddPath(op)
		e.Http.Register(op)
	}
}

// Prime the engine by setting up default endpoints
func (e *Engine) Prime() {
	e.Log.Debug("Priming the engine")

	// Add documentation endpoints
	e.Http.GET("/openapi", e.Docs.SpecHandler)
	e.Http.GET("/docs", e.Docs.ScalarHandler)

	// Register default endpoints
	e.Inject(
		// Register the authentication endpoints w/ documentation
		&HTTPOperation{
			Name:        "User Login",
			Description: "Endpoint for user login. Redirects to the authentication provider.",
			Method:      "GET",
			Path:        "/auth/login",
			Tag:         "Auth",
			Handler:     e.Auth.LoginHandler,
			Response: &HTTPResponse{
				Status: http.StatusTemporaryRedirect,
			},
			Auth: true,
		},
		&HTTPOperation{
			Name:        "Callback",
			Description: "Callback endpoint for the authentication provider. Handles the response after user login.",
			Method:      "POST",
			Path:        "/auth/callback",
			Tag:         "Auth",
			Handler:     e.Auth.CallbackHandler,
			Auth:        false,
		},
		&HTTPOperation{
			Name:        "User Logout",
			Description: "Endpoint for user logout. Clears the session and redirects to the home page.",
			Method:      "GET",
			Path:        "/auth/logout",
			Tag:         "Auth",
			Handler:     e.Auth.LogoutHandler,
			Auth:        false,
		},

		// Register the health check endpoint w/ documentation
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
