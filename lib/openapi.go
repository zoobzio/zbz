package zbz

// OpenAPIInfo represents the metadata for an OpenAPI specification
type OpenAPIInfo struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// OpenAPISchema represents a schema in an OpenAPI specification
type OpenAPISchema struct {
	Type        string                    `yaml:"type,omitempty"`
	Description string                    `yaml:"description,omitempty"`
	Ref         string                    `yaml:"$ref,omitempty"`
	Required    []string                  `yaml:"required,omitempty"`
	Properties  map[string]*OpenAPISchema `yaml:"properties,omitempty"`
	Example     any                       `yaml:"example,omitempty"`
}

// OpenAPIResponseApplicationJSON represents the application/json response in an OpenAPI specification
type OpenAPIResponseApplicationJSON struct {
	Schema *OpenAPISchema `yaml:"schema,omitempty"`
}

// OpenAPIResponseContentItem represents a single content item in an OpenAPI responses
type OpenAPIResponseContent struct {
	ApplicationJSON *OpenAPIResponseApplicationJSON `yaml:"application/json,omitempty"`
}

// OpenAPIResponse represents a response in an OpenAPI specification
type OpenAPIResponse struct {
	Description string                  `yaml:"description"`
	Content     *OpenAPIResponseContent `yaml:"content,omitempty"`
}

// OpenAPISecurityScheme represents a security scheme in an OpenAPI specification
type OpenAPISecurityScheme struct {
	Type   string `yaml:"type"`
	Scheme string `yaml:"scheme,omitempty"`
}

// OpenAPIParameter represents a parameter in an OpenAPI specification
type OpenAPIParameter struct {
	Name        string         `yaml:"name"`
	In          string         `yaml:"in"`
	Description string         `yaml:"description,omitempty"`
	Required    bool           `yaml:"required,omitempty"`
	Schema      *OpenAPISchema `yaml:"schema,omitempty"`
}

// OpenAPIComponents holds the components of an OpenAPI specification, including security schemes and schemas
type OpenAPIComponents struct {
	SecuritySchemes map[string]*OpenAPISecurityScheme `yaml:"securitySchemes"`
	Schemas         map[string]*OpenAPISchema         `yaml:"schemas"`
}

// OpenAPIPath represents a single path in an OpenAPI specification, including its operations and parameters
type OpenAPIPath struct {
	Summary     string                   `yaml:"summary,omitempty"`
	Description string                   `yaml:"description,omitempty"`
	OperationId string                   `yaml:"operationId,omitempty"`
	Parameters  []*OpenAPIParameter      `yaml:"parameters,omitempty"`
	Tags        []string                 `yaml:"tags,omitempty"`
	Responses   map[int]*OpenAPIResponse `yaml:"responses,omitempty"`
	Security    []map[string][]string    `yaml:"security,omitempty"`
}

// OpenAPISpec represents the entire OpenAPI specification, including metadata, components, paths, and tags
type OpenAPISpec struct {
	OpenAPI    string                             `yaml:"openapi"`
	Info       *OpenAPIInfo                       `yaml:"info"`
	Components *OpenAPIComponents                 `yaml:"components"`
	Paths      map[string]map[string]*OpenAPIPath `yaml:"paths"`
	Tags       []map[string]string                `yaml:"tags"`
}
