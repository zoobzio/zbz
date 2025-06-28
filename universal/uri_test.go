package universal

import (
	"testing"
)

func TestResourceURI(t *testing.T) {
	// Test basic resource URI
	userURI := NewResourceURI("db://users/123")
	if userURI.Service() != "db" {
		t.Errorf("Expected service 'db', got '%s'", userURI.Service())
	}
	if userURI.ResourcePath() != "users" {
		t.Errorf("Expected resource 'users', got '%s'", userURI.ResourcePath())
	}
	if userURI.Identifier() != "123" {
		t.Errorf("Expected identifier '123', got '%s'", userURI.Identifier())
	}

	// Test template URI
	templateURI := NewResourceURI("db://users/{id}")
	if !templateURI.HasTemplates() {
		t.Error("Expected URI to have templates")
	}
	
	filledURI := templateURI.With("id", 456)
	if filledURI.Identifier() != "456" {
		t.Errorf("Expected identifier '456', got '%s'", filledURI.Identifier())
	}

	// Test pattern URI
	patternURI := NewResourceURI("bucket://content/*.md")
	if !patternURI.IsPattern() {
		t.Error("Expected URI to be a pattern")
	}

	// Test hierarchical resource
	hierarchicalURI := NewResourceURI("bucket://assets/images/logo.png")
	if hierarchicalURI.ResourcePath() != "assets/images" {
		t.Errorf("Expected resource 'assets/images', got '%s'", hierarchicalURI.ResourcePath())
	}
	if hierarchicalURI.Identifier() != "logo.png" {
		t.Errorf("Expected identifier 'logo.png', got '%s'", hierarchicalURI.Identifier())
	}
}

func TestOperationURI(t *testing.T) {
	// Test basic operation URI
	queryURI := NewOperationURI("db://queries/find-by-email")
	if queryURI.Service() != "db" {
		t.Errorf("Expected service 'db', got '%s'", queryURI.Service())
	}
	if queryURI.Category() != "queries" {
		t.Errorf("Expected category 'queries', got '%s'", queryURI.Category())
	}
	if queryURI.Operation() != "find-by-email" {
		t.Errorf("Expected operation 'find-by-email', got '%s'", queryURI.Operation())
	}

	// Test helper functions
	helperURI := QueryURI("db", "find-active-users")
	expected := "db://queries/find-active-users"
	if helperURI.String() != expected {
		t.Errorf("Expected URI '%s', got '%s'", expected, helperURI.String())
	}
}

func TestURIValidation(t *testing.T) {
	// Test invalid resource URIs
	invalidResourceURIs := []string{
		"I am a banana",           // No scheme
		"db://",                   // Empty path is ok for service-level
		"invalid_service://users", // Invalid service name
	}

	for _, uri := range invalidResourceURIs {
		_, err := ParseResourceURI(uri)
		if err == nil && uri != "db://" { // Empty path is valid
			t.Errorf("Expected error for invalid URI: %s", uri)
		}
	}

	// Test invalid operation URIs
	invalidOperationURIs := []string{
		"db://",                     // Empty path
		"db://queries",              // Missing operation
		"db://queries/find/{id}",    // Templates not allowed
		"db://queries/find/*",       // Patterns not allowed
		"db://too/many/path/parts",  // Too many path parts
	}

	for _, uri := range invalidOperationURIs {
		_, err := ParseOperationURI(uri)
		if err == nil {
			t.Errorf("Expected error for invalid operation URI: %s", uri)
		}
	}
}

func TestResourceURIImmutability(t *testing.T) {
	original := NewResourceURI("db://users/{id}")
	modified := original.With("id", 123)

	// Original should be unchanged
	if !original.HasTemplates() {
		t.Error("Original URI should still have templates")
	}
	if original.String() != "db://users/{id}" {
		t.Errorf("Original URI changed: %s", original.String())
	}

	// Modified should have filled template
	if modified.HasTemplates() {
		t.Error("Modified URI should not have templates")
	}
	if modified.Identifier() != "123" {
		t.Errorf("Modified URI identifier should be '123', got '%s'", modified.Identifier())
	}
}