package astql

import (
	"fmt"
	"strings"

	"zbz/catalog"
)

// ParseFromMetadata generates a QueryAST from catalog metadata
func ParseFromMetadata(metadata catalog.ModelMetadata, operation OperationType) (*QueryAST, error) {
	ast := &QueryAST{
		Operation:  operation,
		Target:     strings.ToLower(metadata.TypeName) + "s", // Simple pluralization
		Fields:     []Field{},
		Conditions: []Condition{},
		Ordering:   []Order{},
		Hints:      []Hint{},
	}

	// Parse fields from metadata
	for _, field := range metadata.Fields {
		// Skip fields marked as internal or write-only
		if field.JSONName == "-" {
			continue
		}

		// Use JSON name if available, otherwise use field name
		fieldName := field.Name
		if field.JSONName != "" && field.JSONName != "-" {
			fieldName = field.JSONName
		}

		// Use database column name if available
		if field.DBColumn != "" {
			fieldName = field.DBColumn
		}

		// Add field to AST
		ast.Fields = append(ast.Fields, Field{
			Name: fieldName,
		})

		// Check for special tags that affect query generation
		for tagKey, tagValue := range field.Tags {
			switch tagKey {
			case "astql":
				parseASTQLTag(ast, field, tagValue)
			case "db":
				// Override field name with database column name
				if tagValue != "" && tagValue != "-" {
					ast.Fields[len(ast.Fields)-1].Name = tagValue
				}
			}
		}
	}

	// Add default conditions based on operation
	switch operation {
	case OpSelect, OpUpdate, OpDelete:
		// Add soft delete check if applicable
		if hasField(metadata, "deleted_at") {
			ast.Conditions = append(ast.Conditions, Condition{
				Field:     "deleted_at",
				Operator:  IS_NULL,
				Value:     nil,
				Logical:   AND,
				ParamName: "deleted_check",
			})
		}
		
		// For single-record operations, add ID condition
		if operation != OpSelect || !isListOperation(metadata) {
			ast.Conditions = append(ast.Conditions, Condition{
				Field:     "id",
				Operator:  EQ,
				Value:     nil, // Will be filled at execution time
				Logical:   AND,
				ParamName: "id",
			})
		}
		
	case OpInsert:
		// Prepare values map for insert
		ast.Values = []map[string]any{{}}
	}

	// Add default ordering for list operations
	if operation == OpSelect && isListOperation(metadata) {
		// Default to created_at DESC if field exists
		if hasField(metadata, "created_at") {
			ast.Ordering = append(ast.Ordering, Order{
				Field:     "created_at",
				Direction: DESC,
			})
		}
	}

	return ast, nil
}

// parseASTQLTag parses astql struct tags for query hints
func parseASTQLTag(ast *QueryAST, field catalog.FieldMetadata, tagValue string) {
	parts := strings.Split(tagValue, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		
		if strings.HasPrefix(part, "index:") {
			indexType := strings.TrimPrefix(part, "index:")
			ast.Hints = append(ast.Hints, Hint{
				Provider: "sql",
				Type:     "index",
				Value:    fmt.Sprintf("%s:%s", field.Name, indexType),
			})
		} else if strings.HasPrefix(part, "relation:") {
			// Parse relationship hints for JOIN generation
			_ = strings.TrimPrefix(part, "relation:")
			// TODO: Generate JOIN clauses based on relationships
		} else if part == "unique" {
			ast.Hints = append(ast.Hints, Hint{
				Provider: "sql",
				Type:     "unique",
				Value:    field.Name,
			})
		} else if strings.HasPrefix(part, "security:") {
			// Security hints for multi-tenancy
			secType := strings.TrimPrefix(part, "security:")
			if secType == "tenant" {
				// Add tenant condition to all queries
				ast.Conditions = append(ast.Conditions, Condition{
					Field:     field.Name,
					Operator:  EQ,
					Value:     nil, // Will be filled from context
					Logical:   AND,
					ParamName: "tenant_id",
				})
			}
		}
	}
}

// hasField checks if a field exists in the metadata
func hasField(metadata catalog.ModelMetadata, fieldName string) bool {
	for _, field := range metadata.Fields {
		if field.Name == fieldName || field.JSONName == fieldName || field.DBColumn == fieldName {
			return true
		}
		// Check database column name in tags
		if dbCol, exists := field.Tags["db"]; exists && dbCol == fieldName {
			return true
		}
	}
	return false
}

// isListOperation determines if this is a list/collection query
func isListOperation(metadata catalog.ModelMetadata) bool {
	// For now, we'll generate both single and list queries
	// This could be enhanced with struct tags or conventions
	return true
}

// GenerateCRUDQueries generates all CRUD queries for a type
func GenerateCRUDQueries(metadata catalog.ModelMetadata) map[string]*QueryAST {
	queries := make(map[string]*QueryAST)

	// Get single record
	if ast, err := ParseFromMetadata(metadata, OpSelect); err == nil {
		queries["get"] = ast
	}

	// List records
	if ast, err := ParseFromMetadata(metadata, OpSelect); err == nil {
		// Remove ID condition for list operation
		filtered := []Condition{}
		for _, cond := range ast.Conditions {
			if cond.Field != "id" {
				filtered = append(filtered, cond)
			}
		}
		ast.Conditions = filtered
		
		// Add pagination
		limit := 100
		offset := 0
		ast.Limit = &limit
		ast.Offset = &offset
		
		queries["list"] = ast
	}

	// Create record
	if ast, err := ParseFromMetadata(metadata, OpInsert); err == nil {
		ast.Returning = []string{"*"}
		queries["create"] = ast
	}

	// Update record
	if ast, err := ParseFromMetadata(metadata, OpUpdate); err == nil {
		ast.Returning = []string{"*"}
		queries["update"] = ast
	}

	// Delete record (soft delete if deleted_at exists)
	if ast, err := ParseFromMetadata(metadata, OpDelete); err == nil {
		if hasField(metadata, "deleted_at") {
			// Convert to UPDATE for soft delete
			ast.Operation = OpUpdate
			ast.Updates = map[string]any{
				"deleted_at": nil, // Will be filled with current timestamp
			}
		}
		queries["delete"] = ast
	}

	// Count records
	if ast, err := ParseFromMetadata(metadata, OpCount); err == nil {
		queries["count"] = ast
	}

	return queries
}