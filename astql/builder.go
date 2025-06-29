package astql

import (
	"fmt"
	"strings"
)

// Builder provides a fluent interface for constructing QueryAST
type Builder struct {
	ast *QueryAST
}

// NewBuilder creates a new query builder
func NewBuilder() *Builder {
	return &Builder{
		ast: &QueryAST{
			Fields:     []Field{},
			Conditions: []Condition{},
			Joins:      []Join{},
			Ordering:   []Order{},
			Grouping:   []Group{},
			Having:     []Condition{},
			Hints:      []Hint{},
		},
	}
}

// Select starts a SELECT query
func Select(target string) *Builder {
	b := NewBuilder()
	b.ast.Operation = OpSelect
	b.ast.Target = target
	return b
}

// Insert starts an INSERT query
func Insert(target string) *Builder {
	b := NewBuilder()
	b.ast.Operation = OpInsert
	b.ast.Target = target
	b.ast.Values = []map[string]any{}
	return b
}

// Update starts an UPDATE query
func Update(target string) *Builder {
	b := NewBuilder()
	b.ast.Operation = OpUpdate
	b.ast.Target = target
	b.ast.Updates = make(map[string]any)
	return b
}

// Delete starts a DELETE query
func Delete(target string) *Builder {
	b := NewBuilder()
	b.ast.Operation = OpDelete
	b.ast.Target = target
	return b
}

// Count starts a COUNT query
func Count(target string) *Builder {
	b := NewBuilder()
	b.ast.Operation = OpCount
	b.ast.Target = target
	return b
}

// Fields specifies which fields to select
func (b *Builder) Fields(fields ...string) *Builder {
	for _, f := range fields {
		b.ast.Fields = append(b.ast.Fields, Field{Name: f})
	}
	return b
}

// Field adds a single field with optional alias
func (b *Builder) Field(name, alias string) *Builder {
	b.ast.Fields = append(b.ast.Fields, Field{Name: name, Alias: alias})
	return b
}

// Where adds a condition
func (b *Builder) Where(field string, op Operator, value any) *Builder {
	// Generate parameter name from field
	paramName := strings.ReplaceAll(field, ".", "_")
	
	b.ast.Conditions = append(b.ast.Conditions, Condition{
		Field:     field,
		Operator:  op,
		Value:     value,
		Logical:   AND,
		ParamName: paramName,
	})
	return b
}

// WhereRaw adds a condition with custom parameter name
func (b *Builder) WhereRaw(field string, op Operator, value any, paramName string) *Builder {
	b.ast.Conditions = append(b.ast.Conditions, Condition{
		Field:     field,
		Operator:  op,
		Value:     value,
		Logical:   AND,
		ParamName: paramName,
	})
	return b
}

// OrWhere adds an OR condition
func (b *Builder) OrWhere(field string, op Operator, value any) *Builder {
	paramName := strings.ReplaceAll(field, ".", "_")
	
	// If this is the first condition, treat it as AND
	if len(b.ast.Conditions) == 0 {
		return b.Where(field, op, value)
	}
	
	// Change the previous condition's logical operator to OR
	b.ast.Conditions[len(b.ast.Conditions)-1].Logical = OR
	
	b.ast.Conditions = append(b.ast.Conditions, Condition{
		Field:     field,
		Operator:  op,
		Value:     value,
		Logical:   AND,
		ParamName: paramName,
	})
	return b
}

// WhereNull adds an IS NULL condition
func (b *Builder) WhereNull(field string) *Builder {
	return b.Where(field, IS_NULL, nil)
}

// WhereNotNull adds an IS NOT NULL condition
func (b *Builder) WhereNotNull(field string) *Builder {
	return b.Where(field, IS_NOT_NULL, nil)
}

// WhereIn adds an IN condition
func (b *Builder) WhereIn(field string, values any) *Builder {
	return b.Where(field, IN, values)
}

// WhereBetween adds a BETWEEN condition
func (b *Builder) WhereBetween(field string, start, end any) *Builder {
	return b.Where(field, BETWEEN, []any{start, end})
}

// Join adds a JOIN clause
func (b *Builder) Join(joinType, target, condition string) *Builder {
	b.ast.Joins = append(b.ast.Joins, Join{
		Type:      joinType,
		Target:    target,
		Condition: condition,
	})
	return b
}

// InnerJoin adds an INNER JOIN
func (b *Builder) InnerJoin(target, condition string) *Builder {
	return b.Join("INNER", target, condition)
}

// LeftJoin adds a LEFT JOIN
func (b *Builder) LeftJoin(target, condition string) *Builder {
	return b.Join("LEFT", target, condition)
}

// OrderBy adds an ORDER BY clause
func (b *Builder) OrderBy(field string, direction SortDirection) *Builder {
	b.ast.Ordering = append(b.ast.Ordering, Order{
		Field:     field,
		Direction: direction,
	})
	return b
}

// OrderByAsc adds ascending order
func (b *Builder) OrderByAsc(field string) *Builder {
	return b.OrderBy(field, ASC)
}

// OrderByDesc adds descending order
func (b *Builder) OrderByDesc(field string) *Builder {
	return b.OrderBy(field, DESC)
}

// GroupBy adds a GROUP BY clause
func (b *Builder) GroupBy(fields ...string) *Builder {
	for _, f := range fields {
		b.ast.Grouping = append(b.ast.Grouping, Group{Field: f})
	}
	return b
}

// Having adds a HAVING condition
func (b *Builder) Having(field string, op Operator, value any) *Builder {
	paramName := fmt.Sprintf("having_%s", strings.ReplaceAll(field, ".", "_"))
	
	b.ast.Having = append(b.ast.Having, Condition{
		Field:     field,
		Operator:  op,
		Value:     value,
		Logical:   AND,
		ParamName: paramName,
	})
	return b
}

// Limit sets the result limit
func (b *Builder) Limit(limit int) *Builder {
	b.ast.Limit = &limit
	return b
}

// Offset sets the result offset
func (b *Builder) Offset(offset int) *Builder {
	b.ast.Offset = &offset
	return b
}

// Paginate sets limit and offset for pagination
func (b *Builder) Paginate(page, pageSize int) *Builder {
	offset := (page - 1) * pageSize
	return b.Limit(pageSize).Offset(offset)
}

// Values adds values for INSERT
func (b *Builder) Values(values map[string]any) *Builder {
	if b.ast.Operation != OpInsert {
		return b
	}
	b.ast.Values = append(b.ast.Values, values)
	return b
}

// Set adds a field update for UPDATE queries
func (b *Builder) Set(field string, value any) *Builder {
	if b.ast.Operation != OpUpdate {
		return b
	}
	b.ast.Updates[field] = value
	return b
}

// Returning specifies fields to return after INSERT/UPDATE/DELETE
func (b *Builder) Returning(fields ...string) *Builder {
	b.ast.Returning = fields
	return b
}

// Hint adds a provider-specific hint
func (b *Builder) Hint(provider, hintType, value string) *Builder {
	b.ast.Hints = append(b.ast.Hints, Hint{
		Provider: provider,
		Type:     hintType,
		Value:    value,
	})
	return b
}

// Build returns the constructed AST
func (b *Builder) Build() (*QueryAST, error) {
	if err := b.ast.Validate(); err != nil {
		return nil, err
	}
	return b.ast.Clone(), nil
}

// MustBuild returns the AST or panics on error
func (b *Builder) MustBuild() *QueryAST {
	ast, err := b.Build()
	if err != nil {
		panic(err)
	}
	return ast
}