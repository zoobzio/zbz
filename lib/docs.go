package zbz

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// Docs is an interface for API documentation functionality
type Docs interface {
	AddTag(name, description string)
	AddPath(op *HTTPOperation)
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

// toPascalCase converts a string to PascalCase
func (d *ZbzDocs) toPascalCase(s string) string {
	words := strings.Fields(s) // splits by whitespace
	for i, word := range words {
		if len(word) > 0 {
			// capitalize first rune, add rest as-is
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, "")
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

	path := &OpenAPIPath{
		Summary:     op.Name,
		Description: op.Description,
		OperationId: d.toPascalCase(op.Name),
		Tags:        []string{op.Tag},
		Responses: map[string]*OpenAPIResponse{
			"200": {
				Description: "Successful operation",
			},
		},
	}

	if op.Auth {
		path.Security = []map[string][]string{
			{"BearerAuth": {}},
		}
	}

	d.spec.Paths[op.Path][strings.ToLower(op.Method)] = path
}

// AddSchema adds a new schema to the OpenAPI specification
func (d *ZbzDocs) AddSchema(name string, schema *OpenAPISchema) {
	d.spec.Components.Schemas[name] = schema
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
