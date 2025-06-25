package zbz

import (
	"fmt"
	"time"

	"zbz/api/auth"
	"zbz/api/cache"
	"zbz/api/http"
	"zbz/zlog"
)

// ZlogContract represents a configuration for logging assignment
type ZlogContract struct {
	BaseContract
	Driver      string         // Logging provider ("simple", "noop", "zap", etc.)
	Config      map[string]any // Provider-specific configuration
}

// Zlog resolves this contract to a ZlogService instance and sets global singleton
func (c ZlogContract) Zlog() zlog.ZlogService {
	// TODO: Implement with new factory pattern
	// For now, return the global singleton which should be set by provider factories
	return zlog.Zlog
}

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
		zlog.Zlog.Debug("Retrieved existing database", zlog.String("key", key))
		return db
	}
	
	// Create new Database service using the user-provided driver
	db := NewDatabase(c.Driver, c.Name, c.Description)
	
	// Load SQL macros from the internal/macros directory
	if err := db.LoadMacros("internal/macros"); err != nil {
		zlog.Zlog.Warn("Failed to load macros", zlog.Err(err))
	}
	
	// Store in registry
	databaseRegistry[key] = db
	zlog.Zlog.Info("Registered new database", 
		zlog.String("key", key),
		zlog.String("driver", c.Driver.DriverName()),
		zlog.String("name", c.Name))
	
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
		zlog.Zlog.Debug("Retrieved existing cache", zlog.String("key", key))
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
		zlog.Zlog.Fatal("Unsupported cache service", zlog.String("service", c.Service))
	}
	
	// Store in registry
	cacheRegistry[key] = cacheInstance
	zlog.Zlog.Info("Registered new cache", 
		zlog.String("key", key),
		zlog.String("service", c.Service),
		zlog.String("name", c.Name))
	
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
		zlog.Zlog.Debug("Retrieved existing auth", zlog.String("key", key))
		return auth
	}
	
	// Create new Auth service using the user-provided driver
	auth := NewAuth(c.Driver, c.Name, c.Description)
	
	// Store in registry
	authRegistry[key] = auth
	zlog.Zlog.Info("Registered new auth", 
		zlog.String("key", key),
		zlog.String("driver", c.Driver.DriverName()),
		zlog.String("name", c.Name))
	
	return auth
}

func (c AuthContract) contractKey() string {
	return fmt.Sprintf("auth:%s:%s", c.Name, c.Driver.DriverName())
}

// CoreContract represents a configuration for how a Core should be exposed via HTTP
type CoreContract[T BaseModel] struct {
	BaseContract                   // Description used for API resource/tag description
	ModelDescription string        // Description used for the data model schema
	Handlers         []string       // Handler names to enable (nil = all handlers)
	DatabaseContract DatabaseContract // Database assignment
}

// Core resolves this contract to a Core instance
func (c CoreContract[T]) Core() Core {
	key := c.contractKey()
	
	// Return existing instance if found
	if core, exists := coreRegistry[key]; exists {
		zlog.Zlog.Debug("Retrieved existing core", zlog.String("key", key))
		return core
	}
	
	// Create new instance based on contract
	zlog.Zlog.Debug("Creating new core from contract", 
		zlog.String("key", key),
		zlog.String("name", c.Name))
	
	// This is the magic - generic contract knows its T type!
	// Use ModelDescription for the data schema, BaseContract.Description for the resource
	modelDesc := c.ModelDescription
	if modelDesc == "" {
		modelDesc = c.Description // Fallback to resource description if model description not set
	}
	
	// Resolve model description from remark if available
	resolvedModelDesc := Remark.MightGet(modelDesc, modelDesc)
	zlog.Zlog.Debug("Resolved model description", 
		zlog.String("key", modelDesc),
		zlog.String("resolved", resolvedModelDesc[:min(50, len(resolvedModelDesc))]),
		zlog.String("model", c.Name))
	
	// Create core with resolved model description
	core := NewCore[T](resolvedModelDesc)
	
	// Resolve resource description from remark if available
	resolvedResourceDesc := Remark.MightGet(c.Description, c.Description)
	zlog.Zlog.Debug("Resolved resource description", 
		zlog.String("key", c.Description),
		zlog.String("resolved", resolvedResourceDesc[:min(50, len(resolvedResourceDesc))]),
		zlog.String("model", c.Name))
	
	// Set contract metadata with resolved descriptions
	core.setContractMetadata(c.Name, resolvedResourceDesc)
	
	// Store in registry
	coreRegistry[key] = core
	zlog.Zlog.Info("Registered new core", 
		zlog.String("key", key),
		zlog.String("name", c.Name))
	
	return core
}

func (c CoreContract[T]) contractKey() string {
	return fmt.Sprintf("core:%s:%s", c.Name, c.DatabaseContract.Name)
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
		zlog.Zlog.Debug("Retrieved existing HTTP", zlog.String("key", key))
		return httpService
	}
	
	// Create HTTP driver based on contract configuration
	var driver HTTPDriver
	switch c.Driver {
	case "gin", "default", "":
		driver = http.NewGinDriver(c.DevMode)
		zlog.Zlog.Debug("Created Gin HTTP driver", zlog.Bool("devMode", c.DevMode))
	default:
		zlog.Zlog.Fatal("Unsupported HTTP driver", zlog.String("driver", c.Driver))
	}
	
	// Load templates if specified
	if c.TemplatesDir != "" {
		err := driver.LoadTemplates(c.TemplatesDir)
		if err != nil {
			zlog.Zlog.Fatal("Failed to load templates", 
				zlog.String("templates_dir", c.TemplatesDir),
				zlog.Err(err))
		}
	}
	
	// Create HTTP service (includes internal docs)
	httpService := NewHTTP(driver, c.Name, c.Description)
	
	// Store in registry
	httpRegistry[key] = httpService
	zlog.Zlog.Info("Registered new HTTP service with driver", 
		zlog.String("key", key),
		zlog.String("name", c.Name),
		zlog.String("driver", driver.DriverName()),
		zlog.String("templates_dir", c.TemplatesDir))
	
	return httpService
}

func (c HTTPContract) contractKey() string {
	return fmt.Sprintf("http:%s:%s:%s", c.Driver, c.Host, c.Port)
}