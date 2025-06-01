package zbz

type OpenAPIInfo struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

type OpenAPIResponse struct {
	Description string `yaml:"description"`
}

type OpenAPISecurityScheme struct {
	Type   string `yaml:"type"`
	Scheme string `yaml:"scheme,omitempty"`
}

type OpenAPISecurity struct {
}

type OpenAPISchema struct {
	Type string `yaml:"type"`
}

type OpenAPIComponents struct {
	SecuritySchemes map[string]*OpenAPISecurityScheme `yaml:"securitySchemes"`
	Schemas         map[string]*OpenAPISchema         `yaml:"schemas"`
}

type OpenAPIPath struct {
	Summary     string                      `yaml:"summary,omitempty"`
	Description string                      `yaml:"description,omitempty"`
	OperationId string                      `yaml:"operationId,omitempty"`
	Parameters  []map[string]any            `yaml:"parameters,omitempty"`
	Tags        []string                    `yaml:"tags,omitempty"`
	Responses   map[string]*OpenAPIResponse `yaml:"responses,omitempty"`
	Security    []map[string][]string       `yaml:"security,omitempty"`
}

type OpenAPISpec struct {
	OpenAPI    string                             `yaml:"openapi"`
	Info       *OpenAPIInfo                       `yaml:"info"`
	Components *OpenAPIComponents                 `yaml:"components"`
	Paths      map[string]map[string]*OpenAPIPath `yaml:"paths"`
	Tags       []map[string]string                `yaml:"tags"`
}
