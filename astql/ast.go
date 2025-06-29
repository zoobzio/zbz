package astql

import "fmt"

// OperationType represents the type of query operation
type OperationType int

const (
	OpSelect OperationType = iota
	OpInsert
	OpUpdate
	OpDelete
	OpCount
	OpAggregate
)

func (o OperationType) String() string {
	switch o {
	case OpSelect:
		return "SELECT"
	case OpInsert:
		return "INSERT"
	case OpUpdate:
		return "UPDATE"
	case OpDelete:
		return "DELETE"
	case OpCount:
		return "COUNT"
	case OpAggregate:
		return "AGGREGATE"
	default:
		return "UNKNOWN"
	}
}

// Operator represents comparison operators
type Operator int

const (
	EQ Operator = iota // =
	NE                 // !=
	GT                 // >
	GE                 // >=
	LT                 // <
	LE                 // <=
	IN                 // IN
	NOT_IN             // NOT IN
	LIKE               // LIKE
	REGEX              // REGEX/~
	IS_NULL            // IS NULL
	IS_NOT_NULL        // IS NOT NULL
	BETWEEN            // BETWEEN
	EXISTS             // EXISTS
	CONTAINS           // CONTAINS (for arrays/json)
)

func (o Operator) String() string {
	switch o {
	case EQ:
		return "="
	case NE:
		return "!="
	case GT:
		return ">"
	case GE:
		return ">="
	case LT:
		return "<"
	case LE:
		return "<="
	case IN:
		return "IN"
	case NOT_IN:
		return "NOT IN"
	case LIKE:
		return "LIKE"
	case REGEX:
		return "~"
	case IS_NULL:
		return "IS NULL"
	case IS_NOT_NULL:
		return "IS NOT NULL"
	case BETWEEN:
		return "BETWEEN"
	case EXISTS:
		return "EXISTS"
	case CONTAINS:
		return "CONTAINS"
	default:
		return "?"
	}
}

// LogicalOp represents logical operators for combining conditions
type LogicalOp int

const (
	AND LogicalOp = iota
	OR
	NOT
)

// SortDirection represents sort order
type SortDirection string

const (
	ASC  SortDirection = "ASC"
	DESC SortDirection = "DESC"
)

// QueryAST represents a universal query structure that can be rendered to any query language
type QueryAST struct {
	Operation  OperationType    // What kind of query
	Target     string           // Table/Collection/Index name
	Fields     []Field          // Selected/Inserted/Updated fields
	Conditions []Condition      // WHERE/Filter conditions
	Joins      []Join           // JOIN operations
	Ordering   []Order          // ORDER BY / sort
	Grouping   []Group          // GROUP BY
	Having     []Condition      // HAVING conditions
	Limit      *int             // Result limit
	Offset     *int             // Result offset
	Returning  []string         // RETURNING clause fields
	Values     []map[string]any // For INSERT operations
	Updates    map[string]any   // For UPDATE operations
	Hints      []Hint           // Provider-specific hints
}

// Field represents a field in a query
type Field struct {
	Name     string // Field name
	Alias    string // Optional alias
	Function string // Optional function (COUNT, SUM, etc.)
}

// Condition represents a query condition
type Condition struct {
	Field     string      // Field name
	Operator  Operator    // Comparison operator
	Value     any         // Comparison value (can be nil for IS NULL)
	Logical   LogicalOp   // How this condition connects to the next
	Nested    []Condition // For nested conditions (OR/AND groups)
	ParamName string      // Named parameter for this condition
}

// Join represents a JOIN operation
type Join struct {
	Type      string // INNER, LEFT, RIGHT, FULL
	Target    string // Table to join
	Alias     string // Table alias
	Condition string // Join condition
}

// Order represents an ORDER BY clause
type Order struct {
	Field     string
	Direction SortDirection
}

// Group represents a GROUP BY clause
type Group struct {
	Field string
}

// Hint represents a provider-specific hint
type Hint struct {
	Provider string // Which provider this hint is for
	Type     string // Hint type (index, parallel, etc.)
	Value    string // Hint value
}

// Validate checks if the AST is valid
func (ast *QueryAST) Validate() error {
	if ast.Target == "" {
		return fmt.Errorf("target (table/collection) is required")
	}

	switch ast.Operation {
	case OpSelect:
		// SELECT can have empty fields (SELECT *)
	case OpInsert:
		if len(ast.Values) == 0 {
			return fmt.Errorf("INSERT requires at least one value")
		}
	case OpUpdate:
		if len(ast.Updates) == 0 {
			return fmt.Errorf("UPDATE requires at least one field to update")
		}
	case OpDelete:
		// DELETE doesn't require fields
	}

	return nil
}

// Clone creates a deep copy of the AST
func (ast *QueryAST) Clone() *QueryAST {
	clone := &QueryAST{
		Operation: ast.Operation,
		Target:    ast.Target,
		Fields:    make([]Field, len(ast.Fields)),
		Conditions: make([]Condition, len(ast.Conditions)),
		Joins:     make([]Join, len(ast.Joins)),
		Ordering:  make([]Order, len(ast.Ordering)),
		Grouping:  make([]Group, len(ast.Grouping)),
		Having:    make([]Condition, len(ast.Having)),
		Hints:     make([]Hint, len(ast.Hints)),
	}

	copy(clone.Fields, ast.Fields)
	copy(clone.Conditions, ast.Conditions)
	copy(clone.Joins, ast.Joins)
	copy(clone.Ordering, ast.Ordering)
	copy(clone.Grouping, ast.Grouping)
	copy(clone.Having, ast.Having)
	copy(clone.Hints, ast.Hints)

	if ast.Limit != nil {
		limit := *ast.Limit
		clone.Limit = &limit
	}
	if ast.Offset != nil {
		offset := *ast.Offset
		clone.Offset = &offset
	}

	// Deep copy values and updates
	if ast.Values != nil {
		clone.Values = make([]map[string]any, len(ast.Values))
		for i, v := range ast.Values {
			clone.Values[i] = make(map[string]any)
			for k, val := range v {
				clone.Values[i][k] = val
			}
		}
	}

	if ast.Updates != nil {
		clone.Updates = make(map[string]any)
		for k, v := range ast.Updates {
			clone.Updates[k] = v
		}
	}

	return clone
}