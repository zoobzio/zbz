package zbz

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Core is an interface that defines the basic CRUD operations for a resource.
type Core interface {
	Description() string
	Meta() *Meta
	Operations() []*HTTPOperation

	CreateHandler(ctx *gin.Context)
	ReadHandler(ctx *gin.Context)
	UpdateHandler(ctx *gin.Context)
	DeleteHandler(ctx *gin.Context)
}

// ZbzCore is a generic implementation for core CRUD operations.
type ZbzCore[T BaseModel] struct {
	description string
}

// NewCore creates a new instance of ZbzCore with the provided logger, config, and database.
func NewCore[T BaseModel](desc string) Core {
	return &ZbzCore[T]{
		description: desc,
	}
}

// Description returns the description of the core resource.
func (c *ZbzCore[T]) Description() string {
	return c.description
}

func (c *ZbzCore[T]) Meta() *Meta {
	return ExtractMeta[T]()
}

// Operations returns the HTTP operations for creating, reading, updating, and deleting the core resource.
func (c *ZbzCore[T]) Operations() []*HTTPOperation {
	meta := c.Meta()
	return []*HTTPOperation{
		{
			Name:        fmt.Sprintf("Create %s", meta.Name),
			Description: fmt.Sprintf("Create a new `%s` in the database.", meta.Name),
			Method:      "POST",
			Path:        fmt.Sprintf("/%s", strings.ToLower(meta.Name)),
			Tag:         meta.Name,
			RequestBody: fmt.Sprintf("Create%sPayload", meta.Name),
			Response: &HTTPResponse{
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
			Response: &HTTPResponse{
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
			Response: &HTTPResponse{
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
			Response: &HTTPResponse{
				Status: http.StatusNoContent,
			},
			Handler: c.DeleteHandler,
			Auth:    true,
		},
	}
}

// Create a record
func (c *ZbzCore[T]) CreateHandler(ctx *gin.Context) {
	log := ctx.MustGet("log").(Logger)
	db := ctx.MustGet("db").(Database)

	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		log.Errorf("Failed to bind JSON: %v", err)
		ctx.Status(http.StatusBadRequest)
		return
	}

	// TODO extract known editable fields to avoid side-effects

	if err := db.Create(&record).Error; err != nil {
		log.Errorf("Failed to create record: %v", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	err := db.IsValid(record)
	if err != nil {
		log.Errorf("Validation failed: %v", err)
		ctx.Status(http.StatusBadRequest)
		return
	}

	ctx.JSON(http.StatusCreated, record)
}

// Read a record by ID
func (c *ZbzCore[T]) ReadHandler(ctx *gin.Context) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	recordID := ctx.Param("id")
	err := db.IsValidID(recordID)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := db.First(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	err = db.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// Update a record by ID
func (c *ZbzCore[T]) UpdateHandler(ctx *gin.Context) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	recordID := ctx.Param("id")

	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := db.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var model T
	if err := db.Model(&model).Where("id = ?", recordID).Updates(record).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// Delete a record by ID
func (c *ZbzCore[T]) DeleteHandler(ctx *gin.Context) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	recordID := ctx.Param("id")
	err := db.IsValidID(recordID)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := db.Delete(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
