package zbz

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Docs interface {
	RegisterPath(uri string, dp *DocsPath)
	RegisterModel(ref string, dm *DocsModel)
	GenerateSpec() map[string]any
	SpecHandler(ctx *gin.Context)
	ScalarHandler(ctx *gin.Context)
}

// DocsModel represents a model in the API documentation
type DocsModel struct {
	Ref     string
	Summary string
	Example any
}

// DocsPath represents a path in the API documentation
type DocsPath struct {
	Uri     string
	Summary string
}

// Docs represents the documentation structure for an API
type ZbzDocs struct {
	config Config
	log    Logger
	models map[string]*DocsModel
	paths  map[string]*DocsPath
}

// NewDocs creates a new Docs instance
func NewDocs(l Logger, c Config) Docs {
	return &ZbzDocs{
		config: c,
		log:    l,
		models: make(map[string]*DocsModel),
		paths:  make(map[string]*DocsPath),
	}
}

// RegisterPath adds a new path to the Docs instance
func (d *ZbzDocs) RegisterPath(uri string, dp *DocsPath) {
	d.paths[uri] = dp
}

// RegisterModel adds a new model to the Docs instance
func (d *ZbzDocs) RegisterModel(ref string, dm *DocsModel) {
	d.models[ref] = dm
}

// GenerateSpec generates the OpenAPI specification for the API documentation
func (d *ZbzDocs) GenerateSpec() map[string]any {
	spec := make(map[string]any)
	spec["openapi"] = "3.0.0"
	spec["info"] = map[string]string{
		"title":       d.config.Title(),
		"version":     d.config.Version(),
		"description": d.config.Description(),
	}
	spec["paths"] = make(map[string]any)

	for uri, path := range d.paths {
		pathSpec := make(map[string]any)
		pathSpec["summary"] = path.Summary
		spec["paths"].(map[string]any)[uri] = pathSpec
	}

	return spec
}

// spec returns the Docs specification for the API
func (d *ZbzDocs) SpecHandler(ctx *gin.Context) {
	s := d.GenerateSpec()
	ctx.JSON(200, s)
}

// docs renders the documentation page
func (d *ZbzDocs) ScalarHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "scalar.tmpl", gin.H{
		"title":   d.config.Title,
		"openapi": "/openapi",
	})
}
