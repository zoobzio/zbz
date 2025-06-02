package zbz

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Core is an interface that defines the basic CRUD operations for a resource.
type Core interface {
	Create(ctx *gin.Context)
	Read(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

// CoreOperation defines the operations for a core resource, including Create, Read, Update, and Delete.
type CoreOperation struct {
	Create *HTTPOperation
	Read   *HTTPOperation
	Update *HTTPOperation
	Delete *HTTPOperation
}

// ZbzCore is a generic implementation for core CRUD operations.
type ZbzCore[T any] struct {
	engine *Engine
}

// NewCore creates a new instance of ZbzCore with the provided logger, config, and engine.Database.
func NewCore[T any](e *Engine, op *CoreOperation) Core {
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
		engine: e,
	}
}

// Create a record
func (c *ZbzCore[T]) Create(ctx *gin.Context) {
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
func (c *ZbzCore[T]) Read(ctx *gin.Context) {
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
func (c *ZbzCore[T]) Update(ctx *gin.Context) {
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
func (c *ZbzCore[T]) Delete(ctx *gin.Context) {
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
