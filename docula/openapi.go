package docula

import (
	"zbz/cereal"
)

// OpenAPISpec represents a complete OpenAPI 3.1.0 specification
type OpenAPISpec struct {
	OpenAPI string                       `json:"openapi" yaml:"openapi"`
	Info    OpenAPIInfo                  `json:"info" yaml:"info"`
	Paths   map[string]OpenAPIPath       `json:"paths,omitempty" yaml:"paths,omitempty"`
	Components *OpenAPIComponents       `json:"components,omitempty" yaml:"components,omitempty"`
}

// OpenAPIInfo contains API metadata
type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

// OpenAPIPath represents an API path with operations
type OpenAPIPath struct {
	Get    *OpenAPIOperation `json:"get,omitempty" yaml:"get,omitempty"`
	Post   *OpenAPIOperation `json:"post,omitempty" yaml:"post,omitempty"`
	Put    *OpenAPIOperation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete *OpenAPIOperation `json:"delete,omitempty" yaml:"delete,omitempty"`
}

// OpenAPIOperation represents an API operation
type OpenAPIOperation struct {
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Summary     string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// OpenAPIComponents contains reusable components
type OpenAPIComponents struct {
	Schemas map[string]OpenAPISchema `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

// OpenAPISchema represents a data schema
type OpenAPISchema struct {
	Type        string                       `json:"type,omitempty" yaml:"type,omitempty"`
	Description string                       `json:"description,omitempty" yaml:"description,omitempty"`
	Properties  map[string]OpenAPISchema     `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required    []string                     `json:"required,omitempty" yaml:"required,omitempty"`
}

// SpecGenerator manages OpenAPI spec generation and caching
type SpecGenerator struct {
	spec        *OpenAPISpec
	processor   *MarkdownProcessor
	
	// Simple cache
	cachedYAML  []byte
	cachedJSON  []byte
}

// NewSpecGenerator creates a new OpenAPI spec generator
func NewSpecGenerator() *SpecGenerator {
	return &SpecGenerator{
		spec: &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: OpenAPIInfo{
				Title:   "API Documentation",
				Version: "1.0.0",
			},
			Paths:      make(map[string]OpenAPIPath),
			Components: &OpenAPIComponents{
				Schemas: make(map[string]OpenAPISchema),
			},
		},
		processor: NewMarkdownProcessor(),
	}
}

// RegisterEndpoint adds an endpoint to the OpenAPI spec
func (sg *SpecGenerator) RegisterEndpoint(method, path, operationID, summary, description string) {
	if sg.spec.Paths == nil {
		sg.spec.Paths = make(map[string]OpenAPIPath)
	}
	
	operation := &OpenAPIOperation{
		OperationID: operationID,
		Summary:     summary,
		Description: description,
		Tags:        []string{"api"},
	}
	
	pathItem := sg.spec.Paths[path]
	switch method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	}
	
	sg.spec.Paths[path] = pathItem
	sg.clearCache()
}

// RegisterModel adds a model schema to the OpenAPI spec
func (sg *SpecGenerator) RegisterModel(name, description string) {
	if sg.spec.Components == nil {
		sg.spec.Components = &OpenAPIComponents{
			Schemas: make(map[string]OpenAPISchema),
		}
	}
	
	sg.spec.Components.Schemas[name] = OpenAPISchema{
		Type:        "object",
		Description: description,
		Properties: map[string]OpenAPISchema{
			"id": {
				Type:        "string",
				Description: "Unique identifier",
			},
			"created_at": {
				Type:        "string",
				Description: "Creation timestamp",
			},
		},
		Required: []string{"id"},
	}
	
	sg.clearCache()
}

// EnhanceWithMarkdown updates the spec with markdown content
func (sg *SpecGenerator) EnhanceWithMarkdown(pages map[string]*DocPage) {
	// Update API description from api.md
	if apiPage, exists := pages["api.md"]; exists {
		sg.spec.Info.Description = sg.processor.ProcessForAPI([]byte(apiPage.Content))
	}
	
	// Enhance operations with {operationId}.md files
	for path, pathItem := range sg.spec.Paths {
		if pathItem.Get != nil {
			sg.enhanceOperation(pathItem.Get, pages)
		}
		if pathItem.Post != nil {
			sg.enhanceOperation(pathItem.Post, pages)
		}
		if pathItem.Put != nil {
			sg.enhanceOperation(pathItem.Put, pages)
		}
		if pathItem.Delete != nil {
			sg.enhanceOperation(pathItem.Delete, pages)
		}
		sg.spec.Paths[path] = pathItem
	}
	
	// Enhance schemas with {model}.md files
	for modelName, schema := range sg.spec.Components.Schemas {
		if modelPage, exists := pages[modelName+".md"]; exists {
			schema.Description = sg.processor.ProcessForAPI([]byte(modelPage.Content))
			sg.spec.Components.Schemas[modelName] = schema
		}
	}
	
	sg.clearCache()
}

// enhanceOperation updates an operation with markdown content
func (sg *SpecGenerator) enhanceOperation(operation *OpenAPIOperation, pages map[string]*DocPage) {
	if operation.OperationID == "" {
		return
	}
	
	// Look for {operationId}.md file
	if opPage, exists := pages[operation.OperationID+".md"]; exists {
		operation.Description = sg.processor.ProcessForAPI([]byte(opPage.Content))
		if opPage.Title != "" {
			operation.Summary = opPage.Title
		}
	}
}

// GetYAML returns the cached YAML representation
func (sg *SpecGenerator) GetYAML() ([]byte, error) {
	if sg.cachedYAML != nil {
		return sg.cachedYAML, nil
	}
	
	yamlData, err := cereal.YAML.Marshal(sg.spec)
	if err != nil {
		return nil, err
	}
	
	sg.cachedYAML = yamlData
	return yamlData, nil
}

// GetJSON returns the cached JSON representation
func (sg *SpecGenerator) GetJSON() ([]byte, error) {
	if sg.cachedJSON != nil {
		return sg.cachedJSON, nil
	}
	
	jsonData, err := cereal.JSON.Marshal(sg.spec)
	if err != nil {
		return nil, err
	}
	
	sg.cachedJSON = jsonData
	return jsonData, nil
}

// clearCache invalidates the cached representations
func (sg *SpecGenerator) clearCache() {
	sg.cachedYAML = nil
	sg.cachedJSON = nil
}

// SetInfo updates the API info
func (sg *SpecGenerator) SetInfo(title, description, version string) {
	sg.spec.Info.Title = title
	sg.spec.Info.Description = description
	sg.spec.Info.Version = version
	sg.clearCache()
}