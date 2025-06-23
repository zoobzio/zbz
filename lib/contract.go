package zbz

import (
	"fmt"
	"time"

	"zbz/lib/auth"
	"zbz/lib/cache"
	"zbz/lib/http"
	"zbz/shared/logger"
)

// BaseContract provides common fields for all contract types across the framework
// Contracts represent configurations that get exchanged for service implementations
type BaseContract struct {
	Name        string // Generic identifier for the contracted resource
	Description string // Human-readable description
}

// PoolConfig represents connection pool configuration
type PoolConfig struct {
	MaxOpen     int           // Maximum open connections (0 = unlimited)
	MaxIdle     int           // Maximum idle connections
	MaxLifetime time.Duration // Maximum connection lifetime
	MaxIdleTime time.Duration // Maximum connection idle time
}


// Response represents a response structure for an HTTP operation
type Response struct {
	Status int
	Ref    string
	Type   string
	Errors []int
}

// HandlerContract represents a given API action that can be used to register endpoints & spawn documentation
type HandlerContract struct {
	Name        string
	Description string
	Tag         string
	Method      string
	Path        string
	Handler     func(RequestContext) `json:"-"`  // Self-contained function signature
	Parameters  []string
	Query       []string
	RequestBody string
	Response    *Response
	Auth        bool
	Scope       string // Auth permission scope (e.g., "read:users")
}

// TrustedSQLIdentifier represents a SQL identifier that has been validated
// Can only be created through validation functions, preventing SQL injection
type TrustedSQLIdentifier struct {
	value string
}

// String returns the validated SQL identifier value
func (t TrustedSQLIdentifier) String() string {
	return t.value
}

// MacroEmbeds holds validated SQL identifiers for macro interpolation
// Prevents user input from reaching raw SQL interpolation
type MacroEmbeds struct {
	Table   TrustedSQLIdentifier  // Table name: "users"
	Columns TrustedSQLIdentifier  // Column list: "id, name, email"
	Values  TrustedSQLIdentifier  // Named params: ":id, :name, :email"
	Updates TrustedSQLIdentifier  // Update assignments: "name = :name, email = :email"
}

// MacroContract defines the necessary data to implement a macro as a query
type MacroContract struct {
	Name  string
	Macro string
	Embed MacroEmbeds
}

// GetName returns the contract name (implements database.MacroContract interface)
func (m *MacroContract) GetName() string {
	return m.Name
}

// GetMacro returns the macro name (implements database.MacroContract interface)
func (m *MacroContract) GetMacro() string {
	return m.Macro
}

// GetEmbeds returns the macro embeds (implements database.MacroContract interface)
func (m *MacroContract) GetEmbeds() any {
	return m.Embed
}


// Internal registries - contracts manage their own service instances
var (
	databaseRegistry = make(map[string]Database)
	authRegistry     = make(map[string]Auth)
	httpRegistry     = make(map[string]HTTP)
	cacheRegistry    = make(map[string]Cache)
	coreRegistry     = make(map[string]Core)
)

// DatabaseContract represents a configuration for database assignment
type DatabaseContract struct {
	BaseContract
	Driver DatabaseDriver  // User-initialized database driver
}

// Database resolves this contract to a Database instance
func (c DatabaseContract) Database() Database {
	key := c.contractKey()
	
	// Return existing instance if found
	if db, exists := databaseRegistry[key]; exists {
		logger.Debug("Retrieved existing database", logger.String("key", key))
		return db
	}
	
	// Create new Database service using the user-provided driver
	db := NewDatabase(c.Driver, c.Name, c.Description)
	
	// Load SQL macros from the internal/macros directory
	if err := db.LoadMacros("internal/macros"); err != nil {
		logger.Warn("Failed to load macros", logger.Err(err))
	}
	
	// Store in registry
	databaseRegistry[key] = db
	logger.Info("Registered new database", 
		logger.String("key", key),
		logger.String("driver", c.Driver.DriverName()),
		logger.String("name", c.Name))
	
	return db
}

func (c DatabaseContract) contractKey() string {
	return fmt.Sprintf("db:%s:%s", c.Name, c.Driver.DriverName())
}

// CacheContract represents a configuration for cache assignment
type CacheContract struct {
	BaseContract
	Service string            // Cache service ("redis", "memory")
	URL     string            // Connection URL (for Redis)
	Config  map[string]any    // Service-specific configuration
}

// Cache resolves this contract to a Cache instance
func (c CacheContract) Cache() Cache {
	key := c.contractKey()
	
	// Return existing instance if found
	if cache, exists := cacheRegistry[key]; exists {
		logger.Debug("Retrieved existing cache", logger.String("key", key))
		return cache
	}
	
	// Create new instance using factory function
	var cacheInstance Cache
	switch c.Service {
	case "redis":
		cacheInstance = cache.NewRedisCache(c.URL)
	case "memory":
		cacheInstance = cache.NewMemoryCache()
	default:
		logger.Fatal("Unsupported cache service", logger.String("service", c.Service))
	}
	
	// Store in registry
	cacheRegistry[key] = cacheInstance
	logger.Info("Registered new cache", 
		logger.String("key", key),
		logger.String("service", c.Service),
		logger.String("name", c.Name))
	
	return cacheInstance
}

func (c CacheContract) contractKey() string {
	return fmt.Sprintf("cache:%s:%s", c.Name, c.Service)
}

// AuthContract represents a configuration for auth service assignment
type AuthContract struct {
	BaseContract
	Driver auth.AuthDriver  // User-initialized auth driver
}

// Auth resolves this contract to an Auth instance
func (c AuthContract) Auth() Auth {
	key := c.contractKey()
	
	// Return existing instance if found
	if auth, exists := authRegistry[key]; exists {
		logger.Debug("Retrieved existing auth", logger.String("key", key))
		return auth
	}
	
	// Create new Auth service using the user-provided driver
	auth := NewAuth(c.Driver, c.Name, c.Description)
	
	// Store in registry
	authRegistry[key] = auth
	logger.Info("Registered new auth", 
		logger.String("key", key),
		logger.String("driver", c.Driver.DriverName()),
		logger.String("name", c.Name))
	
	return auth
}

func (c AuthContract) contractKey() string {
	return fmt.Sprintf("auth:%s:%s", c.Name, c.Driver.DriverName())
}

// CoreContract represents a configuration for how a Core should be exposed via HTTP
type CoreContract[T BaseModel] struct {
	BaseContract
	Handlers         []string         // Handler names to enable (nil = all handlers)
	DatabaseContract DatabaseContract // Database assignment
}

// Core resolves this contract to a Core instance
func (c CoreContract[T]) Core() Core {
	key := c.contractKey()
	
	// Return existing instance if found
	if core, exists := coreRegistry[key]; exists {
		logger.Debug("Retrieved existing core", logger.String("key", key))
		return core
	}
	
	// Create new instance based on contract
	logger.Debug("Creating new core from contract", 
		logger.String("key", key),
		logger.String("name", c.Name))
	
	// This is the magic - generic contract knows its T type!
	core := NewCore[T](c.Description)
	
	// Store in registry
	coreRegistry[key] = core
	logger.Info("Registered new core", 
		logger.String("key", key),
		logger.String("name", c.Name))
	
	return core
}

func (c CoreContract[T]) contractKey() string {
	return fmt.Sprintf("core:%s:%s", c.Name, c.DatabaseContract.Name)
}

// Inject sets up this contract's complete service stack
func (c CoreContract[T]) Inject() {
	logger.Debug("Injecting core contract", 
		logger.String("name", c.Name),
		logger.String("database", c.DatabaseContract.Name))
	
	// Self-resolve to get core and database instances
	core := c.Core()
	database := c.DatabaseContract.Database()
	
	// Set the resolved database and contract metadata on the core
	core.setDatabase(database)
	core.setContractMetadata(c.Name, c.Description)
	
	// Register model metadata with the database schema  
	meta := core.Meta()
	database.GetSchema().AddMeta(meta)
	
	// Create table on the assigned database using one-shot execution
	_, err := database.ExecuteOnce(core.Table(), map[string]any{})
	if err != nil {
		logger.Fatal("Failed to create table during injection", 
			logger.String("model", meta.Name),
			logger.String("database", c.DatabaseContract.Name),
			logger.Err(err))
	}
	
	// Build and validate macro embeds after table exists
	embeds, err := BuildMacroEmbeds(database.GetSchema(), meta)
	if err != nil {
		logger.Fatal("Failed to build macro embeds during injection", 
			logger.String("model", meta.Name),
			logger.String("database", c.DatabaseContract.Name),
			logger.Err(err))
	}
	
	// Store embeds in core and prepare statements on the contract's database
	core.setEmbeds(embeds)
	for _, contract := range core.MacroContracts() {
		err := database.Prepare(contract)
		if err != nil {
			logger.Fatal("Failed to prepare statement during injection", 
				logger.String("model", meta.Name),
				logger.String("contract", contract.Name),
				logger.Err(err))
		}
	}
	
	logger.Info("Successfully created core from contract", 
		logger.String("name", c.Name),
		logger.String("database", c.DatabaseContract.Name))
}

// ContractInjector interface allows system components to work with any contract type
type ContractInjector interface {
	Inject()
}

// HTTPContract represents a configuration for HTTP server setup
type HTTPContract struct {
	BaseContract
	Driver       string            // HTTP framework driver ("gin", "fasthttp", "echo", etc.)
	Port         string            // HTTP server port (e.g., "8080")
	Host         string            // HTTP server host (e.g., "0.0.0.0")
	DevMode      bool              // Development mode flag
	TemplatesDir string            // HTML templates directory
	Headers      map[string]string // Default headers to apply
}

// HTTP resolves this contract to an HTTP instance with internal docs service
func (c HTTPContract) HTTP() HTTP {
	key := c.contractKey()
	
	// Return existing instance if found
	if httpService, exists := httpRegistry[key]; exists {
		logger.Debug("Retrieved existing HTTP", logger.String("key", key))
		return httpService
	}
	
	// Create HTTP driver based on contract configuration
	var driver HTTPDriver
	switch c.Driver {
	case "gin", "default", "":
		driver = http.NewGinDriver(c.DevMode)
		logger.Debug("Created Gin HTTP driver", logger.Bool("devMode", c.DevMode))
	default:
		logger.Fatal("Unsupported HTTP driver", logger.String("driver", c.Driver))
	}
	
	// Create HTTP service (includes internal docs)
	httpService := NewHTTP(driver, c.Name, c.Description)
	
	// Store in registry
	httpRegistry[key] = httpService
	logger.Info("Registered new HTTP service with driver", 
		logger.String("key", key),
		logger.String("name", c.Name),
		logger.String("driver", driver.DriverName()))
	
	return httpService
}

func (c HTTPContract) contractKey() string {
	return fmt.Sprintf("http:%s:%s:%s", c.Driver, c.Host, c.Port)
}