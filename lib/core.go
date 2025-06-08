package zbz

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
)

// Core is an interface that defines the basic CRUD operations for a resource.
type Core interface {
	Meta() *CoreMeta
	CreateHandler(ctx *gin.Context)
	ReadHandler(ctx *gin.Context)
	UpdateHandler(ctx *gin.Context)
	DeleteHandler(ctx *gin.Context)
}

// CoreMeta defines the metadata for a core resource, including its name, description, and example.
type CoreMeta struct {
	Name        string
	Description string
	Example     any
	Fields      []*CoreMeta
}

// CoreModel is a generic model for core resources, which can be extended to include specific fields or methods.
type CoreModel struct {
	Name        string
	Description string
	Example     any
}

// CoreOperation defines the operations for a core resource, including Create, Read, Update, and Delete.
type CoreOperation struct {
	Model  *CoreModel
	Create *HTTPOperation
	Read   *HTTPOperation
	Update *HTTPOperation
	Delete *HTTPOperation
}

// ZbzCore is a generic implementation for core CRUD operations.
type ZbzCore[T any] struct {
	engine  *Engine
	example *T
}

// NewCore creates a new instance of ZbzCore with the provided logger, config, and database.
func NewCore[T any](e *Engine, op *CoreOperation, ex *T) Core {
	if op.Create != nil {
		e.Inject(op.Create)
	}
	if op.Read != nil {
		e.Inject(op.Read)
	}
	if op.Update != nil {
		e.Inject(op.Update)
	}
	if op.Delete != nil {
		e.Inject(op.Delete)
	}
	return &ZbzCore[T]{
		engine:  e,
		example: ex,
	}
}

// Meta returns the metadata for the core resource, including its name, description, and example.
func (c *ZbzCore[T]) Meta() *CoreMeta {
	d := "blah blah blah"
	t := reflect.TypeOf(*c.example)
	m := &CoreMeta{
		Name:        t.Name(),
		Description: d,
		Example:     *c.example,
		Fields:      make([]*CoreMeta, 0, t.NumField()),
	}
	for i := range t.NumField() {
		field := t.Field(i)
		s := field.Tag.Get("desc")
		e := field.Tag.Get("ex")
		m.Fields = append(m.Fields, &CoreMeta{
			Name:        field.Name,
			Description: s,
			Example:     e,
		})
	}
	return m
}

// Create a record
func (c *ZbzCore[T]) CreateHandler(ctx *gin.Context) {
	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := c.engine.Database.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := c.engine.Database.Create(&record).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusCreated, record)
}

// Read a record by ID
func (c *ZbzCore[T]) ReadHandler(ctx *gin.Context) {
	recordID := ctx.Param("id")
	err := c.engine.Database.IsValidID(recordID)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := c.engine.Database.First(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	err = c.engine.Database.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// Update a record by ID
func (c *ZbzCore[T]) UpdateHandler(ctx *gin.Context) {
	recordID := ctx.Param("id")

	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := c.engine.Database.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var model T
	if err := c.engine.Database.Model(&model).Where("id = ?", recordID).Updates(record).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// Delete a record by ID
func (c *ZbzCore[T]) DeleteHandler(ctx *gin.Context) {
	recordID := ctx.Param("id")
	err := c.engine.Database.IsValidID(recordID)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := c.engine.Database.Delete(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
