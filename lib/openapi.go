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
	Format      string                    `yaml:"format,omitempty"`
	Description string                    `yaml:"description,omitempty"`
	Ref         string                    `yaml:"$ref,omitempty"`
	Required    []string                  `yaml:"required,omitempty"`
	Properties  map[string]*OpenAPISchema `yaml:"properties,omitempty"`
	Example     any                       `yaml:"example,omitempty"`
}

// OpenAPIResponseApplicationJSON represents the application/json response in an OpenAPI specification
type OpenAPIApplicationJSON struct {
	Schema *OpenAPISchema `yaml:"schema,omitempty"`
}

// OpenAPIResponseContentItem represents a single content item in an OpenAPI responses
type OpenAPIContent struct {
	ApplicationJSON *OpenAPIApplicationJSON `yaml:"application/json,omitempty"`
}

// OpenAPIResponse represents a response in an OpenAPI specification
type OpenAPIResponse struct {
	Description string          `yaml:"description"`
	Content     *OpenAPIContent `yaml:"content,omitempty"`
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

type OpenAPIRef struct {
	Ref string `yaml:"$ref"`
}

// OpenAPIComponents holds the components of an OpenAPI specification, including security schemes and schemas
type OpenAPIComponents struct {
	Parameters      map[string]*OpenAPIParameter      `yaml:"parameters"`
	RequestBodies   map[string]*OpenAPISchema         `yaml:"requestBodies"`
	SecuritySchemes map[string]*OpenAPISecurityScheme `yaml:"securitySchemes"`
	Schemas         map[string]*OpenAPISchema         `yaml:"schemas"`
}

// OpenAPIRequestBody represents a request body in an OpenAPI specification
type OpenAPIRequestBody struct {
	Description string          `yaml:"description,omitempty"`
	Required    bool            `yaml:"required,omitempty"`
	Content     *OpenAPIContent `yaml:"content,omitempty"`
}

// OpenAPIPath represents a single path in an OpenAPI specification, including its operations and parameters
type OpenAPIPath struct {
	Summary     string                   `yaml:"summary,omitempty"`
	Description string                   `yaml:"description,omitempty"`
	OperationId string                   `yaml:"operationId,omitempty"`
	Parameters  []*OpenAPIRef            `yaml:"parameters,omitempty"`
	Tags        []string                 `yaml:"tags,omitempty"`
	Responses   map[int]*OpenAPIResponse `yaml:"responses,omitempty"`
	RequestBody *OpenAPIRequestBody      `yaml:"requestBody,omitempty"`
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
