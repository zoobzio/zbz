package controllers

import (
	"zbz/lib"
	"zbz/lib/middleware"

	"net/http"

	"github.com/gin-gonic/gin"
)

// CoreController is a generic controller for CRUD operations
type CoreController[T any] struct {
	e *zbz.Engine
}

// NewCoreController creates a new core controller for the given model type
func NewCoreController[T any](p string, e *zbz.Engine) *CoreController[T] {
	core := &CoreController[T]{e: e}
	core.register(p)
	return core
}

// Register the routes for the core controller
func (c *CoreController[T]) register(path string) {
	router := c.e.R.Group(path).Use(middleware.IsAuthenticated)
	{
		router.POST("", c.create)
		router.GET("/:id", c.get)
		router.GET("", c.list)
		router.PUT("/:id", c.update)
		router.DELETE("/:id", c.delete)
	}
}

// Create a record
func (c *CoreController[T]) create(ctx *gin.Context) {
	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := c.e.V.Struct(record)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := c.e.D.Create(&record).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusCreated, record)
}

// Get a record by ID
func (c *CoreController[T]) get(ctx *gin.Context) {
	recordID := ctx.Param("id")
	err := c.e.V.Var(recordID, "uuid")
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := c.e.D.First(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	err = c.e.V.Struct(record)
	if err != nil {
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// List all records
func (c *CoreController[T]) list(ctx *gin.Context) {
	var records []T
	if err := c.e.D.Find(&records).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	for _, record := range records {
		err := c.e.V.Struct(record)
		if err != nil {
			ctx.Status(http.StatusBadRequest)
			return
		}
	}

	ctx.JSON(http.StatusOK, records)
}

// Update a record by ID
func (c *CoreController[T]) update(ctx *gin.Context) {
	recordID := ctx.Param("id")

	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := c.e.V.Struct(record)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var model T
	if err := c.e.D.Model(&model).Where("id = ?", recordID).Updates(record).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// Delete a record by ID
func (c *CoreController[T]) delete(ctx *gin.Context) {
	recordID := ctx.Param("id")
	err := c.e.V.Var(recordID, "uuid")
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := c.e.D.Delete(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
