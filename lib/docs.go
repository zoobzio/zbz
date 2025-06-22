package zbz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Docs is an interface for API documentation functionality
type Docs interface {
	AddTag(name, description string)
	AddPath(op *Operation, errorManager ErrorManager)
	AddParameter(param *OpenAPIParameter)
	AddSchema(meta *Meta)

	SpecHandler(ctx *gin.Context)
	ScalarHandler(ctx *gin.Context)
}

// Docs represents the documentation structure for an API
type zDocs struct {
	spec *OpenAPISpec
	yaml []byte
}

// NewDocs creates a new Docs instance
func NewDocs() Docs {
	return &zDocs{
		spec: &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: &OpenAPIInfo{
				Title:       config.Title(),
				Version:     config.Version(),
				Description: config.Description(),
			},
			Components: &OpenAPIComponents{
				Parameters:    make(map[string]*OpenAPIParameter),
				RequestBodies: make(map[string]*OpenAPIRequestBody),
				SecuritySchemes: map[string]*OpenAPISecurityScheme{
					"Bearer": {
						Type:   "http",
						Scheme: "bearer",
					},
				},
				Schemas: make(map[string]*OpenAPISchema),
			},
			Paths: make(map[string]map[string]*OpenAPIPath),
			Tags:  []map[string]string{},
		},
	}
}

// AddTag adds a new tag to the OpenAPI specification
func (d *zDocs) AddTag(name, description string) {
	d.spec.Tags = append(d.spec.Tags, map[string]string{
		"name":        name,
		"description": description,
	})
}

// AddParameter adds a new parameter to the OpenAPI specification
func (d *zDocs) AddParameter(param *OpenAPIParameter) {
	if d.spec.Components.Parameters == nil {
		d.spec.Components.Parameters = make(map[string]*OpenAPIParameter)
	}
	d.spec.Components.Parameters[param.Name] = param
}

// AddBody adds a new request body to the OpenAPI specification
func (d *zDocs) AddBody(name string, body *OpenAPIRequestBody) {
	if d.spec.Components.RequestBodies == nil {
		d.spec.Components.RequestBodies = make(map[string]*OpenAPIRequestBody)
	}
	d.spec.Components.RequestBodies[name] = body
}

// AddPath adds a new path to the OpenAPI specification
func (d *zDocs) AddPath(op *Operation, errorManager ErrorManager) {
	if d.spec.Paths[op.Path] == nil {
		d.spec.Paths[op.Path] = make(map[string]*OpenAPIPath)
	}

	responses := make(map[int]*OpenAPIResponse)
	if op.Response != nil {
		responses[op.Response.Status] = &OpenAPIResponse{
			Description: http.StatusText(op.Response.Status),
		}

		if op.Response.Ref != "" {
			responses[op.Response.Status].Content = &OpenAPIContent{
				ApplicationJSON: &OpenAPIApplicationJSON{
					Schema: &OpenAPISchema{
						Ref: fmt.Sprintf("#/components/schemas/%s", op.Response.Ref),
					},
				},
			}
		}

		if op.Response.Errors != nil {
			for _, status := range op.Response.Errors {
				errorSchema := GetErrorSchema(errorManager, status)
				if errorSchema != nil {
					responses[status] = &OpenAPIResponse{
						Description: http.StatusText(status),
						Content: &OpenAPIContent{
							ApplicationJSON: &OpenAPIApplicationJSON{
								Schema: errorSchema,
							},
						},
					}
				} else {
					responses[status] = &OpenAPIResponse{
						Description: http.StatusText(status),
					}
				}
			}
		}

		if op.Auth {
			unauthorizedSchema := GetErrorSchema(errorManager, http.StatusUnauthorized)
			if unauthorizedSchema != nil {
				responses[http.StatusUnauthorized] = &OpenAPIResponse{
					Description: http.StatusText(http.StatusUnauthorized),
					Content: &OpenAPIContent{
						ApplicationJSON: &OpenAPIApplicationJSON{
							Schema: unauthorizedSchema,
						},
					},
				}
			}
			
			forbiddenSchema := GetErrorSchema(errorManager, http.StatusForbidden)
			if forbiddenSchema != nil {
				responses[http.StatusForbidden] = &OpenAPIResponse{
					Description: http.StatusText(http.StatusForbidden),
					Content: &OpenAPIContent{
						ApplicationJSON: &OpenAPIApplicationJSON{
							Schema: forbiddenSchema,
						},
					},
				}
			}
		}

		internalErrorSchema := GetErrorSchema(errorManager, http.StatusInternalServerError)
		if internalErrorSchema != nil {
			responses[http.StatusInternalServerError] = &OpenAPIResponse{
				Description: http.StatusText(http.StatusInternalServerError),
				Content: &OpenAPIContent{
					ApplicationJSON: &OpenAPIApplicationJSON{
						Schema: internalErrorSchema,
					},
				},
			}
		}
	}

	path := &OpenAPIPath{
		Summary:     op.Name,
		Description: op.Description,
		OperationId: op.Name,
		Tags:        []string{op.Tag},
		Parameters:  []*OpenAPIRef{},
		Responses:   responses,
	}

	if op.Auth {
		path.Security = []map[string][]string{
			{"Bearer": {}},
		}
	}

	if op.Parameters != nil {
		for _, param := range op.Parameters {
			path.Parameters = append(path.Parameters, &OpenAPIRef{
				Ref: fmt.Sprintf("#/components/parameters/%s", param),
			})
		}
	}

	if op.RequestBody != "" {
		path.RequestBody = &OpenAPIRequestBody{
			Ref: fmt.Sprintf("#/components/requestBodies/%s", op.RequestBody),
		}
	}

	d.spec.Paths[op.Path][strings.ToLower(op.Method)] = path
}

// AddSchema adds a new schema to the OpenAPI specification
func (d *zDocs) AddSchema(meta *Meta) {
	example, err := json.Marshal(meta.Example)
	if err != nil {
		Log.Fatal("Failed to marshal example for model", zap.Any("meta", meta), zap.Error(err))
	}

	schema := &OpenAPISchema{
		Type:        "object",
		Description: meta.Description,
		Example:     string(example),
		Properties:  map[string]*OpenAPISchema{},
		Required:    []string{},
	}

	createPayload := &OpenAPISchema{
		Type:        "object",
		Description: fmt.Sprintf("`Create` payload for %s", meta.Name),
		Properties:  map[string]*OpenAPISchema{},
		Required:    []string{},
	}

	updatePayload := &OpenAPISchema{
		Type:        "object",
		Description: fmt.Sprintf("`Update` payload for %s", meta.Name),
		Properties:  map[string]*OpenAPISchema{},
	}

	for _, field := range meta.Fields {
		var ft string
		var ff string
		switch field.Type {
		case "int", "int32":
			ft = "integer"
			ff = "int32"
		case "int64":
			ft = "integer"
			ff = "int64"
		case "float32":
			ft = "number"
			ff = "float"
		case "float64":
			ft = "number"
			ff = "double"
		case "string":
			ft = "string"
			if strings.Contains(field.Validate, "email") {
				ff = "email"
			} else if strings.Contains(field.Validate, "uuid") {
				ff = "uuid"
			} else if strings.Contains(field.Validate, "url") {
				ff = "uri"
			}
		case "bool":
			ft = "boolean"
		case "time.Time":
			ft = "string"
			ff = "date-time"
		case "[]byte":
			ft = "string"
			ff = "byte"
		}

		fieldSchema := &OpenAPISchema{
			Type:        ft,
			Format:      ff,
			Description: field.Description,
			Example:     field.Example,
		}
		
		// Apply validation constraints from struct tags using our validation system
		rules := validate.ParseValidationRules(field.Validate)
		constraints := validate.GetOpenAPIConstraints(rules, ft)
		for key, value := range constraints {
			switch key {
			case "minLength":
				if v, ok := value.(int); ok {
					fieldSchema.MinLength = &v
				}
			case "maxLength":
				if v, ok := value.(int); ok {
					fieldSchema.MaxLength = &v
				}
			case "minimum":
				if v, ok := value.(float64); ok {
					fieldSchema.Minimum = &v
				}
			case "maximum":
				if v, ok := value.(float64); ok {
					fieldSchema.Maximum = &v
				}
			case "exclusiveMinimum":
				if v, ok := value.(float64); ok {
					fieldSchema.ExclusiveMinimum = &v
				}
			case "exclusiveMaximum":
				if v, ok := value.(float64); ok {
					fieldSchema.ExclusiveMaximum = &v
				}
			case "pattern":
				if v, ok := value.(string); ok {
					fieldSchema.Pattern = v
				}
			case "format":
				if v, ok := value.(string); ok {
					fieldSchema.Format = v
				}
			case "enum":
				if v, ok := value.([]any); ok {
					fieldSchema.Enum = v
				}
			}
		}
		
		schema.Properties[field.DstName] = fieldSchema

		if field.Required {
			schema.Required = append(schema.Required, field.DstName)
		}

		// TODO we can handle edit permissions here using the value of field.Edit
		if field.Edit != "" {
			payload := &OpenAPISchema{
				Type:        ft,
				Format:      ff,
				Description: field.Description,
				Example:     field.Example,
			}
			
			// Apply validation constraints to payload schemas too using our validation system
			payloadConstraints := validate.GetOpenAPIConstraints(rules, ft)
			for key, value := range payloadConstraints {
				switch key {
				case "minLength":
					if v, ok := value.(int); ok {
						payload.MinLength = &v
					}
				case "maxLength":
					if v, ok := value.(int); ok {
						payload.MaxLength = &v
					}
				case "minimum":
					if v, ok := value.(float64); ok {
						payload.Minimum = &v
					}
				case "maximum":
					if v, ok := value.(float64); ok {
						payload.Maximum = &v
					}
				case "exclusiveMinimum":
					if v, ok := value.(float64); ok {
						payload.ExclusiveMinimum = &v
					}
				case "exclusiveMaximum":
					if v, ok := value.(float64); ok {
						payload.ExclusiveMaximum = &v
					}
				case "pattern":
					if v, ok := value.(string); ok {
						payload.Pattern = v
					}
				case "format":
					if v, ok := value.(string); ok {
						payload.Format = v
					}
				case "enum":
					if v, ok := value.([]any); ok {
						payload.Enum = v
					}
				}
			}

			createPayload.Properties[field.DstName] = payload
			updatePayload.Properties[field.DstName] = payload

			if field.Required {
				createPayload.Required = append(createPayload.Required, field.DstName)
			}
		}
	}

	d.spec.Components.Schemas[meta.Name] = schema
	
	// Store request bodies in the proper section
	d.spec.Components.RequestBodies[fmt.Sprintf("Create%sPayload", meta.Name)] = &OpenAPIRequestBody{
		Description: fmt.Sprintf("Request body for creating a %s", meta.Name),
		Required:    true,
		Content: &OpenAPIContent{
			ApplicationJSON: &OpenAPIApplicationJSON{
				Schema: createPayload,
			},
		},
	}
	
	d.spec.Components.RequestBodies[fmt.Sprintf("Update%sPayload", meta.Name)] = &OpenAPIRequestBody{
		Description: fmt.Sprintf("Request body for updating a %s", meta.Name),
		Required:    true,
		Content: &OpenAPIContent{
			ApplicationJSON: &OpenAPIApplicationJSON{
				Schema: updatePayload,
			},
		},
	}
}

// GenerateYAML generates the OpenAPI specification in YAML format
func (d *zDocs) GenerateYAML(spec *OpenAPISpec) {
	data, err := yaml.Marshal(spec)
	if err != nil {
		Log.Fatal("failed to marshal OpenAPI spec to YAML", zap.Error(err))
	}
	d.yaml = data
}

// SpecHandler generates and returns the OpenAPI specification in YAML format
func (d *zDocs) SpecHandler(ctx *gin.Context) {
	if d.yaml == nil {
		d.GenerateYAML(d.spec)
	}
	ctx.Data(http.StatusOK, "text/yaml; charset=utf-8", d.yaml)
}

// ScalarHandler renders a documentation site built using Scalar: https://scalar.com/
func (d *zDocs) ScalarHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "scalar.tmpl", gin.H{
		"title":   config.Title(),
		"openapi": "/openapi",
	})
}

