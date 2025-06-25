package docula

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	
	"zbz/zlog"
	"gopkg.in/yaml.v3"
)

// DoculaService defines the interface for documentation generation
type DoculaService interface {
	// GetSpec returns the complete OpenAPI specification
	GetSpec() *OpenAPISpec
	
	// RegisterModel adds a model to the OpenAPI schema
	RegisterModel(name string, model any) error
	
	// RegisterEndpoint adds an endpoint to the OpenAPI paths
	RegisterEndpoint(method, path, tag, summary, description string, params []*OpenAPIParameter, requestBody *OpenAPIRequestBody, responses map[string]*OpenAPIResponse) error
	
	// SetInfo updates the API info section
	SetInfo(info *OpenAPIInfo) error
	
	// AddServer adds a server to the servers list
	AddServer(server *OpenAPIServer) error
	
	// AddTag adds a tag to the tags list
	AddTag(tag *OpenAPITag) error
	
	// AddSecurityScheme adds a security scheme to components
	AddSecurityScheme(name string, scheme *OpenAPISecurityScheme) error
	
	// ToYAML exports the specification as YAML
	ToYAML() ([]byte, error)
	
	// ToJSON exports the specification as JSON (via YAML conversion)
	ToJSON() ([]byte, error)
}

// DoculaContract defines the configuration for a Docula instance
type DoculaContract struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Info        *OpenAPIInfo `yaml:"info,omitempty"`
	Servers     []*OpenAPIServer `yaml:"servers,omitempty"`
	Security    []map[string][]string `yaml:"security,omitempty"`
}

// doculaService implements DoculaService
type doculaService struct {
	mu   sync.RWMutex
	spec *OpenAPISpec
}

// Global singleton instance
var (
	Docula DoculaService
	once   sync.Once
)

// Initialize creates the global Docula instance
func Initialize(contract DoculaContract) {
	once.Do(func() {
		spec := &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: &OpenAPIInfo{
				Title:   "API Documentation",
				Version: "1.0.0",
			},
			Paths:      make(map[string]*OpenAPIPathItem),
			Components: &OpenAPIComponents{
				Schemas:         make(map[string]*OpenAPISchema),
				Responses:       make(map[string]*OpenAPIResponse),
				Parameters:      make(map[string]*OpenAPIParameter),
				SecuritySchemes: make(map[string]*OpenAPISecurityScheme),
				RequestBodies:   make(map[string]*OpenAPIRequestBody),
				Headers:         make(map[string]*OpenAPIHeader),
				Examples:        make(map[string]*OpenAPIExample),
				Links:           make(map[string]*OpenAPILink),
				Callbacks:       make(map[string]*OpenAPICallback),
			},
		}
		
		// Apply contract configuration
		if contract.Info != nil {
			spec.Info = contract.Info
		}
		if contract.Servers != nil {
			spec.Servers = contract.Servers
		}
		if contract.Security != nil {
			spec.Security = contract.Security
		}
		
		Docula = &doculaService{spec: spec}
		
		zlog.Info("Docula documentation service initialized", 
			zlog.String("name", contract.Name),
			zlog.String("description", contract.Description))
	})
}

// Contract returns a DoculaContract and initializes the service
func (c DoculaContract) Docula() DoculaService {
	Initialize(c)
	return Docula
}

// GetSpec returns the complete OpenAPI specification
func (d *doculaService) GetSpec() *OpenAPISpec {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.spec
}

// RegisterModel adds a model to the OpenAPI schema
func (d *doculaService) RegisterModel(name string, model any) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	schema, err := d.generateSchemaFromStruct(model)
	if err != nil {
		return fmt.Errorf("failed to generate schema for %s: %w", name, err)
	}
	
	d.spec.Components.Schemas[name] = schema
	
	zlog.Debug("Registered model schema", 
		zlog.String("name", name),
		zlog.String("type", reflect.TypeOf(model).String()))
	
	return nil
}

// RegisterEndpoint adds an endpoint to the OpenAPI paths
func (d *doculaService) RegisterEndpoint(method, path, tag, summary, description string, params []*OpenAPIParameter, requestBody *OpenAPIRequestBody, responses map[string]*OpenAPIResponse) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Ensure path exists
	if d.spec.Paths[path] == nil {
		d.spec.Paths[path] = &OpenAPIPathItem{}
	}
	
	operation := &OpenAPIOperation{
		Tags:        []string{tag},
		Summary:     summary,
		Description: description,
		Parameters:  params,
		RequestBody: requestBody,
		Responses:   responses,
	}
	
	// Set operation on the correct HTTP method
	switch strings.ToUpper(method) {
	case "GET":
		d.spec.Paths[path].Get = operation
	case "POST":
		d.spec.Paths[path].Post = operation
	case "PUT":
		d.spec.Paths[path].Put = operation
	case "DELETE":
		d.spec.Paths[path].Delete = operation
	case "PATCH":
		d.spec.Paths[path].Patch = operation
	case "HEAD":
		d.spec.Paths[path].Head = operation
	case "OPTIONS":
		d.spec.Paths[path].Options = operation
	case "TRACE":
		d.spec.Paths[path].Trace = operation
	default:
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}
	
	zlog.Debug("Registered endpoint", 
		zlog.String("method", method),
		zlog.String("path", path),
		zlog.String("tag", tag))
	
	return nil
}

// SetInfo updates the API info section
func (d *doculaService) SetInfo(info *OpenAPIInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.spec.Info = info
	
	zlog.Debug("Updated API info", 
		zlog.String("title", info.Title),
		zlog.String("version", info.Version))
	
	return nil
}

// AddServer adds a server to the servers list
func (d *doculaService) AddServer(server *OpenAPIServer) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.spec.Servers = append(d.spec.Servers, server)
	
	zlog.Debug("Added server", 
		zlog.String("url", server.URL),
		zlog.String("description", server.Description))
	
	return nil
}

// AddTag adds a tag to the tags list
func (d *doculaService) AddTag(tag *OpenAPITag) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.spec.Tags = append(d.spec.Tags, tag)
	
	zlog.Debug("Added tag", 
		zlog.String("name", tag.Name),
		zlog.String("description", tag.Description))
	
	return nil
}

// AddSecurityScheme adds a security scheme to components
func (d *doculaService) AddSecurityScheme(name string, scheme *OpenAPISecurityScheme) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.spec.Components.SecuritySchemes[name] = scheme
	
	zlog.Debug("Added security scheme", 
		zlog.String("name", name),
		zlog.String("type", scheme.Type))
	
	return nil
}

// ToYAML exports the specification as YAML
func (d *doculaService) ToYAML() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	return yaml.Marshal(d.spec)
}

// ToJSON exports the specification as JSON (via YAML conversion)
func (d *doculaService) ToJSON() ([]byte, error) {
	yamlData, err := d.ToYAML()
	if err != nil {
		return nil, err
	}
	
	// Convert YAML to JSON by unmarshaling and remarshaling
	var jsonObj any
	if err := yaml.Unmarshal(yamlData, &jsonObj); err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}
	
	// Use yaml package to marshal as JSON (it handles the conversion properly)
	return yaml.Marshal(jsonObj)
}

// generateSchemaFromStruct creates an OpenAPI schema from a Go struct
func (d *doculaService) generateSchemaFromStruct(model any) (*OpenAPISchema, error) {
	t := reflect.TypeOf(model)
	
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model must be a struct, got %s", t.Kind())
	}
	
	schema := &OpenAPISchema{
		Type:       "object",
		Properties: make(map[string]*OpenAPISchema),
		Required:   []string{},
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip private fields
		if !field.IsExported() {
			continue
		}
		
		// Get field name from json tag or use field name
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}
		
		// Generate schema for field type
		fieldSchema, err := d.generateSchemaFromType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for field %s: %w", field.Name, err)
		}
		
		// Add description from tag if available
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}
		
		schema.Properties[fieldName] = fieldSchema
		
		// Check if field is required (no omitempty tag)
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if !strings.Contains(jsonTag, "omitempty") {
				schema.Required = append(schema.Required, fieldName)
			}
		}
	}
	
	return schema, nil
}

// generateSchemaFromType creates an OpenAPI schema from a Go type
func (d *doculaService) generateSchemaFromType(t reflect.Type) (*OpenAPISchema, error) {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		return d.generateSchemaFromType(t.Elem())
	}
	
	switch t.Kind() {
	case reflect.String:
		return &OpenAPISchema{Type: "string"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &OpenAPISchema{Type: "integer"}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &OpenAPISchema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &OpenAPISchema{Type: "number"}, nil
	case reflect.Bool:
		return &OpenAPISchema{Type: "boolean"}, nil
	case reflect.Slice, reflect.Array:
		itemSchema, err := d.generateSchemaFromType(t.Elem())
		if err != nil {
			return nil, err
		}
		return &OpenAPISchema{
			Type:  "array",
			Items: itemSchema,
		}, nil
	case reflect.Map:
		valueSchema, err := d.generateSchemaFromType(t.Elem())
		if err != nil {
			return nil, err
		}
		return &OpenAPISchema{
			Type:                 "object",
			AdditionalProperties: valueSchema,
		}, nil
	case reflect.Struct:
		// For nested structs, we could either inline them or create a reference
		// For now, let's inline them
		return d.generateSchemaFromStruct(reflect.New(t).Elem().Interface())
	case reflect.Interface:
		// For interfaces, use a generic schema
		return &OpenAPISchema{Type: "object"}, nil
	default:
		return &OpenAPISchema{Type: "string"}, nil // Fallback
	}
}