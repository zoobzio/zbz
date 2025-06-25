package zbz

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"zbz/zlog"
)

// Core is an interface that defines the basic CRUD operations for a resource.
type Core interface {
	Description() string
	Meta() *Meta

	Table() *MacroContract
	MacroContracts() []*MacroContract
	HandlerContracts() []*HandlerContract

	// Business logic methods (return data, framework-agnostic)
	Create(ctx Context) (any, error)
	Read(ctx Context) (any, error)
	Update(ctx Context) (any, error)
	Delete(ctx Context) error
	
	// Contract metadata access (for engine orchestration)
	ContractName() string
	ContractDescription() string
	
	// Internal methods for setting resolved dependencies during injection
	setEmbeds(embeds MacroEmbeds)
	setDatabase(database Database)
	setContractMetadata(name, description string)
}

// zCore is a generic implementation for core CRUD operations.
type zCore[T BaseModel] struct {
	meta        *Meta
	description string
	embeds      MacroEmbeds
	macroContracts   map[string]*MacroContract
	handlerContracts  map[string]*HandlerContract
	validator   Validate // Each core has its own validator instance
	handler     *Handler[T] // HTTP handler bound to this core
	database    Database // Database service resolved from contract during injection
	
	// Contract metadata (from CoreContract that created this service)
	contractName        string
	contractDescription string
}

// NewCore creates a new instance of zCore with the provided description
func NewCore[T BaseModel](desc string) Core {
	meta := extractMeta[T](desc)
	core := &zCore[T]{
		meta:        meta,
		description: desc,
		macroContracts:   make(map[string]*MacroContract),
		handlerContracts:  make(map[string]*HandlerContract),
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

// MacroContracts returns the SQL contracts for the core resource.
func (c *zCore[T]) MacroContracts() []*MacroContract {
	if len(c.macroContracts) == 0 {
		c.createMacroContracts()
	}
	contracts := []*MacroContract{}
	for _, contract := range c.macroContracts {
		contracts = append(contracts, contract)
	}
	return contracts
}

// createMacroContracts returns the SQL statements associated with the core resource.
func (c *zCore[T]) createMacroContracts() {
	// Use the pre-built validated embeds for all contracts
	c.macroContracts["create"] = &MacroContract{
		Name:  fmt.Sprintf("Create%s", c.meta.Name),
		Macro: "create_record",
		Embed: c.embeds,
	}
	c.macroContracts["select"] = &MacroContract{
		Name:  fmt.Sprintf("Select%s", c.meta.Name),
		Macro: "select_record",
		Embed: c.embeds,
	}
	c.macroContracts["update"] = &MacroContract{
		Name:  fmt.Sprintf("Update%s", c.meta.Name),
		Macro: "update_record",
		Embed: c.embeds,
	}
	c.macroContracts["delete"] = &MacroContract{
		Name:  fmt.Sprintf("Delete%s", c.meta.Name),
		Macro: "delete_record",
		Embed: c.embeds,
	}
}

// getOperationDescription attempts to load an operation description from remarks,
// falling back silently to an empty string if not found
func (c *zCore[T]) getOperationDescription(operation string) string {
	remarkKey := strings.ToLower(c.meta.Name) + "_" + operation
	return Remark.MightGet(remarkKey)
}

// createHandlerContracts creates HTTP handler contracts for CRUD operations
func (c *zCore[T]) createHandlerContracts() {
	c.handlerContracts["create"] = &HandlerContract{
		Name:        fmt.Sprintf("Create %s", c.meta.Name),
		Description: c.getOperationDescription("create"),
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
	c.handlerContracts["read"] = &HandlerContract{
		Name:        fmt.Sprintf("Get %s", c.meta.Name),
		Description: c.getOperationDescription("read"),
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
	c.handlerContracts["update"] = &HandlerContract{
		Name:        fmt.Sprintf("Update %s", c.meta.Name),
		Description: c.getOperationDescription("update"),
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
	c.handlerContracts["delete"] = &HandlerContract{
		Name:        fmt.Sprintf("Delete %s", c.meta.Name),
		Description: c.getOperationDescription("delete"),
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

// HandlerContracts returns the HTTP handler contracts for the core resource.
func (c *zCore[T]) HandlerContracts() []*HandlerContract {
	if len(c.handlerContracts) == 0 {
		c.createHandlerContracts()
	}
	contracts := []*HandlerContract{}
	for _, contract := range c.handlerContracts {
		contracts = append(contracts, contract)
	}
	return contracts
}

// setEmbeds stores validated macro embeds in the core (called during injection)
func (c *zCore[T]) setEmbeds(embeds MacroEmbeds) {
	c.embeds = embeds
}

func (c *zCore[T]) setDatabase(database Database) {
	c.database = database
}

func (c *zCore[T]) setContractMetadata(name, description string) {
	c.contractName = name
	c.contractDescription = description
}

func (c *zCore[T]) ContractName() string {
	return c.contractName
}

func (c *zCore[T]) ContractDescription() string {
	return c.contractDescription
}

// Create handles the business logic for creating a new record
func (c *zCore[T]) Create(ctx Context) (any, error) {
	db := c.database
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	// Get raw JSON from request body
	jsonData, err := ctx.BodyBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Create a new model instance
	var payload T

	// Use scoped deserialization to validate field access permissions for creation
	err = DeserializeWithScopes(ctx, jsonData, &payload, OperationCreate)
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

	zlog.Zlog.Info("Creating a new record",
		zlog.Any("meta", c.meta),
		zlog.Any("payload", payload))

	rows, err := db.Execute(c.macroContracts["create"], payload)
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
	db := c.database
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	id := ctx.PathParam("id")
	if id == "" {
		return nil, fmt.Errorf("ID parameter is required")
	}

	err := c.validator.IsValidID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}

	zlog.Zlog.Info("Retrieving a record",
		zlog.Any("meta", c.meta),
		zlog.String("id", id))

	rows, err := db.Execute(c.macroContracts["select"], map[string]any{"id": id})
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
	db := c.database
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	id := ctx.PathParam("id")
	if id == "" {
		return nil, fmt.Errorf("ID parameter is required")
	}

	err := c.validator.IsValidID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}

	// First, get the existing record
	existing, err := db.Execute(c.macroContracts["select"], map[string]any{"id": id})
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
	jsonData, err := ctx.BodyBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Create a copy of existing record for patching
	payload := eresults[0]

	// Use scoped deserialization to validate field access permissions
	if len(jsonData) > 0 {
		err = DeserializeWithScopes(ctx, jsonData, &payload, OperationUpdate)
		if err != nil {
			return nil, fmt.Errorf("deserialization failed: %w", err)
		}
	}

	err = c.validator.IsValid(payload)
	if err != nil {
		return nil, err // Return validation error as-is
	}

	zlog.Zlog.Info("Updating a record",
		zlog.Any("meta", c.meta),
		zlog.String("id", id),
		zlog.Any("payload", payload))

	rows, err := db.Execute(c.macroContracts["update"], payload)
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
	db := c.database
	if db == nil {
		return fmt.Errorf("database connection is not available")
	}

	id := ctx.PathParam("id")
	if id == "" {
		return fmt.Errorf("ID parameter is required")
	}

	err := c.validator.IsValidID(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	zlog.Zlog.Info("Deleting a record",
		zlog.Any("meta", c.meta),
		zlog.String("id", id))

	_, err = db.Execute(c.macroContracts["delete"], map[string]any{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// Legacy gin handlers removed - use framework-agnostic handlers in Handler[T] instead

