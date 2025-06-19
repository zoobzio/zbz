package zbz

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// Core is an interface that defines the basic CRUD operations for a resource.
type Core interface {
	Description() string
	Meta() *Meta

	Table() *MacroContract
	Contracts() []*MacroContract
	Operations() []*Operation

	CreateHandler(ctx *gin.Context)
	ReadHandler(ctx *gin.Context)
	UpdateHandler(ctx *gin.Context)
	DeleteHandler(ctx *gin.Context)
}

// zCore is a generic implementation for core CRUD operations.
type zCore[T BaseModel] struct {
	description string
}

// NewCore creates a new instance of zCore with the provided logger, config, and database.
func NewCore[T BaseModel](desc string) Core {
	return &zCore[T]{
		description: desc,
	}
}

// Description returns the description of the core resource.
func (c *zCore[T]) Description() string {
	return c.description
}

// Meta extracts the metadata for the core resource, which includes its name and other properties.
func (c *zCore[T]) Meta() *Meta {
	return extractMeta[T]()
}

// Table returns the SQL statement to create the table for the core resource.
func (c *zCore[T]) Table() *MacroContract {
	meta := c.Meta()
	defs := []string{}
	for _, field := range meta.Fields {
		if field.Type == "zbz.Model" || field.SourceName == "-" {
			continue
		}
		def := fmt.Sprintf("%s %s", field.SourceName, field.SourceType)
		if field.SourceName == "id" {
			def += " PRIMARY KEY"
		}
		if field.Required {
			def += " NOT NULL"
		}
		defs = append(defs, def)
	}
	return &MacroContract{
		Name:  fmt.Sprintf("Create%sTable", meta.Name),
		Macro: "create_table",
		Embed: map[string]string{
			"table":   strings.ToLower(meta.Name),
			"columns": strings.Join(defs, ", "),
		},
	}
}

// MacroContracts returns the SQL statements associated with the core resource.
func (c *zCore[T]) Contracts() []*MacroContract {
	meta := c.Meta()

	cols := meta.Columns
	vals := []string{}
	for _, col := range meta.Fields {
		if slices.Contains(cols, col.SourceName) {
			vals = append(vals, fmt.Sprintf(":%s", col.Name))
		}
	}

	return []*MacroContract{
		{
			Name:  fmt.Sprintf("Create%s", meta.Name),
			Macro: "create_record",
			Embed: map[string]string{
				"table":   strings.ToLower(meta.Name),
				"columns": strings.Join(cols, ", "),
				"values":  strings.Join(vals, ", "),
			},
		},
	}
}

// Operations returns the HTTP operations for creating, reading, updating, and deleting the core resource.
func (c *zCore[T]) Operations() []*Operation {
	meta := c.Meta()
	return []*Operation{
		{
			Name:        fmt.Sprintf("Create %s", meta.Name),
			Description: fmt.Sprintf("Create a new `%s` in the database.", meta.Name),
			Method:      "POST",
			Path:        fmt.Sprintf("/%s", strings.ToLower(meta.Name)),
			Tag:         meta.Name,
			RequestBody: fmt.Sprintf("Create%sPayload", meta.Name),
			Response: &Response{
				Status: http.StatusCreated,
				Ref:    meta.Name,
				Errors: []int{
					http.StatusBadRequest,
				},
			},
			Handler: c.CreateHandler,
			Auth:    true,
		},
		{
			Name:        fmt.Sprintf("Get %s", meta.Name),
			Description: fmt.Sprintf("Get a specific `%s` by ID.", meta.Name),
			Method:      "GET",
			Path:        fmt.Sprintf("/%s/{id}", strings.ToLower(meta.Name)),
			Tag:         meta.Name,
			Parameters:  []string{"Id"},
			Response: &Response{
				Status: http.StatusOK,
				Ref:    meta.Name,
				Errors: []int{
					http.StatusBadRequest,
					http.StatusNotFound,
				},
			},
			Handler: c.ReadHandler,
			Auth:    true,
		},
		{
			Name:        fmt.Sprintf("Update %s", meta.Name),
			Description: fmt.Sprintf("Update a specific `%s` by ID.", meta.Name),
			Method:      "PUT",
			Path:        fmt.Sprintf("/%s/{id}", strings.ToLower(meta.Name)),
			Tag:         meta.Name,
			Parameters:  []string{"Id"},
			RequestBody: fmt.Sprintf("Update%sPayload", meta.Name),
			Response: &Response{
				Status: http.StatusOK,
				Ref:    meta.Name,
				Errors: []int{
					http.StatusBadRequest,
					http.StatusNotFound,
				},
			},
			Handler: c.UpdateHandler,
			Auth:    true,
		},
		{
			Name:        fmt.Sprintf("Delete %s", meta.Name),
			Description: fmt.Sprintf("Delete a specific `%s` by ID.", meta.Name),
			Method:      "DELETE",
			Path:        fmt.Sprintf("/%s/{id}", strings.ToLower(meta.Name)),
			Tag:         meta.Name,
			Parameters:  []string{"Id"},
			Response: &Response{
				Status: http.StatusNoContent,
			},
			Handler: c.DeleteHandler,
			Auth:    true,
		},
	}
}

// CreateHandler handles the creation of a new record.
func (c *zCore[T]) CreateHandler(ctx *gin.Context) {
}

// ReadHandler retrieves a record by ID.
func (c *zCore[T]) ReadHandler(ctx *gin.Context) {
}

// UpdateHandler updates a record by ID.
func (c *zCore[T]) UpdateHandler(ctx *gin.Context) {
}

// DeleteHandler deletes a record by ID.
func (c *zCore[T]) DeleteHandler(ctx *gin.Context) {
}
