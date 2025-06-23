package zbz

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Core is an interface that defines the basic CRUD operations for a resource.
type Core interface {
	Description() string
	Meta() *Meta

	Table() *MacroContract
	Contracts() []*MacroContract
	Operations() []*Operation

	// Business logic methods (return data, framework-agnostic)
	Create(ctx Context) (any, error)
	Read(ctx Context) (any, error)
	Update(ctx Context) (any, error)
	Delete(ctx Context) error

	// HTTP handlers (for backward compatibility, will be deprecated)
	CreateHandler(ctx *gin.Context)
	ReadHandler(ctx *gin.Context)
	UpdateHandler(ctx *gin.Context)
	DeleteHandler(ctx *gin.Context)
	
	// Internal method for setting validated embeds during injection
	setEmbeds(embeds MacroEmbeds)
}

// zCore is a generic implementation for core CRUD operations.
type zCore[T BaseModel] struct {
	meta        *Meta
	description string
	embeds      MacroEmbeds
	contracts   map[string]*MacroContract
	operations  map[string]*Operation
	validator   Validate // Each core has its own validator instance
	handler     *Handler[T] // HTTP handler bound to this core
}

// NewCore creates a new instance of zCore with the provided description
func NewCore[T BaseModel](desc string) Core {
	meta := extractMeta[T](desc)
	core := &zCore[T]{
		meta:        meta,
		description: desc,
		contracts:   make(map[string]*MacroContract),
		operations:  make(map[string]*Operation),
		validator:   NewValidate(), // Each core gets its own validator
	}
	
	// Create and bind the HTTP handler to this core
	core.handler = NewHandler[T](core)
	
	return core
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
	for _, field := range meta.FieldMetadata {
		if field.GoType == "zbz.Model" || field.DatabaseColumnName == "-" {
			continue
		}
		def := fmt.Sprintf("%s %s", field.DatabaseColumnName, field.DatabaseType)
		if field.DatabaseColumnName == "id" {
			def += " PRIMARY KEY"
		}
		if field.IsRequired {
			def += " NOT NULL"
		}
		defs = append(defs, def)
	}
	// For table creation, we need basic embeds without validation since table doesn't exist yet
	tableEmbeds := MacroEmbeds{
		Table:   TrustedSQLIdentifier{value: strings.ToLower(meta.Name)},
		Columns: TrustedSQLIdentifier{value: strings.Join(defs, ", ")},
		Values:  TrustedSQLIdentifier{value: ""},
		Updates: TrustedSQLIdentifier{value: ""},
	}
	
	return &MacroContract{
		Name:  fmt.Sprintf("Create%sTable", meta.Name),
		Macro: "create_table",
		Embed: tableEmbeds,
	}
}

// createContracts returns the SQL statements associated with the core resource.
func (c *zCore[T]) createContracts() {
	// Use the pre-built validated embeds for all contracts
	c.contracts["create"] = &MacroContract{
		Name:  fmt.Sprintf("Create%s", c.meta.Name),
		Macro: "create_record",
		Embed: c.embeds,
	}
	c.contracts["select"] = &MacroContract{
		Name:  fmt.Sprintf("Select%s", c.meta.Name),
		Macro: "select_record",
		Embed: c.embeds,
	}
	c.contracts["update"] = &MacroContract{
		Name:  fmt.Sprintf("Update%s", c.meta.Name),
		Macro: "update_record",
		Embed: c.embeds,
	}
	c.contracts["delete"] = &MacroContract{
		Name:  fmt.Sprintf("Delete%s", c.meta.Name),
		Macro: "delete_record",
		Embed: c.embeds,
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
		Handler: c.handler.CreateHandler,
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
		Handler: c.handler.ReadHandler,
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
		Handler: c.handler.UpdateHandler,
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
		Handler: c.handler.DeleteHandler,
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

// setEmbeds stores validated macro embeds in the core (called during injection)
func (c *zCore[T]) setEmbeds(embeds MacroEmbeds) {
	c.embeds = embeds
}

// Create handles the business logic for creating a new record
func (c *zCore[T]) Create(ctx Context) (any, error) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	// Get raw JSON from request body
	jsonData, err := ctx.GetRawData()
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Create a new model instance
	var payload T

	// Use scoped deserialization to validate field access permissions for creation
	err = DeserializeWithScopes(ctx.(*gin.Context), jsonData, &payload, OperationCreate)
	if err != nil {
		return nil, fmt.Errorf("deserialization failed: %w", err)
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

	err = c.validator.IsValid(payload)
	if err != nil {
		return nil, err // Return validation error as-is
	}

	Log.Info("Creating a new record", zap.String("model", c.meta.Name), zap.Any("payload", payload))

	rows, err := db.Execute(c.contracts["create"], payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("failed to create record")
	}

	// Validate created record
	err = c.validator.IsValid(results[0])
	if err != nil {
		return nil, fmt.Errorf("created record failed validation: %w", err)
	}

	return results[0], nil
}

// Read handles the business logic for retrieving a record by ID
func (c *zCore[T]) Read(ctx Context) (any, error) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	id := ctx.Param("id")
	if id == "" {
		return nil, fmt.Errorf("ID parameter is required")
	}

	err := c.validator.IsValidID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}

	Log.Info("Retrieving a record", zap.String("model", c.meta.Name), zap.String("id", id))

	rows, err := db.Execute(c.contracts["select"], map[string]any{"id": id})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve record: %w", err)
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("record not found")
	}

	return results[0], nil
}

// Update handles the business logic for updating a record by ID
func (c *zCore[T]) Update(ctx Context) (any, error) {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	id := ctx.Param("id")
	if id == "" {
		return nil, fmt.Errorf("ID parameter is required")
	}

	err := c.validator.IsValidID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}

	// First, get the existing record
	existing, err := db.Execute(c.contracts["select"], map[string]any{"id": id})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve existing record: %w", err)
	}
	defer existing.Close()

	var eresults []T
	for existing.Next() {
		var row T
		if err := existing.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan existing row: %w", err)
		}
		eresults = append(eresults, row)
	}

	if len(eresults) == 0 {
		return nil, fmt.Errorf("record not found")
	}

	// Get raw JSON from request body
	jsonData, err := ctx.GetRawData()
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Create a copy of existing record for patching
	payload := eresults[0]

	// Use scoped deserialization to validate field access permissions
	if len(jsonData) > 0 {
		err = DeserializeWithScopes(ctx.(*gin.Context), jsonData, &payload, OperationUpdate)
		if err != nil {
			return nil, fmt.Errorf("deserialization failed: %w", err)
		}
	}

	err = c.validator.IsValid(payload)
	if err != nil {
		return nil, err // Return validation error as-is
	}

	Log.Info("Updating a record", zap.String("model", c.meta.Name), zap.String("id", id), zap.Any("payload", payload))

	rows, err := db.Execute(c.contracts["update"], payload)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan updated row: %w", err)
		}
		results = append(results, row)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("record not found after update")
	}

	// Validate updated record
	err = c.validator.IsValid(results[0])
	if err != nil {
		return nil, fmt.Errorf("updated record failed validation: %w", err)
	}

	return results[0], nil
}

// Delete handles the business logic for deleting a record by ID
func (c *zCore[T]) Delete(ctx Context) error {
	db := ctx.MustGet("db").(Database)
	if db == nil {
		return fmt.Errorf("database connection is not available")
	}

	id := ctx.Param("id")
	if id == "" {
		return fmt.Errorf("ID parameter is required")
	}

	err := c.validator.IsValidID(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	Log.Info("Deleting a record", zap.String("model", c.meta.Name), zap.String("id", id))

	_, err = db.Execute(c.contracts["delete"], map[string]any{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
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
	jsonData, err := ctx.GetRawData()
	if err != nil {
		Log.Error("Failed to read request body", zap.Error(err))
		ctx.Set("error_message", "Failed to read request body")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Create a new model instance
	var payload T

	// Use scoped deserialization to validate field access permissions for creation
	// Note: Type assert to gin.Context since cereal functions haven't been abstracted yet
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

	err = c.validator.IsValid(payload)
	if err != nil {
		Log.Error("Payload validation failed", zap.Error(err))
		validationErrors := c.validator.ExtractErrors(err)
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
		err = c.validator.IsValid(result)
		if err != nil {
			Log.Error("Validation failed for created record", zap.Error(err), zap.Int("index", i))
			ctx.Set("error_message", "Created record failed validation")
			ctx.Status(http.StatusInternalServerError)
			return
		}
	}

	// Use scoped serialization for the response
	// Note: This will be removed once Handler fully replaces direct HTTP handling in Core
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

	err := c.validator.IsValidID(id)
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
	// Note: This will be removed once Handler fully replaces direct HTTP handling in Core
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

	err := c.validator.IsValidID(id)
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
	jsonData, err := ctx.GetRawData()
	if err != nil {
		Log.Error("Failed to read request body", zap.Error(err))
		ctx.Set("error_message", "Failed to read request body")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Create a copy of existing record for patching
	payload := eresults[0]

	// Use scoped deserialization to validate field access permissions
	if len(jsonData) > 0 {
		// Note: Type assert to gin.Context since cereal functions haven't been abstracted yet
		err = DeserializeWithScopes(ctx, jsonData, &payload, OperationUpdate)
		if err != nil {
			Log.Error("Scoped deserialization failed", zap.Error(err))
			ctx.Set("error_message", err.Error())
			ctx.Status(http.StatusForbidden)
			return
		}
	}

	err = c.validator.IsValid(payload)
	if err != nil {
		Log.Error("Payload validation failed", zap.Error(err))
		validationErrors := c.validator.ExtractErrors(err)
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
		err = c.validator.IsValid(result)
		if err != nil {
			Log.Error("Validation failed for updated record", zap.Error(err), zap.Int("index", i))
			ctx.Set("error_message", "Updated record failed validation")
			ctx.Status(http.StatusInternalServerError)
			return
		}
	}

	// Use scoped serialization for the response
	// Note: This will be removed once Handler fully replaces direct HTTP handling in Core
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

	err := c.validator.IsValidID(id)
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
