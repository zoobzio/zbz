package zbz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// Docs is an interface for API documentation functionality
type Docs interface {
	AddTag(name, description string)
	AddPath(op *HTTPOperation)
	AddParameter(param *OpenAPIParameter)
	AddSchema(meta *Meta)

	SpecHandler(ctx *gin.Context)
	ScalarHandler(ctx *gin.Context)
}

// Docs represents the documentation structure for an API
type ZbzDocs struct {
	config Config
	log    Logger
	spec   *OpenAPISpec
}

// NewDocs creates a new Docs instance
func NewDocs(l Logger, c Config) Docs {
	return &ZbzDocs{
		config: c,
		log:    l,
		spec: &OpenAPISpec{
			OpenAPI: "3.0.0",
			Info: &OpenAPIInfo{
				Title:       c.Title(),
				Version:     c.Version(),
				Description: c.Description(),
			},
			Components: &OpenAPIComponents{
				Parameters:    make(map[string]*OpenAPIParameter),
				RequestBodies: make(map[string]*OpenAPISchema),
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
func (d *ZbzDocs) AddTag(name, description string) {
	d.spec.Tags = append(d.spec.Tags, map[string]string{
		"name":        name,
		"description": description,
	})
}

// AddParameter adds a new parameter to the OpenAPI specification
func (d *ZbzDocs) AddParameter(param *OpenAPIParameter) {
	if d.spec.Components.Parameters == nil {
		d.spec.Components.Parameters = make(map[string]*OpenAPIParameter)
	}
	d.spec.Components.Parameters[toPascalCase(param.Name)] = param
}

// AddBody adds a new request body to the OpenAPI specification
func (d *ZbzDocs) AddBody(body *OpenAPISchema) {
	if d.spec.Components.RequestBodies == nil {
		d.spec.Components.RequestBodies = make(map[string]*OpenAPISchema)
	}
	d.spec.Components.RequestBodies[toPascalCase(body.Description)] = body
}

// AddPath adds a new path to the OpenAPI specification
func (d *ZbzDocs) AddPath(op *HTTPOperation) {
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
						Ref: fmt.Sprintf("#/components/schemas/%s", toPascalCase(op.Response.Ref)),
					},
				},
			}
		}

		if op.Response.Errors != nil {
			for _, status := range op.Response.Errors {
				responses[status] = &OpenAPIResponse{
					Description: http.StatusText(status),
				}
			}
		}

		if op.Auth {
			responses[http.StatusUnauthorized] = &OpenAPIResponse{
				Description: http.StatusText(http.StatusUnauthorized),
			}
			responses[http.StatusForbidden] = &OpenAPIResponse{
				Description: http.StatusText(http.StatusForbidden),
			}
		}

		responses[http.StatusInternalServerError] = &OpenAPIResponse{
			Description: http.StatusText(http.StatusInternalServerError),
		}
	}

	path := &OpenAPIPath{
		Summary:     op.Name,
		Description: op.Description,
		OperationId: toPascalCase(op.Name),
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
				Ref: fmt.Sprintf("#/components/parameters/%s", toPascalCase(param)),
			})
		}
	}

	if op.RequestBody != "" {
		path.RequestBody = &OpenAPIRequestBody{
			Description: fmt.Sprintf("Request body for %s", op.Name),
			Required:    true,
			Content: &OpenAPIContent{
				ApplicationJSON: &OpenAPIApplicationJSON{
					Schema: &OpenAPISchema{
						Ref: fmt.Sprintf("#/components/requestBodies/%s", toPascalCase(op.RequestBody)),
					},
				},
			},
		}
	}

	d.spec.Paths[op.Path][toLowerCase(op.Method)] = path
}

// AddSchema adds a new schema to the OpenAPI specification
func (d *ZbzDocs) AddSchema(meta *Meta) {
	example, err := json.Marshal(meta.Example)
	if err != nil {
		d.log.Fatalf("Failed to marshal example for model %s: %v", meta.Name, err)
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

		schema.Properties[field.Name] = &OpenAPISchema{
			Type:        ft,
			Format:      ff,
			Description: field.Description,
			Example:     field.Example,
		}

		if field.Required {
			schema.Required = append(schema.Required, field.Name)
		}

		// TODO we can handle edit permissions here using the value of field.Edit
		if field.Edit != "" {
			payload := &OpenAPISchema{
				Type:        ft,
				Format:      ff,
				Description: field.Description,
				Example:     field.Example,
			}

			createPayload.Properties[field.Name] = payload
			updatePayload.Properties[field.Name] = payload

			if field.Required {
				createPayload.Required = append(createPayload.Required, field.Name)
			}
		}
	}

	d.spec.Components.Schemas[toPascalCase(meta.Name)] = schema
	d.spec.Components.RequestBodies[fmt.Sprintf("Create%sPayload", meta.Name)] = createPayload
	d.spec.Components.RequestBodies[fmt.Sprintf("Update%sPayload", meta.Name)] = updatePayload
}

// SpecHandler generates and returns the OpenAPI specification in YAML format
func (d *ZbzDocs) SpecHandler(ctx *gin.Context) {
	spec, err := yaml.Marshal(d.spec)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OpenAPI spec"})
	}
	ctx.Data(http.StatusOK, "text/yaml; charset=utf-8", spec)
}

// ScalarHandler renders a documentation site built using Scalar: https://scalar.com/
func (d *ZbzDocs) ScalarHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "scalar.tmpl", gin.H{
		"title":   d.config.Title(),
		"openapi": "/openapi",
	})
}
