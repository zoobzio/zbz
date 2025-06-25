# ZBZ Docula

*"Docula documents things!"* 📖

Standalone OpenAPI documentation generation and management system for Go applications. Docula provides comprehensive schema generation, endpoint registration, and spec export capabilities with contract-based configuration.

## Features

- **Complete OpenAPI 3.1.0 Support**: Full specification generation with modern JSON Schema features
- **Automatic Schema Generation**: Generate schemas from Go structs with validation constraints
- **Endpoint Registration**: Declarative API endpoint documentation
- **Multiple Export Formats**: YAML and JSON export capabilities
- **Contract Configuration**: Declarative service setup and management
- **Thread Safe**: Concurrent operations with proper synchronization
- **Zero Dependencies**: Core functionality with minimal external dependencies

## Quick Start

```go
package main

import (
    "zbz/docula"
)

func main() {
    // Define a documentation contract
    contract := docula.DoculaContract{
        Name:        "api-docs",
        Description: "API documentation service",
        Info: &docula.OpenAPIInfo{
            Title:       "My API",
            Version:     "1.0.0",
            Description: "A sample API built with ZBZ framework",
        },
    }
    
    // Get Docula instance
    docs := contract.Docula()
    
    // Register a model schema
    type User struct {
        ID    int    `json:"id" description:"User ID"`
        Name  string `json:"name" description:"User name"`
        Email string `json:"email" description:"User email"`
    }
    
    docs.RegisterModel("User", User{})
    
    // Register an endpoint
    docs.RegisterEndpoint(
        "GET", "/users/{id}", "users", 
        "Get User", "Retrieve a user by ID",
        []*docula.OpenAPIParameter{
            {
                Name: "id",
                In: "path", 
                Description: "User ID",
                Required: true,
                Schema: &docula.OpenAPISchema{Type: "integer"},
            },
        },
        nil, // no request body
        map[string]*docula.OpenAPIResponse{
            "200": {
                Description: "User found",
                Content: map[string]*docula.OpenAPIMediaType{
                    "application/json": {
                        Schema: &docula.OpenAPISchema{
                            Ref: "#/components/schemas/User",
                        },
                    },
                },
            },
        },
    )
    
    // Export specification
    yamlSpec, err := docs.ToYAML()
    if err != nil {
        panic(err)
    }
    
    // Save or serve the specification
    fmt.Println(string(yamlSpec))
}
```

## Contract-Based Configuration

Docula uses contracts for declarative documentation setup:

```go
// Production API documentation
prodContract := docula.DoculaContract{
    Name:        "production-api",
    Description: "Production API documentation",
    Info: &docula.OpenAPIInfo{
        Title:       "Production API",
        Version:     "2.1.0",
        Description: "Production-ready API with full documentation",
        Contact: &docula.OpenAPIContact{
            Name:  "API Team",
            Email: "api@company.com",
            URL:   "https://company.com/support",
        },
        License: &docula.OpenAPILicense{
            Name: "MIT",
            URL:  "https://opensource.org/licenses/MIT",
        },
    },
    Servers: []*docula.OpenAPIServer{
        {
            URL:         "https://api.company.com/v2",
            Description: "Production server",
        },
        {
            URL:         "https://staging.api.company.com/v2", 
            Description: "Staging server",
        },
    },
}

// Get instance (cached by contract)
docs := prodContract.Docula()
```

## DoculaService Interface

Core documentation operations:

```go
type DoculaService interface {
    // Get the complete OpenAPI specification
    GetSpec() *OpenAPISpec
    
    // Register a Go struct as an OpenAPI schema
    RegisterModel(name string, model any) error
    
    // Register an API endpoint with full documentation
    RegisterEndpoint(method, path, tag, summary, description string, 
        params []*OpenAPIParameter, 
        requestBody *OpenAPIRequestBody, 
        responses map[string]*OpenAPIResponse) error
    
    // Update API metadata
    SetInfo(info *OpenAPIInfo) error
    
    // Add servers
    AddServer(server *OpenAPIServer) error
    
    // Add tags for organization
    AddTag(tag *OpenAPITag) error
    
    // Add security schemes
    AddSecurityScheme(name string, scheme *OpenAPISecurityScheme) error
    
    // Export as YAML
    ToYAML() ([]byte, error)
    
    // Export as JSON
    ToJSON() ([]byte, error)
}
```

## Schema Generation

Docula automatically generates OpenAPI schemas from Go structs:

```go
type Product struct {
    ID          int       `json:"id" description:"Product ID"`
    Name        string    `json:"name" description:"Product name" validate:"required,min=3,max=100"`
    Price       float64   `json:"price" description:"Product price" validate:"gt=0"`
    Category    string    `json:"category" description:"Product category" validate:"oneof=electronics clothing books"`
    InStock     bool      `json:"in_stock" description:"Whether product is in stock"`
    CreatedAt   time.Time `json:"created_at" description:"Creation timestamp"`
    Tags        []string  `json:"tags,omitempty" description:"Product tags"`
}

docs.RegisterModel("Product", Product{})
```

**Schema Features:**
- Automatic type mapping (string, integer, number, boolean, array, object)
- Validation constraint extraction from struct tags
- Optional/required field detection
- Nested struct support
- Array and map handling
- Custom descriptions via tags

## API Documentation

Register endpoints with comprehensive documentation:

```go
// POST /products endpoint
docs.RegisterEndpoint(
    "POST", "/products", "products",
    "Create Product", "Create a new product in the catalog",
    nil, // no path/query parameters
    &docula.OpenAPIRequestBody{
        Description: "Product data",
        Required:    true,
        Content: map[string]*docula.OpenAPIMediaType{
            "application/json": {
                Schema: &docula.OpenAPISchema{
                    Ref: "#/components/schemas/Product",
                },
            },
        },
    },
    map[string]*docula.OpenAPIResponse{
        "201": {
            Description: "Product created successfully",
            Content: map[string]*docula.OpenAPIMediaType{
                "application/json": {
                    Schema: &docula.OpenAPISchema{
                        Ref: "#/components/schemas/Product",
                    },
                },
            },
        },
        "400": {
            Description: "Invalid product data",
        },
        "500": {
            Description: "Internal server error",
        },
    },
)
```

## Framework Integration

Docula integrates seamlessly with frameworks and applications:

```go
// Framework usage - integrated via contracts
import "zbz/api"  // Docula included automatically

// Standalone usage
import "zbz/docula"
```

## Singleton Pattern

Docula uses a singleton pattern with contract-based initialization:

```go
// Multiple contracts can be used, but each creates a singleton
contract := docula.DoculaContract{Name: "my-api"}
docs1 := contract.Docula() // Creates singleton
docs2 := contract.Docula() // Returns same instance

// Different contracts would create different instances if needed
```

## Export and Integration

Export specifications in multiple formats:

```go
// Export as YAML (preferred for human reading)
yamlData, err := docs.ToYAML()

// Export as JSON (for programmatic use)
jsonData, err := docs.ToJSON()

// Integrate with documentation tools
// - Serve via HTTP endpoints
// - Generate static documentation sites
// - Import into API management platforms
```

Perfect for applications requiring comprehensive API documentation generation with minimal setup and maximum flexibility.