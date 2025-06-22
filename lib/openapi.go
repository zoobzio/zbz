package zbz

// OpenAPIInfo represents the metadata for an OpenAPI specification
type OpenAPIInfo struct {
	Title          string          `yaml:"title"`
	Summary        string          `yaml:"summary,omitempty"`
	Description    string          `yaml:"description,omitempty"`
	Version        string          `yaml:"version"`
	TermsOfService string          `yaml:"termsOfService,omitempty"`
	Contact        *OpenAPIContact `yaml:"contact,omitempty"`
	License        *OpenAPILicense `yaml:"license,omitempty"`
}

// OpenAPIContact represents contact information
type OpenAPIContact struct {
	Name  string `yaml:"name,omitempty"`
	URL   string `yaml:"url,omitempty"`
	Email string `yaml:"email,omitempty"`
}

// OpenAPILicense represents license information
type OpenAPILicense struct {
	Name       string `yaml:"name"`
	Identifier string `yaml:"identifier,omitempty"`
	URL        string `yaml:"url,omitempty"`
}

// OpenAPISchema represents a schema in an OpenAPI specification
type OpenAPISchema struct {
	Type                 string                    `yaml:"type,omitempty"`
	Format               string                    `yaml:"format,omitempty"`
	Description          string                    `yaml:"description,omitempty"`
	Ref                  string                    `yaml:"$ref,omitempty"`
	Required             []string                  `yaml:"required,omitempty"`
	Properties           map[string]*OpenAPISchema `yaml:"properties,omitempty"`
	Example              any                       `yaml:"example,omitempty"`
	AdditionalProperties *OpenAPISchema            `yaml:"additionalProperties,omitempty"`
	
	// OpenAPI 3.1.0 / JSON Schema 2020-12 enhancements
	Const       any                  `yaml:"const,omitempty"`
	AnyOf       []*OpenAPISchema     `yaml:"anyOf,omitempty"`
	OneOf       []*OpenAPISchema     `yaml:"oneOf,omitempty"`
	Not         *OpenAPISchema       `yaml:"not,omitempty"`
	If          *OpenAPISchema       `yaml:"if,omitempty"`
	Then        *OpenAPISchema       `yaml:"then,omitempty"`
	Else        *OpenAPISchema       `yaml:"else,omitempty"`
	
	// Validation keywords
	Minimum          *float64 `yaml:"minimum,omitempty"`
	Maximum          *float64 `yaml:"maximum,omitempty"`
	ExclusiveMinimum *float64 `yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `yaml:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `yaml:"multipleOf,omitempty"`
	
	// String validation
	MinLength *int    `yaml:"minLength,omitempty"`
	MaxLength *int    `yaml:"maxLength,omitempty"`
	Pattern   string  `yaml:"pattern,omitempty"`
	
	// Array validation
	Items           *OpenAPISchema   `yaml:"items,omitempty"`
	PrefixItems     []*OpenAPISchema `yaml:"prefixItems,omitempty"`
	Contains        *OpenAPISchema   `yaml:"contains,omitempty"`
	MinItems        *int             `yaml:"minItems,omitempty"`
	MaxItems        *int             `yaml:"maxItems,omitempty"`
	UniqueItems     bool             `yaml:"uniqueItems,omitempty"`
	UnevaluatedItems *OpenAPISchema  `yaml:"unevaluatedItems,omitempty"`
	
	// Object validation
	MinProperties        *int                      `yaml:"minProperties,omitempty"`
	MaxProperties        *int                      `yaml:"maxProperties,omitempty"`
	PropertyNames        *OpenAPISchema            `yaml:"propertyNames,omitempty"`
	PatternProperties    map[string]*OpenAPISchema `yaml:"patternProperties,omitempty"`
	UnevaluatedProperties *OpenAPISchema           `yaml:"unevaluatedProperties,omitempty"`
	
	// Generic validation
	Enum     []any `yaml:"enum,omitempty"`
	Default  any   `yaml:"default,omitempty"`
	ReadOnly bool  `yaml:"readOnly,omitempty"`
	WriteOnly bool `yaml:"writeOnly,omitempty"`
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
	RequestBodies   map[string]*OpenAPIRequestBody    `yaml:"requestBodies"`
	SecuritySchemes map[string]*OpenAPISecurityScheme `yaml:"securitySchemes"`
	Schemas         map[string]*OpenAPISchema         `yaml:"schemas"`
}

// OpenAPIRequestBody represents a request body in an OpenAPI specification
type OpenAPIRequestBody struct {
	Ref         string          `yaml:"$ref,omitempty"`
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

// OpenAPIPathItem represents a path item object that can contain operations
type OpenAPIPathItem struct {
	Ref         string                   `yaml:"$ref,omitempty"`
	Summary     string                   `yaml:"summary,omitempty"`
	Description string                   `yaml:"description,omitempty"`
	Get         *OpenAPIPath             `yaml:"get,omitempty"`
	Put         *OpenAPIPath             `yaml:"put,omitempty"`
	Post        *OpenAPIPath             `yaml:"post,omitempty"`
	Delete      *OpenAPIPath             `yaml:"delete,omitempty"`
	Options     *OpenAPIPath             `yaml:"options,omitempty"`
	Head        *OpenAPIPath             `yaml:"head,omitempty"`
	Patch       *OpenAPIPath             `yaml:"patch,omitempty"`
	Trace       *OpenAPIPath             `yaml:"trace,omitempty"`
	Servers     []*OpenAPIServer         `yaml:"servers,omitempty"`
	Parameters  []*OpenAPIRef            `yaml:"parameters,omitempty"`
}

// OpenAPIServer represents a server object
type OpenAPIServer struct {
	URL         string                    `yaml:"url"`
	Description string                    `yaml:"description,omitempty"`
	Variables   map[string]*OpenAPIServerVariable `yaml:"variables,omitempty"`
}

// OpenAPIServerVariable represents a server variable
type OpenAPIServerVariable struct {
	Enum        []string `yaml:"enum,omitempty"`
	Default     string   `yaml:"default"`
	Description string   `yaml:"description,omitempty"`
}

// OpenAPISpec represents the entire OpenAPI specification, including metadata, components, paths, and tags
type OpenAPISpec struct {
	OpenAPI    string                             `yaml:"openapi"`
	Info       *OpenAPIInfo                       `yaml:"info"`
	Servers    []*OpenAPIServer                   `yaml:"servers,omitempty"`
	Components *OpenAPIComponents                 `yaml:"components"`
	Paths      map[string]map[string]*OpenAPIPath `yaml:"paths"`
	Webhooks   map[string]*OpenAPIPathItem        `yaml:"webhooks,omitempty"`
	Tags       []map[string]string                `yaml:"tags"`
}
