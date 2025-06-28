# Cereal - Validation-Enhanced Serialization

Cereal provides validation-enhanced serialization for JSON, YAML, and TOML formats with built-in scoping and struct validation using `go-playground/validator`.

## Features

- **Automatic Validation**: Validates structs on both marshal and unmarshal operations
- **Multiple Formats**: JSON, YAML, and TOML support with consistent validation
- **Field Scoping**: Permission-based field filtering
- **Pluggable Validation**: Custom validator implementations
- **Rich Error Messages**: Human-readable validation error formatting

## Basic Usage

### Simple Validation

```go
type User struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=0,max=120"`
}

user := User{
    Name:  "John Doe",
    Email: "john@example.com", 
    Age:   30,
}

// Validates automatically during marshal
data, err := cereal.JSON.Marshal(user)
if err != nil {
    // Handle validation or serialization error
    log.Printf("Validation failed: %v", err)
}

// Validates automatically during unmarshal
var result User
err = cereal.JSON.Unmarshal(data, &result)
if err != nil {
    // Handle validation or deserialization error
    log.Printf("Validation failed: %v", err)
}
```

### Manual Validation

```go
// Validate without serializing
err := cereal.Validate(user)
if err != nil {
    // Get formatted error details
    errors := cereal.FormatValidationErrors(err)
    for _, e := range errors {
        fmt.Printf("Field %s: %s\n", e.Field, e.Message)
    }
}
```

### Available Validation Tags

Common validation tags supported by go-playground/validator:

- `required` - Field is required
- `min=n` - Minimum length/value
- `max=n` - Maximum length/value  
- `len=n` - Exact length
- `email` - Valid email format
- `url` - Valid URL format
- `uuid` - Valid UUID format
- `oneof=a b c` - Value must be one of the specified options
- `gt=n` - Greater than
- `gte=n` - Greater than or equal
- `lt=n` - Less than
- `lte=n` - Less than or equal
- `omitempty` - Skip validation if field is empty

### Custom Validation

```go
// Implement custom validator
type MyValidator struct{}

func (v *MyValidator) Validate(s interface{}) error {
    // Custom validation logic
    return nil
}

// Set custom validator globally
cereal.SetValidator(&MyValidator{})

// Or disable validation entirely
cereal.SetValidator(nil)
```

### Error Handling

```go
err := cereal.JSON.Marshal(invalidStruct)
if err != nil {
    // Check if it's a validation error
    if validationErrors := cereal.FormatValidationErrors(err); len(validationErrors) > 0 {
        for _, e := range validationErrors {
            fmt.Printf("Validation error in field '%s': %s (tag: %s, value: %s)\n", 
                e.Field, e.Message, e.Tag, e.Value)
        }
    } else {
        // Other serialization error
        fmt.Printf("Serialization error: %v\n", err)
    }
}
```

### Integration with Scoping

Validation works seamlessly with cereal's permission-based scoping using **redaction instead of omission**:

```go
type SecureData struct {
    PublicField  string `json:"public" validate:"required"`
    PrivateField string `json:"private" scope:"admin" validate:"required,min=10"`
    SSN          string `json:"ssn" scope:"admin" validate:"required,len=11"`
    Email        string `json:"email" scope:"admin" validate:"required,email"`
}

data := SecureData{
    PublicField:  "visible",
    PrivateField: "secret123456",
    SSN:          "123-45-6789",
    Email:        "user@company.com",
}

// Validates redacted fields, then applies scoping with format-aware redaction
jsonData, err := cereal.JSON.Marshal(data, "user") // No admin permission
// Result: {
//   "public": "visible",
//   "private": "[REDACTED]",      // Satisfies min=10 requirement
//   "ssn": "XXX-XX-XXXX",        // Satisfies len=11 requirement  
//   "email": "redacted@example.com" // Satisfies email format requirement
// }
```

## Redaction Policy

When users lack permissions to view scoped fields, cereal uses **intelligent redaction** instead of field omission to prevent validation failures:

### Format-Aware Redaction

- **Email fields** (`validate:"email"`): `redacted@example.com`
- **URL fields** (`validate:"url"`): `https://redacted.example.com`  
- **UUID fields** (`validate:"uuid"`): `00000000-0000-0000-0000-000000000000`
- **Length-specific patterns**:
  - `len=11`: `XXX-XX-XXXX` (SSN format)
  - `len=12`: `XXX-XXX-XXXX` (phone format)
  - `len=16`: `0000000000000000` (credit card format)
  - `len=36`: UUID format with dashes
- **Type-specific formats**:
  - `validate:"alpha"`: Pure alphabetic (e.g., `REDACTED`) 
  - `validate:"alphanum"`: Alphanumeric (e.g., `REDACTED123`)
  - `validate:"numeric"`: Pure numeric (e.g., `000000`)
  - `validate:"json"`: `{"redacted":true}`

### Type-Specific Redaction

- **Strings**: Format-aware or `[REDACTED]` with appropriate length
- **Integers/Floats**: `0` (satisfies common `min=0` constraints)
- **Booleans**: `true` (avoids `required` validation issues)
- **Slices**: Empty slice `[]` (not nil)
- **Maps**: Empty map `{}` (not nil)
- **Pointers**: `nil`

### Why Redaction Instead of Omission?

The old approach of omitting restricted fields broke validation when:

```go
type User struct {
    Name string `json:"name" validate:"required"`
    SSN  string `json:"ssn" scope:"admin" validate:"required,len=11"`
}

// OLD (broken): {"name": "John"} - missing SSN fails validation
// NEW (works): {"name": "John", "ssn": "XXX-XX-XXXX"} - passes validation
```

## Custom Validation System

Cereal supports custom validators that work seamlessly with scoping and redaction:

### Built-in Business Validators

Instead of generic constraints like `len=11`, use semantic validators:

```go
type User struct {
    Name       string `json:"name" validate:"required"`
    SSN        string `json:"ssn" scope:"admin" validate:"ssn"`        // Instead of len=11
    Phone      string `json:"phone" scope:"hr" validate:"phone"`       // Instead of len=12  
    CreditCard string `json:"cc" scope:"finance" validate:"creditcard"` // Includes Luhn validation
    BusinessID string `json:"bid" scope:"admin" validate:"businessid"`  // Custom format
}
```

**Built-in Validators:**
- `ssn`: Social Security Number (XXX-XX-XXXX format)
- `phone`: Phone number (XXX-XXX-XXXX format)
- `creditcard`: Credit card with Luhn algorithm validation
- `businessid`: Alphanumeric business identifier (6-20 chars)

### Custom Validator Registration

```go
// Register custom validator
cereal.RegisterCustomValidator("employee_id", func(ctx context.Context, field reflect.Value, param string) error {
    empID := field.String()
    
    // Accept redacted pattern as valid
    if empID == "EMP-XXXXX" {
        return nil
    }
    
    // Validate format: EMP-12345
    if !strings.HasPrefix(empID, "EMP-") || len(empID) != 9 {
        return fmt.Errorf("employee_id must follow EMP-XXXXX format")
    }
    
    // Could call external validation service here
    return validateWithExternalService(ctx, empID)
})

// Use in struct
type Employee struct {
    Name string `json:"name" validate:"required"`
    ID   string `json:"id" scope:"hr" validate:"employee_id"`
}
```

### Event Handling for External Integration

Since cereal cannot directly use capitan (circular dependency), it provides an event interface that adapters can bridge:

```go
// In an adapter package
cereal.SetValidationEventHandler(func(ctx context.Context, eventType string, data cereal.ValidationEventData) {
    // Bridge to capitan or other external systems
    capitan.Emit(ctx, "validation.executed", "cereal-adapter", data, nil)
})
```

**Event Data Structure:**
```go
type ValidationEventData struct {
    FieldName     string      `json:"field_name"`
    FieldValue    interface{} `json:"field_value"`
    ValidationTag string      `json:"validation_tag"`
    Parameter     string      `json:"parameter,omitempty"`
    StructType    string      `json:"struct_type"`
    Result        string      `json:"result"` // "success", "failure", "error"
    Error         string      `json:"error,omitempty"`
}
```

### Smart Redaction for Custom Validators

Custom validators automatically get appropriate redacted values:

```go
// Original: {"id": "EMP-12345"}
// Redacted: {"id": "EMP-XXXXX"} - still passes employee_id validation
```

The redaction system:
1. Checks for custom validator redaction patterns first
2. Falls back to format-specific patterns (email, uuid, etc.)
3. Uses length-based patterns as last resort

### Advantages Over Generic Validation

**Before (generic):**
```go
type User struct {
    SSN string `json:"ssn" scope:"admin" validate:"required,len=11"`
}
// Issues: len=11 doesn't validate format, redacted value might not satisfy constraint
```

**After (semantic):**
```go
type User struct {
    SSN string `json:"ssn" scope:"admin" validate:"ssn"`
}
// Benefits: validates actual SSN format, redacted "XXX-XX-XXXX" passes validation
```

## Integration Points

Cereal validation integrates automatically with:

- **Universal Data Access**: All universal operations validate data
- **Docula Content**: Document validation on save/load
- **API Handlers**: Request/response validation
- **Configuration**: Config struct validation

## Performance Notes

- Validation occurs on every marshal/unmarshal operation
- Use `omitempty` for optional fields to improve performance
- Consider caching validation rules for frequently used structs
- Set validator to `nil` to disable validation if needed

## Architecture

The validation layer sits between scoping and serialization:

1. **Marshal Flow**: Validate → Apply Scoping → Serialize
2. **Unmarshal Flow**: Deserialize → Apply Scoping → Validate

This ensures data integrity while maintaining security through scoping.