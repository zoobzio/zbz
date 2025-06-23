# ZBZ Framework - Claude Context

## Project Overview
ZBZ is an opinionated Go framework for building APIs with automatic CRUD generation, OpenAPI documentation, and structured error handling. The framework follows a "convention-over-configuration" approach with strong type safety and performance-first design principles.

## Architecture Principles
- **Zero Dependencies Between Services**: Each service (HTTP, Database, Auth, Docs) operates independently to support "bring-your-own-X" implementations
- **Interface-Driven Design**: All core functionality uses interfaces with concrete implementations (e.g., `HTTP` interface → `zHTTP` implementation)
- **Convention Over Configuration**: Generate CRUD operations, docs, and error handling automatically from model definitions
- **Performance First**: Use standard zap logger with typed fields, avoid reflection where possible

## Current Architecture

### Core Components
```
lib/
├── core.go        # Generic CRUD operations via Core[T] interface
├── engine.go      # Service orchestration and dependency injection
├── http.go        # HTTP server with auto-error middleware
├── database.go    # PostgreSQL with macro-based queries
├── docs.go        # Auto-generated OpenAPI documentation
├── error.go       # Standardized error handling system
├── auth.go        # OIDC authentication
├── logger.go      # Structured logging with zap
├── model.go       # Base model with common fields
├── validate.go    # Validation with go-playground/validator
└── macros/        # SQL query templates (.sqlx files)
```

### Key Patterns
- **Generic CRUD**: `NewCore[T BaseModel]()` auto-generates CRUD endpoints
- **SQL Macros**: Template-based queries with embedded parameters
- **Auto-Error Handling**: Middleware automatically converts `ctx.Status(404)` to structured error responses
- **Type-Safe Logging**: `Log.Info("message", zap.String("key", value))`

## Current State & Recent Changes

### ✅ Completed
- **Error Handling System**: Two-layer approach with auto-error middleware
  - Layer 1: `ctx.Set("error_message", "custom")` for per-request messages
  - Layer 2: ErrorManager for system-wide defaults
  - Handlers now just call `ctx.Status(http.StatusNotFound)` 
- **Logger Migration**: Moved from sugared to standard zap logger for performance
- **Service Independence**: Error handling moved from Engine to HTTP layer
- **Multi-Database Architecture**: Support for multiple database connections
  - Database adapters in `lib/database/` directory with PostgreSQL implementation
  - Each database owns its own Schema instance (no shared schemas)
  - DatabaseContract system for assigning cores to specific databases
  - Engine supports registering multiple databases with string keys
  - Schema endpoints exposed per database: `/schema/{database_key}`

### 🔧 Current Architecture Strengths
- Clean separation of concerns between services
- Automatic CRUD generation reduces boilerplate
- Built-in observability (metrics, tracing, logging)
- Strong type safety with Go generics

### ⚠️ Known Technical Debt
1. **SQL Injection Risk**: Macro interpolation at `lib/macro.go:56`
   - TODO comment about sanitization
   - Raw SQL replacement without validation

2. **Limited Query Flexibility**: 
   - Only basic CRUD operations supported
   - No filtering, pagination, or complex joins
   - No custom query support

3. **Credentials Logging Risk**: Database contracts contain sensitive DSN strings
   - DSN contains database passwords and connection details
   - Currently manually avoiding logging DSN fields
   - **NEEDS**: Privatization service to automatically redact sensitive fields from logs
   - Should support struct tags like `json:"-"` but for logging (`log:"-"` or `private:"true"`)
   - Must integrate with zap logger to automatically filter sensitive data

## Planned Features & Architecture Decisions

### 🎯 Near-Term Priorities

#### 1. Contract Matrix System (Critical Architecture Improvement)
**Goal**: Complete contract-based dependency injection system
**Vision**: Every service (Database, Handler, Validator, Auth, Logger, etc.) becomes contract-driven, enabling true "bring-your-own-X" architecture through a contract matrix where users can mix and match any combination of services.

**Benefits**:
- True BYOB architecture - users exchange contracts for configured instances
- Engine becomes contract registry/factory: `engine.GetCore[User](CoreContract{...})`
- Complete service flexibility - any service can be replaced through contracts
- Consistent patterns across all framework components

**Implementation**: All services get corresponding contracts (DatabaseContract, HandlerContract, ValidatorContract, etc.) and CoreContract orchestrates the complete service matrix.

#### 2. ZBZ CLI & TUI Development Tools
**Goal**: Create comprehensive developer experience tools
**Architecture Vision**:
- **CLI**: Command-line interface for scaffolding, monitoring, and operations
- **TUI**: Terminal user interface for interactive development environment

**CLI Command Structure**:
```bash
# Project Management
zbz init <project-name>           # Scaffold new zbz project
zbz generate model <ModelName>    # Generate model with CRUD endpoints
zbz generate migration <name>     # Create database migration
zbz dev                          # Hot-reload development server

# Monitoring & Logs  
zbz logs [--tail] [--service=name] [--grep=pattern] [--json]
zbz status                       # Health check all services
zbz metrics [--service=name]     # View service metrics

# Database Operations
zbz db query "SQL"              # Execute SQL query with formatted output
zbz db migrate [up|down|status] # Migration management  
zbz db seed                     # Run database seeders
zbz db shell                    # Interactive SQL shell
zbz db backup/restore           # Database backup operations

# API & Documentation
zbz docs serve [--port=8081]    # Serve docs locally
zbz docs generate [--output=./docs] # Export static documentation
zbz api test [--endpoint=/user] # API endpoint testing
zbz api validate               # Validate OpenAPI spec

# Deployment & Operations
zbz deploy [environment]        # Deploy to staging/production
zbz config validate           # Validate configuration
```

**TUI Interface Design**:
Multi-pane dashboard similar to lazydocker/lazygit:
- **Main View**: Service overview with health status
- **Logs Panel**: Live, filterable log streaming with syntax highlighting
- **Database Explorer**: Interactive table browser and query execution
- **API Tester**: Send requests, view responses, inspect headers
- **Metrics Dashboard**: Real-time performance graphs
- **Config Manager**: Environment variable and setting management

**Implementation Recommendations**:

*CLI Framework*:
- Use [Cobra](https://github.com/spf13/cobra) for command structure
- [Viper](https://github.com/spf13/viper) for configuration management
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for styled terminal output
- [Table Writer](https://github.com/olekukonko/tablewriter) for formatted data display

*TUI Framework*:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) (modern alternative to gocui)
- [Bubbles](https://github.com/charmbracelet/bubbles) for pre-built components
- Alternative: [tview](https://github.com/rivo/tview) for more traditional approach

*Service Integration*:
- HTTP API client for interacting with running zbz applications
- Direct database connection for db commands (reuse existing database layer)
- Loki API client for log aggregation (enhance existing cmd/zlog)
- OpenTelemetry client for metrics collection
- File system operations for scaffolding and generation

*Project Structure*:
```
cmd/
├── zbz/              # Main CLI entry point
│   ├── main.go
│   ├── cmd/          # Cobra command definitions
│   │   ├── init.go
│   │   ├── generate.go
│   │   ├── logs.go
│   │   ├── db.go
│   │   └── tui.go
│   └── internal/     # CLI-specific logic
│       ├── scaffolder/
│       ├── client/   # API clients
│       └── tui/      # TUI components
└── zlog/             # Enhanced from existing implementation
```

#### 2. Query System Enhancement
**Current Limitation**: Only basic CRUD via SQL macros
**Planned Approach**: Design flexible query builder or enhanced macro system
- Support filtering, sorting, pagination
- Maintain type safety
- Consider relationship handling

#### 3. Migration System
**Current Gap**: Tables created on startup, no versioning
**Requirements**: 
- Schema evolution support
- Version tracking
- Rollback capabilities

#### 4. Natural Language Query (NLQ) System
**Goal**: AI-first search capabilities for simplifying UI and data access
**Architecture Vision**:
- **Per-Core NLQ**: Natural language queries on individual model types
- **Global NLQ**: Cross-model search across all registered cores
- **Query Translation**: Convert natural language to type-safe database queries
- **UI Integration**: Simple search interface that replaces complex filtering UIs

**NLQ Implementation Strategy**:
```go
// Per-core natural language queries
userCore := zbz.NewCore[User]("User management")
results, err := userCore.NLQ("find all users from california who joined last month")

// Global search across all models
engine := zbz.NewEngine()
results, err := engine.GlobalNLQ("show me recent activity for john@example.com")
```

**Technical Requirements**:
- **Query Parser**: LLM integration for natural language interpretation
- **Schema Awareness**: AI model understands available fields and relationships
- **Type Safety**: Generated queries maintain compile-time safety
- **Caching**: Cache parsed queries for performance
- **Fallback**: Graceful degradation to traditional filtering when NLQ fails

**Integration Points**:
- Extend existing `Core[T]` interface with NLQ methods
- Integrate with SQL macro system for query generation
- Add NLQ endpoints to auto-generated REST APIs
- Include NLQ examples in OpenAPI documentation

**Benefits for UI Development**:
- Replace complex filter forms with simple search boxes
- Reduce frontend complexity for data querying
- Enable non-technical users to perform advanced searches
- Provide contextual search across related data models

### 🔮 Long-Term Vision

#### Bring-Your-Own-X Architecture
- **Database**: Support MySQL, SQLite beyond PostgreSQL
- **Auth**: Pluggable auth providers beyond OIDC
- **HTTP**: Custom router implementations
- **Validation**: Pluggable validation providers

#### Advanced Features
- **Relationship Support**: Foreign keys, joins, eager loading
- **Caching Layer**: Built-in caching with configurable backends
- **Rate Limiting**: Request throttling and protection
- **API Versioning**: Built-in version management

## Development Guidelines

### Code Style
- Use interface + implementation pattern (`Interface` → `zImplementation`)
- Keep services dependency-free from each other
- Log with structured fields: `Log.Info("msg", zap.String("key", val))`
- Use `ctx.Status(code)` for errors, not manual JSON responses

### Adding New Features
1. Design interface first
2. Consider how it integrates with existing services
3. Maintain service independence
4. Update this CLAUDE.md with architectural decisions

### Current Build Commands
```bash
go build ./cmd/example  # Build example application
go run ./cmd/example    # Run development server
```

### Testing Strategy
- Look for existing test patterns in codebase
- Check README for test commands
- Framework currently lacks comprehensive test coverage (improvement opportunity)

## Files to Know

### Entry Points
- `cmd/example/main.go` - Example application using the framework
- `lib/engine.go` - Main orchestration and service setup

### Core Interfaces
- `lib/core.go` - CRUD operations interface
- `lib/http.go` - HTTP server interface
- `lib/error.go` - Error management interface

### Key Implementation Files
- `lib/database.go` - PostgreSQL + macro system
- `lib/docs.go` - OpenAPI generation
- `lib/model.go` - Base model and helpers

### Configuration
- Environment variables handled in `lib/config.go`
- No explicit config files mentioned yet

---

*This document should be updated as architectural decisions are made and features are implemented.*