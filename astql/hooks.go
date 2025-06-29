package astql

import (
	"time"

	"zbz/catalog"
)

// ASTQLHookType represents hook types for ASTQL events
type ASTQLHookType int

// Hook types for ASTQL operations
const (
	// AST generation events
	UniversalASTGenerated ASTQLHookType = 7000
	QueryGenerated        ASTQLHookType = 7001
	QueryGenerationFailed ASTQLHookType = 7002

	// Query execution events
	QueryExecuting ASTQLHookType = 7010
	QueryExecuted  ASTQLHookType = 7011
	QueryFailed    ASTQLHookType = 7012

	// Query optimization events
	QueryOptimizationSuggested ASTQLHookType = 7020
	QueryValidated             ASTQLHookType = 7021
	QueryCached                ASTQLHookType = 7022

	// File operation events
	QueryFileWritten ASTQLHookType = 7030
	QueryFileUpdated ASTQLHookType = 7031
	QueryFileDeleted ASTQLHookType = 7032

	// Provider lifecycle events
	ProviderRegistered   ASTQLHookType = 7100
	ProviderConnected    ASTQLHookType = 7101
	ProviderDisconnected ASTQLHookType = 7102
	ProviderHealthCheck  ASTQLHookType = 7103
)

// String implements capitan.HookType interface
func (h ASTQLHookType) String() string {
	switch h {
	case UniversalASTGenerated:
		return "UniversalASTGenerated"
	case QueryGenerated:
		return "QueryGenerated"
	case QueryGenerationFailed:
		return "QueryGenerationFailed"
	case QueryExecuting:
		return "QueryExecuting"
	case QueryExecuted:
		return "QueryExecuted"
	case QueryFailed:
		return "QueryFailed"
	case QueryOptimizationSuggested:
		return "QueryOptimizationSuggested"
	case QueryValidated:
		return "QueryValidated"
	case QueryCached:
		return "QueryCached"
	case QueryFileWritten:
		return "QueryFileWritten"
	case QueryFileUpdated:
		return "QueryFileUpdated"
	case QueryFileDeleted:
		return "QueryFileDeleted"
	case ProviderRegistered:
		return "ProviderRegistered"
	case ProviderConnected:
		return "ProviderConnected"
	case ProviderDisconnected:
		return "ProviderDisconnected"
	case ProviderHealthCheck:
		return "ProviderHealthCheck"
	default:
		return "UnknownASTQLHookType"
	}
}

// Event structures for hooks

// ASTGeneratedEvent is emitted when a universal AST is generated from a type
type ASTGeneratedEvent struct {
	TypeName  string                 `json:"type_name"`
	Operation string                 `json:"operation"`
	AST       *QueryAST              `json:"ast"`
	Metadata  catalog.ModelMetadata  `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// QueryGeneratedEvent is emitted when a query is generated for a specific provider
type QueryGeneratedEvent struct {
	Provider    string                 `json:"provider"`      // sql, mongo, elastic, etc.
	TypeName    string                 `json:"type_name"`
	Operation   string                 `json:"operation"`     // get, list, create, update, delete
	QueryAST    *QueryAST              `json:"query_ast"`
	Query       string                 `json:"query"`         // The actual query string
	QueryHash   string                 `json:"query_hash"`    // Hash for caching
	NamedParams []string               `json:"named_params"`  // List of parameter names
	Language    string                 `json:"language"`      // sql, mongodb, elasticsearch
	Metadata    catalog.ModelMetadata  `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// QueryExecutingEvent is emitted before query execution
type QueryExecutingEvent struct {
	Provider  string         `json:"provider"`
	TypeName  string         `json:"type_name"`
	Operation string         `json:"operation"`
	Query     string         `json:"query"`
	Params    map[string]any `json:"params"`
	Timestamp time.Time      `json:"timestamp"`
}

// QueryExecutedEvent is emitted after query execution
type QueryExecutedEvent struct {
	Provider  string         `json:"provider"`
	TypeName  string         `json:"type_name"`
	Operation string         `json:"operation"`
	Query     string         `json:"query"`
	Params    map[string]any `json:"params"`
	Duration  time.Duration  `json:"duration"`
	RowCount  int            `json:"row_count"`
	Error     error          `json:"error,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// QueryOptimizationEvent suggests query improvements
type QueryOptimizationEvent struct {
	Provider       string    `json:"provider"`
	TypeName       string    `json:"type_name"`
	Operation      string    `json:"operation"`
	OriginalQuery  string    `json:"original_query"`
	OptimizedQuery string    `json:"optimized_query"`
	Reason         string    `json:"reason"`
	Impact         string    `json:"impact"` // "high", "medium", "low"
	Timestamp      time.Time `json:"timestamp"`
}

// QueryFileEvent is emitted for file operations
type QueryFileEvent struct {
	Provider  string    `json:"provider"`
	TypeName  string    `json:"type_name"`
	Operation string    `json:"operation"`
	FilePath  string    `json:"file_path"`
	Content   string    `json:"content"`
	Action    string    `json:"action"` // written, updated, deleted
	Timestamp time.Time `json:"timestamp"`
}

// ASTQLHookType implements capitan.HookType by having ~int underlying type and String() method