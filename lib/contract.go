package zbz

import (
	"fmt"
	"go.uber.org/zap"
	"strings"
	"time"
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

// Internal registries - contracts manage their own service instances
var (
	databaseRegistry = make(map[string]Database)
	coreRegistry     = make(map[string]Core)
)

// DatabaseContract represents a configuration for database assignment
type DatabaseContract struct {
	BaseContract
	Key    string     // Database registry key ("primary", "analytics", etc.)
	DSN    string     // Database connection string
	Driver string     // Database driver ("postgres", "mysql", etc.)
	Pool   PoolConfig // Connection pool configuration
}

// Database resolves this contract to a Database instance
func (c DatabaseContract) Database() Database {
	key := c.contractKey()
	
	// Return existing instance if found
	if db, exists := databaseRegistry[key]; exists {
		Log.Debug("Retrieved existing database", zap.String("key", key))
		return db
	}
	
	// Create new instance using factory function
	db := CreateDatabaseFromContract(c)
	
	// Store in registry
	databaseRegistry[key] = db
	Log.Info("Registered new database", 
		zap.String("key", key),
		zap.String("driver", c.Driver),
		zap.String("name", c.Name))
	
	return db
}

func (c DatabaseContract) contractKey() string {
	return fmt.Sprintf("db:%s:%s", c.Name, c.Driver)
}

// CoreContract represents a configuration for how a Core should be exposed via HTTP
type CoreContract[T BaseModel] struct {
	BaseContract
	Handlers []string         // Handler names to enable (nil = all handlers)
	DatabaseContract DatabaseContract // Database assignment
}

// Core resolves this contract to a Core instance
func (c CoreContract[T]) Core() Core {
	key := c.contractKey()
	
	// Return existing instance if found
	if core, exists := coreRegistry[key]; exists {
		Log.Debug("Retrieved existing core", zap.String("key", key))
		return core
	}
	
	// Create new instance based on contract
	Log.Debug("Creating new core from contract", 
		zap.String("key", key),
		zap.String("name", c.Name))
	
	// This is the magic - generic contract knows its T type!
	core := NewCore[T](c.Description)
	
	// Store in registry
	coreRegistry[key] = core
	Log.Info("Registered new core", 
		zap.String("key", key),
		zap.String("name", c.Name))
	
	return core
}

func (c CoreContract[T]) contractKey() string {
	return fmt.Sprintf("core:%s:%s", c.Name, c.DatabaseContract.Name)
}

// Inject sets up this contract's complete service stack
func (c CoreContract[T]) Inject() {
	Log.Debug("Injecting core contract", 
		zap.String("name", c.Name),
		zap.String("database", c.DatabaseContract.Name))
	
	// Self-resolve to get core and database instances
	core := c.Core()
	database := c.DatabaseContract.Database()
	
	// Get the global engine instance for HTTP/docs services
	// TODO: This dependency on engine should be removed in favor of direct service access
	engine := GetEngine()
	
	// Register core in engine registry for lookup by name (convention-over-configuration)
	meta := core.Meta()
	engine.RegisterCore(meta.Name, core)
	
	// Register model metadata with the database schema
	database.GetSchema().AddMeta(meta)
	
	// Add to documentation service  
	engine.GetDocs().AddSchema(meta)
	engine.GetDocs().AddTag(meta.Name, c.Description)
	
	// Create table on the assigned database using one-shot execution
	_, err := database.ExecuteOnce(core.Table(), map[string]any{})
	if err != nil {
		Log.Fatal("Failed to create table during injection", 
			zap.String("model", meta.Name),
			zap.String("database", c.DatabaseContract.Name),
			zap.Error(err))
	}
	
	// Register this database with the engine so it can be found by key
	engine.RegisterDatabase(c.DatabaseContract.Key, database)
	
	// Build and validate macro embeds after table exists
	embeds, err := BuildMacroEmbeds(database.GetSchema(), meta)
	if err != nil {
		Log.Fatal("Failed to build macro embeds during injection", 
			zap.String("model", meta.Name),
			zap.String("database", c.DatabaseContract.Name),
			zap.Error(err))
	}
	
	// Store embeds in core and prepare statements on the contract's database
	core.setEmbeds(embeds)
	for _, contract := range core.Contracts() {
		err := database.Prepare(contract)
		if err != nil {
			Log.Fatal("Failed to prepare statement during injection", 
				zap.String("model", meta.Name),
				zap.String("contract", contract.Name),
				zap.Error(err))
		}
	}
	
	// Filter operations based on enabled handlers
	operations := core.Operations()
	if c.Handlers != nil && len(c.Handlers) > 0 {
		filteredOps := make([]*Operation, 0)
		for _, op := range operations {
			for _, enabledHandler := range c.Handlers {
				if strings.Contains(op.Name, enabledHandler) {
					filteredOps = append(filteredOps, op)
					break
				}
			}
		}
		operations = filteredOps
	}
	
	// Add HTTP routes and documentation
	http := engine.GetHTTP()
	docs := engine.GetDocs()
	errorManager := http.GetErrorManager()
	
	for _, op := range operations {
		docs.AddPath(op, errorManager)
		http.AddRoute(op)
	}
	
	Log.Info("Successfully injected core contract", 
		zap.String("name", c.Name),
		zap.Int("operations", len(operations)),
		zap.String("database", c.DatabaseContract.Name))
}

// ContractInjector interface allows system components to work with any contract type
type ContractInjector interface {
	Inject()
}

// TODO: Future contract types following the same pattern:
// AuthContract, HTTPContract, DocsContract, etc.