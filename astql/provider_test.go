package astql

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Test basic provider functionality without imports

// BasicSQLProvider simple SQL generator for testing
type BasicSQLProvider struct{}

func (p *BasicSQLProvider) RenderSQL(ast *QueryAST) (string, map[string]any, error) {
	if err := ast.Validate(); err != nil {
		return "", nil, fmt.Errorf("invalid AST: %w", err)
	}

	params := make(map[string]any)
	var sql strings.Builder

	switch ast.Operation {
	case OpSelect:
		sql.WriteString("SELECT ")
		if len(ast.Fields) == 0 {
			sql.WriteString("*")
		} else {
			fields := make([]string, len(ast.Fields))
			for i, field := range ast.Fields {
				fields[i] = field.Name
			}
			sql.WriteString(strings.Join(fields, ", "))
		}

		sql.WriteString(" FROM " + ast.Target)

		if len(ast.Conditions) > 0 {
			sql.WriteString(" WHERE ")
			conds := make([]string, len(ast.Conditions))
			for i, cond := range ast.Conditions {
				switch cond.Operator {
				case EQ:
					conds[i] = fmt.Sprintf("%s = :%s", cond.Field, cond.ParamName)
					params[cond.ParamName] = cond.Value
				case GT:
					conds[i] = fmt.Sprintf("%s > :%s", cond.Field, cond.ParamName)
					params[cond.ParamName] = cond.Value
				case IS_NULL:
					conds[i] = fmt.Sprintf("%s IS NULL", cond.Field)
				case IN:
					conds[i] = fmt.Sprintf("%s = ANY(:%s)", cond.Field, cond.ParamName)
					params[cond.ParamName] = cond.Value
				}
			}
			sql.WriteString(strings.Join(conds, " AND "))
		}

		if len(ast.Ordering) > 0 {
			sql.WriteString(" ORDER BY ")
			orders := make([]string, len(ast.Ordering))
			for i, order := range ast.Ordering {
				orders[i] = fmt.Sprintf("%s %s", order.Field, order.Direction)
			}
			sql.WriteString(strings.Join(orders, ", "))
		}

		if ast.Limit != nil {
			sql.WriteString(fmt.Sprintf(" LIMIT %d", *ast.Limit))
		}

	case OpInsert:
		if len(ast.Values) == 0 {
			return "", nil, fmt.Errorf("INSERT requires values")
		}

		firstRow := ast.Values[0]
		columns := make([]string, 0, len(firstRow))
		placeholders := make([]string, 0, len(firstRow))

		for col, val := range firstRow {
			columns = append(columns, col)
			paramName := fmt.Sprintf("%s_0", col)
			placeholders = append(placeholders, ":"+paramName)
			params[paramName] = val
		}

		sql.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			ast.Target,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", ")))

		if len(ast.Returning) > 0 {
			sql.WriteString(" RETURNING " + strings.Join(ast.Returning, ", "))
		}

	case OpUpdate:
		if len(ast.Updates) == 0 {
			return "", nil, fmt.Errorf("UPDATE requires fields to update")
		}

		sql.WriteString("UPDATE " + ast.Target + " SET ")
		sets := make([]string, 0, len(ast.Updates))
		for field, value := range ast.Updates {
			paramName := "update_" + field
			sets = append(sets, fmt.Sprintf("%s = :%s", field, paramName))
			params[paramName] = value
		}
		sql.WriteString(strings.Join(sets, ", "))

		if len(ast.Conditions) > 0 {
			sql.WriteString(" WHERE ")
			conds := make([]string, len(ast.Conditions))
			for i, cond := range ast.Conditions {
				if cond.Operator == EQ {
					conds[i] = fmt.Sprintf("%s = :%s", cond.Field, cond.ParamName)
					params[cond.ParamName] = cond.Value
				}
			}
			sql.WriteString(strings.Join(conds, " AND "))
		}

		if len(ast.Returning) > 0 {
			sql.WriteString(" RETURNING " + strings.Join(ast.Returning, ", "))
		}

	case OpDelete:
		sql.WriteString("DELETE FROM " + ast.Target)

		if len(ast.Conditions) > 0 {
			sql.WriteString(" WHERE ")
			conds := make([]string, len(ast.Conditions))
			for i, cond := range ast.Conditions {
				if cond.Operator == EQ {
					conds[i] = fmt.Sprintf("%s = :%s", cond.Field, cond.ParamName)
					params[cond.ParamName] = cond.Value
				}
			}
			sql.WriteString(strings.Join(conds, " AND "))
		}

	case OpCount:
		sql.WriteString("SELECT COUNT(*) FROM " + ast.Target)

		if len(ast.Conditions) > 0 {
			sql.WriteString(" WHERE ")
			conds := make([]string, len(ast.Conditions))
			for i, cond := range ast.Conditions {
				switch cond.Operator {
				case EQ:
					conds[i] = fmt.Sprintf("%s = :%s", cond.Field, cond.ParamName)
					params[cond.ParamName] = cond.Value
				case GE:
					conds[i] = fmt.Sprintf("%s >= :%s", cond.Field, cond.ParamName)
					params[cond.ParamName] = cond.Value
				}
			}
			sql.WriteString(strings.Join(conds, " AND "))
		}

	default:
		return "", nil, fmt.Errorf("unsupported operation: %s", ast.Operation)
	}

	return sql.String(), params, nil
}

// BasicMongoProvider simple MongoDB generator for testing
type BasicMongoProvider struct{}

func (p *BasicMongoProvider) RenderMongo(ast *QueryAST) (string, error) {
	if err := ast.Validate(); err != nil {
		return "", fmt.Errorf("invalid AST: %w", err)
	}

	query := map[string]any{
		"collection": ast.Target,
	}

	switch ast.Operation {
	case OpSelect:
		query["operation"] = "find"
		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, cond := range ast.Conditions {
				switch cond.Operator {
				case EQ:
					filter[cond.Field] = cond.Value
				case GT:
					filter[cond.Field] = map[string]any{"$gt": cond.Value}
				case GE:
					filter[cond.Field] = map[string]any{"$gte": cond.Value}
				case IS_NULL:
					filter[cond.Field] = nil
				case IN:
					filter[cond.Field] = map[string]any{"$in": cond.Value}
				}
			}
			query["filter"] = filter
		}

		options := make(map[string]any)
		if len(ast.Ordering) > 0 {
			sort := make(map[string]any)
			for _, order := range ast.Ordering {
				if order.Direction == ASC {
					sort[order.Field] = 1
				} else {
					sort[order.Field] = -1
				}
			}
			options["sort"] = sort
		}

		if ast.Limit != nil {
			options["limit"] = *ast.Limit
		}
		if ast.Offset != nil {
			options["skip"] = *ast.Offset
		}

		if len(options) > 0 {
			query["options"] = options
		}

	case OpInsert:
		if len(ast.Values) == 1 {
			query["operation"] = "insertOne"
			query["document"] = ast.Values[0]
		} else {
			query["operation"] = "insertMany"
			query["documents"] = ast.Values
		}

	case OpUpdate:
		query["operation"] = "updateOne"
		query["update"] = map[string]any{"$set": ast.Updates}

		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, cond := range ast.Conditions {
				if cond.Operator == EQ {
					filter[cond.Field] = cond.Value
				}
			}
			query["filter"] = filter
		}

	case OpDelete:
		query["operation"] = "deleteOne"
		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, cond := range ast.Conditions {
				if cond.Operator == EQ {
					filter[cond.Field] = cond.Value
				}
			}
			query["filter"] = filter
		}

	case OpCount:
		query["operation"] = "countDocuments"
		if len(ast.Conditions) > 0 {
			filter := make(map[string]any)
			for _, cond := range ast.Conditions {
				switch cond.Operator {
				case EQ:
					filter[cond.Field] = cond.Value
				case GE:
					filter[cond.Field] = map[string]any{"$gte": cond.Value}
				}
			}
			query["filter"] = filter
		}

	default:
		return "", fmt.Errorf("unsupported operation: %s", ast.Operation)
	}

	jsonBytes, err := json.MarshalIndent(query, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal query: %w", err)
	}

	return string(jsonBytes), nil
}

func TestProviderIntegration(t *testing.T) {
	// Test data
	userAST := Select("users").
		Fields("id", "email", "name", "created_at").
		Where("tenant_id", EQ, "tenant123").
		Where("status", EQ, "active").
		Where("deleted_at", IS_NULL, nil).
		OrderByDesc("created_at").
		Limit(25).
		MustBuild()

	insertAST := Insert("users").
		Values(map[string]any{
			"id":        "user456",
			"email":     "new@example.com",
			"name":      "New User",
			"tenant_id": "tenant123",
			"status":    "active",
		}).
		Returning("id", "created_at").
		MustBuild()

	updateAST := Update("users").
		Set("name", "Updated Name").
		Set("updated_at", "now()").
		Where("id", EQ, "user456").
		Where("tenant_id", EQ, "tenant123").
		Returning("*").
		MustBuild()

	countAST := Count("users").
		Where("tenant_id", EQ, "tenant123").
		Where("status", EQ, "active").
		MustBuild()

	t.Run("SQL Provider", func(t *testing.T) {
		sqlProvider := &BasicSQLProvider{}

		t.Run("SELECT Query", func(t *testing.T) {
			query, params, err := sqlProvider.RenderSQL(userAST)
			if err != nil {
				t.Fatalf("SQL generation failed: %v", err)
			}

			expectedSQL := "SELECT id, email, name, created_at FROM users WHERE tenant_id = :tenant_id AND status = :status AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 25"
			if normalizeSQL(query) != normalizeSQL(expectedSQL) {
				t.Errorf("SQL mismatch\nGot:  %s\nWant: %s", query, expectedSQL)
			}

			if len(params) != 2 {
				t.Errorf("Expected 2 parameters, got %d: %v", len(params), params)
			}

			t.Logf("✅ SQL: %s", query)
			t.Logf("✅ Params: %v", params)
		})

		t.Run("INSERT Query", func(t *testing.T) {
			query, params, err := sqlProvider.RenderSQL(insertAST)
			if err != nil {
				t.Fatalf("SQL generation failed: %v", err)
			}

			if !strings.Contains(query, "INSERT INTO users") {
				t.Errorf("Expected INSERT query, got: %s", query)
			}

			if !strings.Contains(query, "RETURNING id, created_at") {
				t.Errorf("Expected RETURNING clause, got: %s", query)
			}

			if len(params) != 5 {
				t.Errorf("Expected 5 parameters, got %d: %v", len(params), params)
			}

			t.Logf("✅ SQL: %s", query)
			t.Logf("✅ Params: %v", params)
		})

		t.Run("UPDATE Query", func(t *testing.T) {
			query, params, err := sqlProvider.RenderSQL(updateAST)
			if err != nil {
				t.Fatalf("SQL generation failed: %v", err)
			}

			if !strings.Contains(query, "UPDATE users SET") {
				t.Errorf("Expected UPDATE query, got: %s", query)
			}

			if !strings.Contains(query, "RETURNING *") {
				t.Errorf("Expected RETURNING clause, got: %s", query)
			}

			t.Logf("✅ SQL: %s", query)
			t.Logf("✅ Params: %v", params)
		})

		t.Run("COUNT Query", func(t *testing.T) {
			query, params, err := sqlProvider.RenderSQL(countAST)
			if err != nil {
				t.Fatalf("SQL generation failed: %v", err)
			}

			expectedSQL := "SELECT COUNT(*) FROM users WHERE tenant_id = :tenant_id AND status = :status"
			if normalizeSQL(query) != normalizeSQL(expectedSQL) {
				t.Errorf("SQL mismatch\nGot:  %s\nWant: %s", query, expectedSQL)
			}

			t.Logf("✅ SQL: %s", query)
			t.Logf("✅ Params: %v", params)
		})
	})

	t.Run("MongoDB Provider", func(t *testing.T) {
		mongoProvider := &BasicMongoProvider{}

		t.Run("Find Query", func(t *testing.T) {
			queryJSON, err := mongoProvider.RenderMongo(userAST)
			if err != nil {
				t.Fatalf("MongoDB generation failed: %v", err)
			}

			var query map[string]any
			if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if query["operation"] != "find" {
				t.Errorf("Expected find operation, got: %v", query["operation"])
			}

			if query["collection"] != "users" {
				t.Errorf("Expected users collection, got: %v", query["collection"])
			}

			// Check filter
			if filter, ok := query["filter"].(map[string]any); ok {
				if filter["tenant_id"] != "tenant123" {
					t.Errorf("Expected tenant_id filter, got: %v", filter)
				}
			} else {
				t.Errorf("Expected filter object, got: %T", query["filter"])
			}

			t.Logf("✅ MongoDB Query:\n%s", queryJSON)
		})

		t.Run("Insert Query", func(t *testing.T) {
			queryJSON, err := mongoProvider.RenderMongo(insertAST)
			if err != nil {
				t.Fatalf("MongoDB generation failed: %v", err)
			}

			var query map[string]any
			if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if query["operation"] != "insertOne" {
				t.Errorf("Expected insertOne operation, got: %v", query["operation"])
			}

			if doc, ok := query["document"].(map[string]any); ok {
				if doc["email"] != "new@example.com" {
					t.Errorf("Expected email in document, got: %v", doc)
				}
			} else {
				t.Errorf("Expected document object, got: %T", query["document"])
			}

			t.Logf("✅ MongoDB Query:\n%s", queryJSON)
		})

		t.Run("Update Query", func(t *testing.T) {
			queryJSON, err := mongoProvider.RenderMongo(updateAST)
			if err != nil {
				t.Fatalf("MongoDB generation failed: %v", err)
			}

			var query map[string]any
			if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if query["operation"] != "updateOne" {
				t.Errorf("Expected updateOne operation, got: %v", query["operation"])
			}

			t.Logf("✅ MongoDB Query:\n%s", queryJSON)
		})

		t.Run("Count Query", func(t *testing.T) {
			queryJSON, err := mongoProvider.RenderMongo(countAST)
			if err != nil {
				t.Fatalf("MongoDB generation failed: %v", err)
			}

			var query map[string]any
			if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if query["operation"] != "countDocuments" {
				t.Errorf("Expected countDocuments operation, got: %v", query["operation"])
			}

			t.Logf("✅ MongoDB Query:\n%s", queryJSON)
		})
	})
}

func TestComplexQueries(t *testing.T) {
	t.Run("JOIN Query (SQL)", func(t *testing.T) {
		ast := Select("users").
			Fields("u.id", "u.name", "p.name AS profile_name").
			InnerJoin("profiles p", "u.id = p.user_id").
			Where("u.tenant_id", EQ, "tenant123").
			Where("u.deleted_at", IS_NULL, nil).
			OrderByAsc("u.name").
			MustBuild()

		sqlProvider := &BasicSQLProvider{}
		query, params, err := sqlProvider.RenderSQL(ast)
		if err != nil {
			t.Fatalf("SQL generation failed: %v", err)
		}

		if !strings.Contains(query, "INNER JOIN") {
			t.Errorf("Expected JOIN in query, got: %s", query)
		}

		t.Logf("✅ JOIN SQL: %s", query)
		t.Logf("✅ Params: %v", params)
	})

	t.Run("Pagination Query", func(t *testing.T) {
		ast := Select("orders").
			Fields("id", "total", "created_at").
			Where("user_id", EQ, "user123").
			OrderByDesc("created_at").
			Paginate(2, 10). // Page 2, 10 items per page
			MustBuild()

		if *ast.Limit != 10 {
			t.Errorf("Expected limit 10, got %d", *ast.Limit)
		}

		if *ast.Offset != 10 { // (page-1) * pageSize = (2-1) * 10 = 10
			t.Errorf("Expected offset 10, got %d", *ast.Offset)
		}

		t.Logf("✅ Pagination: LIMIT %d OFFSET %d", *ast.Limit, *ast.Offset)
	})

	t.Run("Batch Insert", func(t *testing.T) {
		ast := Insert("orders").
			Values(map[string]any{"id": "order1", "total": 100.50}).
			Values(map[string]any{"id": "order2", "total": 75.25}).
			Values(map[string]any{"id": "order3", "total": 200.00}).
			MustBuild()

		if len(ast.Values) != 3 {
			t.Errorf("Expected 3 value sets, got %d", len(ast.Values))
		}

		mongoProvider := &BasicMongoProvider{}
		queryJSON, err := mongoProvider.RenderMongo(ast)
		if err != nil {
			t.Fatalf("MongoDB generation failed: %v", err)
		}

		var query map[string]any
		if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}

		if query["operation"] != "insertMany" {
			t.Errorf("Expected insertMany for multiple values, got: %v", query["operation"])
		}

		t.Logf("✅ Batch Insert MongoDB:\n%s", queryJSON)
	})
}