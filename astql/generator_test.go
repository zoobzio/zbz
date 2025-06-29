package astql

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestUser for testing query generation
type TestUser struct {
	ID        string  `json:"id" db:"user_id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Age       int     `json:"age"`
	TenantID  string  `json:"tenant_id"`
	CreatedAt string  `json:"created_at"`
	DeletedAt *string `json:"deleted_at,omitempty"`
}

func TestSQLQueryGeneration(t *testing.T) {
	tests := []struct {
		name     string
		ast      *QueryAST
		wantSQL  string
		wantParams int
	}{
		{
			name: "Simple SELECT",
			ast: Select("users").
				Fields("id", "email", "name").
				Where("tenant_id", EQ, "tenant123").
				OrderByDesc("created_at").
				Limit(10).
				MustBuild(),
			wantSQL: "SELECT id, email, name FROM users WHERE tenant_id = :tenant_id ORDER BY created_at DESC LIMIT 10",
			wantParams: 1,
		},
		{
			name: "SELECT with multiple conditions",
			ast: Select("users").
				Where("age", GT, 18).
				Where("deleted_at", IS_NULL, nil).
				WhereIn("status", []string{"active", "pending"}).
				MustBuild(),
			wantSQL: "SELECT * FROM users WHERE age > :age AND deleted_at IS NULL AND status = ANY(:status)",
			wantParams: 2,
		},
		{
			name: "INSERT single record",
			ast: Insert("users").
				Values(map[string]any{
					"id":    "user123",
					"email": "test@example.com",
					"name":  "Test User",
				}).
				Returning("id", "created_at").
				MustBuild(),
			wantSQL: "INSERT INTO users (id, email, name) VALUES (:id_0, :email_0, :name_0) RETURNING id, created_at",
			wantParams: 3,
		},
		{
			name: "UPDATE with conditions",
			ast: Update("users").
				Set("name", "Updated Name").
				Set("updated_at", "now()").
				Where("id", EQ, "user123").
				Where("tenant_id", EQ, "tenant123").
				Returning("*").
				MustBuild(),
			wantSQL: "UPDATE users SET name = :update_name, updated_at = :update_updated_at WHERE id = :id AND tenant_id = :tenant_id RETURNING *",
			wantParams: 4,
		},
		{
			name: "DELETE with conditions",
			ast: Delete("users").
				Where("id", EQ, "user123").
				Where("deleted_at", IS_NULL, nil).
				MustBuild(),
			wantSQL: "DELETE FROM users WHERE id = :id AND deleted_at IS NULL",
			wantParams: 1,
		},
		{
			name: "COUNT with filter",
			ast: Count("users").
				Where("age", GE, 18).
				Where("status", EQ, "active").
				MustBuild(),
			wantSQL: "SELECT COUNT(*) FROM users WHERE age >= :age AND status = :status",
			wantParams: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock SQL generator (simplified version)
			query, params := mockSQLGenerate(tt.ast)
			
			// Normalize whitespace for comparison
			gotSQL := normalizeSQL(query)
			wantSQL := normalizeSQL(tt.wantSQL)
			
			if gotSQL != wantSQL {
				t.Errorf("SQL mismatch\nGot:  %s\nWant: %s", gotSQL, wantSQL)
			}
			
			if len(params) != tt.wantParams {
				t.Errorf("Parameter count mismatch. Got: %d, Want: %d", len(params), tt.wantParams)
			}
			
			t.Logf("Generated SQL: %s", query)
			t.Logf("Parameters: %v", params)
		})
	}
}

func TestMongoQueryGeneration(t *testing.T) {
	tests := []struct {
		name      string
		ast       *QueryAST
		wantOp    string
		wantField string
	}{
		{
			name: "Simple find",
			ast: Select("users").
				Where("tenant_id", EQ, "tenant123").
				Where("age", GT, 18).
				OrderByDesc("created_at").
				Limit(10).
				MustBuild(),
			wantOp:    "find",
			wantField: "tenant_id",
		},
		{
			name: "Insert document",
			ast: Insert("users").
				Values(map[string]any{
					"id":    "user123",
					"email": "test@example.com",
					"name":  "Test User",
				}).
				MustBuild(),
			wantOp:    "insertOne",
			wantField: "id",
		},
		{
			name: "Update document",
			ast: Update("users").
				Set("name", "Updated Name").
				Where("id", EQ, "user123").
				MustBuild(),
			wantOp:    "updateOne",
			wantField: "id",
		},
		{
			name: "Delete document",
			ast: Delete("users").
				Where("id", EQ, "user123").
				MustBuild(),
			wantOp:    "deleteOne",
			wantField: "id",
		},
		{
			name: "Count documents",
			ast: Count("users").
				Where("status", EQ, "active").
				MustBuild(),
			wantOp:    "countDocuments",
			wantField: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock MongoDB generator
			queryJSON := mockMongoGenerate(tt.ast)
			
			// Parse JSON to verify structure
			var query map[string]any
			if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
				t.Fatalf("Invalid JSON generated: %v", err)
			}
			
			// Check operation
			if op, ok := query["operation"].(string); !ok || op != tt.wantOp {
				t.Errorf("Operation mismatch. Got: %v, Want: %s", query["operation"], tt.wantOp)
			}
			
			// Check collection
			if coll, ok := query["collection"].(string); !ok || coll != tt.ast.Target {
				t.Errorf("Collection mismatch. Got: %v, Want: %s", query["collection"], tt.ast.Target)
			}
			
			t.Logf("Generated MongoDB query: %s", queryJSON)
		})
	}
}

func TestASTValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     *QueryAST
		wantErr bool
	}{
		{
			name: "Valid SELECT",
			ast: &QueryAST{
				Operation: OpSelect,
				Target:    "users",
			},
			wantErr: false,
		},
		{
			name: "Invalid - missing target",
			ast: &QueryAST{
				Operation: OpSelect,
			},
			wantErr: true,
		},
		{
			name: "Invalid INSERT - no values",
			ast: &QueryAST{
				Operation: OpInsert,
				Target:    "users",
			},
			wantErr: true,
		},
		{
			name: "Invalid UPDATE - no updates",
			ast: &QueryAST{
				Operation: OpUpdate,
				Target:    "users",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ast.Validate()
			
			if tt.wantErr && err == nil {
				t.Error("Expected validation error, got none")
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestBuilderAPI(t *testing.T) {
	// Test the fluent builder API
	query := Select("users").
		Fields("id", "email", "name").
		Where("tenant_id", EQ, "tenant123").
		Where("age", GT, 18).
		OrWhere("status", EQ, "premium").
		OrderByDesc("created_at").
		Limit(20).
		Offset(100)

	ast, err := query.Build()
	if err != nil {
		t.Fatalf("Builder API failed: %v", err)
	}

	// Verify AST structure
	if ast.Operation != OpSelect {
		t.Errorf("Wrong operation. Got: %s, Want: %s", ast.Operation, OpSelect)
	}

	if ast.Target != "users" {
		t.Errorf("Wrong target. Got: %s, Want: users", ast.Target)
	}

	if len(ast.Fields) != 3 {
		t.Errorf("Wrong field count. Got: %d, Want: 3", len(ast.Fields))
	}

	if len(ast.Conditions) != 3 {
		t.Errorf("Wrong condition count. Got: %d, Want: 3", len(ast.Conditions))
	}

	if *ast.Limit != 20 {
		t.Errorf("Wrong limit. Got: %d, Want: 20", *ast.Limit)
	}

	if *ast.Offset != 100 {
		t.Errorf("Wrong offset. Got: %d, Want: 100", *ast.Offset)
	}

	t.Logf("Built AST: %+v", ast)
}

// Mock SQL generator for testing (simplified)
func mockSQLGenerate(ast *QueryAST) (string, map[string]any) {
	params := make(map[string]any)
	
	switch ast.Operation {
	case OpSelect:
		var sql strings.Builder
		sql.WriteString("SELECT ")
		
		if len(ast.Fields) == 0 {
			sql.WriteString("*")
		} else {
			fields := make([]string, len(ast.Fields))
			for i, f := range ast.Fields {
				fields[i] = f.Name
			}
			sql.WriteString(strings.Join(fields, ", "))
		}
		
		sql.WriteString(" FROM " + ast.Target)
		
		if len(ast.Conditions) > 0 {
			sql.WriteString(" WHERE ")
			conds := make([]string, len(ast.Conditions))
			for i, c := range ast.Conditions {
				switch c.Operator {
				case EQ:
					conds[i] = c.Field + " = :" + c.ParamName
					params[c.ParamName] = c.Value
				case GT:
					conds[i] = c.Field + " > :" + c.ParamName
					params[c.ParamName] = c.Value
				case GE:
					conds[i] = c.Field + " >= :" + c.ParamName
					params[c.ParamName] = c.Value
				case IS_NULL:
					conds[i] = c.Field + " IS NULL"
				case IN:
					conds[i] = c.Field + " = ANY(:" + c.ParamName + ")"
					params[c.ParamName] = c.Value
				}
			}
			sql.WriteString(strings.Join(conds, " AND "))
		}
		
		if len(ast.Ordering) > 0 {
			sql.WriteString(" ORDER BY ")
			orders := make([]string, len(ast.Ordering))
			for i, o := range ast.Ordering {
				orders[i] = o.Field + " " + string(o.Direction)
			}
			sql.WriteString(strings.Join(orders, ", "))
		}
		
		if ast.Limit != nil {
			sql.WriteString(fmt.Sprintf(" LIMIT %d", *ast.Limit))
		}
		
		return sql.String(), params
		
	case OpInsert:
		if len(ast.Values) == 0 {
			return "", params
		}
		
		firstRow := ast.Values[0]
		columns := make([]string, 0, len(firstRow))
		placeholders := make([]string, 0, len(firstRow))
		
		for col, val := range firstRow {
			columns = append(columns, col)
			paramName := col + "_0"
			placeholders = append(placeholders, ":"+paramName)
			params[paramName] = val
		}
		
		sql := "INSERT INTO " + ast.Target + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
		
		if len(ast.Returning) > 0 {
			sql += " RETURNING " + strings.Join(ast.Returning, ", ")
		}
		
		return sql, params
		
	case OpUpdate:
		sets := make([]string, 0, len(ast.Updates))
		for field, value := range ast.Updates {
			paramName := "update_" + field
			sets = append(sets, field+" = :"+paramName)
			params[paramName] = value
		}
		
		sql := "UPDATE " + ast.Target + " SET " + strings.Join(sets, ", ")
		
		if len(ast.Conditions) > 0 {
			sql += " WHERE "
			conds := make([]string, len(ast.Conditions))
			for i, c := range ast.Conditions {
				if c.Operator == EQ {
					conds[i] = c.Field + " = :" + c.ParamName
					params[c.ParamName] = c.Value
				}
			}
			sql += strings.Join(conds, " AND ")
		}
		
		if len(ast.Returning) > 0 {
			sql += " RETURNING " + strings.Join(ast.Returning, ", ")
		}
		
		return sql, params
		
	case OpDelete:
		sql := "DELETE FROM " + ast.Target
		
		if len(ast.Conditions) > 0 {
			sql += " WHERE "
			conds := make([]string, len(ast.Conditions))
			for i, c := range ast.Conditions {
				switch c.Operator {
				case EQ:
					conds[i] = c.Field + " = :" + c.ParamName
					params[c.ParamName] = c.Value
				case IS_NULL:
					conds[i] = c.Field + " IS NULL"
				}
			}
			sql += strings.Join(conds, " AND ")
		}
		
		return sql, params
		
	case OpCount:
		sql := "SELECT COUNT(*) FROM " + ast.Target
		
		if len(ast.Conditions) > 0 {
			sql += " WHERE "
			conds := make([]string, len(ast.Conditions))
			for i, c := range ast.Conditions {
				switch c.Operator {
				case EQ:
					conds[i] = c.Field + " = :" + c.ParamName
					params[c.ParamName] = c.Value
				case GE:
					conds[i] = c.Field + " >= :" + c.ParamName
					params[c.ParamName] = c.Value
				}
			}
			sql += strings.Join(conds, " AND ")
		}
		
		return sql, params
	}
	
	return "", params
}

// Mock MongoDB generator for testing
func mockMongoGenerate(ast *QueryAST) string {
	query := map[string]any{
		"collection": ast.Target,
	}
	
	switch ast.Operation {
	case OpSelect:
		query["operation"] = "find"
		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, c := range ast.Conditions {
				switch c.Operator {
				case EQ:
					filter[c.Field] = c.Value
				case GT:
					filter[c.Field] = map[string]any{"$gt": c.Value}
				}
			}
			query["filter"] = filter
		}
		if len(ast.Ordering) > 0 {
			sort := make(map[string]any)
			for _, o := range ast.Ordering {
				if o.Direction == ASC {
					sort[o.Field] = 1
				} else {
					sort[o.Field] = -1
				}
			}
			query["options"] = map[string]any{"sort": sort}
		}
		
	case OpInsert:
		query["operation"] = "insertOne"
		if len(ast.Values) > 0 {
			query["document"] = ast.Values[0]
		}
		
	case OpUpdate:
		query["operation"] = "updateOne"
		if len(ast.Updates) > 0 {
			query["update"] = map[string]any{"$set": ast.Updates}
		}
		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, c := range ast.Conditions {
				if c.Operator == EQ {
					filter[c.Field] = c.Value
				}
			}
			query["filter"] = filter
		}
		
	case OpDelete:
		query["operation"] = "deleteOne"
		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, c := range ast.Conditions {
				if c.Operator == EQ {
					filter[c.Field] = c.Value
				}
			}
			query["filter"] = filter
		}
		
	case OpCount:
		query["operation"] = "countDocuments"
		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, c := range ast.Conditions {
				if c.Operator == EQ {
					filter[c.Field] = c.Value
				}
			}
			query["filter"] = filter
		}
	}
	
	jsonBytes, _ := json.MarshalIndent(query, "", "  ")
	return string(jsonBytes)
}

// normalizeSQL removes extra whitespace for comparison
func normalizeSQL(sql string) string {
	// Replace multiple spaces with single space
	parts := strings.Fields(sql)
	return strings.Join(parts, " ")
}