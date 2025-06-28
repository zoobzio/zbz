package universal

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ResourceURI represents a URI that points to a specific resource or resource pattern
// Used for direct CRUD operations: Get, Set, Delete, List, Exists
// Supports templates and patterns for flexible resource addressing
//
// Examples:
//   - db://users/123 (specific user record)
//   - bucket://config.yaml (specific file)
//   - cache://sessions/{id} (templated cache key)
//   - db://posts/* (pattern for all posts)
//   - bucket://content/*.md (pattern for all markdown files)
type ResourceURI struct {
	raw        string
	service    string
	resource   []string
	identifier string
	templates  map[string]string
	isPattern  bool
}

// ResourceURI creates a new ResourceURI with validation
// Panics if the URI format is invalid - use for package-level constants
func NewResourceURI(uri string) ResourceURI {
	parsed, err := parseResourceURI(uri)
	if err != nil {
		panic(fmt.Sprintf("invalid resource URI: %v", err))
	}
	return parsed
}

// ParseResourceURI creates a ResourceURI with error handling
// Use when parsing dynamic/user-provided URIs
func ParseResourceURI(uri string) (ResourceURI, error) {
	return parseResourceURI(uri)
}

// parseResourceURI handles the actual parsing logic
func parseResourceURI(uri string) (ResourceURI, error) {
	// Validate basic URI format
	if !strings.Contains(uri, "://") {
		return ResourceURI{}, fmt.Errorf("missing scheme separator '://' in URI: %s", uri)
	}

	// Split scheme and path
	schemeParts := strings.SplitN(uri, "://", 2)
	if len(schemeParts) != 2 {
		return ResourceURI{}, fmt.Errorf("invalid URI format: %s", uri)
	}

	service := schemeParts[0]
	path := schemeParts[1]

	// Validate service name (alphanumeric + hyphens only)
	if !isValidServiceName(service) {
		return ResourceURI{}, fmt.Errorf("invalid service name '%s': must be alphanumeric with optional hyphens", service)
	}

	// Handle empty path (service-level resource)
	if path == "" {
		return ResourceURI{
			raw:        uri,
			service:    service,
			resource:   []string{},
			identifier: "",
		}, nil
	}

	// Split path into resource hierarchy and identifier
	pathParts := strings.Split(path, "/")
	var resource []string
	var identifier string

	if len(pathParts) == 1 {
		// Single segment - could be resource or identifier
		identifier = pathParts[0]
	} else {
		// Multiple segments - last is identifier, rest is resource hierarchy
		resource = pathParts[:len(pathParts)-1]
		identifier = pathParts[len(pathParts)-1]
	}

	// Extract templates from identifier
	templates := extractTemplates(identifier)

	// Check if this is a pattern (contains wildcards)
	isPattern := strings.Contains(identifier, "*") || strings.Contains(identifier, "?")

	// Extract query parameters if present
	if strings.Contains(identifier, "?") {
		idParts := strings.SplitN(identifier, "?", 2)
		identifier = idParts[0]
		// Parse query parameters and merge with templates
		if queryParams, err := url.ParseQuery(idParts[1]); err == nil {
			for key, values := range queryParams {
				if len(values) > 0 {
					templates[key] = values[0]
				}
			}
		}
	}

	return ResourceURI{
		raw:        uri,
		service:    service,
		resource:   resource,
		identifier: identifier,
		templates:  templates,
		isPattern:  isPattern,
	}, nil
}

// String returns the URI as a string
func (r ResourceURI) String() string {
	return r.raw
}

// Service returns the service name (user-defined when registering provider)
func (r ResourceURI) Service() string {
	return r.service
}

// Resource returns the resource hierarchy as a slice
func (r ResourceURI) Resource() []string {
	return append([]string(nil), r.resource...) // Return copy
}

// ResourcePath returns the resource hierarchy as a single path string
func (r ResourceURI) ResourcePath() string {
	return strings.Join(r.resource, "/")
}

// Identifier returns the resource identifier (last path segment)
func (r ResourceURI) Identifier() string {
	return r.identifier
}

// IsPattern returns true if the URI contains wildcard patterns
func (r ResourceURI) IsPattern() bool {
	return r.isPattern
}

// HasTemplates returns true if the URI contains template placeholders
func (r ResourceURI) HasTemplates() bool {
	return len(r.templates) > 0
}

// Templates returns a copy of the template placeholders found in the URI
func (r ResourceURI) Templates() map[string]string {
	result := make(map[string]string)
	for k, v := range r.templates {
		result[k] = v
	}
	return result
}

// With returns a new ResourceURI with template placeholders filled in
// This is immutable - the original URI is unchanged
func (r ResourceURI) With(key string, value any) ResourceURI {
	// Create new URI by replacing template placeholder
	newURI := strings.ReplaceAll(r.raw, "{"+key+"}", fmt.Sprintf("%v", value))

	// Parse the new URI
	parsed, err := parseResourceURI(newURI)
	if err != nil {
		// If parsing fails, return original (this shouldn't happen with valid templates)
		return r
	}

	return parsed
}

// WithParams returns a new ResourceURI with multiple template placeholders filled in
func (r ResourceURI) WithParams(params map[string]any) ResourceURI {
	newURI := r.raw
	for key, value := range params {
		newURI = strings.ReplaceAll(newURI, "{"+key+"}", fmt.Sprintf("%v", value))
	}

	parsed, err := parseResourceURI(newURI)
	if err != nil {
		return r
	}

	return parsed
}

// WithQuery returns a new ResourceURI with query parameters added
func (r ResourceURI) WithQuery(params map[string]string) ResourceURI {
	if len(params) == 0 {
		return r
	}

	// Build query string
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	queryString := values.Encode()

	// Append to URI
	separator := "?"
	if strings.Contains(r.raw, "?") {
		separator = "&"
	}
	newURI := r.raw + separator + queryString

	parsed, err := parseResourceURI(newURI)
	if err != nil {
		return r
	}

	return parsed
}

// Join appends path segments to the resource hierarchy
func (r ResourceURI) Join(segments ...string) ResourceURI {
	if len(segments) == 0 {
		return r
	}

	// Build new path by joining segments
	newPath := r.ResourcePath()
	for _, segment := range segments {
		if newPath != "" {
			newPath += "/"
		}
		newPath += segment
	}

	// Add identifier back if it exists
	if r.identifier != "" {
		if newPath != "" {
			newPath += "/"
		}
		newPath += r.identifier
	}

	// Build new URI
	newURI := r.service + "://" + newPath

	parsed, err := parseResourceURI(newURI)
	if err != nil {
		return r
	}

	return parsed
}

// extractTemplates finds template placeholders in a string
func extractTemplates(s string) map[string]string {
	templates := make(map[string]string)
	
	// Find all {placeholder} patterns
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(s, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			templates[match[1]] = match[0] // key -> {key}
		}
	}
	
	return templates
}

// isValidServiceName checks if a service name is valid
func isValidServiceName(name string) bool {
	// Service names must be alphanumeric with optional hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, name)
	return matched && name != ""
}