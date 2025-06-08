package zbz

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// Docs is an interface for API documentation functionality
type Docs interface {
	AddTag(name, description string)
	AddPath(op *HTTPOperation)
	AddSchema(meta *CoreMeta)
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
				SecuritySchemes: map[string]*OpenAPISecurityScheme{
					"BearerAuth": {
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

// AddPath adds a new path to the OpenAPI specification
func (d *ZbzDocs) AddPath(op *HTTPOperation) {
	if d.spec.Paths[op.Path] == nil {
		d.spec.Paths[op.Path] = make(map[string]*OpenAPIPath)
	}

	responses := make(map[int]*OpenAPIResponse)
	if op.Response != nil {
		responses[op.Response.Status] = &OpenAPIResponse{
			Description: http.StatusText(op.Response.Status),
			Content: &OpenAPIResponseContent{
				ApplicationJSON: &OpenAPIResponseApplicationJSON{
					Schema: &OpenAPISchema{
						Ref: fmt.Sprintf("#/components/schemas/%s", toPascalCase(op.Response.Ref)),
					},
				},
			},
		}

		if op.Response.Errors != nil {
			for _, status := range op.Response.Errors {
				responses[status] = &OpenAPIResponse{
					Description: http.StatusText(status),
				}
			}
		}
	}

	path := &OpenAPIPath{
		Summary:     op.Name,
		Description: op.Description,
		OperationId: toPascalCase(op.Name),
		Tags:        []string{op.Tag},
		Responses:   responses,
	}

	if op.Auth {
		path.Security = []map[string][]string{
			{"BearerAuth": {}},
		}
	}

	d.spec.Paths[op.Path][toLowerCase(op.Method)] = path
}

// AddSchema adds a new schema to the OpenAPI specification
func (d *ZbzDocs) AddSchema(meta *CoreMeta) {
	example, err := json.Marshal(meta.Example)
	if err != nil {
		d.log.Fatalf("Failed to marshal example for model %s: %v", meta.Name, err)
	}

	d.spec.Components.Schemas[toPascalCase(meta.Name)] = &OpenAPISchema{
		Type:        "object",
		Description: meta.Description,
		Example:     string(example),
		Properties: map[string]*OpenAPISchema{
			"id": {
				Type:        "string",
				Description: "Unique identifier for the resource",
				Example:     "123e4567-e89b-12d3-a456-426614174000",
			},
		},
		Required: []string{"id", "createdAt", "updatedAt"},
	}
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
