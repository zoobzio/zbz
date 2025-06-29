package sqlprovider

import (
	"fmt"
	"strings"

	"zbz/astql"
)

// SQLGenerator renders AST to SQL queries
type SQLGenerator struct{}

// NewSQLGenerator creates a new SQL query generator
func NewSQLGenerator() *SQLGenerator {
	return &SQLGenerator{}
}

// Render converts a QueryAST to SQL with named parameters
func (g *SQLGenerator) Render(ast *astql.QueryAST) (query string, params map[string]any, err error) {
	if err := ast.Validate(); err != nil {
		return "", nil, fmt.Errorf("invalid AST: %w", err)
	}

	params = make(map[string]any)

	switch ast.Operation {
	case astql.OpSelect:
		query, err = g.renderSelect(ast, params)
	case astql.OpInsert:
		query, err = g.renderInsert(ast, params)
	case astql.OpUpdate:
		query, err = g.renderUpdate(ast, params)
	case astql.OpDelete:
		query, err = g.renderDelete(ast, params)
	case astql.OpCount:
		query, err = g.renderCount(ast, params)
	default:
		return "", nil, fmt.Errorf("unsupported operation: %s", ast.Operation)
	}

	return query, params, err
}

// renderSelect generates SELECT queries
func (g *SQLGenerator) renderSelect(ast *astql.QueryAST, params map[string]any) (string, error) {
	var sql strings.Builder

	// SELECT clause
	sql.WriteString("SELECT ")
	if len(ast.Fields) == 0 {
		sql.WriteString("*")
	} else {
		fields := make([]string, len(ast.Fields))
		for i, field := range ast.Fields {
			if field.Alias != "" {
				fields[i] = fmt.Sprintf("%s AS %s", field.Name, field.Alias)
			} else {
				fields[i] = field.Name
			}
		}
		sql.WriteString(strings.Join(fields, ", "))
	}

	// FROM clause
	sql.WriteString(" FROM ")
	sql.WriteString(ast.Target)

	// JOIN clauses
	for _, join := range ast.Joins {
		sql.WriteString(fmt.Sprintf(" %s JOIN %s", join.Type, join.Target))
		if join.Alias != "" {
			sql.WriteString(" AS " + join.Alias)
		}
		if join.Condition != "" {
			sql.WriteString(" ON " + join.Condition)
		}
	}

	// WHERE clause
	if len(ast.Conditions) > 0 {
		sql.WriteString(" WHERE ")
		whereClause, err := g.renderConditions(ast.Conditions, params)
		if err != nil {
			return "", err
		}
		sql.WriteString(whereClause)
	}

	// GROUP BY clause
	if len(ast.Grouping) > 0 {
		sql.WriteString(" GROUP BY ")
		groups := make([]string, len(ast.Grouping))
		for i, group := range ast.Grouping {
			groups[i] = group.Field
		}
		sql.WriteString(strings.Join(groups, ", "))
	}

	// HAVING clause
	if len(ast.Having) > 0 {
		sql.WriteString(" HAVING ")
		havingClause, err := g.renderConditions(ast.Having, params)
		if err != nil {
			return "", err
		}
		sql.WriteString(havingClause)
	}

	// ORDER BY clause
	if len(ast.Ordering) > 0 {
		sql.WriteString(" ORDER BY ")
		orders := make([]string, len(ast.Ordering))
		for i, order := range ast.Ordering {
			orders[i] = fmt.Sprintf("%s %s", order.Field, order.Direction)
		}
		sql.WriteString(strings.Join(orders, ", "))
	}

	// LIMIT clause
	if ast.Limit != nil {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", *ast.Limit))
	}

	// OFFSET clause
	if ast.Offset != nil {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", *ast.Offset))
	}

	return sql.String(), nil
}

// renderInsert generates INSERT queries
func (g *SQLGenerator) renderInsert(ast *astql.QueryAST, params map[string]any) (string, error) {
	if len(ast.Values) == 0 {
		return "", fmt.Errorf("INSERT requires at least one value set")
	}

	var sql strings.Builder
	sql.WriteString("INSERT INTO ")
	sql.WriteString(ast.Target)

	// Get column names from first value set
	firstRow := ast.Values[0]
	columns := make([]string, 0, len(firstRow))
	for col := range firstRow {
		columns = append(columns, col)
	}

	// Columns clause
	sql.WriteString(" (")
	sql.WriteString(strings.Join(columns, ", "))
	sql.WriteString(")")

	// VALUES clause
	sql.WriteString(" VALUES ")
	valueSets := make([]string, len(ast.Values))
	
	for i, valueSet := range ast.Values {
		placeholders := make([]string, len(columns))
		for j, col := range columns {
			paramName := fmt.Sprintf("%s_%d", col, i)
			placeholders[j] = ":" + paramName
			params[paramName] = valueSet[col]
		}
		valueSets[i] = "(" + strings.Join(placeholders, ", ") + ")"
	}
	sql.WriteString(strings.Join(valueSets, ", "))

	// RETURNING clause
	if len(ast.Returning) > 0 {
		sql.WriteString(" RETURNING ")
		sql.WriteString(strings.Join(ast.Returning, ", "))
	}

	return sql.String(), nil
}

// renderUpdate generates UPDATE queries
func (g *SQLGenerator) renderUpdate(ast *astql.QueryAST, params map[string]any) (string, error) {
	if len(ast.Updates) == 0 {
		return "", fmt.Errorf("UPDATE requires at least one field to update")
	}

	var sql strings.Builder
	sql.WriteString("UPDATE ")
	sql.WriteString(ast.Target)
	sql.WriteString(" SET ")

	// SET clause
	sets := make([]string, 0, len(ast.Updates))
	for field, value := range ast.Updates {
		paramName := "update_" + field
		sets = append(sets, fmt.Sprintf("%s = :%s", field, paramName))
		params[paramName] = value
	}
	sql.WriteString(strings.Join(sets, ", "))

	// WHERE clause
	if len(ast.Conditions) > 0 {
		sql.WriteString(" WHERE ")
		whereClause, err := g.renderConditions(ast.Conditions, params)
		if err != nil {
			return "", err
		}
		sql.WriteString(whereClause)
	}

	// RETURNING clause
	if len(ast.Returning) > 0 {
		sql.WriteString(" RETURNING ")
		sql.WriteString(strings.Join(ast.Returning, ", "))
	}

	return sql.String(), nil
}

// renderDelete generates DELETE queries
func (g *SQLGenerator) renderDelete(ast *astql.QueryAST, params map[string]any) (string, error) {
	var sql strings.Builder
	sql.WriteString("DELETE FROM ")
	sql.WriteString(ast.Target)

	// WHERE clause
	if len(ast.Conditions) > 0 {
		sql.WriteString(" WHERE ")
		whereClause, err := g.renderConditions(ast.Conditions, params)
		if err != nil {
			return "", err
		}
		sql.WriteString(whereClause)
	}

	// RETURNING clause
	if len(ast.Returning) > 0 {
		sql.WriteString(" RETURNING ")
		sql.WriteString(strings.Join(ast.Returning, ", "))
	}

	return sql.String(), nil
}

// renderCount generates COUNT queries
func (g *SQLGenerator) renderCount(ast *astql.QueryAST, params map[string]any) (string, error) {
	var sql strings.Builder
	sql.WriteString("SELECT COUNT(*) FROM ")
	sql.WriteString(ast.Target)

	// WHERE clause
	if len(ast.Conditions) > 0 {
		sql.WriteString(" WHERE ")
		whereClause, err := g.renderConditions(ast.Conditions, params)
		if err != nil {
			return "", err
		}
		sql.WriteString(whereClause)
	}

	return sql.String(), nil
}

// renderConditions generates WHERE/HAVING conditions
func (g *SQLGenerator) renderConditions(conditions []astql.Condition, params map[string]any) (string, error) {
	if len(conditions) == 0 {
		return "", nil
	}

	var conditionParts []string
	
	for i, cond := range conditions {
		var condStr string
		
		// Build the condition based on operator
		switch cond.Operator {
		case astql.EQ:
			condStr = fmt.Sprintf("%s = :%s", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.NE:
			condStr = fmt.Sprintf("%s != :%s", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.GT:
			condStr = fmt.Sprintf("%s > :%s", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.GE:
			condStr = fmt.Sprintf("%s >= :%s", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.LT:
			condStr = fmt.Sprintf("%s < :%s", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.LE:
			condStr = fmt.Sprintf("%s <= :%s", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.LIKE:
			condStr = fmt.Sprintf("%s LIKE :%s", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.IN:
			condStr = fmt.Sprintf("%s = ANY(:%s)", cond.Field, cond.ParamName)
			params[cond.ParamName] = cond.Value
		case astql.IS_NULL:
			condStr = fmt.Sprintf("%s IS NULL", cond.Field)
		case astql.IS_NOT_NULL:
			condStr = fmt.Sprintf("%s IS NOT NULL", cond.Field)
		case astql.BETWEEN:
			if values, ok := cond.Value.([]any); ok && len(values) == 2 {
				startParam := cond.ParamName + "_start"
				endParam := cond.ParamName + "_end"
				condStr = fmt.Sprintf("%s BETWEEN :%s AND :%s", cond.Field, startParam, endParam)
				params[startParam] = values[0]
				params[endParam] = values[1]
			} else {
				return "", fmt.Errorf("BETWEEN requires array of 2 values")
			}
		default:
			return "", fmt.Errorf("unsupported operator: %s", cond.Operator)
		}

		// Add logical operator for all but the last condition
		if i < len(conditions)-1 {
			switch cond.Logical {
			case astql.AND:
				condStr += " AND"
			case astql.OR:
				condStr += " OR"
			}
		}

		conditionParts = append(conditionParts, condStr)
	}

	return strings.Join(conditionParts, " "), nil
}