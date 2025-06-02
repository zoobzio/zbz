package zbz

// OpenAPIInfo represents the metadata for an OpenAPI specification
type OpenAPIInfo struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// OpenAPIResponse represents a response in an OpenAPI specification
type OpenAPIResponse struct {
	Description string `yaml:"description"`
}

// OpenAPISecurityScheme represents a security scheme in an OpenAPI specification
type OpenAPISecurityScheme struct {
	Type   string `yaml:"type"`
	Scheme string `yaml:"scheme,omitempty"`
}

// OpenAPISchema represents a schema in an OpenAPI specification
type OpenAPISchema struct {
	Type        string                    `yaml:"type"`
	Description string                    `yaml:"description,omitempty"`
	Ref         string                    `yaml:"$ref,omitempty"` // Reference to another schema
	Required    []string                  `yaml:"required,omitempty"`
	Properties  map[string]*OpenAPISchema `yaml:"properties,omitempty"`
	Example     any                       `yaml:"example,omitempty"`
}

// OpenAPIComponents holds the components of an OpenAPI specification, including security schemes and schemas
type OpenAPIComponents struct {
	SecuritySchemes map[string]*OpenAPISecurityScheme `yaml:"securitySchemes"`
	Schemas         map[string]*OpenAPISchema         `yaml:"schemas"`
}

// OpenAPIPath represents a single path in an OpenAPI specification, including its operations and parameters
type OpenAPIPath struct {
	Summary     string                      `yaml:"summary,omitempty"`
	Description string                      `yaml:"description,omitempty"`
	OperationId string                      `yaml:"operationId,omitempty"`
	Parameters  []map[string]any            `yaml:"parameters,omitempty"`
	Tags        []string                    `yaml:"tags,omitempty"`
	Responses   map[string]*OpenAPIResponse `yaml:"responses,omitempty"`
	Security    []map[string][]string       `yaml:"security,omitempty"`
}

// OpenAPISpec represents the entire OpenAPI specification, including metadata, components, paths, and tags
type OpenAPISpec struct {
	OpenAPI    string                             `yaml:"openapi"`
	Info       *OpenAPIInfo                       `yaml:"info"`
	Components *OpenAPIComponents                 `yaml:"components"`
	Paths      map[string]map[string]*OpenAPIPath `yaml:"paths"`
	Tags       []map[string]string                `yaml:"tags"`
}
