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
	meta        *Meta
	description string
	contracts   map[string]*MacroContract
	operations  map[string]*Operation
}

// NewCore creates a new instance of zCore with the provided logger, config, and database.
func NewCore[T BaseModel](desc string) Core {
	meta := extractMeta[T](desc)
	return &zCore[T]{
		meta:        meta,
		description: desc,
		contracts:   make(map[string]*MacroContract),
		operations:  make(map[string]*Operation),
	}
}

// Description returns the description of the core resource.
func (c *zCore[T]) Description() string {
	return c.description
}

// Meta extracts the metadata for the core resource, which includes its name and other properties.
func (c *zCore[T]) Meta() *Meta {
	return c.meta
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
func (c *zCore[T]) createContracts() {
	cols := c.meta.Columns
	vals := []string{}
	valupdates := []string{}
	for _, col := range c.meta.Columns {
		if slices.Contains(cols, col) {
			vals = append(vals, fmt.Sprintf(":%s", col))
			if col != "id" {
				valupdates = append(valupdates, fmt.Sprintf("%s = :%s", col, col))
			}
		}
	}

	tblstring := strings.ToLower(c.meta.Name)
	colstring := strings.Join(cols, ", ")
	valstring := strings.Join(vals, ", ")
	valupdatestring := strings.Join(valupdates, ", ")

	c.contracts["create"] = &MacroContract{
		Name:  fmt.Sprintf("Create%s", c.meta.Name),
		Macro: "create_record",
		Embed: map[string]string{
			"table":   tblstring,
			"columns": colstring,
			"values":  valstring,
		},
	}
	c.contracts["select"] = &MacroContract{
		Name:  fmt.Sprintf("Select%s", c.meta.Name),
		Macro: "select_record",
		Embed: map[string]string{
			"table":   tblstring,
			"columns": colstring,
		},
	}
	c.contracts["update"] = &MacroContract{
		Name:  fmt.Sprintf("Update%s", c.meta.Name),
		Macro: "update_record",
		Embed: map[string]string{
			"table":   tblstring,
			"columns": colstring,
			"updates": valupdatestring,
		},
	}
	c.contracts["delete"] = &MacroContract{
		Name:  fmt.Sprintf("Delete%s", c.meta.Name),
		Macro: "delete_record",
		Embed: map[string]string{
			"table": tblstring,
		},
	}
}

// Contracts returns the SQL contracts for the core resource.
func (c *zCore[T]) Contracts() []*MacroContract {
	if len(c.contracts) == 0 {
		c.createContracts()
	}
	contracts := []*MacroContract{}
	for _, contract := range c.contracts {
		contracts = append(contracts, contract)
	}
	return contracts
}

// Operations returns the HTTP operations for creating, reading, updating, and deleting the core resource.
func (c *zCore[T]) createOperations() {
	c.operations["create"] = &Operation{
		Name:        fmt.Sprintf("Create %s", c.meta.Name),
		Description: fmt.Sprintf("Create a new `%s` in the database.", c.meta.Name),
		Method:      "POST",
		Path:        fmt.Sprintf("/%s", strings.ToLower(c.meta.Name)),
		Tag:         c.meta.Name,
		RequestBody: fmt.Sprintf("Create%sPayload", c.meta.Name),
		Response: &Response{
			Status: http.StatusCreated,
			Ref:    c.meta.Name,
			Errors: []int{
				http.StatusBadRequest,
			},
		},
		Handler: c.CreateHandler,
		Auth:    true,
	}
	c.operations["read"] = &Operation{
		Name:        fmt.Sprintf("Get %s", c.meta.Name),
		Description: fmt.Sprintf("Get a specific `%s` by ID.", c.meta.Name),
		Method:      "GET",
		Path:        fmt.Sprintf("/%s/{id}", strings.ToLower(c.meta.Name)),
		Tag:         c.meta.Name,
		Parameters:  []string{"Id"},
		Response: &Response{
			Status: http.StatusOK,
			Ref:    c.meta.Name,
			Errors: []int{
				http.StatusBadRequest,
				http.StatusNotFound,
			},
		},
		Handler: c.ReadHandler,
		Auth:    true,
	}
	c.operations["update"] = &Operation{
		Name:        fmt.Sprintf("Update %s", c.meta.Name),
		Description: fmt.Sprintf("Update a specific `%s` by ID.", c.meta.Name),
		Method:      "PUT",
		Path:        fmt.Sprintf("/%s/{id}", strings.ToLower(c.meta.Name)),
		Tag:         c.meta.Name,
		Parameters:  []string{"Id"},
		RequestBody: fmt.Sprintf("Update%sPayload", c.meta.Name),
		Response: &Response{
			Status: http.StatusOK,
			Ref:    c.meta.Name,
			Errors: []int{
				http.StatusBadRequest,
				http.StatusNotFound,
			},
		},
		Handler: c.UpdateHandler,
		Auth:    true,
	}
	c.operations["delete"] = &Operation{
		Name:        fmt.Sprintf("Delete %s", c.meta.Name),
		Description: fmt.Sprintf("Delete a specific `%s` by ID.", c.meta.Name),
		Method:      "DELETE",
		Path:        fmt.Sprintf("/%s/{id}", strings.ToLower(c.meta.Name)),
		Tag:         c.meta.Name,
		Parameters:  []string{"Id"},
		Response: &Response{
			Status: http.StatusNoContent,
		},
		Handler: c.DeleteHandler,
		Auth:    true,
	}
}

// Operations returns the HTTP operations for the core resource.
func (c *zCore[T]) Operations() []*Operation {
	if len(c.operations) == 0 {
		c.createOperations()
	}
	ops := []*Operation{}
	for _, op := range c.operations {
		ops = append(ops, op)
	}
	return ops
}

// CreateHandler handles the creation of a new record.
func (c *zCore[T]) CreateHandler(ctx *gin.Context) {

	db := ctx.MustGet("db").(Database)
	if db == nil {
		Log.Error("Database connection is not available")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	payload, err := NewModel[T](ctx)
	if err != nil {
		Log.Errorw("Failed to bind JSON payload", "error", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	err = validate.IsValid(payload)
	if err != nil {
		Log.Errorw("Payload validation failed", "error", err)
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Infow("Creating a new record", "model", c.meta.Name, "payload", payload)

	rows, err := db.Execute(c.contracts["create"], payload)
	if err != nil {
		Log.Errorw("Failed to create record", "error", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []T

	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			Log.Errorw("Failed to scan row", "error", err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		Log.Warn("No records created")
		ctx.Status(http.StatusNoContent)
		return
	}

	for i, result := range results {
		err = validate.IsValid(result)
		if err != nil {
			Log.Errorw("Validation failed for created record", "error", err, "index", i)
			ctx.Status(http.StatusBadRequest)
			return
		}
	}

	ctx.JSON(http.StatusCreated, results[0])
}

// ReadHandler retrieves a record by ID.
func (c *zCore[T]) ReadHandler(ctx *gin.Context) {

	db := ctx.MustGet("db").(Database)
	if db == nil {
		Log.Error("Database connection is not available")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	id := ctx.Param("id")
	if id == "" {
		Log.Error("ID parameter is missing")
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := validate.IsValidID(id)
	if err != nil {
		Log.Errorw("Invalid ID", "error", err, "id", id)
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Infow("Retrieving a record", "model", c.meta.Name, "id", id)

	rows, err := db.Execute(c.contracts["select"], map[string]any{"id": id})
	if err != nil {
		Log.Errorw("Failed to retrieve record", "error", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []T

	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			Log.Errorw("Failed to scan row", "error", err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		Log.Warn("No records found for ID", "id", id)
		ctx.Status(http.StatusNotFound)
		return
	}

	ctx.JSON(http.StatusOK, results[0])
}

// UpdateHandler updates a record by ID.
func (c *zCore[T]) UpdateHandler(ctx *gin.Context) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		Log.Error("Database connection is not available")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	id := ctx.Param("id")
	if id == "" {
		Log.Error("ID parameter is missing")
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := validate.IsValidID(id)
	if err != nil {
		Log.Errorw("Invalid ID", "error", err, "id", id)
		ctx.Status(http.StatusBadRequest)
		return
	}

	existing, err := db.Execute(c.contracts["select"], map[string]any{"id": id})
	if err != nil {
		Log.Errorw("Failed to retrieve existing record", "error", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer existing.Close()

	var eresults []T

	for existing.Next() {
		var row T
		if err := existing.StructScan(&row); err != nil {
			Log.Errorw("Failed to scan row", "error", err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		eresults = append(eresults, row)
	}

	if len(eresults) == 0 {
		Log.Warn("No records found for ID", "id", id)
		ctx.Status(http.StatusNotFound)
		return
	}

	payload, err := PatchModel(ctx, &eresults[0])
	if err != nil {
		Log.Errorw("Failed to bind JSON payload", "error", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	err = validate.IsValid(payload)
	if err != nil {
		Log.Errorw("Payload validation failed", "error", err)
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Infow("Updating a record", "model", c.meta.Name, "id", id, "payload", payload)

	rows, err := db.Execute(c.contracts["update"], payload)
	if err != nil {
		Log.Errorw("Failed to update record", "error", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []T

	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			Log.Errorw("Failed to scan row", "error", err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		Log.Warn("No records updated for ID", "id", id)
		ctx.Status(http.StatusNotFound)
		return
	}

	for i, result := range results {
		err = validate.IsValid(result)
		if err != nil {
			Log.Errorw("Validation failed for updated record", "error", err, "index", i)
			ctx.Status(http.StatusBadRequest)
			return
		}
	}

	ctx.JSON(http.StatusOK, results[0])
}

// DeleteHandler deletes a record by ID.
func (c *zCore[T]) DeleteHandler(ctx *gin.Context) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		Log.Error("Database connection is not available")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	id := ctx.Param("id")
	if id == "" {
		Log.Error("ID parameter is missing")
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := validate.IsValidID(id)
	if err != nil {
		Log.Errorw("Invalid ID", "error", err, "id", id)
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Infow("Deleting a record", "model", c.meta.Name, "id", id)

	_, err = db.Execute(c.contracts["delete"], map[string]any{"id": id})
	if err != nil {
		Log.Errorw("Failed to delete record", "error", err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
