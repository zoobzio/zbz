package astql

import (
	"context"
	"time"

	"zbz/capitan"
	"zbz/catalog"
	"zbz/zlog"
)

// GenerateFromType generates queries for a type and emits events
// This is the core function that kicks off the event-driven query generation
func GenerateFromType[T any]() {
	ctx := context.Background()
	metadata := catalog.Select[T]()
	
	zlog.Info("Generating queries from type",
		zlog.String("type_name", metadata.TypeName),
		zlog.Int("field_count", len(metadata.Fields)))

	// Generate all CRUD queries
	queries := GenerateCRUDQueries(metadata)

	// Emit events for each generated query
	for operation, ast := range queries {
		// First emit the universal AST
		capitan.Emit(ctx, UniversalASTGenerated, "astql", ASTGeneratedEvent{
			TypeName:  metadata.TypeName,
			Operation: operation,
			AST:       ast,
			Metadata:  metadata,
			Timestamp: time.Now(),
		}, nil)

		zlog.Debug("Generated universal AST",
			zlog.String("type_name", metadata.TypeName),
			zlog.String("operation", operation),
			zlog.String("target", ast.Target),
			zlog.Int("field_count", len(ast.Fields)),
			zlog.Int("condition_count", len(ast.Conditions)))
	}

	zlog.Info("Query generation completed",
		zlog.String("type_name", metadata.TypeName),
		zlog.Int("query_count", len(queries)))
}

// GenerateFromMetadata generates queries from existing metadata
func GenerateFromMetadata(metadata catalog.ModelMetadata) {
	ctx := context.Background()
	
	zlog.Info("Generating queries from metadata",
		zlog.String("type_name", metadata.TypeName),
		zlog.Int("field_count", len(metadata.Fields)))

	// Generate all CRUD queries
	queries := GenerateCRUDQueries(metadata)

	// Emit events for each generated query
	for operation, ast := range queries {
		capitan.Emit(ctx, UniversalASTGenerated, "astql", ASTGeneratedEvent{
			TypeName:  metadata.TypeName,
			Operation: operation,
			AST:       ast,
			Metadata:  metadata,
			Timestamp: time.Now(),
		}, nil)

		zlog.Debug("Generated universal AST from metadata",
			zlog.String("type_name", metadata.TypeName),
			zlog.String("operation", operation))
	}
}

// ValidateAST validates a query AST and emits validation events
func ValidateAST(ast *QueryAST, metadata catalog.ModelMetadata) error {
	ctx := context.Background()
	
	zlog.Debug("Validating AST",
		zlog.String("operation", ast.Operation.String()),
		zlog.String("target", ast.Target))

	err := ast.Validate()
	
	if err != nil {
		zlog.Warn("AST validation failed",
			zlog.String("operation", ast.Operation.String()),
			zlog.String("target", ast.Target),
			zlog.Err(err))
		return err
	}

	// Emit validation success event
	capitan.Emit(ctx, QueryValidated, "astql", struct {
		TypeName  string    `json:"type_name"`
		Operation string    `json:"operation"`
		Target    string    `json:"target"`
		Valid     bool      `json:"valid"`
		Timestamp time.Time `json:"timestamp"`
	}{
		TypeName:  metadata.TypeName,
		Operation: ast.Operation.String(),
		Target:    ast.Target,
		Valid:     true,
		Timestamp: time.Now(),
	}, nil)

	zlog.Debug("AST validation passed",
		zlog.String("operation", ast.Operation.String()),
		zlog.String("target", ast.Target))

	return nil
}

// GenerateWithCustomAST allows for custom AST generation and emits appropriate events
func GenerateWithCustomAST(typeName string, operation string, ast *QueryAST, metadata catalog.ModelMetadata) {
	ctx := context.Background()
	
	zlog.Info("Generating with custom AST",
		zlog.String("type_name", typeName),
		zlog.String("operation", operation))

	// Validate the custom AST
	if err := ast.Validate(); err != nil {
		zlog.Error("Custom AST validation failed",
			zlog.String("type_name", typeName),
			zlog.String("operation", operation),
			zlog.Err(err))
		return
	}

	// Emit the universal AST event
	capitan.Emit(ctx, UniversalASTGenerated, "astql", ASTGeneratedEvent{
		TypeName:  typeName,
		Operation: operation,
		AST:       ast,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}, nil)

	zlog.Info("Custom AST generated successfully",
		zlog.String("type_name", typeName),
		zlog.String("operation", operation))
}

// GetSupportedOperations returns the list of operations that ASTQL can generate
func GetSupportedOperations() []string {
	return []string{
		"get",    // Single record retrieval
		"list",   // Multiple record retrieval with pagination
		"create", // Insert new record
		"update", // Update existing record
		"delete", // Delete record (soft or hard)
		"count",  // Count records
	}
}

// GetSupportedHints returns provider-specific hints that ASTQL recognizes
func GetSupportedHints() map[string][]string {
	return map[string][]string{
		"sql": {
			"index:btree",
			"index:hash",
			"index:gin",
			"index:unique",
			"unique",
			"security:tenant",
		},
		"mongo": {
			"index:text",
			"index:compound",
			"shard_key",
		},
		"elastic": {
			"type:text",
			"type:keyword",
			"type:date",
			"analyzer:standard",
		},
	}
}