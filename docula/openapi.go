package docula

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
	Name       string `yaml:"name,omitempty"`
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
	MinLength        *int     `yaml:"minLength,omitempty"`
	MaxLength        *int     `yaml:"maxLength,omitempty"`
	Pattern          string   `yaml:"pattern,omitempty"`
	MinItems         *int     `yaml:"minItems,omitempty"`
	MaxItems         *int     `yaml:"maxItems,omitempty"`
	UniqueItems      bool     `yaml:"uniqueItems,omitempty"`
	MinProperties    *int     `yaml:"minProperties,omitempty"`
	MaxProperties    *int     `yaml:"maxProperties,omitempty"`
	
	// Array/Object types
	Items    *OpenAPISchema `yaml:"items,omitempty"`
	Contains *OpenAPISchema `yaml:"contains,omitempty"`
	
	// String enums
	Enum []any `yaml:"enum,omitempty"`
	
	// Deprecated flag
	Deprecated bool `yaml:"deprecated,omitempty"`
}

// OpenAPIParameter represents a parameter in an OpenAPI specification
type OpenAPIParameter struct {
	Name            string         `yaml:"name"`
	In              string         `yaml:"in"`
	Description     string         `yaml:"description,omitempty"`
	Required        bool           `yaml:"required,omitempty"`
	Deprecated      bool           `yaml:"deprecated,omitempty"`
	AllowEmptyValue bool           `yaml:"allowEmptyValue,omitempty"`
	Schema          *OpenAPISchema `yaml:"schema,omitempty"`
	Example         any            `yaml:"example,omitempty"`
}

// OpenAPIRequestBody represents a request body in an OpenAPI specification
type OpenAPIRequestBody struct {
	Description string                       `yaml:"description,omitempty"`
	Content     map[string]*OpenAPIMediaType `yaml:"content"`
	Required    bool                         `yaml:"required,omitempty"`
}

// OpenAPIMediaType represents a media type in an OpenAPI specification
type OpenAPIMediaType struct {
	Schema   *OpenAPISchema `yaml:"schema,omitempty"`
	Example  any            `yaml:"example,omitempty"`
	Examples map[string]any `yaml:"examples,omitempty"`
}

// OpenAPIResponse represents a response in an OpenAPI specification
type OpenAPIResponse struct {
	Description string                       `yaml:"description"`
	Headers     map[string]*OpenAPIHeader    `yaml:"headers,omitempty"`
	Content     map[string]*OpenAPIMediaType `yaml:"content,omitempty"`
	Links       map[string]*OpenAPILink      `yaml:"links,omitempty"`
}

// OpenAPIHeader represents a header in an OpenAPI specification
type OpenAPIHeader struct {
	Description     string         `yaml:"description,omitempty"`
	Required        bool           `yaml:"required,omitempty"`
	Deprecated      bool           `yaml:"deprecated,omitempty"`
	AllowEmptyValue bool           `yaml:"allowEmptyValue,omitempty"`
	Schema          *OpenAPISchema `yaml:"schema,omitempty"`
	Example         any            `yaml:"example,omitempty"`
}

// OpenAPILink represents a link in an OpenAPI specification
type OpenAPILink struct {
	OperationRef string            `yaml:"operationRef,omitempty"`
	OperationId  string            `yaml:"operationId,omitempty"`
	Parameters   map[string]any    `yaml:"parameters,omitempty"`
	RequestBody  any               `yaml:"requestBody,omitempty"`
	Description  string            `yaml:"description,omitempty"`
	Server       *OpenAPIServer    `yaml:"server,omitempty"`
}

// OpenAPITag represents a tag in an OpenAPI specification
type OpenAPITag struct {
	Name         string                `yaml:"name"`
	Description  string                `yaml:"description,omitempty"`
	ExternalDocs *OpenAPIExternalDocs  `yaml:"externalDocs,omitempty"`
}

// OpenAPIExternalDocs represents external documentation
type OpenAPIExternalDocs struct {
	Description string `yaml:"description,omitempty"`
	URL         string `yaml:"url"`
}

// OpenAPISecurityScheme represents a security scheme
type OpenAPISecurityScheme struct {
	Type             string            `yaml:"type"`
	Description      string            `yaml:"description,omitempty"`
	Name             string            `yaml:"name,omitempty"`
	In               string            `yaml:"in,omitempty"`
	Scheme           string            `yaml:"scheme,omitempty"`
	BearerFormat     string            `yaml:"bearerFormat,omitempty"`
	Flows            *OpenAPIOAuthFlows `yaml:"flows,omitempty"`
	OpenIdConnectUrl string            `yaml:"openIdConnectUrl,omitempty"`
}

// OpenAPIOAuthFlows represents OAuth flows
type OpenAPIOAuthFlows struct {
	Implicit          *OpenAPIOAuthFlow `yaml:"implicit,omitempty"`
	Password          *OpenAPIOAuthFlow `yaml:"password,omitempty"`
	ClientCredentials *OpenAPIOAuthFlow `yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OpenAPIOAuthFlow `yaml:"authorizationCode,omitempty"`
}

// OpenAPIOAuthFlow represents a single OAuth flow
type OpenAPIOAuthFlow struct {
	AuthorizationUrl string            `yaml:"authorizationUrl,omitempty"`
	TokenUrl         string            `yaml:"tokenUrl,omitempty"`
	RefreshUrl       string            `yaml:"refreshUrl,omitempty"`
	Scopes          map[string]string `yaml:"scopes"`
}

// OpenAPIComponents represents the components section
type OpenAPIComponents struct {
	Schemas         map[string]*OpenAPISchema         `yaml:"schemas,omitempty"`
	Responses       map[string]*OpenAPIResponse       `yaml:"responses,omitempty"`
	Parameters      map[string]*OpenAPIParameter      `yaml:"parameters,omitempty"`
	Examples        map[string]*OpenAPIExample        `yaml:"examples,omitempty"`
	RequestBodies   map[string]*OpenAPIRequestBody    `yaml:"requestBodies,omitempty"`
	Headers         map[string]*OpenAPIHeader         `yaml:"headers,omitempty"`
	SecuritySchemes map[string]*OpenAPISecurityScheme `yaml:"securitySchemes,omitempty"`
	Links           map[string]*OpenAPILink           `yaml:"links,omitempty"`
	Callbacks       map[string]*OpenAPICallback       `yaml:"callbacks,omitempty"`
}

// OpenAPIExample represents an example
type OpenAPIExample struct {
	Summary       string `yaml:"summary,omitempty"`
	Description   string `yaml:"description,omitempty"`
	Value         any    `yaml:"value,omitempty"`
	ExternalValue string `yaml:"externalValue,omitempty"`
}

// OpenAPICallback represents a callback
type OpenAPICallback map[string]*OpenAPIPathItem

// OpenAPIPathItem represents a path item
type OpenAPIPathItem struct {
	Ref         string              `yaml:"$ref,omitempty"`
	Summary     string              `yaml:"summary,omitempty"`
	Description string              `yaml:"description,omitempty"`
	Get         *OpenAPIOperation   `yaml:"get,omitempty"`
	Put         *OpenAPIOperation   `yaml:"put,omitempty"`
	Post        *OpenAPIOperation   `yaml:"post,omitempty"`
	Delete      *OpenAPIOperation   `yaml:"delete,omitempty"`
	Options     *OpenAPIOperation   `yaml:"options,omitempty"`
	Head        *OpenAPIOperation   `yaml:"head,omitempty"`
	Patch       *OpenAPIOperation   `yaml:"patch,omitempty"`
	Trace       *OpenAPIOperation   `yaml:"trace,omitempty"`
	Servers     []*OpenAPIServer    `yaml:"servers,omitempty"`
	Parameters  []*OpenAPIParameter `yaml:"parameters,omitempty"`
}

// OpenAPIOperation represents an operation
type OpenAPIOperation struct {
	Tags         []string                         `yaml:"tags,omitempty"`
	Summary      string                           `yaml:"summary,omitempty"`
	Description  string                           `yaml:"description,omitempty"`
	ExternalDocs *OpenAPIExternalDocs             `yaml:"externalDocs,omitempty"`
	OperationId  string                           `yaml:"operationId,omitempty"`
	Parameters   []*OpenAPIParameter              `yaml:"parameters,omitempty"`
	RequestBody  *OpenAPIRequestBody              `yaml:"requestBody,omitempty"`
	Responses    map[string]*OpenAPIResponse      `yaml:"responses"`
	Callbacks    map[string]*OpenAPICallback      `yaml:"callbacks,omitempty"`
	Deprecated   bool                             `yaml:"deprecated,omitempty"`
	Security     []map[string][]string            `yaml:"security,omitempty"`
	Servers      []*OpenAPIServer                 `yaml:"servers,omitempty"`
}

// OpenAPIServer represents a server
type OpenAPIServer struct {
	URL         string                      `yaml:"url"`
	Description string                      `yaml:"description,omitempty"`
	Variables   map[string]*OpenAPIVariable `yaml:"variables,omitempty"`
}

// OpenAPIVariable represents a server variable
type OpenAPIVariable struct {
	Enum        []string `yaml:"enum,omitempty"`
	Default     string   `yaml:"default"`
	Description string   `yaml:"description,omitempty"`
}

// OpenAPISpec represents the root OpenAPI specification
type OpenAPISpec struct {
	OpenAPI      string                            `yaml:"openapi"`
	Info         *OpenAPIInfo                      `yaml:"info"`
	Servers      []*OpenAPIServer                  `yaml:"servers,omitempty"`
	Paths        map[string]*OpenAPIPathItem       `yaml:"paths,omitempty"`
	Components   *OpenAPIComponents                `yaml:"components,omitempty"`
	Security     []map[string][]string             `yaml:"security,omitempty"`
	Tags         []*OpenAPITag                     `yaml:"tags,omitempty"`
	ExternalDocs *OpenAPIExternalDocs              `yaml:"externalDocs,omitempty"`
}