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

// Prime the engine by setting up default endpoints
func (e *Engine) Prime() {
	e.Log.Info("Priming the engine...")

	// Register the authentication endpoints
	e.Http.GET("/auth/login", e.Auth.LoginHandler)
	e.Http.POST("/auth/logout", e.Auth.LogoutHandler)
	e.Http.POST("/auth/callback", e.Auth.CallbackHandler)

	// Register the documentation endpoints
	e.Http.GET("/openapi", e.Docs.SpecHandler)
	e.Http.GET("/docs", e.Docs.ScalarHandler)

	// Register the health check endpoints
	e.Http.GET("/health", e.Health.HealthCheckHandler)
}

// Start the engine by running an HTTP server
func (e *Engine) Start() {
	e.Log.Info("Starting the engine...")
	e.Http.Serve()
}
