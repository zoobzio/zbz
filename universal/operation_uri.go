package universal

import (
	"fmt"
	"regexp"
	"strings"
)

// OperationURI represents a URI that points to a specific operation or query
// Used for complex operations via Execute(): queries, batch operations, aggregations
// Does NOT support templates or patterns - operations must be explicitly defined
//
// Examples:
//   - db://queries/find-by-email (named query)
//   - db://operations/bulk-insert (batch operation)
//   - search://queries/faceted-search (search operation)
//   - cache://operations/invalidate-pattern (cache operation)
type OperationURI struct {
	raw       string
	service   string
	category  string
	operation string
}

// NewOperationURI creates a new OperationURI with validation
// Panics if the URI format is invalid - use for package-level constants
func NewOperationURI(uri string) OperationURI {
	parsed, err := parseOperationURI(uri)
	if err != nil {
		panic(fmt.Sprintf("invalid operation URI: %v", err))
	}
	return parsed
}

// ParseOperationURI creates an OperationURI with error handling
// Use when parsing dynamic/user-provided URIs
func ParseOperationURI(uri string) (OperationURI, error) {
	return parseOperationURI(uri)
}

// parseOperationURI handles the actual parsing logic
func parseOperationURI(uri string) (OperationURI, error) {
	// Validate basic URI format
	if !strings.Contains(uri, "://") {
		return OperationURI{}, fmt.Errorf("missing scheme separator '://' in URI: %s", uri)
	}

	// Split scheme and path
	schemeParts := strings.SplitN(uri, "://", 2)
	if len(schemeParts) != 2 {
		return OperationURI{}, fmt.Errorf("invalid URI format: %s", uri)
	}

	service := schemeParts[0]
	path := schemeParts[1]

	// Validate service name
	if !isValidServiceName(service) {
		return OperationURI{}, fmt.Errorf("invalid service name '%s': must be alphanumeric with optional hyphens", service)
	}

	// Operation URIs cannot have empty paths
	if path == "" {
		return OperationURI{}, fmt.Errorf("operation URI must specify a category and operation: %s", uri)
	}

	// Operation URIs cannot contain templates or patterns
	if strings.Contains(path, "{") || strings.Contains(path, "}") {
		return OperationURI{}, fmt.Errorf("operation URI cannot contain templates: %s", uri)
	}
	if strings.Contains(path, "*") || strings.Contains(path, "?") {
		return OperationURI{}, fmt.Errorf("operation URI cannot contain patterns: %s", uri)
	}

	// Parse category and operation from path
	pathParts := strings.Split(path, "/")
	if len(pathParts) != 2 {
		return OperationURI{}, fmt.Errorf("operation URI must have format 'service://category/operation': %s", uri)
	}

	category := pathParts[0]
	operation := pathParts[1]

	// Validate category and operation names
	if !isValidOperationName(category) {
		return OperationURI{}, fmt.Errorf("invalid category name '%s': must be alphanumeric with optional hyphens/underscores", category)
	}
	if !isValidOperationName(operation) {
		return OperationURI{}, fmt.Errorf("invalid operation name '%s': must be alphanumeric with optional hyphens/underscores", operation)
	}

	return OperationURI{
		raw:       uri,
		service:   service,
		category:  category,
		operation: operation,
	}, nil
}

// String returns the URI as a string
func (o OperationURI) String() string {
	return o.raw
}

// Service returns the service name (user-defined when registering provider)
func (o OperationURI) Service() string {
	return o.service
}

// Category returns the operation category (queries, operations, etc.)
func (o OperationURI) Category() string {
	return o.category
}

// Operation returns the specific operation name
func (o OperationURI) Operation() string {
	return o.operation
}

// FullName returns the category and operation as "category/operation"
func (o OperationURI) FullName() string {
	return o.category + "/" + o.operation
}

// isValidOperationName checks if an operation or category name is valid
func isValidOperationName(name string) bool {
	// Operation names can contain alphanumeric chars, hyphens, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched && name != ""
}

// Common operation categories as constants for convenience
const (
	CategoryQueries    = "queries"
	CategoryOperations = "operations"
	CategoryBatch      = "batch"
	CategoryAggregate  = "aggregate"
	CategoryTx         = "tx"
)

// Helper functions for building common operation URIs

// QueryURI builds a query operation URI
func QueryURI(service, queryName string) OperationURI {
	return NewOperationURI(fmt.Sprintf("%s://%s/%s", service, CategoryQueries, queryName))
}

// BatchURI builds a batch operation URI
func BatchURI(service, operationName string) OperationURI {
	return NewOperationURI(fmt.Sprintf("%s://%s/%s", service, CategoryBatch, operationName))
}

// AggregateURI builds an aggregate operation URI
func AggregateURI(service, aggregateName string) OperationURI {
	return NewOperationURI(fmt.Sprintf("%s://%s/%s", service, CategoryAggregate, aggregateName))
}

// TransactionURI builds a transaction operation URI
func TransactionURI(service, txOperation string) OperationURI {
	return NewOperationURI(fmt.Sprintf("%s://%s/%s", service, CategoryTx, txOperation))
}