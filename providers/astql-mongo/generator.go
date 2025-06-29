package mongoprovider

import (
	"encoding/json"
	"fmt"

	"zbz/astql"
)

// MongoGenerator renders AST to MongoDB queries
type MongoGenerator struct{}

// NewMongoGenerator creates a new MongoDB query generator
func NewMongoGenerator() *MongoGenerator {
	return &MongoGenerator{}
}

// MongoQuery represents a MongoDB query structure
type MongoQuery struct {
	Operation  string         `json:"operation"`  // "find", "insertOne", "updateOne", etc.
	Collection string         `json:"collection"` // Collection name
	Filter     map[string]any `json:"filter,omitempty"`
	Document   map[string]any `json:"document,omitempty"`
	Update     map[string]any `json:"update,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
}

// Render converts a QueryAST to MongoDB query structure
func (g *MongoGenerator) Render(ast *astql.QueryAST) (string, map[string]any, error) {
	if err := ast.Validate(); err != nil {
		return "", nil, fmt.Errorf("invalid AST: %w", err)
	}

	var mongoQuery MongoQuery
	var err error

	switch ast.Operation {
	case astql.OpSelect:
		mongoQuery, err = g.renderFind(ast)
	case astql.OpInsert:
		mongoQuery, err = g.renderInsert(ast)
	case astql.OpUpdate:
		mongoQuery, err = g.renderUpdate(ast)
	case astql.OpDelete:
		mongoQuery, err = g.renderDelete(ast)
	case astql.OpCount:
		mongoQuery, err = g.renderCount(ast)
	default:
		return "", nil, fmt.Errorf("unsupported operation: %s", ast.Operation)
	}

	if err != nil {
		return "", nil, err
	}

	// Convert to JSON string
	queryBytes, err := json.MarshalIndent(mongoQuery, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal MongoDB query: %w", err)
	}

	return string(queryBytes), nil, nil
}

// renderFind generates find queries
func (g *MongoGenerator) renderFind(ast *astql.QueryAST) (MongoQuery, error) {
	query := MongoQuery{
		Operation:  "find",
		Collection: ast.Target,
		Filter:     make(map[string]any),
		Options:    make(map[string]any),
	}

	// Build filter from conditions
	if len(ast.Conditions) > 0 {
		filter, err := g.buildFilter(ast.Conditions)
		if err != nil {
			return query, err
		}
		query.Filter = filter
	}

	// Build projection (field selection)
	if len(ast.Fields) > 0 {
		projection := make(map[string]any)
		for _, field := range ast.Fields {
			projection[field.Name] = 1
		}
		query.Options["projection"] = projection
	}

	// Build sort
	if len(ast.Ordering) > 0 {
		sort := make(map[string]any)
		for _, order := range ast.Ordering {
			if order.Direction == astql.ASC {
				sort[order.Field] = 1
			} else {
				sort[order.Field] = -1
			}
		}
		query.Options["sort"] = sort
	}

	// Add limit and skip
	if ast.Limit != nil {
		query.Options["limit"] = *ast.Limit
	}
	if ast.Offset != nil {
		query.Options["skip"] = *ast.Offset
	}

	return query, nil
}

// renderInsert generates insertOne/insertMany queries
func (g *MongoGenerator) renderInsert(ast *astql.QueryAST) (MongoQuery, error) {
	if len(ast.Values) == 0 {
		return MongoQuery{}, fmt.Errorf("INSERT requires at least one document")
	}

	query := MongoQuery{
		Collection: ast.Target,
	}

	if len(ast.Values) == 1 {
		query.Operation = "insertOne"
		query.Document = ast.Values[0]
	} else {
		query.Operation = "insertMany"
		documents := make([]map[string]any, len(ast.Values))
		for i, values := range ast.Values {
			documents[i] = values
		}
		query.Document = map[string]any{"documents": documents}
	}

	return query, nil
}

// renderUpdate generates updateOne/updateMany queries
func (g *MongoGenerator) renderUpdate(ast *astql.QueryAST) (MongoQuery, error) {
	if len(ast.Updates) == 0 {
		return MongoQuery{}, fmt.Errorf("UPDATE requires at least one field to update")
	}

	query := MongoQuery{
		Operation:  "updateOne", // Could be updateMany based on conditions
		Collection: ast.Target,
		Filter:     make(map[string]any),
		Update:     map[string]any{"$set": ast.Updates},
		Options:    make(map[string]any),
	}

	// Build filter from conditions
	if len(ast.Conditions) > 0 {
		filter, err := g.buildFilter(ast.Conditions)
		if err != nil {
			return query, err
		}
		query.Filter = filter
	}

	return query, nil
}

// renderDelete generates deleteOne/deleteMany queries
func (g *MongoGenerator) renderDelete(ast *astql.QueryAST) (MongoQuery, error) {
	query := MongoQuery{
		Operation:  "deleteOne", // Could be deleteMany based on conditions
		Collection: ast.Target,
		Filter:     make(map[string]any),
	}

	// Build filter from conditions
	if len(ast.Conditions) > 0 {
		filter, err := g.buildFilter(ast.Conditions)
		if err != nil {
			return query, err
		}
		query.Filter = filter
	}

	return query, nil
}

// renderCount generates countDocuments queries
func (g *MongoGenerator) renderCount(ast *astql.QueryAST) (MongoQuery, error) {
	query := MongoQuery{
		Operation:  "countDocuments",
		Collection: ast.Target,
		Filter:     make(map[string]any),
	}

	// Build filter from conditions
	if len(ast.Conditions) > 0 {
		filter, err := g.buildFilter(ast.Conditions)
		if err != nil {
			return query, err
		}
		query.Filter = filter
	}

	return query, nil
}

// buildFilter converts conditions to MongoDB filter
func (g *MongoGenerator) buildFilter(conditions []astql.Condition) (map[string]any, error) {
	if len(conditions) == 0 {
		return make(map[string]any), nil
	}

	// Simple case: all AND conditions
	if g.allAndConditions(conditions) {
		filter := make(map[string]any)
		for _, cond := range conditions {
			fieldFilter, err := g.buildCondition(cond)
			if err != nil {
				return nil, err
			}
			filter[cond.Field] = fieldFilter
		}
		return filter, nil
	}

	// Complex case: mixed AND/OR conditions
	// Build as $and/$or expression
	return g.buildComplexFilter(conditions)
}

// buildCondition converts a single condition to MongoDB filter
func (g *MongoGenerator) buildCondition(cond astql.Condition) (any, error) {
	switch cond.Operator {
	case astql.EQ:
		return cond.Value, nil
	case astql.NE:
		return map[string]any{"$ne": cond.Value}, nil
	case astql.GT:
		return map[string]any{"$gt": cond.Value}, nil
	case astql.GE:
		return map[string]any{"$gte": cond.Value}, nil
	case astql.LT:
		return map[string]any{"$lt": cond.Value}, nil
	case astql.LE:
		return map[string]any{"$lte": cond.Value}, nil
	case astql.IN:
		return map[string]any{"$in": cond.Value}, nil
	case astql.NOT_IN:
		return map[string]any{"$nin": cond.Value}, nil
	case astql.LIKE:
		// Convert SQL LIKE to MongoDB regex
		pattern := g.convertLikeToRegex(cond.Value.(string))
		return map[string]any{"$regex": pattern, "$options": "i"}, nil
	case astql.REGEX:
		return map[string]any{"$regex": cond.Value}, nil
	case astql.IS_NULL:
		return nil, nil
	case astql.IS_NOT_NULL:
		return map[string]any{"$ne": nil}, nil
	case astql.BETWEEN:
		if values, ok := cond.Value.([]any); ok && len(values) == 2 {
			return map[string]any{
				"$gte": values[0],
				"$lte": values[1],
			}, nil
		}
		return nil, fmt.Errorf("BETWEEN requires array of 2 values")
	case astql.EXISTS:
		return map[string]any{"$exists": true}, nil
	default:
		return nil, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}
}

// allAndConditions checks if all conditions use AND logic
func (g *MongoGenerator) allAndConditions(conditions []astql.Condition) bool {
	for _, cond := range conditions {
		if cond.Logical == astql.OR {
			return false
		}
	}
	return true
}

// buildComplexFilter handles mixed AND/OR conditions
func (g *MongoGenerator) buildComplexFilter(conditions []astql.Condition) (map[string]any, error) {
	// For now, build as $and array
	// TODO: Optimize for $or conditions
	andConditions := make([]map[string]any, len(conditions))
	
	for i, cond := range conditions {
		fieldFilter, err := g.buildCondition(cond)
		if err != nil {
			return nil, err
		}
		andConditions[i] = map[string]any{cond.Field: fieldFilter}
	}

	return map[string]any{"$and": andConditions}, nil
}

// convertLikeToRegex converts SQL LIKE pattern to MongoDB regex
func (g *MongoGenerator) convertLikeToRegex(pattern string) string {
	// Replace SQL wildcards with regex equivalents
	// % -> .*
	// _ -> .
	regex := pattern
	regex = fmt.Sprintf("^%s$", regex) // Anchor the pattern
	regex = fmt.Sprintf("%s", regex)   // Escape special chars would go here
	return regex
}