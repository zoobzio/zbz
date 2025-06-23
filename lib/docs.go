package zbz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Docs is an interface for API documentation functionality
type Docs interface {
	AddTag(name, description string)
	AddPath(op *Operation, errorManager ErrorManager)
	AddParameter(param *OpenAPIParameter)
	AddSchema(meta *Meta)

	SpecHandler(ctx *gin.Context)
	ScalarHandler(ctx *gin.Context)
}

// Docs represents the documentation structure for an API
type zDocs struct {
	spec      *OpenAPISpec
	yaml      []byte
	validator Validate
}

// NewDocs creates a new Docs instance
func NewDocs() Docs {
	return &zDocs{
		validator: NewValidate(),
		spec: &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: &OpenAPIInfo{
				Title:       config.Title(),
				Version:     config.Version(),
				Description: config.Description(),
			},
			Components: &OpenAPIComponents{
				Parameters:    make(map[string]*OpenAPIParameter),
				RequestBodies: make(map[string]*OpenAPIRequestBody),
				SecuritySchemes: map[string]*OpenAPISecurityScheme{
					"Bearer": {
						Type:   "http",
						Scheme: "bearer",
					},
				},
				Schemas: make(map[string]*OpenAPISchema),
			},
			Paths: make(map[string]map[string]*OpenAPIPath),
			Tags:  []map[string]string{},
		},
	}
}

// AddTag adds a new tag to the OpenAPI specification
func (d *zDocs) AddTag(name, description string) {
	d.spec.Tags = append(d.spec.Tags, map[string]string{
		"name":        name,
		"description": description,
	})
}

// AddParameter adds a new parameter to the OpenAPI specification
func (d *zDocs) AddParameter(param *OpenAPIParameter) {
	if d.spec.Components.Parameters == nil {
		d.spec.Components.Parameters = make(map[string]*OpenAPIParameter)
	}
	d.spec.Components.Parameters[param.Name] = param
}

// AddBody adds a new request body to the OpenAPI specification
func (d *zDocs) AddBody(name string, body *OpenAPIRequestBody) {
	if d.spec.Components.RequestBodies == nil {
		d.spec.Components.RequestBodies = make(map[string]*OpenAPIRequestBody)
	}
	d.spec.Components.RequestBodies[name] = body
}

// AddPath adds a new path to the OpenAPI specification
func (d *zDocs) AddPath(op *Operation, errorManager ErrorManager) {
	if d.spec.Paths[op.Path] == nil {
		d.spec.Paths[op.Path] = make(map[string]*OpenAPIPath)
	}

	responses := make(map[int]*OpenAPIResponse)
	if op.Response != nil {
		responses[op.Response.Status] = &OpenAPIResponse{
			Description: http.StatusText(op.Response.Status),
		}

		if op.Response.Ref != "" {
			responses[op.Response.Status].Content = &OpenAPIContent{
				ApplicationJSON: &OpenAPIApplicationJSON{
					Schema: &OpenAPISchema{
						Ref: fmt.Sprintf("#/components/schemas/%s", op.Response.Ref),
					},
				},
			}
		}

		if op.Response.Errors != nil {
			for _, status := range op.Response.Errors {
				errorSchema := GetErrorSchema(errorManager, status)
				if errorSchema != nil {
					responses[status] = &OpenAPIResponse{
						Description: http.StatusText(status),
						Content: &OpenAPIContent{
							ApplicationJSON: &OpenAPIApplicationJSON{
								Schema: errorSchema,
							},
						},
					}
				} else {
					responses[status] = &OpenAPIResponse{
						Description: http.StatusText(status),
					}
				}
			}
		}

		if op.Auth {
			unauthorizedSchema := GetErrorSchema(errorManager, http.StatusUnauthorized)
			if unauthorizedSchema != nil {
				responses[http.StatusUnauthorized] = &OpenAPIResponse{
					Description: http.StatusText(http.StatusUnauthorized),
					Content: &OpenAPIContent{
						ApplicationJSON: &OpenAPIApplicationJSON{
							Schema: unauthorizedSchema,
						},
					},
				}
			}
			
			forbiddenSchema := GetErrorSchema(errorManager, http.StatusForbidden)
			if forbiddenSchema != nil {
				responses[http.StatusForbidden] = &OpenAPIResponse{
					Description: http.StatusText(http.StatusForbidden),
					Content: &OpenAPIContent{
						ApplicationJSON: &OpenAPIApplicationJSON{
							Schema: forbiddenSchema,
						},
					},
				}
			}
		}

		internalErrorSchema := GetErrorSchema(errorManager, http.StatusInternalServerError)
		if internalErrorSchema != nil {
			responses[http.StatusInternalServerError] = &OpenAPIResponse{
				Description: http.StatusText(http.StatusInternalServerError),
				Content: &OpenAPIContent{
					ApplicationJSON: &OpenAPIApplicationJSON{
						Schema: internalErrorSchema,
					},
				},
			}
		}
	}

	path := &OpenAPIPath{
		Summary:     op.Name,
		Description: op.Description,
		OperationId: op.Name,
		Tags:        []string{op.Tag},
		Parameters:  []*OpenAPIRef{},
		Responses:   responses,
	}

	if op.Auth {
		path.Security = []map[string][]string{
			{"Bearer": {}},
		}
	}

	if op.Parameters != nil {
		for _, param := range op.Parameters {
			path.Parameters = append(path.Parameters, &OpenAPIRef{
				Ref: fmt.Sprintf("#/components/parameters/%s", param),
			})
		}
	}

	if op.RequestBody != "" {
		path.RequestBody = &OpenAPIRequestBody{
			Ref: fmt.Sprintf("#/components/requestBodies/%s", op.RequestBody),
		}
	}

	d.spec.Paths[op.Path][strings.ToLower(op.Method)] = path
}

// AddSchema adds a new schema to the OpenAPI specification
func (d *zDocs) AddSchema(meta *Meta) {
	example, err := json.Marshal(meta.ExampleValue)
	if err != nil {
		Log.Fatal("Failed to marshal example for model", zap.Any("meta", meta), zap.Error(err))
	}

	schema := &OpenAPISchema{
		Type:        "object",
		Description: meta.Description,
		Example:     string(example),
		Properties:  map[string]*OpenAPISchema{},
		Required:    []string{},
	}

	createPayload := &OpenAPISchema{
		Type:        "object",
		Description: fmt.Sprintf("`Create` payload for %s", meta.Name),
		Properties:  map[string]*OpenAPISchema{},
		Required:    []string{},
	}

	updatePayload := &OpenAPISchema{
		Type:        "object",
		Description: fmt.Sprintf("`Update` payload for %s", meta.Name),
		Properties:  map[string]*OpenAPISchema{},
	}

	for _, field := range meta.FieldMetadata {
		var ft string
		var ff string
		switch field.GoType {
		case "int", "int32":
			ft = "integer"
			ff = "int32"
		case "int64":
			ft = "integer"
			ff = "int64"
		case "float32":
			ft = "number"
			ff = "float"
		case "float64":
			ft = "number"
			ff = "double"
		case "string":
			ft = "string"
			if strings.Contains(field.ValidationRules, "email") {
				ff = "email"
			} else if strings.Contains(field.ValidationRules, "uuid") {
				ff = "uuid"
			} else if strings.Contains(field.ValidationRules, "url") {
				ff = "uri"
			}
		case "bool":
			ft = "boolean"
		case "time.Time":
			ft = "string"
			ff = "date-time"
		case "[]byte":
			ft = "string"
			ff = "byte"
		}

		fieldSchema := &OpenAPISchema{
			Type:        ft,
			Format:      ff,
			Description: field.Description,
			Example:     field.ExampleValue,
		}
		
		// Handle scoped fields - mark as optional for SDK generation
		isScoped := field.ScopeRules != ""
		originallyRequired := field.IsRequired
		
		if isScoped {
			// Add scope extensions for SDK generators to understand field permissions
			if fieldSchema.Extensions == nil {
				fieldSchema.Extensions = make(map[string]any)
			}
			fieldSchema.Extensions["x-scope"] = field.ScopeRules
			fieldSchema.Extensions["x-scope-required"] = originallyRequired
			fieldSchema.Extensions["x-scope-description"] = "This field may be undefined if the user lacks the required permissions"
		}
		
		// Apply validation constraints from struct tags using our validation system
		// Parse validation rules using validator, then generate OpenAPI constraints
		rules := d.validator.ParseValidationRules(field.ValidationRules)
		parsedRules := ParsedValidationRules{
			Rules:     rules,
			FieldType: ft,
			FieldName: field.JSONFieldName,
		}
		constraints := d.getOpenAPIConstraints(parsedRules)
		for key, value := range constraints {
			switch key {
			case "minLength":
				if v, ok := value.(int); ok {
					fieldSchema.MinLength = &v
				}
			case "maxLength":
				if v, ok := value.(int); ok {
					fieldSchema.MaxLength = &v
				}
			case "minimum":
				if v, ok := value.(float64); ok {
					fieldSchema.Minimum = &v
				}
			case "maximum":
				if v, ok := value.(float64); ok {
					fieldSchema.Maximum = &v
				}
			case "exclusiveMinimum":
				if v, ok := value.(float64); ok {
					fieldSchema.ExclusiveMinimum = &v
				}
			case "exclusiveMaximum":
				if v, ok := value.(float64); ok {
					fieldSchema.ExclusiveMaximum = &v
				}
			case "pattern":
				if v, ok := value.(string); ok {
					fieldSchema.Pattern = v
				}
			case "format":
				if v, ok := value.(string); ok {
					fieldSchema.Format = v
				}
			case "enum":
				if v, ok := value.([]any); ok {
					fieldSchema.Enum = v
				}
			}
		}
		
		schema.Properties[field.JSONFieldName] = fieldSchema

		// For response schemas, scoped fields are always optional (even if DB requires them)
		// since they might be filtered out based on user permissions
		if field.IsRequired && !isScoped {
			schema.Required = append(schema.Required, field.JSONFieldName)
		}

		// TODO we can handle edit permissions here using the value of field.EditType
		if field.EditType != "" {
			payload := &OpenAPISchema{
				Type:        ft,
				Format:      ff,
				Description: field.Description,
				Example:     field.ExampleValue,
			}
			
			// Apply validation constraints to payload schemas too using our validation system
			payloadConstraints := d.getOpenAPIConstraints(parsedRules)
			for key, value := range payloadConstraints {
				switch key {
				case "minLength":
					if v, ok := value.(int); ok {
						payload.MinLength = &v
					}
				case "maxLength":
					if v, ok := value.(int); ok {
						payload.MaxLength = &v
					}
				case "minimum":
					if v, ok := value.(float64); ok {
						payload.Minimum = &v
					}
				case "maximum":
					if v, ok := value.(float64); ok {
						payload.Maximum = &v
					}
				case "exclusiveMinimum":
					if v, ok := value.(float64); ok {
						payload.ExclusiveMinimum = &v
					}
				case "exclusiveMaximum":
					if v, ok := value.(float64); ok {
						payload.ExclusiveMaximum = &v
					}
				case "pattern":
					if v, ok := value.(string); ok {
						payload.Pattern = v
					}
				case "format":
					if v, ok := value.(string); ok {
						payload.Format = v
					}
				case "enum":
					if v, ok := value.([]any); ok {
						payload.Enum = v
					}
				}
			}

			// Handle scoped fields in payloads
			if isScoped {
				// Add scope extensions to payload fields too
				if payload.Extensions == nil {
					payload.Extensions = make(map[string]any)
				}
				payload.Extensions["x-scope"] = field.ScopeRules
				payload.Extensions["x-scope-required"] = originallyRequired
				payload.Extensions["x-scope-description"] = "This field requires specific permissions to modify"
			}
			
			createPayload.Properties[field.JSONFieldName] = payload
			updatePayload.Properties[field.JSONFieldName] = payload

			// For create payloads, scoped fields that are required in DB still need to be provided
			// (the scoping happens at runtime during deserialization)
			if field.IsRequired {
				createPayload.Required = append(createPayload.Required, field.JSONFieldName)
			}
			// Update payloads are always optional since they're PATCH-style
		}
	}

	d.spec.Components.Schemas[meta.Name] = schema
	
	// Store request bodies in the proper section
	d.spec.Components.RequestBodies[fmt.Sprintf("Create%sPayload", meta.Name)] = &OpenAPIRequestBody{
		Description: fmt.Sprintf("Request body for creating a %s", meta.Name),
		Required:    true,
		Content: &OpenAPIContent{
			ApplicationJSON: &OpenAPIApplicationJSON{
				Schema: createPayload,
			},
		},
	}
	
	d.spec.Components.RequestBodies[fmt.Sprintf("Update%sPayload", meta.Name)] = &OpenAPIRequestBody{
		Description: fmt.Sprintf("Request body for updating a %s", meta.Name),
		Required:    true,
		Content: &OpenAPIContent{
			ApplicationJSON: &OpenAPIApplicationJSON{
				Schema: updatePayload,
			},
		},
	}
}

// getOpenAPIConstraints converts parsed validation rules to OpenAPI constraints
func (d *zDocs) getOpenAPIConstraints(parsed ParsedValidationRules) map[string]any {
	constraints := make(map[string]any)

	for _, rule := range parsed.Rules {
		switch rule.Name {
		case "min":
			if len(rule.Params) > 0 {
				if parsed.FieldType == "string" {
					if minLen, err := strconv.Atoi(rule.Params[0]); err == nil {
						constraints["minLength"] = minLen
					}
				} else if parsed.FieldType == "integer" || parsed.FieldType == "number" {
					if minVal, err := strconv.ParseFloat(rule.Params[0], 64); err == nil {
						constraints["minimum"] = minVal
					}
				}
			}

		case "max":
			if len(rule.Params) > 0 {
				if parsed.FieldType == "string" {
					if maxLen, err := strconv.Atoi(rule.Params[0]); err == nil {
						constraints["maxLength"] = maxLen
					}
				} else if parsed.FieldType == "integer" || parsed.FieldType == "number" {
					if maxVal, err := strconv.ParseFloat(rule.Params[0], 64); err == nil {
						constraints["maximum"] = maxVal
					}
				}
			}

		case "gt":
			if len(rule.Params) > 0 {
				if minVal, err := strconv.ParseFloat(rule.Params[0], 64); err == nil {
					constraints["exclusiveMinimum"] = minVal
				}
			}

		case "gte":
			if len(rule.Params) > 0 {
				if minVal, err := strconv.ParseFloat(rule.Params[0], 64); err == nil {
					constraints["minimum"] = minVal
				}
			}

		case "lt":
			if len(rule.Params) > 0 {
				if maxVal, err := strconv.ParseFloat(rule.Params[0], 64); err == nil {
					constraints["exclusiveMaximum"] = maxVal
				}
			}

		case "lte":
			if len(rule.Params) > 0 {
				if maxVal, err := strconv.ParseFloat(rule.Params[0], 64); err == nil {
					constraints["maximum"] = maxVal
				}
			}

		case "len":
			if len(rule.Params) > 0 && parsed.FieldType == "string" {
				if length, err := strconv.Atoi(rule.Params[0]); err == nil {
					constraints["minLength"] = length
					constraints["maxLength"] = length
				}
			}

		case "oneof":
			if len(rule.Params) > 0 {
				enum := make([]any, len(rule.Params))
				for i, param := range rule.Params {
					enum[i] = param
				}
				constraints["enum"] = enum
			}

		case "regexp":
			if len(rule.Params) > 0 {
				constraints["pattern"] = rule.Params[0]
			}

		case "email":
			if parsed.FieldType == "string" {
				constraints["format"] = "email"
			}

		case "url":
			if parsed.FieldType == "string" {
				constraints["format"] = "uri"
			}

		case "uuid", "uuid4":
			if parsed.FieldType == "string" {
				constraints["format"] = "uuid"
			}

		case "alpha":
			if parsed.FieldType == "string" {
				constraints["pattern"] = "^[a-zA-Z]+$"
			}

		case "alphanum":
			if parsed.FieldType == "string" {
				constraints["pattern"] = "^[a-zA-Z0-9]+$"
			}

		case "numeric":
			if parsed.FieldType == "string" {
				constraints["pattern"] = "^[0-9]+$"
			}
		}
	}

	return constraints
}

// GenerateYAML generates the OpenAPI specification in YAML format
func (d *zDocs) GenerateYAML(spec *OpenAPISpec) {
	data, err := yaml.Marshal(spec)
	if err != nil {
		Log.Fatal("failed to marshal OpenAPI spec to YAML", zap.Error(err))
	}
	d.yaml = data
}

// SpecHandler generates and returns the OpenAPI specification in YAML format
// Shows full schema for admins, scoped schema for regular users
func (d *zDocs) SpecHandler(ctx *gin.Context) {
	// Get user permissions from context
	permissions, exists := ctx.Get("permissions")
	if !exists {
		permissions = []string{} // No permissions
	}

	userPerms, ok := permissions.([]string)
	if !ok {
		userPerms = []string{}
	}

	// Check if user has admin privileges (can see everything)
	if d.hasAdminAccess(userPerms) {
		// Return full static schema for admins
		if d.yaml == nil {
			d.GenerateYAML(d.spec)
		}
		ctx.Data(http.StatusOK, "text/yaml; charset=utf-8", d.yaml)
	} else {
		// Return scoped schema for regular users
		scopedSpec := d.filterSpecByPermissions(d.spec, userPerms)
		
		data, err := yaml.Marshal(scopedSpec)
		if err != nil {
			Log.Error("Failed to marshal scoped OpenAPI spec to YAML", zap.Error(err))
			ctx.Status(http.StatusInternalServerError)
			return
		}
		
		ctx.Data(http.StatusOK, "text/yaml; charset=utf-8", data)
	}
}

// ScalarHandler renders a documentation site built using Scalar: https://scalar.com/
// Shows full docs for admins, scoped docs for regular users
func (d *zDocs) ScalarHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "scalar.tmpl", gin.H{
		"title":   config.Title(),
		"openapi": "/openapi", // Always points to the same endpoint, but it returns different content based on user permissions
	})
}

// hasAdminAccess checks if the user has admin-level access to see everything
func (d *zDocs) hasAdminAccess(userPermissions []string) bool {
	// Check for admin scopes that should see everything
	adminScopes := []string{
		"admin",
		"admin:all", 
		"read:admin",
		"admin:users",
		"read:admin:users",
	}
	
	for _, userPerm := range userPermissions {
		for _, adminScope := range adminScopes {
			if userPerm == adminScope {
				return true
			}
		}
	}
	
	return false
}

// filterSpecByPermissions creates a copy of the OpenAPI spec filtered by user permissions
func (d *zDocs) filterSpecByPermissions(spec *OpenAPISpec, userPermissions []string) *OpenAPISpec {
	// Create a deep copy of the spec
	filteredSpec := &OpenAPISpec{
		OpenAPI: spec.OpenAPI,
		Info:    spec.Info,
		Components: &OpenAPIComponents{
			Parameters:      spec.Components.Parameters,
			SecuritySchemes: spec.Components.SecuritySchemes,
			Schemas:         make(map[string]*OpenAPISchema),
			RequestBodies:   make(map[string]*OpenAPIRequestBody),
		},
		Paths: spec.Paths, // Paths don't need filtering, only schemas
		Tags:  spec.Tags,
	}

	// Filter schemas by removing fields the user can't access
	for name, schema := range spec.Components.Schemas {
		filteredSpec.Components.Schemas[name] = d.filterSchemaByPermissions(schema, userPermissions)
	}
	
	// Filter request bodies
	for name, reqBody := range spec.Components.RequestBodies {
		filteredSpec.Components.RequestBodies[name] = d.filterRequestBodyByPermissions(reqBody, userPermissions)
	}

	return filteredSpec
}

// filterSchemaByPermissions removes fields from a schema that the user cannot access
func (d *zDocs) filterSchemaByPermissions(schema *OpenAPISchema, userPermissions []string) *OpenAPISchema {
	if schema == nil || schema.Properties == nil {
		return schema
	}

	// Create a copy of the schema
	filteredSchema := &OpenAPISchema{
		Type:        schema.Type,
		Description: schema.Description,
		Example:     schema.Example,
		Properties:  make(map[string]*OpenAPISchema),
		Required:    []string{},
	}

	// Check each property for scope requirements
	for fieldName, fieldSchema := range schema.Properties {
		if d.canUserAccessField(fieldSchema, userPermissions, "read") {
			filteredSchema.Properties[fieldName] = fieldSchema
			
			// Only add to required if user can access the field
			for _, reqField := range schema.Required {
				if reqField == fieldName {
					filteredSchema.Required = append(filteredSchema.Required, fieldName)
					break
				}
			}
		}
		// If user can't access field, it's simply omitted from the filtered schema
	}

	return filteredSchema
}

// filterRequestBodyByPermissions filters request body schemas based on user permissions
func (d *zDocs) filterRequestBodyByPermissions(reqBody *OpenAPIRequestBody, userPermissions []string) *OpenAPIRequestBody {
	if reqBody == nil || reqBody.Content == nil || reqBody.Content.ApplicationJSON == nil {
		return reqBody
	}

	// Create a copy and filter the schema
	filteredReqBody := &OpenAPIRequestBody{
		Description: reqBody.Description,
		Required:    reqBody.Required,
		Ref:         reqBody.Ref,
		Content: &OpenAPIContent{
			ApplicationJSON: &OpenAPIApplicationJSON{
				Schema: d.filterRequestBodySchemaByPermissions(reqBody.Content.ApplicationJSON.Schema, userPermissions),
			},
		},
	}

	return filteredReqBody
}

// filterRequestBodySchemaByPermissions filters request body schemas for write permissions
func (d *zDocs) filterRequestBodySchemaByPermissions(schema *OpenAPISchema, userPermissions []string) *OpenAPISchema {
	if schema == nil || schema.Properties == nil {
		return schema
	}

	// Create a copy of the schema
	filteredSchema := &OpenAPISchema{
		Type:        schema.Type,
		Description: schema.Description,
		Properties:  make(map[string]*OpenAPISchema),
		Required:    []string{},
	}

	// Check each property for write scope requirements
	for fieldName, fieldSchema := range schema.Properties {
		if d.canUserAccessField(fieldSchema, userPermissions, "write") {
			filteredSchema.Properties[fieldName] = fieldSchema
			
			// Only add to required if user can write the field
			for _, reqField := range schema.Required {
				if reqField == fieldName {
					filteredSchema.Required = append(filteredSchema.Required, fieldName)
					break
				}
			}
		}
	}

	return filteredSchema
}

// canUserAccessField checks if a user has the required permissions to access a field
func (d *zDocs) canUserAccessField(fieldSchema *OpenAPISchema, userPermissions []string, operation string) bool {
	if fieldSchema.Extensions == nil {
		return true // No scope restrictions
	}

	scopeValue, exists := fieldSchema.Extensions["x-scope"]
	if !exists {
		return true // No scope restrictions
	}

	scopeStr, ok := scopeValue.(string)
	if !ok {
		return true // Invalid scope format, allow access
	}

	// Parse scope requirements similar to cereal.go
	scopes := strings.Split(scopeStr, ",")
	var requiredPerms []string
	
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}

		// Check if this scope matches the operation we're checking
		if operation == "read" {
			if strings.HasPrefix(scope, "read:") || (!strings.Contains(scope, ":write") && !strings.Contains(scope, ":create") && !strings.Contains(scope, ":update")) {
				perm := strings.TrimPrefix(scope, "read:")
				requiredPerms = append(requiredPerms, perm)
			}
		} else if operation == "write" {
			if strings.Contains(scope, ":write") || strings.Contains(scope, ":create") || strings.Contains(scope, ":update") {
				requiredPerms = append(requiredPerms, scope)
			}
		}
	}

	// Check if user has any of the required permissions
	if len(requiredPerms) == 0 {
		return true // No specific permissions required
	}

	for _, userPerm := range userPermissions {
		for _, reqPerm := range requiredPerms {
			if userPerm == reqPerm {
				return true
			}
		}
	}

	return false
}

