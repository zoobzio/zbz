package zbz

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
	h := NewHTTP(l, c)
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

// Inject registers a single HTTP operation with the engine's HTTP router & docuementer.
func (e *Engine) Inject(ops ...*HTTPOperation) {
	for _, op := range ops {
		e.Docs.AddPath(op)
		e.Http.Register(op)
	}
}

// Prime the engine by setting up default endpoints
func (e *Engine) Prime() {
	e.Log.Debug("Priming the engine...")
	e.Inject(
		// Register the authentication endpoints
		&HTTPOperation{
			Name:        "AuthLogin",
			Summary:     "User Login",
			Description: "Endpoint for user login. Redirects to the authentication provider.",
			Method:      "GET",
			Path:        "/auth/login",
			Tag:         "Auth",
			Handler:     e.Auth.LoginHandler,
			Auth:        false,
		},
		&HTTPOperation{
			Name:        "AuthCallback",
			Summary:     "Callback",
			Description: "Callback endpoint for the authentication provider. Handles the response after user login.",
			Method:      "POST",
			Path:        "/auth/callback",
			Tag:         "Auth",
			Handler:     e.Auth.CallbackHandler,
			Auth:        false,
		},
		&HTTPOperation{
			Name:        "AuthLogout",
			Summary:     "User Logout",
			Description: "Endpoint for user logout. Clears the session and redirects to the home page.",
			Method:      "GET",
			Path:        "/auth/logout",
			Tag:         "Auth",
			Handler:     e.Auth.LogoutHandler,
			Auth:        false,
		},

		// Register the health check endpoint
		&HTTPOperation{
			Name:        "HealthCheck",
			Summary:     "Health Check",
			Description: "Endpoint to check the health of the application. Returns a 200 OK status if the application is running.",
			Method:      "GET",
			Path:        "/health",
			Tag:         "Health",
			Handler:     e.Health.HealthCheckHandler,
			Auth:        false,
		},

		// Register the API documentation endpoints
		&HTTPOperation{
			Name:        "OpenAPISpec",
			Summary:     "OpenAPI Specification",
			Description: "Endpoint to retrieve the OpenAPI specification for the API. Returns a YAML representation of the API documentation.",
			Method:      "GET",
			Path:        "/openapi",
			Tag:         "Docs",
			Handler:     e.Docs.SpecHandler,
			Auth:        false,
		},
		&HTTPOperation{
			Name:        "OpenAPIDocs",
			Summary:     "Scalar",
			Description: "Endpoint to serve the documentation site. Returns the HTML page with the API documentation.",
			Method:      "GET",
			Path:        "/docs",
			Tag:         "Docs",
			Handler:     e.Docs.ScalarHandler,
			Auth:        false,
		},
	)
}

// Start the engine by running an HTTP server
func (e *Engine) Start() {
	e.Log.Debug("Starting the engine...")
	e.Http.Serve()
}
