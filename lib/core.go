package zbz

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Core
type Core[T any] struct {
	config   *Config
	database Database
	log      Logger
}

// NewCore
func NewCore[T any](
	l Logger,
	c *Config,
	d Database,
) *Core[T] {
	return &Core[T]{
		config:   c,
		database: d,
		log:      l,
	}
}

// Create a record
func (c *Core[T]) Create(ctx *gin.Context) {
	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := c.database.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := c.database.Create(&record).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusCreated, record)
}

// Read a record by ID
func (c *Core[T]) Read(ctx *gin.Context) {
	recordID := ctx.Param("id")
	err := c.database.IsValidID(recordID)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := c.database.First(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	err = c.database.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// Update a record by ID
func (c *Core[T]) Update(ctx *gin.Context) {
	recordID := ctx.Param("id")

	var record T
	if err := ctx.ShouldBindJSON(&record); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := c.database.IsValid(record)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var model T
	if err := c.database.Model(&model).Where("id = ?", recordID).Updates(record).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, record)
}

// Delete a record by ID
func (c *Core[T]) Delete(ctx *gin.Context) {
	recordID := ctx.Param("id")
	err := c.database.IsValidID(recordID)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var record T
	if err := c.database.Delete(&record, recordID).Error; err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
