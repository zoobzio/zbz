package zbz

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CoreContract represents a configuration for how a Core should be exposed via HTTP
type CoreContract struct {
	Core        Core
	Description string   // Description for the OpenAPI tag
	Handlers    []string // Handler names to enable (nil = all handlers)
}

// Core is an interface that defines the basic CRUD operations for a resource.
type Core interface {
	Description() string
	Meta() *Meta
	Contract(description string, handlers ...string) *CoreContract

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

// Contract creates a CoreContract for HTTP exposure with the given description and optional handler filtering
func (c *zCore[T]) Contract(description string, handlers ...string) *CoreContract {
	return &CoreContract{
		Core:        c,
		Description: description,
		Handlers:    handlers, // nil if no handlers specified (enables all)
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
		Parameters:  []string{"id"},
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
		Parameters:  []string{"id"},
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
		Parameters:  []string{"id"},
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
		ctx.Set("error_message", "Database connection is not available")
		ctx.Status(http.StatusServiceUnavailable)
		return
	}

	// Get raw JSON from request body
	jsonData, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		Log.Error("Failed to read request body", zap.Error(err))
		ctx.Set("error_message", "Failed to read request body")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Create a new model instance
	var payload T

	// Use scoped deserialization to validate field access permissions for creation
	err = DeserializeWithScopes(ctx, jsonData, &payload, OperationCreate)
	if err != nil {
		Log.Error("Scoped deserialization failed", zap.Error(err))
		ctx.Set("error_message", err.Error())
		ctx.Status(http.StatusForbidden)
		return
	}

	// Set model fields (ID, timestamps, etc.) manually since we used scoped deserialization
	val := reflect.ValueOf(&payload).Elem()
	modelField := val.FieldByName("Model")
	if modelField.IsValid() && modelField.CanSet() {
		model := Model{
			ID:        uuid.NewString(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		modelField.Set(reflect.ValueOf(model))
	}

	err = validate.IsValid(payload)
	if err != nil {
		Log.Error("Payload validation failed", zap.Error(err))
		validationErrors := validate.ExtractErrors(err)
		ctx.Set("error_details", validationErrors)
		ctx.Set("error_message", "Validation failed")
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Info("Creating a new record", zap.String("model", c.meta.Name), zap.Any("payload", payload))

	rows, err := db.Execute(c.contracts["create"], payload)
	if err != nil {
		Log.Error("Failed to create record", zap.Error(err))
		if strings.Contains(err.Error(), "duplicate key") {
			ctx.Set("error_message", "Resource already exists")
			ctx.Status(http.StatusConflict)
		} else {
			ctx.Status(http.StatusInternalServerError)
		}
		return
	}
	defer rows.Close()

	var results []T

	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			Log.Error("Failed to scan row", zap.Error(err))
			ctx.Status(http.StatusInternalServerError)
			return
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		Log.Warn("No records created")
		ctx.Set("error_message", "Failed to create record")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	for i, result := range results {
		err = validate.IsValid(result)
		if err != nil {
			Log.Error("Validation failed for created record", zap.Error(err), zap.Int("index", i))
			ctx.Set("error_message", "Created record failed validation")
			ctx.Status(http.StatusInternalServerError)
			return
		}
	}

	// Use scoped serialization for the response
	RespondWithScopedJSON(ctx, http.StatusCreated, results[0])
}

// ReadHandler retrieves a record by ID.
func (c *zCore[T]) ReadHandler(ctx *gin.Context) {

	db := ctx.MustGet("db").(Database)
	if db == nil {
		Log.Error("Database connection is not available")
		ctx.Set("error_message", "Database connection is not available")
		ctx.Status(http.StatusServiceUnavailable)
		return
	}

	id := ctx.Param("id")
	if id == "" {
		Log.Error("ID parameter is missing")
		ctx.Set("error_message", "ID parameter is required")
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := validate.IsValidID(id)
	if err != nil {
		Log.Error("Invalid ID", zap.Error(err), zap.String("id", id))
		ctx.Set("error_message", "Invalid ID format")
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Info("Retrieving a record", zap.String("model", c.meta.Name), zap.String("id", id))

	rows, err := db.Execute(c.contracts["select"], map[string]any{"id": id})
	if err != nil {
		Log.Error("Failed to retrieve record", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []T

	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			Log.Error("Failed to scan row", zap.Error(err))
			ctx.Status(http.StatusInternalServerError)
			return
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		Log.Warn("No records found for ID", zap.String("id", id))
		ctx.Status(http.StatusNotFound)
		return
	}

	// Use scoped serialization to filter fields based on user permissions
	RespondWithScopedJSON(ctx, http.StatusOK, results[0])
}

// UpdateHandler updates a record by ID.
func (c *zCore[T]) UpdateHandler(ctx *gin.Context) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		Log.Error("Database connection is not available")
		ctx.Set("error_message", "Database connection is not available")
		ctx.Status(http.StatusServiceUnavailable)
		return
	}

	id := ctx.Param("id")
	if id == "" {
		Log.Error("ID parameter is missing")
		ctx.Set("error_message", "ID parameter is required")
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := validate.IsValidID(id)
	if err != nil {
		Log.Error("Invalid ID", zap.Error(err), zap.String("id", id))
		ctx.Set("error_message", "Invalid ID format")
		ctx.Status(http.StatusBadRequest)
		return
	}

	existing, err := db.Execute(c.contracts["select"], map[string]any{"id": id})
	if err != nil {
		Log.Error("Failed to retrieve existing record", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer existing.Close()

	var eresults []T

	for existing.Next() {
		var row T
		if err := existing.StructScan(&row); err != nil {
			Log.Error("Failed to scan row", zap.Error(err))
			ctx.Status(http.StatusInternalServerError)
			return
		}
		eresults = append(eresults, row)
	}

	if len(eresults) == 0 {
		Log.Warn("No records found for ID", zap.String("id", id))
		ctx.Status(http.StatusNotFound)
		return
	}

	// Get raw JSON from request body
	var jsonData []byte
	if ctx.Request.Body != nil {
		jsonData, err = io.ReadAll(ctx.Request.Body)
		if err != nil {
			Log.Error("Failed to read request body", zap.Error(err))
			ctx.Set("error_message", "Failed to read request body")
			ctx.Status(http.StatusBadRequest)
			return
		}
	}

	// Create a copy of existing record for patching
	payload := eresults[0]

	// Use scoped deserialization to validate field access permissions
	if len(jsonData) > 0 {
		err = DeserializeWithScopes(ctx, jsonData, &payload, OperationUpdate)
		if err != nil {
			Log.Error("Scoped deserialization failed", zap.Error(err))
			ctx.Set("error_message", err.Error())
			ctx.Status(http.StatusForbidden)
			return
		}
	}

	err = validate.IsValid(payload)
	if err != nil {
		Log.Error("Payload validation failed", zap.Error(err))
		validationErrors := validate.ExtractErrors(err)
		ctx.Set("error_details", validationErrors)
		ctx.Set("error_message", "Validation failed")
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Info("Updating a record", zap.String("model", c.meta.Name), zap.String("id", id), zap.Any("payload", payload))

	rows, err := db.Execute(c.contracts["update"], payload)
	if err != nil {
		Log.Error("Failed to update record", zap.Error(err))
		if strings.Contains(err.Error(), "duplicate key") {
			ctx.Set("error_message", "Update would create a duplicate")
			ctx.Status(http.StatusConflict)
		} else {
			ctx.Status(http.StatusInternalServerError)
		}
		return
	}
	defer rows.Close()

	var results []T

	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			Log.Error("Failed to scan row", zap.Error(err))
			ctx.Status(http.StatusInternalServerError)
			return
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		Log.Warn("No records updated for ID", zap.String("id", id))
		ctx.Status(http.StatusNotFound)
		return
	}

	for i, result := range results {
		err = validate.IsValid(result)
		if err != nil {
			Log.Error("Validation failed for updated record", zap.Error(err), zap.Int("index", i))
			ctx.Set("error_message", "Updated record failed validation")
			ctx.Status(http.StatusInternalServerError)
			return
		}
	}

	// Use scoped serialization for the response
	RespondWithScopedJSON(ctx, http.StatusOK, results[0])
}

// DeleteHandler deletes a record by ID.
func (c *zCore[T]) DeleteHandler(ctx *gin.Context) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		Log.Error("Database connection is not available")
		ctx.Set("error_message", "Database connection is not available")
		ctx.Status(http.StatusServiceUnavailable)
		return
	}

	id := ctx.Param("id")
	if id == "" {
		Log.Error("ID parameter is missing")
		ctx.Set("error_message", "ID parameter is required")
		ctx.Status(http.StatusBadRequest)
		return
	}

	err := validate.IsValidID(id)
	if err != nil {
		Log.Error("Invalid ID", zap.Error(err), zap.String("id", id))
		ctx.Set("error_message", "Invalid ID format")
		ctx.Status(http.StatusBadRequest)
		return
	}

	Log.Info("Deleting a record", zap.String("model", c.meta.Name), zap.String("id", id))

	_, err = db.Execute(c.contracts["delete"], map[string]any{"id": id})
	if err != nil {
		Log.Error("Failed to delete record", zap.Error(err))
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
